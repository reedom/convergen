package parser

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewQualifiedType(t *testing.T) {
	tests := []struct {
		name         string
		packageAlias string
		typeName     string
		importPath   string
		isLocal      bool
		wantErr      bool
		expectErr    error
	}{
		{
			name:         "valid local type",
			packageAlias: "",
			typeName:     "User",
			importPath:   "",
			isLocal:      true,
			wantErr:      false,
		},
		{
			name:         "valid external type",
			packageAlias: "models",
			typeName:     "User",
			importPath:   "./internal/models",
			isLocal:      false,
			wantErr:      false,
		},
		{
			name:         "empty type name",
			packageAlias: "models",
			typeName:     "",
			importPath:   "./internal/models",
			isLocal:      false,
			wantErr:      true,
			expectErr:    ErrInvalidQualifiedTypeName,
		},
		{
			name:         "non-local without package alias",
			packageAlias: "",
			typeName:     "User",
			importPath:   "./internal/models",
			isLocal:      false,
			wantErr:      true,
			expectErr:    ErrInvalidQualifiedTypeName,
		},
		{
			name:         "non-local without import path",
			packageAlias: "models",
			typeName:     "User",
			importPath:   "",
			isLocal:      false,
			wantErr:      true,
			expectErr:    ErrInvalidImportPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qt, err := NewQualifiedType(tt.packageAlias, tt.typeName, tt.importPath, tt.isLocal)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != nil {
					assert.ErrorIs(t, err, tt.expectErr)
				}
				assert.Nil(t, qt)
			} else {
				require.NoError(t, err)
				require.NotNil(t, qt)
				assert.Equal(t, tt.packageAlias, qt.PackageAlias)
				assert.Equal(t, tt.typeName, qt.TypeName)
				assert.Equal(t, tt.importPath, qt.ImportPath)
				assert.Equal(t, tt.isLocal, qt.IsLocal)
			}
		})
	}
}

