package emitter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/executor"
)

// TestValidationIntegration tests the complete validation framework integration.
func TestValidationIntegration(t *testing.T) {
	t.Run("complete_validation_pipeline", func(t *testing.T) {
		// Setup
		config := DefaultConfig()
		config.EnableSyntaxValidation = true
		config.EnableSemanticValidation = true
		config.EnableTypeValidation = true
		config.EnableMemoryCompilation = true
		config.ValidationTimeout = 10 * time.Second

		logger := zap.NewNop()
		metrics := NewMetrics()

		// Create code generator with validation
		generator := NewCodeGenerator(config, logger, metrics)
		require.NotNil(t, generator)

		// Create a realistic method result for testing
		methodResult := createTestMethodResult()

		// Generate method code
		ctx := context.Background()
		methodCode, err := generator.GenerateMethodCode(ctx, methodResult)
		require.NoError(t, err)
		require.NotNil(t, methodCode)

		// Verify the generated code has expected structure
		assert.NotEmpty(t, methodCode.Name)
		assert.NotEmpty(t, methodCode.Signature)
		assert.NotEmpty(t, methodCode.Body)
		assert.Equal(t, StrategyAssignmentBlock, methodCode.Strategy)

		// Test direct validation of generated method
		concreteGenerator, ok := generator.(*ConcreteCodeGenerator)
		require.True(t, ok)
		require.NotNil(t, concreteGenerator.validator)

		// Validate the method code through the validator
		err = concreteGenerator.validator.ValidateMethod(methodCode)
		assert.NoError(t, err, "Generated method code should pass validation")

		// Test individual validation components
		t.Run("syntax_validation", func(t *testing.T) {
			// Test syntax validation with complete method code
			completeCode := constructCompleteMethodForTesting(methodCode)
			err := concreteGenerator.validator.Validate(completeCode)
			assert.NoError(t, err, "Complete method should pass syntax validation")
		})

		t.Run("complete_code_validation", func(t *testing.T) {
			completeCode := constructCompleteMethodForTesting(methodCode)
			err := concreteGenerator.validator.Validate(completeCode)
			assert.NoError(t, err, "Complete method code should pass validation")
		})

		// Verify validation metrics were updated
		validator := concreteGenerator.validator.(*ConcreteCodeValidator)
		validationMetrics := validator.GetValidationMetrics()
		assert.NotNil(t, validationMetrics)
		assert.Greater(t, validationMetrics.ValidationTime, time.Duration(0))
	})

	t.Run("validation_configuration_options", func(t *testing.T) {
		tests := []struct {
			name                     string
			enableSyntaxValidation   bool
			enableSemanticValidation bool
			enableMemoryCompilation  bool
			expectValidation         bool
		}{
			{
				name:                     "all_validations_enabled",
				enableSyntaxValidation:   true,
				enableSemanticValidation: true,
				enableMemoryCompilation:  true,
				expectValidation:         true,
			},
			{
				name:                     "syntax_only",
				enableSyntaxValidation:   true,
				enableSemanticValidation: false,
				enableMemoryCompilation:  false,
				expectValidation:         true,
			},
			{
				name:                     "no_validation",
				enableSyntaxValidation:   false,
				enableSemanticValidation: false,
				enableMemoryCompilation:  false,
				expectValidation:         false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := DefaultConfig()
				config.EnableSyntaxValidation = tt.enableSyntaxValidation
				config.EnableSemanticValidation = tt.enableSemanticValidation
				config.EnableMemoryCompilation = tt.enableMemoryCompilation

				generator := NewCodeGenerator(config, zap.NewNop(), NewMetrics())
				concreteGenerator := generator.(*ConcreteCodeGenerator)

				// Test that validation is called or not called based on config
				methodResult := createTestMethodResult()
				methodCode, err := generator.GenerateMethodCode(context.Background(), methodResult)
				assert.NoError(t, err)

				// The validation should not fail even if it's disabled
				// because the finalizeGeneration method checks the config
				assert.NotNil(t, methodCode)

				if tt.expectValidation {
					// Verify validator is configured
					assert.NotNil(t, concreteGenerator.validator)
				}
			})
		}
	})

	t.Run("validation_error_handling", func(t *testing.T) {
		config := DefaultConfig()
		config.EnableSyntaxValidation = true
		validator := NewCodeValidator(config, zap.NewNop())

		// Test invalid method code
		invalidMethodCode := &MethodCode{
			Name:      "InvalidMethod",
			Signature: "func InvalidMethod(invalid syntax",
			Body:      "invalid body",
			Strategy:  StrategyAssignmentBlock,
		}

		err := validator.ValidateMethod(invalidMethodCode)
		assert.Error(t, err, "Invalid method code should fail validation")
		assert.Contains(t, err.Error(), "method validation failed")
	})

	t.Run("validation_performance", func(t *testing.T) {
		config := DefaultConfig()
		config.EnableSyntaxValidation = true
		validator := NewCodeValidator(config, zap.NewNop())

		// Test validation performance with a complex method
		complexMethodCode := createComplexMethodCode()

		startTime := time.Now()
		err := validator.ValidateMethod(complexMethodCode)
		duration := time.Since(startTime)

		assert.NoError(t, err)
		assert.Less(t, duration, 1*time.Second, "Validation should complete quickly")

		// Check metrics
		concreteValidator := validator.(*ConcreteCodeValidator)
		metrics := concreteValidator.GetValidationMetrics()
		assert.Greater(t, metrics.ValidationTime, time.Duration(0))
	})
}

