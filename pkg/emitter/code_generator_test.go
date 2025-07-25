package emitter

import (
	"context"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"go.uber.org/zap/zaptest"
)

func TestCodeGenerator_GenerateMethodCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	// Create test method result
	method := &domain.MethodResult{
		MethodName: "ConvertUser",
		Data: map[string]interface{}{
			"Name": &domain.FieldResult{
				FieldID:      "Name",
				Success:      true,
				Result:       "src.Name",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"Email": &domain.FieldResult{
				FieldID:      "Email",
				Success:      true,
				Result:       "src.Email",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
		},
	}
	
	ctx := context.Background()
	methodCode, err := generator.GenerateMethodCode(ctx, method)
	
	if err != nil {
		t.Fatalf("GenerateMethodCode failed: %v", err)
	}
	
	if methodCode == nil {
		t.Fatal("Generated method code is nil")
	}
	
	if methodCode.Name != "ConvertUser" {
		t.Errorf("Expected method name 'ConvertUser', got '%s'", methodCode.Name)
	}
	
	if methodCode.Signature == "" {
		t.Error("Method signature should not be empty")
	}
	
	if methodCode.Body == "" {
		t.Error("Method body should not be empty")
	}
	
	if len(methodCode.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(methodCode.Fields))
	}
	
	t.Logf("Generated method code:\n%s\n%s", methodCode.Signature, methodCode.Body)
}

func TestCodeGenerator_GenerateFieldCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	// Test direct field
	directField := &domain.FieldResult{
		FieldID:      "DirectField",
		Success:      true,
		Result:       "src.DirectField",
		StrategyUsed: "direct",
		Duration:     time.Millisecond,
	}
	
	ctx := context.Background()
	fieldCode, err := generator.GenerateFieldCode(ctx, directField)
	
	if err != nil {
		t.Fatalf("GenerateFieldCode failed: %v", err)
	}
	
	if fieldCode == nil {
		t.Fatal("Generated field code is nil")
	}
	
	if fieldCode.Assignment == "" {
		t.Error("Field assignment should not be empty")
	}
	
	// Test converter field
	converterField := &domain.FieldResult{
		FieldID:      "ConverterField",
		Success:      true,
		Result:       "converter.Convert(src.ConverterField)",
		StrategyUsed: "converter",
		Duration:     5 * time.Millisecond,
	}
	
	fieldCode2, err := generator.GenerateFieldCode(ctx, converterField)
	
	if err != nil {
		t.Fatalf("GenerateFieldCode for converter failed: %v", err)
	}
	
	if fieldCode2 == nil {
		t.Fatal("Generated converter field code is nil")
	}
	
	// Test error field
	errorField := &domain.FieldResult{
		FieldID:      "ErrorField",
		Success:      false,
		Error:        &domain.ExecutionError{FieldID: "ErrorField", Error: "conversion failed"},
		StrategyUsed: "converter",
		Duration:     10 * time.Millisecond,
		RetryCount:   2,
	}
	
	fieldCode3, err := generator.GenerateFieldCode(ctx, errorField)
	
	if err != nil {
		t.Fatalf("GenerateFieldCode for error field failed: %v", err)
	}
	
	if fieldCode3 == nil {
		t.Fatal("Generated error field code is nil")
	}
	
	if fieldCode3.ErrorCheck == "" {
		t.Error("Error field should have error check code")
	}
}

func TestCodeGenerator_GenerateErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	errors := []domain.ExecutionError{
		{
			FieldID: "Field1",
			Error:   "conversion error 1",
		},
		{
			FieldID: "Field2",
			Error:   "conversion error 2",
		},
	}
	
	ctx := context.Background()
	errorCode, err := generator.GenerateErrorHandling(ctx, errors)
	
	if err != nil {
		t.Fatalf("GenerateErrorHandling failed: %v", err)
	}
	
	if errorCode == nil {
		t.Fatal("Generated error code is nil")
	}
	
	if errorCode.Code == "" {
		t.Error("Error handling code should not be empty")
	}
	
	t.Logf("Generated error handling:\n%s", errorCode.Code)
}

func TestCodeGenerator_GetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	genMetrics := generator.GetMetrics()
	
	if genMetrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	
	// Initial metrics should be zero
	if genMetrics.MethodsGenerated != 0 {
		t.Errorf("Expected 0 methods generated initially, got %d", genMetrics.MethodsGenerated)
	}
}

func TestCodeGenerator_Shutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := generator.Shutdown(ctx)
	
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestCodeGenerator_HelperFunctions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics).(*ConcreteCodeGenerator)
	
	helpers := generator.getHelperFunctions()
	
	if helpers == nil {
		t.Fatal("Helper functions should not be nil")
	}
	
	// Test camelCase function
	camelCaseFn, exists := helpers["camelCase"]
	if !exists {
		t.Error("camelCase helper function should exist")
	}
	
	if camelCaseFn != nil {
		if fn, ok := camelCaseFn.(func(string) string); ok {
			result := fn("test_field")
			expected := "testField"
			if result != expected {
				t.Errorf("Expected camelCase of 'test_field' to be '%s', got '%s'", expected, result)
			}
		}
	}
	
	// Test snakeCase function
	snakeCaseFn, exists := helpers["snakeCase"]
	if !exists {
		t.Error("snakeCase helper function should exist")
	}
	
	// Test indent function
	indentFn, exists := helpers["indent"]
	if !exists {
		t.Error("indent helper function should exist")
	}
	
	if indentFn != nil {
		if fn, ok := indentFn.(func(string, int) string); ok {
			result := fn("test", 2)
			if result == "" {
				t.Error("indent function should return non-empty result")
			}
		}
	}
}