func TestQualifiedType_String(t *testing.T) {
	tests := []struct {
		name     string
		qt       *QualifiedType
		expected string
	}{
		{
			name: "local type",
			qt: &QualifiedType{
				PackageAlias: "",
				TypeName:     "User",
				ImportPath:   "",
				IsLocal:      true,
			},
			expected: "User",
		},
		{
			name: "external type",
			qt: &QualifiedType{
				PackageAlias: "models",
				TypeName:     "User",
				ImportPath:   "./internal/models",
				IsLocal:      false,
			},
			expected: "models.User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.qt.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQualifiedType_FullName(t *testing.T) {
	tests := []struct {
		name     string
		qt       *QualifiedType
		expected string
	}{
		{
			name: "local type",
			qt: &QualifiedType{
				PackageAlias: "",
				TypeName:     "User",
				ImportPath:   "",
				IsLocal:      true,
			},
			expected: "User",
		},
		{
			name: "external type",
			qt: &QualifiedType{
				PackageAlias: "models",
				TypeName:     "User",
				ImportPath:   "./internal/models",
				IsLocal:      false,
			},
			expected: "./internal/models.User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.qt.FullName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCrossPackageResolver(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	cpr := NewCrossPackageResolver(nil, importMap, logger)

	require.NotNil(t, cpr)
	assert.NotNil(t, cpr.packageLoader)
	assert.NotNil(t, cpr.cache)
	assert.Equal(t, logger, cpr.logger)
	assert.Equal(t, len(importMap), len(cpr.importMap))
	
	// Verify import map was copied (not referenced)
	assert.Equal(t, importMap["models"], cpr.importMap["models"])
	assert.Equal(t, importMap["dto"], cpr.importMap["dto"])
}

func TestCrossPackageResolver_ParseTypeArguments(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	cpr := NewCrossPackageResolver(nil, importMap, logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		typeSpec    string
		expectCount int
		expectTypes []struct {
			packageAlias string
			typeName     string
			isLocal      bool
		}
		wantErr bool
	}{
		{
			name:        "simple local type",
			typeSpec:    "User",
			expectCount: 1,
			expectTypes: []struct {
				packageAlias string
				typeName     string
				isLocal      bool
			}{
				{packageAlias: "", typeName: "User", isLocal: true},
			},
			wantErr: false,
		},
		{
			name:        "generic with local types",
			typeSpec:    "Converter[User,UserDTO]",
			expectCount: 2,
			expectTypes: []struct {
				packageAlias string
				typeName     string
				isLocal      bool
			}{
				{packageAlias: "", typeName: "User", isLocal: true},
				{packageAlias: "", typeName: "UserDTO", isLocal: true},
			},
			wantErr: false,
		},
		{
			name:        "generic with cross-package types",
			typeSpec:    "TypeMapper[models.User,dto.UserDTO]",
			expectCount: 2,
			expectTypes: []struct {
				packageAlias string
				typeName     string
				isLocal      bool
			}{
				{packageAlias: "models", typeName: "User", isLocal: false},
				{packageAlias: "dto", typeName: "UserDTO", isLocal: false},
			},
			wantErr: false,
		},
		{
			name:        "mixed local and external types",
			typeSpec:    "Converter[User,dto.UserDTO]",
			expectCount: 2,
			expectTypes: []struct {
				packageAlias string
				typeName     string
				isLocal      bool
			}{
				{packageAlias: "", typeName: "User", isLocal: true},
				{packageAlias: "dto", typeName: "UserDTO", isLocal: false},
			},
			wantErr: false,
		},
		{
			name:        "unknown package alias",
			typeSpec:    "Converter[unknown.Type,User]",
			expectCount: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qualifiedTypes, err := cpr.ParseTypeArguments(ctx, tt.typeSpec)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, qualifiedTypes, tt.expectCount)

			for i, expected := range tt.expectTypes {
				require.True(t, i < len(qualifiedTypes), "not enough qualified types returned")
				
				qt := qualifiedTypes[i]
				assert.Equal(t, expected.packageAlias, qt.PackageAlias, "package alias mismatch at index %d", i)
				assert.Equal(t, expected.typeName, qt.TypeName, "type name mismatch at index %d", i)
				assert.Equal(t, expected.isLocal, qt.IsLocal, "isLocal mismatch at index %d", i)
			}
		})
	}
}

func TestCrossPackageResolver_parseQualifiedTypeName(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	cpr := NewCrossPackageResolver(nil, importMap, logger)

	tests := []struct {
		name         string
		typeName     string
		expectLocal  bool
		expectAlias  string
		expectType   string
		expectPath   string
		wantErr      bool
		expectErr    error
	}{
		{
			name:        "local type",
			typeName:    "User",
			expectLocal: true,
			expectAlias: "",
			expectType:  "User",
			expectPath:  "",
			wantErr:     false,
		},
		{
			name:        "qualified type",
			typeName:    "models.User",
			expectLocal: false,
			expectAlias: "models",
			expectType:  "User",
			expectPath:  "./internal/models",
			wantErr:     false,
		},
		{
			name:        "unknown package alias",
			typeName:    "unknown.Type",
			wantErr:     true,
			expectErr:   ErrPackageAliasNotFound,
		},
		{
			name:        "too many dots",
			typeName:    "pkg.sub.Type",
			wantErr:     true,
			expectErr:   ErrInvalidQualifiedTypeName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qt, err := cpr.parseQualifiedTypeName(tt.typeName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != nil {
					assert.ErrorIs(t, err, tt.expectErr)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, qt)
			assert.Equal(t, tt.expectLocal, qt.IsLocal)
			assert.Equal(t, tt.expectAlias, qt.PackageAlias)
			assert.Equal(t, tt.expectType, qt.TypeName)
			assert.Equal(t, tt.expectPath, qt.ImportPath)
		})
	}
}

func TestCrossPackageResolver_splitTypeArguments(t *testing.T) {
	logger := zap.NewNop()
	cpr := NewCrossPackageResolver(nil, nil, logger)

	tests := []struct {
		name         string
		input        string
		expected     []string
	}{
		{
			name:     "simple types",
			input:    "User,UserDTO",
			expected: []string{"User", "UserDTO"},
		},
		{
			name:     "qualified types",
			input:    "models.User,dto.UserDTO",
			expected: []string{"models.User", "dto.UserDTO"},
		},
		{
			name:     "mixed types with spaces",
			input:    "User, dto.UserDTO, models.Entity",
			expected: []string{"User", " dto.UserDTO", " models.Entity"},
		},
		{
			name:     "nested generics",
			input:    "Generic[T,U],Simple",
			expected: []string{"Generic[T,U]", "Simple"},
		},
		{
			name:     "complex nested",
			input:    "Mapper[Generic[T,U],Simple],models.User",
			expected: []string{"Mapper[Generic[T,U],Simple]", "models.User"},
		},
		{
			name:     "single type",
			input:    "User",
			expected: []string{"User"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cpr.splitTypeArguments(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCrossPackageResolver_UpdateImportMap(t *testing.T) {
	logger := zap.NewNop()
	initialImports := map[string]string{
		"models": "./internal/models",
	}

	cpr := NewCrossPackageResolver(nil, initialImports, logger)

	newImports := map[string]string{
		"dto":      "./pkg/dto",
		"services": "./internal/services",
	}

	cpr.UpdateImportMap(newImports)

	// Verify initial imports are preserved
	assert.Equal(t, "./internal/models", cpr.importMap["models"])

	// Verify new imports are added
	assert.Equal(t, "./pkg/dto", cpr.importMap["dto"])
	assert.Equal(t, "./internal/services", cpr.importMap["services"])

	// Verify we have the expected total count
	assert.Equal(t, 3, len(cpr.importMap))
}

func TestCrossPackageResolver_ValidateQualifiedTypes(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	cpr := NewCrossPackageResolver(nil, importMap, logger)

	tests := []struct {
		name          string
		qualifiedTypes []*QualifiedType
		wantErr       bool
		expectErr     error
	}{
		{
			name: "valid types",
			qualifiedTypes: []*QualifiedType{
				{PackageAlias: "", TypeName: "User", ImportPath: "", IsLocal: true},
				{PackageAlias: "models", TypeName: "Entity", ImportPath: "./internal/models", IsLocal: false},
			},
			wantErr: false,
		},
		{
			name: "nil type in slice",
			qualifiedTypes: []*QualifiedType{
				{PackageAlias: "", TypeName: "User", ImportPath: "", IsLocal: true},
				nil,
			},
			wantErr: true,
		},
		{
			name: "empty type name",
			qualifiedTypes: []*QualifiedType{
				{PackageAlias: "", TypeName: "", ImportPath: "", IsLocal: true},
			},
			wantErr: true,
		},
		{
			name: "external type with empty package alias",
			qualifiedTypes: []*QualifiedType{
				{PackageAlias: "", TypeName: "User", ImportPath: "./models", IsLocal: false},
			},
			wantErr: true,
		},
		{
			name: "external type with empty import path",
			qualifiedTypes: []*QualifiedType{
				{PackageAlias: "models", TypeName: "User", ImportPath: "", IsLocal: false},
			},
			wantErr: true,
		},
		{
			name: "unknown package alias",
			qualifiedTypes: []*QualifiedType{
				{PackageAlias: "unknown", TypeName: "Type", ImportPath: "./unknown", IsLocal: false},
			},
			wantErr:   true,
			expectErr: ErrPackageAliasNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cpr.ValidateQualifiedTypes(tt.qualifiedTypes)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != nil {
					assert.ErrorIs(t, err, tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCrossPackageResolver_validateImportPath(t *testing.T) {
	logger := zap.NewNop()
	cpr := NewCrossPackageResolver(nil, nil, logger)

	tests := []struct {
		name       string
		importPath string
		wantErr    bool
		expectErr  error
	}{
		{
			name:       "valid path",
			importPath: "./internal/models",
			wantErr:    false,
		},
		{
			name:       "valid package path",
			importPath: "github.com/user/repo/pkg/models",
			wantErr:    false,
		},
		{
			name:       "empty path",
			importPath: "",
			wantErr:    true,
			expectErr:  ErrInvalidImportPath,
		},
		{
			name:       "path with spaces",
			importPath: "./internal models",
			wantErr:    true,
			expectErr:  ErrInvalidImportPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cpr.validateImportPath(tt.importPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != nil {
					assert.ErrorIs(t, err, tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCrossPackageResolver_GetImportMap(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	cpr := NewCrossPackageResolver(nil, importMap, logger)

	// Get import map copy
	result := cpr.GetImportMap()

	// Verify content is the same
	assert.Equal(t, importMap, result)

	// Verify it's a copy (modifying result doesn't affect original)
	result["test"] = "./test"
	assert.NotEqual(t, len(importMap), len(result))
	assert.Equal(t, len(importMap), len(cpr.importMap))
}

func TestCrossPackageResolver_checkCircularDependency(t *testing.T) {
	logger := zap.NewNop()
	cpr := NewCrossPackageResolver(nil, nil, logger)

	// Set up a loading stack to simulate circular dependency
	cpr.loadingStack = []string{"./pkg/a", "./pkg/b"}

	tests := []struct {
		name       string
		importPath string
		wantErr    bool
		expectErr  error
	}{
		{
			name:       "no circular dependency",
			importPath: "./pkg/c",
			wantErr:    false,
		},
		{
			name:       "circular dependency detected",
			importPath: "./pkg/a",
			wantErr:    true,
			expectErr:  ErrCircularPackageDependency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cpr.checkCircularDependency(tt.importPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != nil {
					assert.ErrorIs(t, err, tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPackageLoadCache(t *testing.T) {
	cache := NewPackageLoadCache()

	// Test empty cache
	_, exists := cache.Get("test")
	assert.False(t, exists)

	// Test set and get
	// Note: We can't create a real packages.Package in tests easily,
	// so we'll test with nil for basic cache functionality
	cache.Set("test", nil)
	
	result, exists := cache.Get("test")
	assert.True(t, exists)
	assert.Nil(t, result) // We stored nil for testing
}

func TestGetReflectKind(t *testing.T) {
	tests := []struct {
		typeName string
		expected reflect.Kind
	}{
		{"bool", reflect.Bool},
		{"int", reflect.Int},
		{"string", reflect.String},
		{"interface", reflect.Interface},
		{"struct", reflect.Struct},
		{"CustomType", reflect.Struct}, // Default to struct
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := getReflectKind(tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCrossPackageResolverWithConfig(t *testing.T) {
	logger := zap.NewNop()
	importMap := map[string]string{
		"models": "./internal/models",
	}
	config := &CrossPackageResolverConfig{
		Timeout:      10 * time.Second,
		MaxCacheSize: 50,
		MaxWorkers:   3,
	}

	cpr := NewCrossPackageResolverWithConfig(nil, importMap, logger, config)

	require.NotNil(t, cpr)
	assert.Equal(t, config.Timeout, cpr.timeout)
	assert.Equal(t, config.MaxCacheSize, cpr.maxCacheSize)
	assert.NotNil(t, cpr.packageLoader)
	assert.NotNil(t, cpr.cache)
}

func TestCopyImportMap(t *testing.T) {
	original := map[string]string{
		"models": "./internal/models",
		"dto":    "./pkg/dto",
	}

	// Test with non-nil map
	copied := copyImportMap(original)
	assert.Equal(t, original, copied)

	// Verify it's a copy
	copied["test"] = "./test"
	assert.NotEqual(t, len(original), len(copied))

	// Test with nil map
	nilCopy := copyImportMap(nil)
	assert.NotNil(t, nilCopy)
	assert.Equal(t, 0, len(nilCopy))
}