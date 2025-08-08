package parser

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrInvalidQualifiedTypeName  = errors.New("invalid qualified type name")
	ErrPackageAliasNotFound      = errors.New("package alias not found in import map")
	ErrFailedToLoadPackage       = errors.New("failed to load package")
	ErrTypeNotFoundInPackage     = errors.New("type not found in package")
	ErrInvalidImportPath         = errors.New("invalid import path")
	ErrCircularPackageDependency = errors.New("circular package dependency detected")
	ErrPackageLoadTimeout        = errors.New("package load timeout")
	ErrQualifiedTypeIsNil        = errors.New("qualified type is nil")
	ErrQualifiedTypeEmptyName    = errors.New("qualified type has empty type name")
	ErrEmptyPackageAlias         = errors.New("qualified type has empty package alias for non-local type")
	ErrEmptyImportPath           = errors.New("qualified type has empty import path for non-local type")
)

// QualifiedType represents a type reference that may come from an external package.
type QualifiedType struct {
	PackageAlias string `json:"package_alias"` // "pkg" in "pkg.User"
	TypeName     string `json:"type_name"`     // "User" in "pkg.User"
	ImportPath   string `json:"import_path"`   // resolved from imports map
	IsLocal      bool   `json:"is_local"`      // true if no package prefix (local type)
}

// NewQualifiedType creates a new qualified type with validation.
func NewQualifiedType(packageAlias, typeName, importPath string, isLocal bool) (*QualifiedType, error) {
	if typeName == "" {
		return nil, fmt.Errorf("%w: type name cannot be empty", ErrInvalidQualifiedTypeName)
	}

	if !isLocal && packageAlias == "" {
		return nil, fmt.Errorf("%w: package alias cannot be empty for non-local types", ErrInvalidQualifiedTypeName)
	}

	if !isLocal && importPath == "" {
		return nil, fmt.Errorf("%w: import path cannot be empty for non-local types", ErrInvalidImportPath)
	}

	return &QualifiedType{
		PackageAlias: packageAlias,
		TypeName:     typeName,
		ImportPath:   importPath,
		IsLocal:      isLocal,
	}, nil
}

// String returns the string representation of the qualified type.
func (qt *QualifiedType) String() string {
	if qt.IsLocal {
		return qt.TypeName
	}
	return qt.PackageAlias + "." + qt.TypeName
}

// FullName returns the fully qualified name including import path.
func (qt *QualifiedType) FullName() string {
	if qt.IsLocal {
		return qt.TypeName
	}
	return qt.ImportPath + "." + qt.TypeName
}

// PackageLoadCache provides thread-safe caching for loaded packages.
type PackageLoadCache struct {
	cache map[string]*packages.Package
	mutex sync.RWMutex
}

// NewPackageLoadCache creates a new package load cache.
func NewPackageLoadCache() *PackageLoadCache {
	return &PackageLoadCache{
		cache: make(map[string]*packages.Package),
	}
}

// Get retrieves a cached package by import path.
func (plc *PackageLoadCache) Get(importPath string) (*packages.Package, bool) {
	plc.mutex.RLock()
	defer plc.mutex.RUnlock()
	pkg, exists := plc.cache[importPath]
	return pkg, exists
}

// Set stores a package in the cache.
func (plc *PackageLoadCache) Set(importPath string, pkg *packages.Package) {
	plc.mutex.Lock()
	defer plc.mutex.Unlock()
	plc.cache[importPath] = pkg
}

// CrossPackageResolver provides cross-package type resolution capabilities.
// It extends the type instantiation system to support type arguments from external packages.
type CrossPackageResolver struct {
	packageLoader *PackageLoader
	importMap     map[string]string // alias -> import path
	cache         *PackageLoadCache
	logger        *zap.Logger

	// Configuration
	timeout      time.Duration
	maxCacheSize int

	// Safety mechanisms
	loadingStack []string // Track loading chain for cycle detection
	loadingMutex sync.Mutex
}

// CrossPackageResolverConfig configures the behavior of CrossPackageResolver.
type CrossPackageResolverConfig struct {
	Timeout      time.Duration `json:"timeout"`
	MaxCacheSize int           `json:"max_cache_size"`
	MaxWorkers   int           `json:"max_workers"`
}