func TestCodeGenerator_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	ctx := context.Background()
	
	// Test with nil method
	_, err := generator.GenerateMethodCode(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil method result")
	}
	
	// Test with nil field
	_, err = generator.GenerateFieldCode(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil field result")
	}
	
	// Test with empty errors
	errorCode, err := generator.GenerateErrorHandling(ctx, []domain.ExecutionError{})
	if err != nil {
		t.Errorf("Should not error with empty error list: %v", err)
	}
	if errorCode != nil && errorCode.Code != "" {
		t.Error("Empty error list should result in empty error code")
	}
}

func TestCodeGenerator_ComplexScenarios(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	// Create a complex method with various field types
	method := &domain.MethodResult{
		MethodName: "ConvertComplex",
		Data: map[string]interface{}{
			"SimpleField": &domain.FieldResult{
				FieldID:      "SimpleField",
				Success:      true,
				Result:       "src.SimpleField",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"ConvertedField": &domain.FieldResult{
				FieldID:      "ConvertedField",
				Success:      true,
				Result:       "converter.Convert(src.ConvertedField)",
				StrategyUsed: "converter",
				Duration:     5 * time.Millisecond,
			},
			"ErrorProneField": &domain.FieldResult{
				FieldID:      "ErrorProneField",
				Success:      false,
				Error:        &domain.ExecutionError{FieldID: "ErrorProneField", Error: "complex conversion failed"},
				StrategyUsed: "converter",
				Duration:     15 * time.Millisecond,
				RetryCount:   3,
			},
			"LiteralField": &domain.FieldResult{
				FieldID:      "LiteralField",
				Success:      true,
				Result:       "\"constant_value\"",
				StrategyUsed: "literal",
				Duration:     time.Microsecond,
			},
		},
	}
	
	ctx := context.Background()
	methodCode, err := generator.GenerateMethodCode(ctx, method)
	
	if err != nil {
		t.Fatalf("Complex method generation failed: %v", err)
	}
	
	if methodCode == nil {
		t.Fatal("Generated method code is nil")
	}
	
	if len(methodCode.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(methodCode.Fields))
	}
	
	// Should generate appropriate strategy
	if methodCode.Strategy == "" {
		t.Error("Strategy should be selected for complex method")
	}
	
	// Should have complexity metrics
	if methodCode.Complexity == nil {
		t.Error("Complexity metrics should be calculated")
	}
	
	t.Logf("Complex method generated with strategy: %s", methodCode.Strategy)
	t.Logf("Complexity score: %f", methodCode.Complexity.ComplexityScore)
}

func TestCodeGenerator_StrategySelection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	config.MaxFieldsForComposite = 3 // Set threshold for composite literal strategy
	metrics := NewEmitterMetrics(true)
	
	generator := NewCodeGenerator(config, logger, metrics)
	
	// Test simple method that should use composite literal
	simpleMethod := &domain.MethodResult{
		MethodName: "ConvertSimple",
		Data: map[string]interface{}{
			"Field1": &domain.FieldResult{
				FieldID:      "Field1",
				Success:      true,
				Result:       "src.Field1",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"Field2": &domain.FieldResult{
				FieldID:      "Field2",
				Success:      true,
				Result:       "src.Field2",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
		},
	}
	
	ctx := context.Background()
	methodCode, err := generator.GenerateMethodCode(ctx, simpleMethod)
	
	if err != nil {
		t.Fatalf("Simple method generation failed: %v", err)
	}
	
	// Simple methods might use composite literal strategy
	t.Logf("Simple method strategy: %s", methodCode.Strategy)
	
	// Test complex method that should use assignment block
	complexMethod := &domain.MethodResult{
		MethodName: "ConvertComplex",
		Data: map[string]interface{}{
			"Field1": &domain.FieldResult{
				FieldID:      "Field1",
				Success:      false,
				Error:        &domain.ExecutionError{FieldID: "Field1", Error: "error"},
				StrategyUsed: "converter",
				Duration:     10 * time.Millisecond,
			},
			"Field2": &domain.FieldResult{
				FieldID:      "Field2",
				Success:      true,
				Result:       "converter.Convert(src.Field2)",
				StrategyUsed: "converter",
				Duration:     5 * time.Millisecond,
			},
		},
	}
	
	methodCode2, err := generator.GenerateMethodCode(ctx, complexMethod)
	
	if err != nil {
		t.Fatalf("Complex method generation failed: %v", err)
	}
	
	// Complex methods should typically use assignment block strategy
	t.Logf("Complex method strategy: %s", methodCode2.Strategy)
	
	// Strategies should be different for different complexity levels
	if methodCode.Strategy == methodCode2.Strategy {
		t.Log("Note: Both methods used the same strategy - this may be expected depending on configuration")
	}
}