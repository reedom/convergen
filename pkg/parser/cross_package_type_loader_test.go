package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
)

func TestNewCrossPackageTypeLoaderAdapter(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	resolver := NewCrossPackageResolver(nil, importMap, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	require.NotNil(t, adapter)
	assert.Equal(t, resolver, adapter.resolver)
	assert.Equal(t, logger, adapter.logger)
}

func TestCrossPackageTypeLoaderAdapter_ResolveType(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	resolver := NewCrossPackageResolver(nil, importMap, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)
	ctx := context.Background()

	tests := []struct {
		name              string
		qualifiedTypeName string
		expectError       bool
		expectType        string
	}{
		{
			name:              "local type",
			qualifiedTypeName: "User",
			expectError:       false,
			expectType:        "User",
		},
		{
			name:              "qualified type (will fail package loading)",
			qualifiedTypeName: "models.User",
			expectError:       true, // Package loading will fail
		},
		{
			name:              "unknown package alias",
			qualifiedTypeName: "unknown.Type",
			expectError:       true,
		},
		{
			name:              "invalid format",
			qualifiedTypeName: "pkg.sub.Type",
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter.ResolveType(ctx, tt.qualifiedTypeName)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectType, result.Name())
			}
		})
	}
}

func TestCrossPackageTypeLoaderAdapter_ValidateTypeArguments(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	resolver := NewCrossPackageResolver(nil, importMap, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		typeArguments []string
		expectError   bool
	}{
		{
			name:          "empty arguments",
			typeArguments: []string{},
			expectError:   false,
		},
		{
			name:          "valid local types",
			typeArguments: []string{"User", "UserDTO"},
			expectError:   false,
		},
		{
			name:          "qualified types (will fail package loading)",
			typeArguments: []string{"models.User", "dto.UserDTO"},
			expectError:   true, // Package loading will fail
		},
		{
			name:          "mixed types (will fail package loading)",
			typeArguments: []string{"User", "dto.UserDTO"},
			expectError:   true, // Package loading will fail for dto.UserDTO
		},
		{
			name:          "invalid qualified type",
			typeArguments: []string{"unknown.Type"},
			expectError:   true,
		},
		{
			name:          "invalid format",
			typeArguments: []string{"pkg.sub.Type"},
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.ValidateTypeArguments(ctx, tt.typeArguments)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCrossPackageTypeLoaderAdapter_GetImportPaths(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	resolver := NewCrossPackageResolver(nil, importMap, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	tests := []struct {
		name          string
		typeArguments []string
		expected      []string
	}{
		{
			name:          "empty arguments",
			typeArguments: []string{},
			expected:      []string{},
		},
		{
			name:          "local types only",
			typeArguments: []string{"User", "UserDTO"},
			expected:      []string{},
		},
		{
			name:          "qualified types",
			typeArguments: []string{"models.User", "dto.UserDTO"},
			expected:      []string{"./internal/models", "./pkg/dto"},
		},
		{
			name:          "mixed types",
			typeArguments: []string{"User", "dto.UserDTO"},
			expected:      []string{"./pkg/dto"},
		},
		{
			name:          "duplicate imports",
			typeArguments: []string{"models.User", "models.Entity"},
			expected:      []string{"./internal/models"},
		},
		{
			name:          "with spaces",
			typeArguments: []string{" User ", " dto.UserDTO "},
			expected:      []string{"./pkg/dto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.GetImportPaths(tt.typeArguments)

			// Order may vary, so check length and content
			assert.Equal(t, len(tt.expected), len(result))
			for _, expectedPath := range tt.expected {
				assert.Contains(t, result, expectedPath)
			}
		})
	}
}

func TestCrossPackageTypeLoaderAdapter_UpdateImportMap(t *testing.T) {
	logger := zap.NewNop()
	initialImports := map[string]string{
		"models": "./internal/models",
	}

	resolver := NewCrossPackageResolver(nil, initialImports, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	newImports := map[string]string{
		"dto":      "./pkg/dto",
		"services": "./internal/services",
	}

	adapter.UpdateImportMap(newImports)

	// Verify the underlying resolver was updated
	importMap := resolver.GetImportMap()
	assert.Equal(t, "./internal/models", importMap["models"])
	assert.Equal(t, "./pkg/dto", importMap["dto"])
	assert.Equal(t, "./internal/services", importMap["services"])
}

func TestCrossPackageTypeLoaderAdapter_ClearCache(t *testing.T) {
	logger := zap.NewNop()
	resolver := NewCrossPackageResolver(nil, nil, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	// This is mainly testing that the method doesn't panic
	// and delegates to the underlying resolver
	adapter.ClearCache()

	// If we got here without panicking, the test passes
	assert.True(t, true)
}

func TestCrossPackageTypeLoaderAdapter_GetCacheStats(t *testing.T) {
	logger := zap.NewNop()
	resolver := NewCrossPackageResolver(nil, nil, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	hits, misses := adapter.GetCacheStats()

	// Should return valid statistics (exact values depend on resolver implementation)
	assert.GreaterOrEqual(t, hits, 0)
	assert.GreaterOrEqual(t, misses, 0)
}

func TestCrossPackageTypeLoaderAdapter_GetResolver(t *testing.T) {
	logger := zap.NewNop()
	resolver := NewCrossPackageResolver(nil, nil, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	result := adapter.GetResolver()
	assert.Equal(t, resolver, result)
}

// Integration test with domain.TypeInstantiator.
func TestCrossPackageTypeLoaderAdapter_Integration(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	// Create the full stack
	resolver := NewCrossPackageResolver(nil, importMap, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	// Create a type instantiator with cross-package support
	config := domain.NewTypeInstantiatorConfig()
	config.CrossPackageTypeLoader = adapter

	typeBuilder := domain.NewTypeBuilder()
	instantiator := domain.NewTypeInstantiatorWithConfig(typeBuilder, logger, config)

	// Verify the instantiator has cross-package support
	assert.True(t, instantiator.HasCrossPackageSupport())
	assert.Equal(t, adapter, instantiator.GetCrossPackageLoader())

	// Test resolving type arguments through the instantiator
	ctx := context.Background()
	typeArguments := []string{"User", "models.Entity", "dto.UserDTO"}

	// This would typically be called by the instantiator internally
	// We test the adapter directly here
	// Note: This will fail because the test packages don't exist, which is expected
	err := adapter.ValidateTypeArguments(ctx, typeArguments)
	assert.Error(t, err) // Package loading will fail for non-existent paths

	importPaths := adapter.GetImportPaths(typeArguments)
	expected := []string{"./internal/models", "./pkg/dto"}
	assert.Equal(t, len(expected), len(importPaths))
	for _, expectedPath := range expected {
		assert.Contains(t, importPaths, expectedPath)
	}
}

func TestCrossPackageTypeLoaderAdapter_WithNilLogger(t *testing.T) {
	resolver := NewCrossPackageResolver(nil, nil, nil)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, nil)

	require.NotNil(t, adapter)
	require.NotNil(t, adapter.logger) // Should have created a no-op logger
}

func TestCrossPackageTypeLoaderAdapter_EmptyTypeArguments(t *testing.T) {
	logger := zap.NewNop()
	resolver := NewCrossPackageResolver(nil, nil, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	// Test with empty and whitespace-only type arguments
	typeArguments := []string{"", "  ", "\t", "\n"}
	importPaths := adapter.GetImportPaths(typeArguments)

	assert.Equal(t, 0, len(importPaths))
}

func TestCrossPackageTypeLoaderAdapter_InvalidTypeArguments(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
	}

	resolver := NewCrossPackageResolver(nil, importMap, logger)
	adapter := NewCrossPackageTypeLoaderAdapter(resolver, logger)

	// Test that invalid type arguments don't crash GetImportPaths
	typeArguments := []string{"invalid..type", "pkg.sub.Type", ""}
	importPaths := adapter.GetImportPaths(typeArguments)

	// Should return empty list for invalid arguments
	assert.Equal(t, 0, len(importPaths))
}