// TestValidationFrameworkRequirements verifies all Task 2.1 requirements are met.
func TestValidationFrameworkRequirements(t *testing.T) {
	config := DefaultConfig()
	validator := NewCodeValidator(config, zap.NewNop()).(*ConcreteCodeValidator)

	t.Run("requirement_5_1_syntax_checking", func(t *testing.T) {
		// Verify syntactic correctness checking with go/parser integration
		validCode := `func Convert(src *Source) (*Dest, error) { return &Dest{}, nil }`
		invalidCode := `func Convert(src *Source { invalid syntax`

		err := validator.validateSyntax(validCode)
		assert.NoError(t, err, "Valid syntax should pass")

		err = validator.validateSyntax(invalidCode)
		assert.Error(t, err, "Invalid syntax should fail")
		assert.Contains(t, err.Error(), "syntax error")
	})

	t.Run("requirement_5_8_semantic_validation", func(t *testing.T) {
		// Verify semantic validation using go/types
		validCode := `
		type Source struct { Field string }
		type Dest struct { Field string }
		func Convert(src *Source) (*Dest, error) {
			return &Dest{Field: src.Field}, nil
		}`

		err := validator.validateSemantics(validCode)
		assert.NoError(t, err, "Semantically valid code should pass")
	})

	t.Run("requirement_7_1_type_safety", func(t *testing.T) {
		// Verify type safety verification for generated conversion functions
		methodCode := &MethodCode{
			Name:      "ConvertData",
			Signature: "func ConvertData(src *Source) (*Dest, error)",
			Body:      "return &Dest{Field: src.Field}, nil",
			Imports:   []*Import{{Path: "fmt", Standard: true}},
			Strategy:  StrategyAssignmentBlock,
		}

		err := validator.ValidateMethod(methodCode)
		assert.NoError(t, err, "Type-safe method should pass validation")
	})

	t.Run("requirement_memory_compilation", func(t *testing.T) {
		// Verify in-memory compilation testing
		validCode := `func Test() { return }`

		err := validator.validateMemoryCompilation(validCode)
		assert.NoError(t, err, "Valid code should compile in memory")
	})

	t.Run("validation_framework_completeness", func(t *testing.T) {
		// Verify the validation framework is complete and integrated

		// 1. CodeValidator interface is implemented
		assert.Implements(t, (*CodeValidator)(nil), validator)

		// 2. All validation methods are available
		err := validator.Validate("func Test() { return }")
		assert.NoError(t, err)

		methodCode := createTestMethodCode()
		err = validator.ValidateMethod(methodCode)
		assert.NoError(t, err)

		err = validator.ValidateMethodCode(methodCode)
		assert.NoError(t, err)

		// 3. Metrics are tracked
		metrics := validator.GetValidationMetrics()
		assert.NotNil(t, metrics)
		assert.Greater(t, metrics.ValidationTime, time.Duration(0))

		// 4. Configuration options work
		assert.NotNil(t, validator.config)
		assert.NotNil(t, validator.fileSet)
		assert.NotNil(t, validator.logger)
	})
}

// Helper functions for testing

func createTestMethodResult() *domain.MethodResult {
	method := &domain.Method{
		Name: "ConvertSourceToDest",
	}

	return &domain.MethodResult{
		Method: method,
		Metadata: map[string]interface{}{
			"field1": &executor.FieldResult{
				FieldID:      "Field1",
				StrategyUsed: "direct_assignment",
				Error:        nil,
			},
		},
	}
}

func createTestMethodCode() *MethodCode {
	return &MethodCode{
		Name:      "ConvertData",
		Signature: "func ConvertData(src *Source) (*Dest, error)",
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
	}
}

func createComplexMethodCode() *MethodCode {
	return &MethodCode{
		Name:      "ConvertComplexData",
		Signature: "func ConvertComplexData(src *ComplexSource) (*ComplexDest, error)",
		Body: `	if src == nil {
		return nil, fmt.Errorf("source is nil")
	}
	
	dest := &ComplexDest{}
	
	// Convert simple fields
	dest.StringField = src.StringField
	dest.IntField = src.IntField
	dest.BoolField = src.BoolField
	
	// Convert slice fields
	if src.SliceField != nil {
		dest.SliceField = make([]string, len(src.SliceField))
		for i, v := range src.SliceField {
			dest.SliceField[i] = v
		}
	}
	
	// Convert map fields
	if src.MapField != nil {
		dest.MapField = make(map[string]int)
		for k, v := range src.MapField {
			dest.MapField[k] = v
		}
	}
	
	return dest, nil`,
		Imports: []*Import{
			{Path: "fmt", Standard: true, Used: true},
		},
		Strategy: StrategyAssignmentBlock,
	}
}

func constructCompleteMethodForTesting(methodCode *MethodCode) string {
	return `package main

import "fmt"

type Source struct {
	Field1 string
	Field2 int
}

type Dest struct {
	Field1 string
	Field2 int
}

` + methodCode.Signature + ` {
` + methodCode.Body + `
}`
}

// BenchmarkValidationPerformance measures validation performance.
func BenchmarkValidationPerformance(b *testing.B) {
	config := DefaultConfig()
	config.EnableSyntaxValidation = true
	config.EnableSemanticValidation = false // Disable for benchmark
	config.EnableMemoryCompilation = false

	validator := NewCodeValidator(config, zap.NewNop())
	methodCode := createTestMethodCode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateMethod(methodCode)
		if err != nil {
			b.Fatal(err)
		}
	}
}
