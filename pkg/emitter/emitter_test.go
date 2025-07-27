package emitter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/executor"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// Helper function to create test method with proper field mappings
func createTestMethodWithFields(methodName string, fieldNames ...string) (*domain.Method, error) {
	sourceType := domain.NewBasicType("SourceType", reflect.Struct)
	destType := domain.NewBasicType("DestType", reflect.Struct)

	method, err := domain.NewMethod(methodName, sourceType, destType)
	if err != nil {
		return nil, err
	}

	// Add field mappings using proper domain constructors
	for _, fieldName := range fieldNames {
		sourceSpec, err := domain.NewFieldSpec([]string{fieldName}, domain.NewBasicType("string", reflect.String))
		if err != nil {
			return nil, err
		}

		destSpec, err := domain.NewFieldSpec([]string{fieldName}, domain.NewBasicType("string", reflect.String))
		if err != nil {
			return nil, err
		}

		strategy := &domain.DirectAssignmentStrategy{}
		mapping, err := domain.NewFieldMapping(
			fieldName, // ID
			sourceSpec,
			destSpec,
			strategy,
		)
		if err != nil {
			return nil, err
		}

		method.AddMapping(mapping)
	}

	return method, nil
}

// Helper function to create test executor field results
func createTestFieldResult(fieldID, result, strategy string, success bool) *executor.FieldResult {
	return &executor.FieldResult{
		FieldID:      fieldID,
		Success:      success,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		Duration:     time.Millisecond,
		Result:       result,
		Error:        nil,
		StrategyUsed: strategy,
		RetryCount:   0,
	}
}

func TestEmitter_GenerateCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()

	emitter := NewEmitter(logger, eventBus, config)

	// Create test method with proper field mappings
	method, err := createTestMethodWithFields("ConvertUser", "Name", "Email")
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Create test execution results with proper field results
	results := &domain.ExecutionResults{
		PackageName: "testpkg",
		BaseCode:    "// Base code\n",
		Methods: []*domain.MethodResult{
			{
				Method:  method,
				Success: true,
				Metadata: map[string]interface{}{
					"Name":  createTestFieldResult("Name", "src.Name", "direct", true),
					"Email": createTestFieldResult("Email", "src.Email", "direct", true),
				},
			},
		},
	}

	ctx := context.Background()
	code, err := emitter.GenerateCode(ctx, results)

	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if code == nil {
		t.Fatal("Generated code is nil")
	}

	if code.PackageName != "testpkg" {
		t.Errorf("Expected package name 'testpkg', got '%s'", code.PackageName)
	}

	if len(code.Methods) != 1 {
		t.Errorf("Expected 1 method, got %d", len(code.Methods))
	}

	if code.Source == "" {
		t.Error("Generated source code is empty")
	}

	t.Logf("Generated code:\n%s", code.Source)
}

func TestEmitter_GenerateMethod(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()

	emitter := NewEmitter(logger, eventBus, config)

	// Create test method with proper field mappings
	domainMethod, err := createTestMethodWithFields("ConvertSimple", "ID")
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Create test method result with proper field results
	method := &domain.MethodResult{
		Method:  domainMethod,
		Success: true,
		Metadata: map[string]interface{}{
			"ID": createTestFieldResult("ID", "src.ID", "direct", true),
		},
	}

	ctx := context.Background()
	methodCode, err := emitter.GenerateMethod(ctx, method)

	if err != nil {
		t.Fatalf("GenerateMethod failed: %v", err)
	}

	if methodCode == nil {
		t.Fatal("Generated method code is nil")
	}

	if methodCode.Name != "ConvertSimple" {
		t.Errorf("Expected method name 'ConvertSimple', got '%s'", methodCode.Name)
	}

	if methodCode.Body == "" {
		t.Error("Generated method body is empty")
	}

	t.Logf("Generated method:\n%s\n%s", methodCode.Signature, methodCode.Body)
}

func TestEmitter_OptimizeOutput(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	config.OptimizationLevel = OptimizationAggressive

	emitter := NewEmitter(logger, eventBus, config)

	// Create test generated code
	code := &GeneratedCode{
		PackageName: "testpkg",
		Methods: []*MethodCode{
			{
				Name:      "TestMethod",
				Signature: "func TestMethod(src *Source) (*Dest, error)",
				Body:      "	var dest Dest\n	dest.Field = src.Field\n	return &dest, nil\n",
			},
		},
		Metadata: &GenerationMetadata{},
		Metrics:  NewGenerationMetrics(),
	}

	ctx := context.Background()
	optimized, err := emitter.OptimizeOutput(ctx, code)

	if err != nil {
		t.Fatalf("OptimizeOutput failed: %v", err)
	}

	if optimized == nil {
		t.Fatal("Optimized code is nil")
	}

	// Optimization should not change the basic structure
	if len(optimized.Methods) != len(code.Methods) {
		t.Errorf("Expected %d methods after optimization, got %d",
			len(code.Methods), len(optimized.Methods))
	}
}

