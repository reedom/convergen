package emitter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewCodeValidator(t *testing.T) {
	config := DefaultConfig()
	logger := zap.NewNop()

	validator := NewCodeValidator(config, logger)
	require.NotNil(t, validator)

	concreteValidator, ok := validator.(*ConcreteCodeValidator)
	require.True(t, ok)
	assert.NotNil(t, concreteValidator.fileSet)
	assert.NotNil(t, concreteValidator.logger)
	assert.NotNil(t, concreteValidator.config)
	assert.NotNil(t, concreteValidator.metrics)
}

func TestCodeValidator_ValidateSyntax(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_function",
			code: `func Convert(src *Source) (*Dest, error) {
				return &Dest{
					Field1: src.Field1,
					Field2: src.Field2,
				}, nil
			}`,
			wantErr: false,
		},
		{
			name: "syntax_error_missing_brace",
			code: `func Convert(src *Source) (*Dest, error) {
				return &Dest{
					Field1: src.Field1,
					Field2: src.Field2,
				}, nil`,
			wantErr: true,
			errMsg:  "syntax error",
		},
		{
			name: "syntax_error_invalid_syntax",
			code: `func Convert(src *Source) (*Dest, error) {
				return &Dest{
					Field1: src.Field1 ;;; invalid
				}, nil
			}`,
			wantErr: true,
			errMsg:  "syntax error",
		},
	}

	config := DefaultConfig()
	config.EnableSyntaxValidation = true
	validator := NewCodeValidator(config, zap.NewNop()).(*ConcreteCodeValidator)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSyntax(tt.code)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodeValidator_ValidateSemantics(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_semantics",
			code: `func Convert(src *Source) (*Dest, error) {
				if src == nil {
					return nil, fmt.Errorf("source is nil")
				}
				return &Dest{
					Field1: src.Field1,
				}, nil
			}
			
			type Source struct {
				Field1 string
			}
			
			type Dest struct {
				Field1 string
			}`,
			wantErr: false,
		},
		{
			name: "undefined_type",
			code: `func Convert(src *Source) (*UndefinedType, error) {
				return &UndefinedType{}, nil
			}`,
			wantErr: true,
			errMsg:  "semantic validation failed",
		},
	}

	config := DefaultConfig()
	config.EnableSemanticValidation = true
	validator := NewCodeValidator(config, zap.NewNop()).(*ConcreteCodeValidator)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSemantics(tt.code)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodeValidator_ValidateMethod(t *testing.T) {
	tests := []struct {
		name       string
		methodCode *MethodCode
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid_method",
			methodCode: &MethodCode{
				Name:      "ConvertSourceToDest",
				Signature: "func ConvertSourceToDest(src *Source) (*Dest, error)",
				Body: `	if src == nil {
		return nil, fmt.Errorf("source is nil")
	}
	return &Dest{
		Field1: src.Field1,
	}, nil`,
				Imports: []*Import{
					{Path: "fmt", Standard: true, Used: true},
				},
				Strategy: StrategyAssignmentBlock,
			},
			wantErr: false,
		},
		{
			name:       "nil_method",
			methodCode: nil,
			wantErr:    true,
			errMsg:     "method code is nil",
		},
		{
			name: "invalid_signature",
			methodCode: &MethodCode{
				Name:      "InvalidMethod",
				Signature: "func InvalidMethod(invalid syntax here",
				Body:      "return nil",
				Strategy:  StrategyAssignmentBlock,
			},
			wantErr: true,
			errMsg:  "method validation failed",
		},
	}

	config := DefaultConfig()
	config.EnableSyntaxValidation = true
	validator := NewCodeValidator(config, zap.NewNop())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateMethod(tt.methodCode)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodeValidator_ValidateCompleteCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "syntax_only_validation",
			code: `package main

import "fmt"

func Convert(src *Source) (*Dest, error) {
	return &Dest{Field1: src.Field1}, nil
}`,
			config: &Config{
				EnableSyntaxValidation:   true,
				EnableSemanticValidation: false,
				EnableMemoryCompilation:  false,
				ValidationTimeout:        5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "all_validations_enabled",
			code: `package main

import "fmt"

func Convert(src *Source) (*Dest, error) {
	if src == nil {
		return nil, fmt.Errorf("source is nil")
	}
	return &Dest{Field1: src.Field1}, nil
}

type Source struct {
	Field1 string
}

type Dest struct {
	Field1 string
}`,
			config: &Config{
				EnableSyntaxValidation:   true,
				EnableSemanticValidation: true,
				EnableMemoryCompilation:  true,
				ValidationTimeout:        10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "syntax_error_detected",
			code: `package main

func Convert(src *Source) (*Dest, error) {
	return &Dest{Field1: src.Field1} // missing comma and nil
}`,
			config: &Config{
				EnableSyntaxValidation: true,
				ValidationTimeout:      5 * time.Second,
			},
			wantErr: true,
			errMsg:  "syntax validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCodeValidator(tt.config, zap.NewNop())

			err := validator.Validate(tt.code)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodeValidator_ValidateStructuralElements(t *testing.T) {
	config := DefaultConfig()
	validator := NewCodeValidator(config, zap.NewNop()).(*ConcreteCodeValidator)

	t.Run("balanced_braces", func(t *testing.T) {
		tests := []struct {
			name    string
			code    string
			wantErr bool
		}{
			{"balanced", "{ { } }", false},
			{"unbalanced_extra_open", "{ { }", true},
			{"unbalanced_extra_close", "{ } }", true},
			{"empty", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validator.validateBalancedBraces(tt.code)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("return_statements", func(t *testing.T) {
		tests := []struct {
			name     string
			code     string
			hasWarning bool
		}{
			{"has_return", "return nil", false},
			{"no_return", "fmt.Println(\"hello\")", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Reset metrics before test
				validator.metrics = &ValidationMetrics{}
				
				err := validator.validateReturnStatements(tt.code)
				assert.NoError(t, err) // This function doesn't return errors, just warnings
				
				if tt.hasWarning {
					assert.Greater(t, validator.metrics.WarningsFound, 0)
				}
			})
		}
	})

	t.Run("import_validation", func(t *testing.T) {
		tests := []struct {
			name    string
			imports []*Import
			wantErr bool
		}{
			{
				name: "valid_imports",
				imports: []*Import{
					{Path: "fmt", Standard: true},
					{Path: "github.com/user/repo", Standard: false},
				},
				wantErr: false,
			},
			{
				name: "empty_import_path",
				imports: []*Import{
					{Path: "", Standard: true},
				},
				wantErr: true,
			},
			{
				name: "import_with_spaces",
				imports: []*Import{
					{Path: "invalid path", Standard: false},
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validator.validateMethodImports(tt.imports)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestCodeValidator_Metrics(t *testing.T) {
	config := DefaultConfig()
	validator := NewCodeValidator(config, zap.NewNop()).(*ConcreteCodeValidator)

	// Test metrics tracking
	initialErrors := validator.metrics.ErrorsFound
	_ = validator.metrics.WarningsFound

	// Trigger some validation that should update metrics
	validCode := `func Test() { return }`
	invalidCode := `func Test() { return } }`

	err := validator.Validate(validCode)
	assert.NoError(t, err)

	err = validator.Validate(invalidCode)
	assert.Error(t, err)

	// Check that metrics were updated
	assert.Greater(t, validator.metrics.ErrorsFound, initialErrors)
	assert.Greater(t, validator.metrics.ValidationTime, time.Duration(0))

	// Test metrics retrieval
	metrics := validator.GetValidationMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, validator.metrics, metrics)

	// Test metrics reset
	validator.ResetMetrics()
	assert.Equal(t, 0, validator.metrics.ErrorsFound)
	assert.Equal(t, 0, validator.metrics.WarningsFound)
	assert.Equal(t, time.Duration(0), validator.metrics.ValidationTime)
}

func TestCodeValidator_ConstructCompleteMethodCode(t *testing.T) {
	config := DefaultConfig()
	validator := NewCodeValidator(config, zap.NewNop()).(*ConcreteCodeValidator)

	method := &MethodCode{
		Name:      "ConvertData",
		Signature: "func ConvertData(src *Source) (*Dest, error)",
		Body:      "return &Dest{Field: src.Field}, nil",
		Documentation: "// ConvertData converts Source to Dest\n",
		Imports: []*Import{
			{Path: "fmt", Standard: true},
			{Path: "errors", Standard: true},
		},
	}

	code := validator.constructCompleteMethodCode(method)

	// Verify the constructed code contains expected elements
	assert.Contains(t, code, "package main")
	assert.Contains(t, code, "import (")
	assert.Contains(t, code, "\"fmt\"")
	assert.Contains(t, code, "\"errors\"")
	assert.Contains(t, code, method.Documentation)
	assert.Contains(t, code, method.Signature)
	assert.Contains(t, code, method.Body)

	// Verify it's valid Go code by attempting to parse it
	err := validator.validateSyntax(code)
	assert.NoError(t, err)
}

func TestCodeValidator_ConfigurationOptions(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		validCode      string
		expectedChecks []string
	}{
		{
			name: "syntax_only",
			config: &Config{
				EnableSyntaxValidation:   true,
				EnableSemanticValidation: false,
				EnableMemoryCompilation:  false,
			},
			validCode: "func Test() { return }",
			expectedChecks: []string{"syntax"},
		},
		{
			name: "syntax_and_semantic",
			config: &Config{
				EnableSyntaxValidation:   true,
				EnableSemanticValidation: true,
				EnableMemoryCompilation:  false,
			},
			validCode: "func Test() { return }",
			expectedChecks: []string{"syntax", "semantic"},
		},
		{
			name: "all_validations",
			config: &Config{
				EnableSyntaxValidation:   true,
				EnableSemanticValidation: true,
				EnableMemoryCompilation:  true,
			},
			validCode: "func Test() { return }",
			expectedChecks: []string{"syntax", "semantic", "memory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCodeValidator(tt.config, zap.NewNop())
			
			err := validator.Validate(tt.validCode)
			
			// For now, we just verify no errors occur with valid code
			// In a more comprehensive test, we'd verify which validation paths were taken
			assert.NoError(t, err)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkCodeValidator_ValidateSyntax(b *testing.B) {
	config := DefaultConfig()
	validator := NewCodeValidator(config, zap.NewNop())

	code := `func Convert(src *Source) (*Dest, error) {
		if src == nil {
			return nil, fmt.Errorf("source is nil")
		}
		return &Dest{
			Field1: src.Field1,
			Field2: src.Field2,
			Field3: src.Field3,
		}, nil
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate(code)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodeValidator_ValidateMethod(b *testing.B) {
	config := DefaultConfig()
	validator := NewCodeValidator(config, zap.NewNop())

	method := &MethodCode{
		Name:      "Convert",
		Signature: "func Convert(src *Source) (*Dest, error)",
		Body: `	if src == nil {
		return nil, fmt.Errorf("source is nil")
	}
	return &Dest{
		Field1: src.Field1,
		Field2: src.Field2,
	}, nil`,
		Imports: []*Import{
			{Path: "fmt", Standard: true, Used: true},
		},
		Strategy: StrategyAssignmentBlock,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateMethod(method)
		if err != nil {
			b.Fatal(err)
		}
	}
}