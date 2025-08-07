package domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrGenericInterfaceNil                    = errors.New("generic interface cannot be nil")
	ErrTypeArgumentsNil                       = errors.New("type arguments cannot be nil")
	ErrTypeArgumentCountMismatch              = errors.New("type argument count does not match type parameter count")
	ErrInvalidTypeArgument                    = errors.New("invalid type argument")
	ErrConstraintViolation                    = errors.New("type argument violates constraint")
	ErrRecursiveInstantiation                 = errors.New("recursive instantiation detected")
	ErrInstantiationFailed                    = errors.New("instantiation failed")
	ErrCacheKeyGeneration                     = errors.New("failed to generate cache key")
	ErrCircularTypeDetected                   = errors.New("circular type dependency detected")
	ErrCrossPackageTypeLoader                 = errors.New("cross-package type loader not configured")
	ErrExternalTypeResolution                 = errors.New("failed to resolve external type")
	ErrInterfaceNameCannotBeEmpty             = errors.New("interface name cannot be empty")
	ErrGenericInterfaceMustHaveTypeParameters = errors.New("generic interface must have type parameters")
	ErrEmptyTypeArgument                      = errors.New("empty type argument")
)

// InstantiatedInterface represents a concrete instantiation of a generic interface.
// This is the primary result of the type instantiation process.
type InstantiatedInterface struct {
	// Core instantiation data
	SourceInterface Type            `json:"source_interface"` // Original generic interface
	TypeArguments   map[string]Type `json:"type_arguments"`   // Map of type parameter names to concrete types
	ConcreteType    Type            `json:"concrete_type"`    // The instantiated concrete type
	TypeSignature   string          `json:"type_signature"`   // Unique signature for this instantiation

	// Instantiation metadata
	InstantiatedAt   time.Time         `json:"instantiated_at"`   // When this instantiation was created
	ValidationResult *ValidationResult `json:"validation_result"` // Constraint validation results

	// Performance tracking
	InstantiationDurationMS int64 `json:"instantiation_duration_ms"` // Time taken to instantiate
	CacheHit                bool  `json:"cache_hit"`                 // Whether this was served from cache
}

// ValidationResult captures the results of constraint validation during instantiation.
type ValidationResult struct {
	Valid                bool                        `json:"valid"`
	ViolatedConstraints  []ConstraintViolation       `json:"violated_constraints,omitempty"`
	ValidationDurationMS int64                       `json:"validation_duration_ms"`
	Details              map[string]ValidationDetail `json:"details,omitempty"`
}

// ConstraintViolation represents a specific constraint violation.
type ConstraintViolation struct {
	TypeParamName      string `json:"type_param_name"`
	ExpectedConstraint string `json:"expected_constraint"`
	ActualType         string `json:"actual_type"`
	ViolationMessage   string `json:"violation_message"`
}

// ValidationDetail provides detailed information about constraint validation.
type ValidationDetail struct {
	TypeParamName    string `json:"type_param_name"`
	TypeArgument     string `json:"type_argument"`
	ConstraintType   string `json:"constraint_type"`
	ValidationPassed bool   `json:"validation_passed"`
	Details          string `json:"details,omitempty"`
}

// NewInstantiatedInterface creates a new instantiated interface with proper validation.
func NewInstantiatedInterface(
	sourceInterface Type,
	typeArguments map[string]Type,
	concreteType Type,
	typeSignature string,
) (*InstantiatedInterface, error) {
	if sourceInterface == nil {
		return nil, ErrGenericInterfaceNil
	}

	if typeArguments == nil {
		return nil, ErrTypeArgumentsNil
	}

	if concreteType == nil {
		return nil, ErrInvalidTypeArgument
	}

	if typeSignature == "" {
		return nil, ErrCacheKeyGeneration
	}

	// Validate type arguments are not nil
	for paramName, typeArg := range typeArguments {
		if paramName == "" || typeArg == nil {
			return nil, fmt.Errorf("%w: parameter '%s'", ErrInvalidTypeArgument, paramName)
		}
	}

	return &InstantiatedInterface{
		SourceInterface:         sourceInterface,
		TypeArguments:           copyTypeArguments(typeArguments),
		ConcreteType:            concreteType,
		TypeSignature:           typeSignature,
		InstantiatedAt:          time.Now(),
		ValidationResult:        nil, // Will be set during validation
		InstantiationDurationMS: 0,   // Will be set by instantiator
		CacheHit:                false,
	}, nil
}