func TestEmitter_GetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()

	emitter := NewEmitter(logger, eventBus, config)

	metrics := emitter.GetMetrics()

	if metrics == nil {
		t.Fatal("Metrics is nil")
	}

	// Initial metrics should be zero
	if metrics.TotalMethods != 0 {
		t.Errorf("Expected 0 methods generated initially, got %d", metrics.TotalMethods)
	}
}

func TestEmitter_Shutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()

	emitter := NewEmitter(logger, eventBus, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := emitter.Shutdown(ctx)

	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestDefaultEmitterConfig(t *testing.T) {
	config := DefaultEmitterConfig()

	if config == nil {
		t.Fatal("Default config is nil")
	}

	if config.MaxFieldsForComposite <= 0 {
		t.Error("MaxFieldsForComposite should be positive")
	}

	if config.IndentStyle == "" {
		t.Error("IndentStyle should not be empty")
	}

	if config.LineWidth <= 0 {
		t.Error("LineWidth should be positive")
	}

	if config.GenerationTimeout <= 0 {
		t.Error("GenerationTimeout should be positive")
	}
}

func TestEmitter_ConcurrentGeneration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	config.EnableConcurrentGen = true
	config.MaxConcurrentMethods = 2

	emitter := NewEmitter(logger, eventBus, config)

	// Create multiple test methods
	results := &domain.ExecutionResults{
		PackageName: "testpkg",
		Methods: []*domain.MethodResult{
			{
				Method:  &domain.Method{Name: "ConvertUser1"},
				Success: true,
				Metadata: map[string]interface{}{
					"fields": map[string]interface{}{
						"Name": map[string]interface{}{
							"field_id": "Name",
							"result":   "src.Name",
							"strategy": "direct",
						},
					},
				},
			},
			{
				Method:  &domain.Method{Name: "ConvertUser2"},
				Success: true,
				Metadata: map[string]interface{}{
					"fields": map[string]interface{}{
						"Email": map[string]interface{}{
							"field_id": "Email",
							"result":   "src.Email",
							"strategy": "direct",
						},
					},
				},
			},
			{
				Method:  &domain.Method{Name: "ConvertUser3"},
				Success: true,
				Metadata: map[string]interface{}{
					"fields": map[string]interface{}{
						"ID": map[string]interface{}{
							"field_id": "ID",
							"result":   "src.ID",
							"strategy": "direct",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	code, err := emitter.GenerateCode(ctx, results)

	if err != nil {
		t.Fatalf("Concurrent generation failed: %v", err)
	}

	if len(code.Methods) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(code.Methods))
	}
}

func TestEmitter_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()

	emitter := NewEmitter(logger, eventBus, config)

	// Test with nil input
	ctx := context.Background()
	_, err := emitter.GenerateCode(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil execution results")
	}

	// Test with nil method
	_, err = emitter.GenerateMethod(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil method result")
	}

	// Test with nil code for optimization
	_, err = emitter.OptimizeOutput(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil generated code")
	}
}

func TestEmitter_ComplexFieldHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()

	emitter := NewEmitter(logger, eventBus, config)

	// Create method with complex field scenarios
	method := &domain.MethodResult{
		Method: &domain.Method{
			Name: "ConvertComplex",
		},
		Success: false,
		Error: &domain.ExecutionError{
			Type:      "conversion_error",
			Message:   "conversion failed",
			Component: "emitter",
			Field:     "ErrorField",
			Timestamp: time.Now(),
		},
		Metadata: map[string]interface{}{
			"fields": map[string]interface{}{
				"DirectField": map[string]interface{}{
					"field_id": "DirectField",
					"result":   "src.DirectField",
					"strategy": "direct",
				},
				"ConverterField": map[string]interface{}{
					"field_id": "ConverterField",
					"result":   "converter.Convert(src.ConverterField)",
					"strategy": "converter",
				},
				"ErrorField": map[string]interface{}{
					"field_id": "ErrorField",
					"success":  false,
					"strategy": "converter",
				},
			},
		},
	}

	ctx := context.Background()
	methodCode, err := emitter.GenerateMethod(ctx, method)

	if err != nil {
		t.Fatalf("Complex method generation failed: %v", err)
	}

	if methodCode == nil {
		t.Fatal("Generated method code is nil")
	}

	// Should handle different field types appropriately
	if methodCode.Body == "" {
		t.Error("Generated method body is empty")
	}

	t.Logf("Complex method generated:\n%s\n%s", methodCode.Signature, methodCode.Body)
}
