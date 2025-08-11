package parser

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// CrossPackageTypeLoaderAdapter adapts CrossPackageResolver to implement domain.CrossPackageTypeLoader.
// This bridges the parser and domain packages to avoid circular dependencies.
type CrossPackageTypeLoaderAdapter struct {
	resolver *CrossPackageResolver
	logger   *zap.Logger
}

// NewCrossPackageTypeLoaderAdapter creates a new adapter for cross-package type loading.
func NewCrossPackageTypeLoaderAdapter(
	resolver *CrossPackageResolver,
	logger *zap.Logger,
) *CrossPackageTypeLoaderAdapter {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CrossPackageTypeLoaderAdapter{
		resolver: resolver,
		logger:   logger,
	}
}

// ResolveType resolves a qualified type name to a concrete Type.
// The qualifiedTypeName should be in the format "package.TypeName" or "TypeName" for local types.
func (adapter *CrossPackageTypeLoaderAdapter) ResolveType(
	ctx context.Context,
	qualifiedTypeName string,
) (domain.Type, error) {
	adapter.logger.Debug("resolving type through adapter",
		zap.String("qualified_type_name", qualifiedTypeName))

	// Parse the qualified type name
	qualifiedType, err := adapter.resolver.parseQualifiedTypeName(qualifiedTypeName)
	if err != nil {
		return nil, err
	}

	// Resolve using the cross-package resolver
	domainType, err := adapter.resolver.ResolveType(ctx, qualifiedType)
	if err != nil {
		return nil, err
	}

	adapter.logger.Debug("successfully resolved type through adapter",
		zap.String("qualified_type_name", qualifiedTypeName),
		zap.String("resolved_type", domainType.String()))

	return domainType, nil
}

// ValidateTypeArguments validates that all type arguments can be resolved.
func (adapter *CrossPackageTypeLoaderAdapter) ValidateTypeArguments(
	ctx context.Context,
	typeArguments []string,
) error {
	adapter.logger.Debug("validating type arguments through adapter",
		zap.Int("argument_count", len(typeArguments)),
		zap.Strings("arguments", typeArguments))

	for i, typeArg := range typeArguments {
		// Parse each type argument
		qualifiedType, err := adapter.resolver.parseQualifiedTypeName(strings.TrimSpace(typeArg))
		if err != nil {
			return err
		}

		// Validate by attempting to resolve (without storing result)
		_, err = adapter.resolver.ResolveType(ctx, qualifiedType)
		if err != nil {
			adapter.logger.Error("type argument validation failed",
				zap.Int("argument_index", i),
				zap.String("argument", typeArg),
				zap.Error(err))
			return err
		}
	}

	adapter.logger.Debug("all type arguments validated successfully")
	return nil
}

// GetImportPaths returns the import paths needed for the given type arguments.
func (adapter *CrossPackageTypeLoaderAdapter) GetImportPaths(typeArguments []string) []string {
	importPaths := make([]string, 0)
	importPathSet := make(map[string]bool) // For deduplication

	for _, typeArg := range typeArguments {
		typeArg = strings.TrimSpace(typeArg)
		if typeArg == "" {
			continue
		}

		// Parse the type argument
		qualifiedType, err := adapter.resolver.parseQualifiedTypeName(typeArg)
		if err != nil {
			adapter.logger.Warn("failed to parse type argument for import path extraction",
				zap.String("type_argument", typeArg),
				zap.Error(err))
			continue
		}

		// Only include import paths for external types
		if !qualifiedType.IsLocal && qualifiedType.ImportPath != "" {
			if !importPathSet[qualifiedType.ImportPath] {
				importPaths = append(importPaths, qualifiedType.ImportPath)
				importPathSet[qualifiedType.ImportPath] = true
			}
		}
	}

	adapter.logger.Debug("extracted import paths from type arguments",
		zap.Int("type_argument_count", len(typeArguments)),
		zap.Int("import_path_count", len(importPaths)),
		zap.Strings("import_paths", importPaths))

	return importPaths
}

// GetResolver returns the underlying CrossPackageResolver.
func (adapter *CrossPackageTypeLoaderAdapter) GetResolver() *CrossPackageResolver {
	return adapter.resolver
}

// UpdateImportMap updates the import map in the underlying resolver.
func (adapter *CrossPackageTypeLoaderAdapter) UpdateImportMap(newImports map[string]string) {
	adapter.resolver.UpdateImportMap(newImports)
	adapter.logger.Debug("updated import map through adapter",
		zap.Int("mapping_count", len(newImports)))
}

// ClearCache clears the cache in the underlying resolver.
func (adapter *CrossPackageTypeLoaderAdapter) ClearCache() {
	adapter.resolver.ClearCache()
	adapter.logger.Debug("cleared cache through adapter")
}

// GetCacheStats returns cache statistics from the underlying resolver.
func (adapter *CrossPackageTypeLoaderAdapter) GetCacheStats() (hits, misses int) {
	return adapter.resolver.GetCacheStats()
}