// NewCrossPackageResolverConfig creates a default configuration.
func NewCrossPackageResolverConfig() *CrossPackageResolverConfig {
	return &CrossPackageResolverConfig{
		Timeout:      30 * time.Second,
		MaxCacheSize: 100,
		MaxWorkers:   5,
	}
}

// NewCrossPackageResolver creates a new cross-package resolver with the given dependencies.
func NewCrossPackageResolver(
	packageLoader *PackageLoader,
	importMap map[string]string,
	logger *zap.Logger,
) *CrossPackageResolver {
	return NewCrossPackageResolverWithConfig(
		packageLoader,
		importMap,
		logger,
		NewCrossPackageResolverConfig(),
	)
}

// NewCrossPackageResolverWithConfig creates a new cross-package resolver with custom configuration.
func NewCrossPackageResolverWithConfig(
	packageLoader *PackageLoader,
	importMap map[string]string,
	logger *zap.Logger,
	config *CrossPackageResolverConfig,
) *CrossPackageResolver {
	if packageLoader == nil {
		packageLoader = NewPackageLoader(config.MaxWorkers, config.Timeout)
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	if importMap == nil {
		importMap = make(map[string]string)
	}

	return &CrossPackageResolver{
		packageLoader: packageLoader,
		importMap:     copyImportMap(importMap),
		cache:         NewPackageLoadCache(),
		logger:        logger,
		timeout:       config.Timeout,
		maxCacheSize:  config.MaxCacheSize,
		loadingStack:  make([]string, 0),
	}
}

// copyImportMap creates a defensive copy of the import map.
func copyImportMap(importMap map[string]string) map[string]string {
	if importMap == nil {
		return make(map[string]string)
	}

	cp := make(map[string]string, len(importMap))
	for k, v := range importMap {
		cp[k] = v
	}
	return cp
}

// ParseTypeArguments supports syntax like: "TypeMapper[pkg.User,dto.UserDTO]" and "Converter[User,UserDTO]".
func (cpr *CrossPackageResolver) ParseTypeArguments(
	ctx context.Context,
	typeSpec string,
) ([]*QualifiedType, error) {
	// Regular expression to parse generic type syntax
	// Matches: TypeName[Type1,Type2,pkg.Type3]
	genericTypeRegex := regexp.MustCompile(`^(\w+)\[([^\]]+)\]$`)

	matches := genericTypeRegex.FindStringSubmatch(strings.TrimSpace(typeSpec))
	if len(matches) != 3 {
		// Not a generic type specification, treat as single type
		return cpr.parseSimpleType(typeSpec)
	}

	// Extract type arguments from the bracket expression
	typeArgsStr := matches[2]
	typeArgStrs := cpr.splitTypeArguments(typeArgsStr)

	qualifiedTypes := make([]*QualifiedType, 0, len(typeArgStrs))

	for _, typeArgStr := range typeArgStrs {
		qualifiedType, err := cpr.parseQualifiedTypeName(strings.TrimSpace(typeArgStr))
		if err != nil {
			return nil, fmt.Errorf("failed to parse type argument '%s': %w", typeArgStr, err)
		}
		qualifiedTypes = append(qualifiedTypes, qualifiedType)
	}

	cpr.logger.Debug("parsed type arguments from type specification",
		zap.String("type_spec", typeSpec),
		zap.Int("type_arg_count", len(qualifiedTypes)))

	return qualifiedTypes, nil
}

// parseSimpleType parses a non-generic type specification.
func (cpr *CrossPackageResolver) parseSimpleType(typeSpec string) ([]*QualifiedType, error) {
	qualifiedType, err := cpr.parseQualifiedTypeName(strings.TrimSpace(typeSpec))
	if err != nil {
		return nil, fmt.Errorf("failed to parse type specification '%s': %w", typeSpec, err)
	}
	return []*QualifiedType{qualifiedType}, nil
}

// Handles cases like: "User,pkg.Type,Generic[T,U]".
func (cpr *CrossPackageResolver) splitTypeArguments(typeArgsStr string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, char := range typeArgsStr {
		switch char {
		case '[':
			depth++
			current.WriteRune(char)
		case ']':
			depth--
			current.WriteRune(char)
		case ',':
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Add the last argument
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseQualifiedTypeName parses a qualified type name (e.g., "pkg.User" or "User").
func (cpr *CrossPackageResolver) parseQualifiedTypeName(typeName string) (*QualifiedType, error) {
	// Check for qualified name (package.Type)
	if parts := strings.Split(typeName, "."); 1 < len(parts) {
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: too many dots in type name '%s'", ErrInvalidQualifiedTypeName, typeName)
		}

		packageAlias := parts[0]
		typeNamePart := parts[1]

		// Resolve import path
		importPath, exists := cpr.importMap[packageAlias]
		if !exists {
			return nil, fmt.Errorf("%w: '%s'", ErrPackageAliasNotFound, packageAlias)
		}

		return NewQualifiedType(packageAlias, typeNamePart, importPath, false)
	}

	// Local type (no package prefix)
	return NewQualifiedType("", typeName, "", true)
}

// ResolveType resolves a qualified type to a domain.Type by loading the package if necessary.
func (cpr *CrossPackageResolver) ResolveType(
	ctx context.Context,
	qualifiedType *QualifiedType,
) (domain.Type, error) {
	if qualifiedType.IsLocal {
		// For local types, create a basic type
		// In a full implementation, this would resolve from the current package
		return domain.NewBasicType(qualifiedType.TypeName, getReflectKind(qualifiedType.TypeName)), nil
	}

	// Load the external package
	pkg, err := cpr.loadPackage(ctx, qualifiedType.ImportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package '%s': %w", qualifiedType.ImportPath, err)
	}

	// Find the type in the package
	obj := pkg.Types.Scope().Lookup(qualifiedType.TypeName)
	if obj == nil {
		return nil, fmt.Errorf("%w: '%s' in package '%s'",
			ErrTypeNotFoundInPackage, qualifiedType.TypeName, qualifiedType.ImportPath)
	}

	// Convert to domain type
	domainType := cpr.convertTodomainType(obj, qualifiedType)

	cpr.logger.Debug("resolved cross-package type",
		zap.String("qualified_name", qualifiedType.String()),
		zap.String("import_path", qualifiedType.ImportPath),
		zap.String("domain_type", domainType.String()))

	return domainType, nil
}

// loadPackage loads a package by import path with caching and cycle detection.
func (cpr *CrossPackageResolver) loadPackage(ctx context.Context, importPath string) (*packages.Package, error) {
	// Check cache first
	if pkg, exists := cpr.cache.Get(importPath); exists {
		cpr.logger.Debug("package cache hit", zap.String("import_path", importPath))
		return pkg, nil
	}

	// Check for circular dependencies
	if err := cpr.checkCircularDependency(importPath); err != nil {
		return nil, err
	}

	// Add to loading stack
	cpr.loadingMutex.Lock()
	cpr.loadingStack = append(cpr.loadingStack, importPath)
	cpr.loadingMutex.Unlock()

	defer func() {
		// Remove from loading stack
		cpr.loadingMutex.Lock()
		if 0 < len(cpr.loadingStack) {
			cpr.loadingStack = cpr.loadingStack[:len(cpr.loadingStack)-1]
		}
		cpr.loadingMutex.Unlock()
	}()

	// Create timeout context
	loadCtx, cancel := context.WithTimeout(ctx, cpr.timeout)
	defer cancel()

	cpr.logger.Debug("loading package", zap.String("import_path", importPath))

	// Load package using packages.Load
	config := &packages.Config{
		Context: loadCtx,
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedTypesInfo,
	}

	pkgs, err := packages.Load(config, importPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %s", ErrFailedToLoadPackage, importPath, err.Error())
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%w: no packages found for path '%s'", ErrFailedToLoadPackage, importPath)
	}

	pkg := pkgs[0]
	if 0 < len(pkg.Errors) {
		var errorMsgs []string
		for _, pkgErr := range pkg.Errors {
			errorMsgs = append(errorMsgs, pkgErr.Error())
		}
		return nil, fmt.Errorf("%w: %s: %s", ErrFailedToLoadPackage, importPath, strings.Join(errorMsgs, "; "))
	}

	// Cache the loaded package
	cpr.cache.Set(importPath, pkg)

	cpr.logger.Info("successfully loaded package",
		zap.String("import_path", importPath),
		zap.String("package_name", pkg.Name))

	return pkg, nil
}

// checkCircularDependency checks for circular package loading dependencies.
func (cpr *CrossPackageResolver) checkCircularDependency(importPath string) error {
	cpr.loadingMutex.Lock()
	defer cpr.loadingMutex.Unlock()

	for _, loading := range cpr.loadingStack {
		if loading == importPath {
			return fmt.Errorf("%w: %s -> %s", ErrCircularPackageDependency,
				strings.Join(cpr.loadingStack, " -> "), importPath)
		}
	}

	return nil
}

// convertTodomainType converts a types.Object to a domain.Type.
func (cpr *CrossPackageResolver) convertTodomainType(obj types.Object, qualifiedType *QualifiedType) domain.Type {
	switch t := obj.Type().(type) {
	case *types.Named:
		// Handle named types (structs, interfaces, etc.)
		underlying := t.Underlying()
		kind := getReflectKindFromType(underlying)

		// Create a basic type with the qualified name
		return domain.NewBasicType(qualifiedType.String(), kind)

	case *types.Interface:
		// Handle interface types
		return domain.NewBasicType(qualifiedType.String(), getReflectKind("interface"))

	case *types.Struct:
		// Handle struct types
		return domain.NewBasicType(qualifiedType.String(), getReflectKind("struct"))

	default:
		// For other types, create a basic type
		kind := getReflectKindFromType(t)
		return domain.NewBasicType(qualifiedType.String(), kind)
	}
}

// UpdateImportMap updates the import map with new alias -> import path mappings.
func (cpr *CrossPackageResolver) UpdateImportMap(newImports map[string]string) {
	cpr.loadingMutex.Lock()
	defer cpr.loadingMutex.Unlock()

	for alias, importPath := range newImports {
		// Validate import path
		if err := cpr.validateImportPath(importPath); err != nil {
			cpr.logger.Warn("invalid import path skipped",
				zap.String("alias", alias),
				zap.String("import_path", importPath),
				zap.Error(err))
			continue
		}

		cpr.importMap[alias] = importPath
		cpr.logger.Debug("updated import mapping",
			zap.String("alias", alias),
			zap.String("import_path", importPath))
	}
}

// validateImportPath validates that an import path is valid.
func (cpr *CrossPackageResolver) validateImportPath(importPath string) error {
	if importPath == "" {
		return fmt.Errorf("%w: empty import path", ErrInvalidImportPath)
	}

	// Basic validation - in a full implementation, this would be more thorough
	if strings.Contains(importPath, " ") {
		return fmt.Errorf("%w: import path contains spaces: '%s'", ErrInvalidImportPath, importPath)
	}

	return nil
}

// GetImportMap returns a copy of the current import map.
func (cpr *CrossPackageResolver) GetImportMap() map[string]string {
	cpr.loadingMutex.Lock()
	defer cpr.loadingMutex.Unlock()
	return copyImportMap(cpr.importMap)
}

// ClearCache clears the package cache.
func (cpr *CrossPackageResolver) ClearCache() {
	cpr.cache = NewPackageLoadCache()
	cpr.logger.Debug("cross-package resolver cache cleared")
}

// GetCacheStats returns cache statistics.
func (cpr *CrossPackageResolver) GetCacheStats() (hits, misses int) {
	// This is a simplified implementation
	// In a full implementation, this would track actual hit/miss statistics
	return len(cpr.cache.cache), 0
}

// Helper functions

// getReflectKind returns a reflect.Kind for common type names.
// This is a simplified implementation for basic type mapping.
func getReflectKind(typeName string) reflect.Kind {
	switch strings.ToLower(typeName) {
	case "bool":
		return reflect.Bool
	case "int":
		return reflect.Int
	case "int8":
		return reflect.Int8
	case "int16":
		return reflect.Int16
	case "int32":
		return reflect.Int32
	case "int64":
		return reflect.Int64
	case "uint":
		return reflect.Uint
	case "uint8":
		return reflect.Uint8
	case "uint16":
		return reflect.Uint16
	case "uint32":
		return reflect.Uint32
	case "uint64":
		return reflect.Uint64
	case "float32":
		return reflect.Float32
	case "float64":
		return reflect.Float64
	case "string":
		return reflect.String
	case "interface":
		return reflect.Interface
	case "struct":
		return reflect.Struct
	default:
		return reflect.Struct // Default for custom types
	}
}

// getReflectKindFromType returns a reflect.Kind value from a types.Type.
func getReflectKindFromType(t types.Type) reflect.Kind {
	switch t := t.(type) {
	case *types.Basic:
		return getReflectKind(t.Name())
	case *types.Struct:
		return reflect.Struct
	case *types.Interface:
		return reflect.Interface
	case *types.Pointer:
		return reflect.Ptr
	case *types.Slice:
		return reflect.Slice
	case *types.Array:
		return reflect.Array
	case *types.Map:
		return reflect.Map
	case *types.Chan:
		return reflect.Chan
	case *types.Signature:
		return reflect.Func
	default:
		return reflect.Struct // Default
	}
}

// ValidateQualifiedTypes validates a slice of qualified types.
func (cpr *CrossPackageResolver) ValidateQualifiedTypes(qualifiedTypes []*QualifiedType) error {
	for i, qt := range qualifiedTypes {
		if qt == nil {
			return fmt.Errorf("%w at index %d", ErrQualifiedTypeIsNil, i)
		}

		if qt.TypeName == "" {
			return fmt.Errorf("%w at index %d", ErrQualifiedTypeEmptyName, i)
		}

		if !qt.IsLocal {
			if qt.PackageAlias == "" {
				return fmt.Errorf("%w at index %d", ErrEmptyPackageAlias, i)
			}

			if qt.ImportPath == "" {
				return fmt.Errorf("%w at index %d", ErrEmptyImportPath, i)
			}

			// Validate that the package alias exists in import map
			if _, exists := cpr.importMap[qt.PackageAlias]; !exists {
				return fmt.Errorf("%w: '%s' for type '%s'", ErrPackageAliasNotFound, qt.PackageAlias, qt.TypeName)
			}
		}
	}

	return nil
}

// ResolveAllTypes resolves all qualified types to domain types.
func (cpr *CrossPackageResolver) ResolveAllTypes(
	ctx context.Context,
	qualifiedTypes []*QualifiedType,
) ([]domain.Type, error) {
	if err := cpr.ValidateQualifiedTypes(qualifiedTypes); err != nil {
		return nil, fmt.Errorf("qualified types validation failed: %w", err)
	}

	domainTypes := make([]domain.Type, len(qualifiedTypes))

	for i, qt := range qualifiedTypes {
		domainType, err := cpr.ResolveType(ctx, qt)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve type at index %d (%s): %w", i, qt.String(), err)
		}
		domainTypes[i] = domainType
	}

	cpr.logger.Info("resolved all qualified types",
		zap.Int("type_count", len(domainTypes)))

	return domainTypes, nil
}

// GetPackageByAlias retrieves a loaded package by its alias.
func (cpr *CrossPackageResolver) GetPackageByAlias(ctx context.Context, alias string) (*packages.Package, error) {
	importPath, exists := cpr.importMap[alias]
	if !exists {
		return nil, fmt.Errorf("%w: '%s'", ErrPackageAliasNotFound, alias)
	}

	return cpr.loadPackage(ctx, importPath)
}

// ExtractPackageInfo extracts package information for debugging and validation.
func (cpr *CrossPackageResolver) ExtractPackageInfo(pkg *packages.Package) map[string]interface{} {
	info := map[string]interface{}{
		"name":   pkg.Name,
		"path":   pkg.PkgPath,
		"types":  make([]string, 0),
		"errors": make([]string, 0),
	}

	// Extract type names
	if pkg.Types != nil && pkg.Types.Scope() != nil {
		for _, name := range pkg.Types.Scope().Names() {
			obj := pkg.Types.Scope().Lookup(name)
			if obj != nil && obj.Exported() {
				typeNames := info["types"].([]string)
				info["types"] = append(typeNames, name)
			}
		}
	}

	// Extract errors
	for _, err := range pkg.Errors {
		errorMsgs := info["errors"].([]string)
		info["errors"] = append(errorMsgs, err.Error())
	}

	return info
}