// copyTypeArguments creates a defensive copy of type arguments map.
func copyTypeArguments(typeArgs map[string]Type) map[string]Type {
	cp := make(map[string]Type, len(typeArgs))
	for k, v := range typeArgs {
		cp[k] = v
	}
	return cp
}

// GenericInterface represents a generic interface that can be instantiated.
// This serves as input to the instantiation process.
type GenericInterface struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"type_params"`
	Methods    []*Method   `json:"methods"`
	Package    string      `json:"package"`
}

// NewGenericInterface creates a new generic interface with validation.
func NewGenericInterface(name string, typeParams []TypeParam, methods []*Method, pkg string) (*GenericInterface, error) {
	if name == "" {
		return nil, ErrInterfaceNameCannotBeEmpty
	}

	if len(typeParams) == 0 {
		return nil, ErrGenericInterfaceMustHaveTypeParameters
	}

	// Validate type parameters
	for i, param := range typeParams {
		if !param.IsValid() {
			return nil, fmt.Errorf("%w at index %d: %s", ErrInvalidTypeArgument, i, param.Name)
		}
	}

	return &GenericInterface{
		Name:       name,
		TypeParams: append([]TypeParam(nil), typeParams...), // defensive copy
		Methods:    append([]*Method(nil), methods...),      // defensive copy
		Package:    pkg,
	}, nil
}

// CrossPackageTypeLoader defines the interface for loading types from external packages.
// This abstracts the cross-package type resolution to avoid circular dependencies.
type CrossPackageTypeLoader interface {
	// ResolveType resolves a qualified type name to a concrete Type.
	// The qualifiedTypeName should be in the format "package.TypeName" or "TypeName" for local types.
	ResolveType(ctx context.Context, qualifiedTypeName string) (Type, error)

	// ValidateTypeArguments validates that all type arguments can be resolved.
	ValidateTypeArguments(ctx context.Context, typeArguments []string) error

	// GetImportPaths returns the import paths needed for the given type arguments.
	GetImportPaths(typeArguments []string) []string
}

// TypeInstantiator is the core engine for converting generic types to concrete types.
// It handles constraint validation, caching, and performance optimization.
type TypeInstantiator struct {
	typeBuilder        *TypeBuilder
	substitutionEngine *TypeSubstitutionEngine // Enhanced type substitution engine
	cache              map[string]*InstantiatedInterface
	logger             *zap.Logger
	crossPackageLoader CrossPackageTypeLoader // For resolving external types

	// Performance tracking
	cacheHits           int64
	cacheMisses         int64
	totalInstantiations int64

	// Safety mechanisms
	recursionDepth     int
	maxRecursionDepth  int
	instantiationStack []string // Track current instantiation chain for cycle detection
}

// TypeInstantiatorConfig configures the behavior of TypeInstantiator.
type TypeInstantiatorConfig struct {
	MaxRecursionDepth        int                       `json:"max_recursion_depth"`
	EnableCaching            bool                      `json:"enable_caching"`
	EnablePerformanceTrack   bool                      `json:"enable_performance_tracking"`
	CacheCapacity            int                       `json:"cache_capacity"`
	CrossPackageTypeLoader   CrossPackageTypeLoader    `json:"-"` // Cannot serialize interfaces
	SubstitutionEngineConfig *SubstitutionEngineConfig `json:"substitution_engine_config,omitempty"`
}

// NewTypeInstantiatorConfig creates a default configuration.
func NewTypeInstantiatorConfig() *TypeInstantiatorConfig {
	return &TypeInstantiatorConfig{
		MaxRecursionDepth:        10,
		EnableCaching:            true,
		EnablePerformanceTrack:   true,
		CacheCapacity:            1000,
		CrossPackageTypeLoader:   nil, // Must be set explicitly for cross-package support
		SubstitutionEngineConfig: NewSubstitutionEngineConfig(),
	}
}

// NewTypeInstantiator creates a new type instantiator with the given dependencies.
func NewTypeInstantiator(typeBuilder *TypeBuilder, logger *zap.Logger) *TypeInstantiator {
	return NewTypeInstantiatorWithConfig(typeBuilder, logger, NewTypeInstantiatorConfig())
}

// NewTypeInstantiatorWithConfig creates a new type instantiator with custom configuration.
func NewTypeInstantiatorWithConfig(
	typeBuilder *TypeBuilder,
	logger *zap.Logger,
	config *TypeInstantiatorConfig,
) *TypeInstantiator {
	if typeBuilder == nil {
		typeBuilder = NewTypeBuilder()
	}

	if logger == nil {
		// Create a no-op logger if none provided
		logger = zap.NewNop()
	}

	var cache map[string]*InstantiatedInterface
	if config.EnableCaching {
		cache = make(map[string]*InstantiatedInterface, config.CacheCapacity)
	}

	// Initialize substitution engine with configuration
	substitutionConfig := config.SubstitutionEngineConfig
	if substitutionConfig == nil {
		substitutionConfig = NewSubstitutionEngineConfig()
	}
	substitutionEngine := NewTypeSubstitutionEngineWithConfig(typeBuilder, substitutionConfig, logger)

	return &TypeInstantiator{
		typeBuilder:        typeBuilder,
		substitutionEngine: substitutionEngine,
		cache:              cache,
		logger:             logger,
		crossPackageLoader: config.CrossPackageTypeLoader,
		maxRecursionDepth:  config.MaxRecursionDepth,
		instantiationStack: make([]string, 0),
	}
}

// InstantiateInterface is the main entry point for type instantiation.
// It converts a generic interface with concrete type arguments to a concrete interface.
func (ti *TypeInstantiator) InstantiateInterface(
	genericInterface *GenericInterface,
	typeArgs []Type,
) (*InstantiatedInterface, error) {
	return ti.InstantiateInterfaceWithContext(context.Background(), genericInterface, typeArgs)
}

// InstantiateInterfaceWithContext is the main entry point for type instantiation with context support.
// It converts a generic interface with concrete type arguments to a concrete interface.
func (ti *TypeInstantiator) InstantiateInterfaceWithContext(
	ctx context.Context,
	genericInterface *GenericInterface,
	typeArgs []Type,
) (*InstantiatedInterface, error) {
	startTime := time.Now()

	if genericInterface == nil {
		return nil, ErrGenericInterfaceNil
	}

	if typeArgs == nil {
		return nil, ErrTypeArgumentsNil
	}

	// Validate type argument count
	if len(typeArgs) != len(genericInterface.TypeParams) {
		return nil, fmt.Errorf("%w: expected %d, got %d",
			ErrTypeArgumentCountMismatch,
			len(genericInterface.TypeParams),
			len(typeArgs))
	}

	// Create type argument mapping
	typeArgMap := make(map[string]Type, len(typeArgs))
	for i, typeArg := range typeArgs {
		if typeArg == nil {
			return nil, fmt.Errorf("%w: argument at index %d is nil", ErrInvalidTypeArgument, i)
		}
		typeArgMap[genericInterface.TypeParams[i].Name] = typeArg
	}

	// Generate cache key for this instantiation
	typeSignature := ti.generateTypeSignature(genericInterface, typeArgMap)

	// Check cache first
	if ti.cache != nil {
		if cached, found := ti.cache[typeSignature]; found {
			ti.cacheHits++
			ti.totalInstantiations++
			ti.logger.Debug("cache hit for interface instantiation",
				zap.String("interface", genericInterface.Name),
				zap.String("signature", typeSignature))

			// Create a copy with updated cache hit status
			result := *cached
			result.CacheHit = true
			return &result, nil
		}
		ti.cacheMisses++
	}

	ti.logger.Debug("instantiating generic interface",
		zap.String("interface", genericInterface.Name),
		zap.Int("type_params", len(genericInterface.TypeParams)),
		zap.Int("type_args", len(typeArgs)),
		zap.String("signature", typeSignature))

	// Check for recursion
	if err := ti.checkRecursion(typeSignature); err != nil {
		return nil, err
	}

	// Add to recursion stack
	ti.instantiationStack = append(ti.instantiationStack, typeSignature)
	ti.recursionDepth++

	defer func() {
		// Remove from recursion stack
		if 0 < len(ti.instantiationStack) {
			ti.instantiationStack = ti.instantiationStack[:len(ti.instantiationStack)-1]
		}
		ti.recursionDepth--
	}()

	// Validate constraints
	validationResult, err := ti.validateConstraints(genericInterface.TypeParams, typeArgMap)
	if err != nil {
		ti.logger.Error("constraint validation failed",
			zap.String("interface", genericInterface.Name),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %s", ErrConstraintViolation, err.Error())
	}

	if !validationResult.Valid {
		ti.logger.Error("constraint violations detected",
			zap.String("interface", genericInterface.Name),
			zap.Int("violations", len(validationResult.ViolatedConstraints)))
		return nil, fmt.Errorf("%w: %d constraint violations", ErrConstraintViolation, len(validationResult.ViolatedConstraints))
	}

	// Perform the actual instantiation
	concreteType, err := ti.instantiateType(genericInterface, typeArgMap)
	if err != nil {
		ti.logger.Error("type instantiation failed",
			zap.String("interface", genericInterface.Name),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %s", ErrInstantiationFailed, err.Error())
	}

	// Create the instantiated interface result
	instantiated, err := NewInstantiatedInterface(
		ti.createInterfaceType(genericInterface),
		typeArgMap,
		concreteType,
		typeSignature,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create instantiated interface: %w", err)
	}

	// Set validation result and timing
	instantiated.ValidationResult = validationResult
	instantiated.InstantiationDurationMS = time.Since(startTime).Milliseconds()
	instantiated.CacheHit = false

	// Cache the result
	if ti.cache != nil {
		ti.cache[typeSignature] = instantiated
	}

	ti.totalInstantiations++

	ti.logger.Info("successfully instantiated generic interface",
		zap.String("interface", genericInterface.Name),
		zap.String("signature", typeSignature),
		zap.Int64("duration_ms", instantiated.InstantiationDurationMS),
		zap.Bool("valid", validationResult.Valid))

	return instantiated, nil
}

// InstantiateInterfaceFromStrings instantiates a generic interface using string type arguments.
// This method supports cross-package type resolution through the configured CrossPackageTypeLoader.
func (ti *TypeInstantiator) InstantiateInterfaceFromStrings(
	ctx context.Context,
	genericInterface *GenericInterface,
	typeArguments []string,
) (*InstantiatedInterface, error) {
	if ti.crossPackageLoader == nil && ti.hasExternalTypes(typeArguments) {
		return nil, fmt.Errorf("%w: external types detected but no cross-package loader configured", ErrCrossPackageTypeLoader)
	}

	// Resolve string type arguments to concrete types
	typeArgs, err := ti.resolveTypeArguments(ctx, typeArguments)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve type arguments: %w", err)
	}

	// Use the standard instantiation method with resolved types
	return ti.InstantiateInterfaceWithContext(ctx, genericInterface, typeArgs)
}

// resolveTypeArguments resolves string type arguments to concrete Type objects.
func (ti *TypeInstantiator) resolveTypeArguments(ctx context.Context, typeArguments []string) ([]Type, error) {
	if len(typeArguments) == 0 {
		return []Type{}, nil
	}

	resolvedTypes := make([]Type, len(typeArguments))

	for i, typeArg := range typeArguments {
		resolvedType, err := ti.resolveTypeArgument(ctx, typeArg)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve type argument at index %d (%s): %w", i, typeArg, err)
		}
		resolvedTypes[i] = resolvedType
	}

	ti.logger.Debug("resolved all type arguments",
		zap.Int("argument_count", len(typeArguments)),
		zap.Strings("arguments", typeArguments))

	return resolvedTypes, nil
}

// resolveTypeArgument resolves a single string type argument to a concrete Type.
func (ti *TypeInstantiator) resolveTypeArgument(ctx context.Context, typeArgument string) (Type, error) {
	typeArgument = strings.TrimSpace(typeArgument)

	if typeArgument == "" {
		return nil, ErrEmptyTypeArgument
	}

	// Check if this is a qualified type (contains dot)
	if strings.Contains(typeArgument, ".") {
		// External type - use cross-package loader
		if ti.crossPackageLoader == nil {
			return nil, fmt.Errorf("%w: cannot resolve external type %s", ErrCrossPackageTypeLoader, typeArgument)
		}

		resolvedType, err := ti.crossPackageLoader.ResolveType(ctx, typeArgument)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %s", ErrExternalTypeResolution, typeArgument, err.Error())
		}

		ti.logger.Debug("resolved external type",
			zap.String("type_argument", typeArgument),
			zap.String("resolved_type", resolvedType.String()))

		return resolvedType, nil
	}

	// Local type - create a basic type representation
	// In a full implementation, this would resolve from the current package's scope
	localType := NewBasicType(typeArgument, getReflectKindFromTypeName(typeArgument))

	ti.logger.Debug("resolved local type",
		zap.String("type_argument", typeArgument),
		zap.String("resolved_type", localType.String()))

	return localType, nil
}

// hasExternalTypes checks if any of the type arguments are external (qualified) types.
func (ti *TypeInstantiator) hasExternalTypes(typeArguments []string) bool {
	for _, typeArg := range typeArguments {
		if strings.Contains(strings.TrimSpace(typeArg), ".") {
			return true
		}
	}
	return false
}

const (
	interfaceKeyword = "interface"
)

// getReflectKindFromTypeName returns a reflect.Kind value for common type names.
func getReflectKindFromTypeName(typeName string) reflect.Kind {
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
	case interfaceKeyword:
		return reflect.Interface
	default:
		return reflect.Struct // Default for custom types
	}
}

// SetCrossPackageLoader sets the cross-package type loader for external type resolution.
func (ti *TypeInstantiator) SetCrossPackageLoader(loader CrossPackageTypeLoader) {
	ti.crossPackageLoader = loader
	ti.logger.Debug("cross-package type loader configured")
}

// GetCrossPackageLoader returns the configured cross-package type loader.
func (ti *TypeInstantiator) GetCrossPackageLoader() CrossPackageTypeLoader {
	return ti.crossPackageLoader
}

// HasCrossPackageSupport returns true if a cross-package type loader is configured.
func (ti *TypeInstantiator) HasCrossPackageSupport() bool {
	return ti.crossPackageLoader != nil
}

// checkRecursion validates that we're not in a recursive instantiation cycle.
func (ti *TypeInstantiator) checkRecursion(typeSignature string) error {
	if ti.maxRecursionDepth <= ti.recursionDepth {
		return fmt.Errorf("%w: maximum recursion depth %d exceeded", ErrRecursiveInstantiation, ti.maxRecursionDepth)
	}

	// Check for cycles in instantiation stack
	for _, existing := range ti.instantiationStack {
		if existing == typeSignature {
			return fmt.Errorf("%w: circular dependency detected for %s", ErrCircularTypeDetected, typeSignature)
		}
	}

	return nil
}

// validateConstraints validates that all type arguments satisfy their constraints.
func (ti *TypeInstantiator) validateConstraints(
	typeParams []TypeParam,
	typeArgs map[string]Type,
) (*ValidationResult, error) {
	startTime := time.Now()

	result := &ValidationResult{
		Valid:               true,
		ViolatedConstraints: make([]ConstraintViolation, 0),
		Details:             make(map[string]ValidationDetail),
	}

	for _, p := range typeParams {
		param := p
		typeArg, exists := typeArgs[param.Name]
		if !exists {
			result.Valid = false
			result.ViolatedConstraints = append(result.ViolatedConstraints, ConstraintViolation{
				TypeParamName:      param.Name,
				ExpectedConstraint: param.GetConstraintType(),
				ActualType:         "missing",
				ViolationMessage:   fmt.Sprintf("type argument for parameter %s is missing", param.Name),
			})
			continue
		}

		// Validate constraint satisfaction
		satisfies := param.SatisfiesConstraint(typeArg)

		detail := ValidationDetail{
			TypeParamName:    param.Name,
			TypeArgument:     typeArg.String(),
			ConstraintType:   param.GetConstraintType(),
			ValidationPassed: satisfies,
		}

		if !satisfies {
			result.Valid = false
			violation := ConstraintViolation{
				TypeParamName:      param.Name,
				ExpectedConstraint: param.GetConstraintType(),
				ActualType:         typeArg.String(),
				ViolationMessage:   ti.generateConstraintViolationMessage(&param, typeArg),
			}
			result.ViolatedConstraints = append(result.ViolatedConstraints, violation)
			detail.Details = violation.ViolationMessage
		}

		result.Details[param.Name] = detail
	}

	result.ValidationDurationMS = time.Since(startTime).Milliseconds()

	ti.logger.Debug("constraint validation completed",
		zap.Bool("valid", result.Valid),
		zap.Int("violations", len(result.ViolatedConstraints)),
		zap.Int64("duration_ms", result.ValidationDurationMS))

	return result, nil
}

// generateConstraintViolationMessage creates a detailed error message for constraint violations.
func (ti *TypeInstantiator) generateConstraintViolationMessage(param *TypeParam, typeArg Type) string {
	switch param.GetConstraintType() {
	case "any":
		return "should never happen - any constraint accepts all types"
	case "comparable":
		return fmt.Sprintf("type %s is not comparable", typeArg.String())
	case "union":
		unionTypes := make([]string, len(param.UnionTypes))
		for i, t := range param.UnionTypes {
			unionTypes[i] = t.String()
		}
		return fmt.Sprintf("type %s does not match any of the union types: %s",
			typeArg.String(), strings.Join(unionTypes, " | "))
	case "union_underlying":
		unionTypes := make([]string, len(param.UnionTypes))
		for i, t := range param.UnionTypes {
			unionTypes[i] = "~" + t.String()
		}
		return fmt.Sprintf("type %s does not match any of the underlying union types: %s",
			typeArg.String(), strings.Join(unionTypes, " | "))
	case "underlying":
		return fmt.Sprintf("type %s is not assignable to underlying type ~%s",
			typeArg.String(), param.Underlying.Type.String())
	case "interface":
		return fmt.Sprintf("type %s does not implement interface %s",
			typeArg.String(), param.Constraint.String())
	default:
		return fmt.Sprintf("type %s does not satisfy constraint %s",
			typeArg.String(), param.GetConstraintType())
	}
}

// instantiateType performs the actual type instantiation logic using the substitution engine.
func (ti *TypeInstantiator) instantiateType(
	genericInterface *GenericInterface,
	typeArgs map[string]Type,
) (Type, error) {
	// Create the generic interface type for substitution
	genericInterfaceType := ti.createInterfaceType(genericInterface)

	// Convert type arguments map to ordered slices for substitution engine
	typeParams := genericInterface.TypeParams
	orderedTypeArgs := make([]Type, len(typeParams))
	for i, param := range typeParams {
		orderedTypeArgs[i] = typeArgs[param.Name]
	}

	// Perform type substitution using the substitution engine
	substitutionResult, err := ti.substitutionEngine.SubstituteType(
		genericInterfaceType,
		typeParams,
		orderedTypeArgs,
	)
	if err != nil {
		return nil, fmt.Errorf("type substitution failed: %w", err)
	}

	ti.logger.Debug("performed type substitution",
		zap.String("interface", genericInterface.Name),
		zap.String("original_type", substitutionResult.OriginalType.String()),
		zap.String("substituted_type", substitutionResult.SubstitutedType.String()),
		zap.Int64("substitution_time_ms", substitutionResult.SubstitutionTime),
		zap.Bool("cache_hit", substitutionResult.CacheHit))

	return substitutionResult.SubstitutedType, nil
}

// generateConcreteTypeName creates a name for the concrete instantiated type.
func (ti *TypeInstantiator) generateConcreteTypeName(
	genericInterface *GenericInterface,
	typeArgs map[string]Type,
) string {
	var builder strings.Builder
	builder.WriteString(genericInterface.Name)
	builder.WriteString("[")

	typeArgNames := make([]string, 0, len(typeArgs))
	for i, param := range genericInterface.TypeParams {
		if 0 < i {
			builder.WriteString(", ")
		}
		typeArg := typeArgs[param.Name]
		typeArgName := typeArg.Name()
		builder.WriteString(typeArgName)
		typeArgNames = append(typeArgNames, typeArgName)
	}

	builder.WriteString("]")
	return builder.String()
}

// generateTypeSignature creates a unique signature for caching instantiated interfaces.
func (ti *TypeInstantiator) generateTypeSignature(
	genericInterface *GenericInterface,
	typeArgs map[string]Type,
) string {
	var builder strings.Builder

	// Include package and interface name
	if genericInterface.Package != "" {
		builder.WriteString(genericInterface.Package)
		builder.WriteString(".")
	}
	builder.WriteString(genericInterface.Name)
	builder.WriteString("[")

	// Add type arguments in parameter order for consistency
	for i, param := range genericInterface.TypeParams {
		if 0 < i {
			builder.WriteString(",")
		}
		typeArg := typeArgs[param.Name]
		builder.WriteString(typeArg.String())
	}

	builder.WriteString("]")
	return builder.String()
}

// createInterfaceType creates a Type representation of the generic interface.
func (ti *TypeInstantiator) createInterfaceType(genericInterface *GenericInterface) Type {
	// Create a basic interface type representation
	return NewBasicType(genericInterface.Name, reflect.Interface)
}

// GetCacheStats returns statistics about the instantiation cache.
func (ti *TypeInstantiator) GetCacheStats() CacheStats {
	cacheSize := 0
	if ti.cache != nil {
		cacheSize = len(ti.cache)
	}

	return CacheStats{
		CacheHits:           ti.cacheHits,
		CacheMisses:         ti.cacheMisses,
		TotalInstantiations: ti.totalInstantiations,
		CacheSize:           int64(cacheSize),
		HitRate:             ti.calculateHitRate(),
	}
}

// CacheStats provides statistics about instantiation cache performance.
type CacheStats struct {
	CacheHits           int64   `json:"cache_hits"`
	CacheMisses         int64   `json:"cache_misses"`
	TotalInstantiations int64   `json:"total_instantiations"`
	CacheSize           int64   `json:"cache_size"`
	HitRate             float64 `json:"hit_rate"`
}

// calculateHitRate computes the cache hit rate as a percentage.
func (ti *TypeInstantiator) calculateHitRate() float64 {
	total := ti.cacheHits + ti.cacheMisses
	if total == 0 {
		return 0.0
	}
	return float64(ti.cacheHits) / float64(total) * 100.0
}

// ClearCache removes all cached instantiations.
func (ti *TypeInstantiator) ClearCache() {
	if ti.cache != nil {
		ti.cache = make(map[string]*InstantiatedInterface)
		ti.logger.Debug("instantiation cache cleared")
	}
}

// GetCachedInstantiation retrieves a cached instantiation by type signature.
func (ti *TypeInstantiator) GetCachedInstantiation(typeSignature string) (*InstantiatedInterface, bool) {
	if ti.cache == nil {
		return nil, false
	}

	instantiation, found := ti.cache[typeSignature]
	return instantiation, found
}

// HasCachedInstantiation checks if an instantiation is cached.
func (ti *TypeInstantiator) HasCachedInstantiation(typeSignature string) bool {
	_, found := ti.GetCachedInstantiation(typeSignature)
	return found
}

// SubstituteType is a convenience method that exposes the substitution engine directly.
// This allows for standalone type substitution without full interface instantiation.
func (ti *TypeInstantiator) SubstituteType(
	genericType Type,
	typeParams []TypeParam,
	typeArgs []Type,
) (*SubstitutionResult, error) {
	return ti.substitutionEngine.SubstituteType(genericType, typeParams, typeArgs)
}

// SubstituteTypeWithContext performs type substitution with context support.
func (ti *TypeInstantiator) SubstituteTypeWithContext(
	ctx context.Context,
	genericType Type,
	typeParams []TypeParam,
	typeArgs []Type,
) (*SubstitutionResult, error) {
	return ti.substitutionEngine.SubstituteTypeWithContext(ctx, genericType, typeParams, typeArgs)
}

// GetSubstitutionEngine returns the internal substitution engine for advanced usage.
func (ti *TypeInstantiator) GetSubstitutionEngine() *TypeSubstitutionEngine {
	return ti.substitutionEngine
}

// GetSubstitutionStats returns statistics from the substitution engine.
func (ti *TypeInstantiator) GetSubstitutionStats() *SubstitutionStats {
	return ti.substitutionEngine.GetSubstitutionStats()
}

// ClearSubstitutionCache clears the substitution engine's cache.
func (ti *TypeInstantiator) ClearSubstitutionCache() {
	ti.substitutionEngine.ClearCache()
}

// GetSubstitutionCacheSize returns the size of the substitution cache.
func (ti *TypeInstantiator) GetSubstitutionCacheSize() int {
	return ti.substitutionEngine.GetCacheSize()
}
