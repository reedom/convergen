package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_parseImportMap(t *testing.T) {
	tests := []struct {
		name       string
		importsStr string
		expected   map[string]string
		wantErr    bool
		expectErr  string
	}{
		{
			name:       "empty imports",
			importsStr: "",
			expected:   map[string]string{},
			wantErr:    false,
		},
		{
			name:       "single import mapping",
			importsStr: "models=./internal/models",
			expected: map[string]string{
				"models": "./internal/models",
			},
			wantErr: false,
		},
		{
			name:       "multiple import mappings",
			importsStr: "models=./internal/models,dto=./pkg/dto",
			expected: map[string]string{
				"models": "./internal/models",
				"dto":    "./pkg/dto",
			},
			wantErr: false,
		},
		{
			name:       "mappings with spaces",
			importsStr: "models = ./internal/models , dto = ./pkg/dto",
			expected: map[string]string{
				"models": "./internal/models",
				"dto":    "./pkg/dto",
			},
			wantErr: false,
		},
		{
			name:       "invalid format missing equals",
			importsStr: "models./internal/models",
			wantErr:    true,
			expectErr:  "invalid import mapping format",
		},
		{
			name:       "empty alias",
			importsStr: "=./internal/models",
			wantErr:    true,
			expectErr:  "empty package alias",
		},
		{
			name:       "empty import path",
			importsStr: "models=",
			wantErr:    true,
			expectErr:  "empty import path",
		},
		{
			name:       "invalid alias with numbers at start",
			importsStr: "1models=./internal/models",
			wantErr:    true,
			expectErr:  "valid Go identifier",
		},
		{
			name:       "import path with spaces",
			importsStr: "models=./internal models",
			wantErr:    true,
			expectErr:  "cannot contain spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			result, err := c.parseImportMap(tt.importsStr)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_validatePackageAlias(t *testing.T) {
	tests := []struct {
		name      string
		alias     string
		wantErr   bool
		expectErr string
	}{
		{
			name:    "valid simple alias",
			alias:   "models",
			wantErr: false,
		},
		{
			name:    "valid alias with underscore",
			alias:   "data_models",
			wantErr: false,
		},
		{
			name:    "valid alias starting with underscore",
			alias:   "_internal",
			wantErr: false,
		},
		{
			name:    "valid alias with numbers",
			alias:   "models2",
			wantErr: false,
		},
		{
			name:      "empty alias",
			alias:     "",
			wantErr:   true,
			expectErr: "cannot be empty",
		},
		{
			name:      "alias starting with number",
			alias:     "1models",
			wantErr:   true,
			expectErr: "valid Go identifier",
		},
		{
			name:      "alias with spaces",
			alias:     "data models",
			wantErr:   true,
			expectErr: "valid Go identifier",
		},
		{
			name:      "alias with special characters",
			alias:     "models-data",
			wantErr:   true,
			expectErr: "valid Go identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			err := c.validatePackageAlias(tt.alias)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateImportPath(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		wantErr    bool
		expectErr  string
	}{
		{
			name:       "valid relative path",
			importPath: "./internal/models",
			wantErr:    false,
		},
		{
			name:       "valid absolute package path",
			importPath: "github.com/user/repo/pkg/models",
			wantErr:    false,
		},
		{
			name:       "valid simple path",
			importPath: "models",
			wantErr:    false,
		},
		{
			name:       "empty path",
			importPath: "",
			wantErr:    true,
			expectErr:  "cannot be empty",
		},
		{
			name:       "path with spaces",
			importPath: "./internal models",
			wantErr:    true,
			expectErr:  "cannot contain spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			err := c.validateImportPath(tt.importPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_hasQualifiedTypes(t *testing.T) {
	tests := []struct {
		name     string
		typeSpec string
		expected bool
	}{
		{
			name:     "no type spec",
			typeSpec: "",
			expected: false,
		},
		{
			name:     "simple type",
			typeSpec: "Converter",
			expected: false,
		},
		{
			name:     "generic with local types",
			typeSpec: "Converter[User,UserDTO]",
			expected: false,
		},
		{
			name:     "generic with qualified types",
			typeSpec: "TypeMapper[models.User,dto.UserDTO]",
			expected: true,
		},
		{
			name:     "mixed local and qualified types",
			typeSpec: "Converter[User,dto.UserDTO]",
			expected: true,
		},
		{
			name:     "no brackets but has dot",
			typeSpec: "models.Converter",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{TypeSpec: tt.typeSpec}
			result := c.hasQualifiedTypes()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_getInterfaceName(t *testing.T) {
	tests := []struct {
		name     string
		typeSpec string
		expected string
	}{
		{
			name:     "empty type spec",
			typeSpec: "",
			expected: "",
		},
		{
			name:     "simple interface name",
			typeSpec: "Converter",
			expected: "Converter",
		},
		{
			name:     "generic interface",
			typeSpec: "TypeMapper[User,UserDTO]",
			expected: "TypeMapper",
		},
		{
			name:     "generic interface with spaces",
			typeSpec: " TypeMapper [User,UserDTO] ",
			expected: "TypeMapper",
		},
		{
			name:     "qualified interface name",
			typeSpec: "pkg.Converter[User,UserDTO]",
			expected: "pkg.Converter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{TypeSpec: tt.typeSpec}
			result := c.getInterfaceName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_isValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{
			name:     "valid simple identifier",
			id:       "Converter",
			expected: true,
		},
		{
			name:     "valid identifier with underscore",
			id:       "Type_Mapper",
			expected: true,
		},
		{
			name:     "valid identifier starting with underscore",
			id:       "_internal",
			expected: true,
		},
		{
			name:     "valid identifier with numbers",
			id:       "Converter2",
			expected: true,
		},
		{
			name:     "empty identifier",
			id:       "",
			expected: false,
		},
		{
			name:     "identifier starting with number",
			id:       "2Converter",
			expected: false,
		},
		{
			name:     "identifier with spaces",
			id:       "Type Mapper",
			expected: false,
		},
		{
			name:     "identifier with special characters",
			id:       "Type-Mapper",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			result := c.isValidIdentifier(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_HasCrossPackageTypes(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "no type spec",
			config:   Config{},
			expected: false,
		},
		{
			name: "local types only",
			config: Config{
				TypeSpec: "Converter[User,UserDTO]",
			},
			expected: false,
		},
		{
			name: "qualified types",
			config: Config{
				TypeSpec: "TypeMapper[models.User,dto.UserDTO]",
			},
			expected: true,
		},
		{
			name: "mixed types",
			config: Config{
				TypeSpec: "Converter[User,dto.UserDTO]",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasCrossPackageTypes()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetPackageAlias(t *testing.T) {
	config := Config{
		ImportMap: map[string]string{
			"models": "./internal/models",
			"dto":    "./pkg/dto",
		},
	}

	tests := []struct {
		name         string
		alias        string
		expectedPath string
		expectedOk   bool
	}{
		{
			name:         "existing alias",
			alias:        "models",
			expectedPath: "./internal/models",
			expectedOk:   true,
		},
		{
			name:         "non-existing alias",
			alias:        "unknown",
			expectedPath: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := config.GetPackageAlias(tt.alias)
			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}

	// Test with nil ImportMap
	configNil := Config{ImportMap: nil}
	path, ok := configNil.GetPackageAlias("models")
	assert.Equal(t, "", path)
	assert.False(t, ok)
}

func TestConfig_validateCrossPackageConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantErr   bool
		expectErr string
	}{
		{
			name:    "no cross-package config",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "local types only",
			config: Config{
				TypeSpec: "Converter[User,UserDTO]",
			},
			wantErr: false,
		},
		{
			name: "qualified types with import map",
			config: Config{
				TypeSpec: "TypeMapper[models.User,dto.UserDTO]",
				ImportMap: map[string]string{
					"models": "./internal/models",
					"dto":    "./pkg/dto",
				},
			},
			wantErr: false,
		},
		{
			name: "qualified types without import map",
			config: Config{
				TypeSpec: "TypeMapper[models.User,dto.UserDTO]",
			},
			wantErr:   true,
			expectErr: "no import mappings provided",
		},
		{
			name: "import map without type spec",
			config: Config{
				ImportMap: map[string]string{
					"models": "./internal/models",
				},
			},
			wantErr: false, // This is valid
		},
		{
			name: "invalid interface name",
			config: Config{
				TypeSpec: "2InvalidName[User,UserDTO]",
			},
			wantErr:   true,
			expectErr: "invalid interface name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateCrossPackageConfig()

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	config := Config{
		Input:    "input.go",
		Output:   "output.gen.go",
		Log:      "output.log",
		TypeSpec: "TypeMapper[models.User,dto.UserDTO]",
		ImportMap: map[string]string{
			"models": "./internal/models",
			"dto":    "./pkg/dto",
		},
	}

	result := config.String()

	// Verify essential parts are included
	assert.Contains(t, result, "Input: \"input.go\"")
	assert.Contains(t, result, "Output: \"output.gen.go\"")
	assert.Contains(t, result, "Log: \"output.log\"")
	assert.Contains(t, result, "TypeSpec: \"TypeMapper[models.User,dto.UserDTO]\"")
	assert.Contains(t, result, "ImportMap:")
}

func TestFormatImportMap(t *testing.T) {
	tests := []struct {
		name      string
		importMap map[string]string
		expected  string
	}{
		{
			name:      "empty map",
			importMap: map[string]string{},
			expected:  "{}",
		},
		{
			name: "single mapping",
			importMap: map[string]string{
				"models": "./internal/models",
			},
			expected: `{models: "./internal/models"}`,
		},
		{
			name: "multiple mappings",
			importMap: map[string]string{
				"models": "./internal/models",
				"dto":    "./pkg/dto",
			},
			// Note: order is not guaranteed in Go maps, so we test inclusion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatImportMap(tt.importMap)

			if len(tt.importMap) <= 1 {
				assert.Equal(t, tt.expected, result)
			} else {
				// For multiple mappings, just check structure and content
				assert.True(t, strings.HasPrefix(result, "{"))
				assert.True(t, strings.HasSuffix(result, "}"))
				for alias, path := range tt.importMap {
					assert.Contains(t, result, alias)
					assert.Contains(t, result, path)
				}
			}
		})
	}
}

// Test struct literal configuration validation.
func TestConfig_validateStructLiteralConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantErr   bool
		expectErr string
	}{
		{
			name:    "no flags set",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "only no-struct-literal set",
			config: Config{
				NoStructLiteral: true,
			},
			wantErr: false,
		},
		{
			name: "only struct-literal set",
			config: Config{
				StructLiteral: true,
			},
			wantErr: false,
		},
		{
			name: "both flags set - conflict",
			config: Config{
				NoStructLiteral: true,
				StructLiteral:   true,
			},
			wantErr:   true,
			expectErr: "cannot specify both",
		},
		{
			name: "verbose with no-struct-literal",
			config: Config{
				NoStructLiteral: true,
				Verbose:         true,
			},
			wantErr: false,
		},
		{
			name: "verbose with struct-literal",
			config: Config{
				StructLiteral: true,
				Verbose:       true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateStructLiteralConfig()

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test struct literal helper methods.
func TestConfig_StructLiteralHelpers(t *testing.T) {
	tests := []struct {
		name                    string
		config                  Config
		expectDisabled          bool
		expectExplicitlyEnabled bool
		expectVerbose           bool
	}{
		{
			name:                    "default config",
			config:                  Config{},
			expectDisabled:          false,
			expectExplicitlyEnabled: false,
			expectVerbose:           false,
		},
		{
			name: "no-struct-literal enabled",
			config: Config{
				NoStructLiteral: true,
			},
			expectDisabled:          true,
			expectExplicitlyEnabled: false,
			expectVerbose:           false,
		},
		{
			name: "struct-literal enabled",
			config: Config{
				StructLiteral: true,
			},
			expectDisabled:          false,
			expectExplicitlyEnabled: true,
			expectVerbose:           false,
		},
		{
			name: "verbose enabled",
			config: Config{
				Verbose: true,
			},
			expectDisabled:          false,
			expectExplicitlyEnabled: false,
			expectVerbose:           true,
		},
		{
			name: "all flags set",
			config: Config{
				NoStructLiteral: true,
				StructLiteral:   false, // This should be false to avoid conflict
				Verbose:         true,
			},
			expectDisabled:          true,
			expectExplicitlyEnabled: false,
			expectVerbose:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectDisabled, tt.config.IsStructLiteralDisabled())
			assert.Equal(t, tt.expectExplicitlyEnabled, tt.config.IsStructLiteralExplicitlyEnabled())
			assert.Equal(t, tt.expectVerbose, tt.config.IsVerboseMode())
		})
	}
}

// Test string representation includes new fields.
func TestConfig_StringWithStructLiteralFields(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		contains []string
	}{
		{
			name:     "default config",
			config:   Config{Input: "test.go", Output: "test.gen.go"},
			contains: []string{"Input: \"test.go\"", "Output: \"test.gen.go\""},
		},
		{
			name: "with no-struct-literal",
			config: Config{
				Input:           "test.go",
				Output:          "test.gen.go",
				NoStructLiteral: true,
			},
			contains: []string{"Input: \"test.go\"", "NoStructLiteral: true"},
		},
		{
			name: "with struct-literal",
			config: Config{
				Input:         "test.go",
				Output:        "test.gen.go",
				StructLiteral: true,
			},
			contains: []string{"Input: \"test.go\"", "StructLiteral: true"},
		},
		{
			name: "with verbose",
			config: Config{
				Input:   "test.go",
				Output:  "test.gen.go",
				Verbose: true,
			},
			contains: []string{"Input: \"test.go\"", "Verbose: true"},
		},
		{
			name: "with all struct literal flags",
			config: Config{
				Input:           "test.go",
				Output:          "test.gen.go",
				NoStructLiteral: false, // Only one should be true to avoid conflict
				StructLiteral:   true,
				Verbose:         true,
			},
			contains: []string{"Input: \"test.go\"", "StructLiteral: true", "Verbose: true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.String()
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}
