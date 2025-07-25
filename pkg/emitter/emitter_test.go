package emitter

import (
	"context"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"go.uber.org/zap/zaptest"
)

func TestEmitter_GenerateCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	
	emitter := NewEmitter(logger, eventBus, config)
	
	// Create test execution results
	results := &domain.ExecutionResults{
		PackageName: "testpkg",
		BaseCode:    "// Base code\n",
		Methods: []*domain.MethodResult{
			{
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
	
	// Create test method result
	method := &domain.MethodResult{
		MethodName: "ConvertSimple",
		Data: map[string]interface{}{
			"ID": &domain.FieldResult{
				FieldID:      "ID",
				Success:      true,
				Result:       "src.ID",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
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
	if metrics.MethodsGenerated != 0 {
		t.Errorf("Expected 0 methods generated initially, got %d", metrics.MethodsGenerated)
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
				MethodName: "ConvertUser1",
				Data: map[string]interface{}{
					"Name": &domain.FieldResult{
						FieldID:      "Name",
						Success:      true,
						Result:       "src.Name",
						StrategyUsed: "direct",
						Duration:     time.Millisecond,
					},
				},
			},
			{
				MethodName: "ConvertUser2",
				Data: map[string]interface{}{
					"Email": &domain.FieldResult{
						FieldID:      "Email",
						Success:      true,
						Result:       "src.Email",
						StrategyUsed: "direct",
						Duration:     time.Millisecond,
					},
				},
			},
			{
				MethodName: "ConvertUser3",
				Data: map[string]interface{}{
					"ID": &domain.FieldResult{
						FieldID:      "ID",
						Success:      true,
						Result:       "src.ID",
						StrategyUsed: "direct",
						Duration:     time.Millisecond,
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
		MethodName: "ConvertComplex",
		Data: map[string]interface{}{
			"DirectField": &domain.FieldResult{
				FieldID:      "DirectField",
				Success:      true,
				Result:       "src.DirectField",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"ConverterField": &domain.FieldResult{
				FieldID:      "ConverterField",
				Success:      true,
				Result:       "converter.Convert(src.ConverterField)",
				StrategyUsed: "converter",
				Duration:     5 * time.Millisecond,
			},
			"ErrorField": &domain.FieldResult{
				FieldID:      "ErrorField",
				Success:      false,
				Error:        &domain.ExecutionError{FieldID: "ErrorField", Error: "conversion failed"},
				StrategyUsed: "converter",
				Duration:     10 * time.Millisecond,
				RetryCount:   2,
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