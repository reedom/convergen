package emitter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"go.uber.org/zap/zaptest"
)

// TestEmitterIntegration tests the complete emitter pipeline
func TestEmitterIntegration_CompleteWorkflow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	config.OptimizationLevel = OptimizationBasic
	
	// Create event-aware emitter
	innerEmitter := NewEmitter(logger, eventBus, config)
	eventAwareEmitter := NewEventAwareEmitter(innerEmitter, eventBus, logger)
	
	// Create simple test data with proper domain structure
	sourceType := domain.NewBasicType("User", reflect.Struct)
	destType := domain.NewBasicType("UserDto", reflect.Struct)
	
	method, err := domain.NewMethod("ConvertUser", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}
	
	results := &domain.ExecutionResults{
		PackageName: "converter",
		BaseCode:    "// Generated converter package\n",
		Methods: []*domain.MethodResult{
			{
				Method:      method,
				Code:        "func ConvertUser(src User) UserDto {\n\treturn UserDto{ID: src.ID, Name: src.Name}\n}",
				Imports:     []domain.Import{},
				Success:     true,
				Error:       nil,
				Metadata:    map[string]interface{}{"test": true},
				ProcessedAt: time.Now(),
				DurationMS:  5,
			},
		},
		Success:   true,
		Errors:    []*domain.ExecutionError{},
		TotalTime: 5 * time.Millisecond,
		Metadata:  map[string]interface{}{"test_data": true},
	}
	
	// Track events
	var eventsReceived []string
	eventHandler := events.NewFuncEventHandler("emitter.started", func(ctx context.Context, event events.Event) error {
		eventsReceived = append(eventsReceived, event.Type())
		return nil
	})
	eventBus.Subscribe("emitter.started", eventHandler)
	
	completedHandler := events.NewFuncEventHandler("emitter.completed", func(ctx context.Context, event events.Event) error {
		eventsReceived = append(eventsReceived, event.Type())
		return nil
	})
	eventBus.Subscribe("emitter.completed", completedHandler)
	
	ctx := context.Background()
	startTime := time.Now()
	
	// Execute complete workflow
	generatedCode, err := eventAwareEmitter.GenerateCode(ctx, results)
	
	if err != nil {
		t.Fatalf("Complete workflow failed: %v", err)
	}
	
	duration := time.Since(startTime)
	t.Logf("Complete workflow took: %v", duration)
	
	// Verify generated code structure
	if generatedCode == nil {
		t.Fatal("Generated code is nil")
	}
	
	if generatedCode.PackageName != "converter" {
		t.Errorf("Expected package name 'converter', got '%s'", generatedCode.PackageName)
	}
	
	if len(generatedCode.Methods) != 1 {
		t.Errorf("Expected 1 method, got %d", len(generatedCode.Methods))
	}
	
	// Verify source code is generated
	if generatedCode.Source == "" {
		t.Error("Generated source code is empty")
	}
	
	// Verify imports are generated
	if generatedCode.Imports == nil {
		t.Error("Imports should be generated")
	}
	
	// Verify metrics are collected
	if generatedCode.Metrics == nil {
		t.Error("Metrics should be collected")
	}
	
	if generatedCode.Metrics.MethodsGenerated != 2 {
		t.Errorf("Expected 2 methods in metrics, got %d", generatedCode.Metrics.MethodsGenerated)
	}
	
	// Verify metadata
	if generatedCode.Metadata == nil {
		t.Error("Metadata should be generated")
	}
	
	if generatedCode.Metadata.GenerationDuration <= 0 {
		t.Error("Generation duration should be positive")
	}
	
	// Check events were fired
	if len(eventsReceived) < 2 {
		t.Errorf("Expected at least 2 events, got %d", len(eventsReceived))
	}
	
	t.Logf("Events received: %v", eventsReceived)
	t.Logf("Generated source:\n%s", generatedCode.Source)
}

func TestEmitterIntegration_EventPipeline(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	
	emitter := NewEmitter(logger, eventBus, config)
	eventHandler := NewEmitterEventHandler(emitter, eventBus, logger)
	
	// Register event handlers
	err := eventHandler.RegisterEventHandlers()
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}
	
	// Track all emitter events
	var allEvents []string
	
	eventTypes := []string{
		EventEmitterStarted,
		EventEmitterCompleted,
		EventCodeGenerationStarted,
		EventMethodGenerated,
		EventStrategySelected,
	}
	
	for _, eventType := range eventTypes {
		handler := events.NewFuncEventHandler(eventType, func(ctx context.Context, event events.Event) error {
			allEvents = append(allEvents, event.Type())
			t.Logf("Received event: %s with metadata: %v", event.Type(), event.Metadata())
			return nil
		})
		eventBus.Subscribe(eventType, handler)
	}
	
	// Simulate executor completion event
	sourceType := domain.NewBasicType("SimpleSource", reflect.Struct)
	destType := domain.NewBasicType("SimpleDest", reflect.Struct)
	
	method, err := domain.NewMethod("ConvertSimple", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}
	
	results := &domain.ExecutionResults{
		PackageName: "testpkg",
		Methods: []*domain.MethodResult{
			{
				Method:      method,
				Code:        "func ConvertSimple(src SimpleSource) SimpleDest { return SimpleDest{Field1: src.Field1} }",
				Imports:     []domain.Import{},
				Success:     true,
				Error:       nil,
				Metadata:    map[string]interface{}{"test": true},
				ProcessedAt: time.Now(),
				DurationMS:  1,
			},
		},
		Success:   true,
		Errors:    []*domain.ExecutionError{},
		TotalTime: time.Millisecond,
		Metadata:  map[string]interface{}{"test": true},
	}
	
	executorEvent := events.NewBaseEvent("executor.completed", context.Background())
	executorEvent.WithMetadata("execution_results", results)
	
	// Publish executor completed event
	err = eventBus.Publish(executorEvent)
	if err != nil {
		t.Fatalf("Failed to publish executor event: %v", err)
	}
	
	// Wait for async processing
	time.Sleep(200 * time.Millisecond)
	
	// Verify events were processed
	if len(allEvents) == 0 {
		t.Error("No events were processed")
	}
	
	t.Logf("Total events processed: %d", len(allEvents))
	t.Logf("Events: %v", allEvents)
	
	// Check that emitter started and completed events were fired
	hasStarted := false
	hasCompleted := false
	
	for _, event := range allEvents {
		if event == EventEmitterStarted {
			hasStarted = true
		}
		if event == EventEmitterCompleted {
			hasCompleted = true
		}
	}
	
	if !hasStarted {
		t.Error("Emitter started event was not fired")
	}
	
	if !hasCompleted {
		t.Error("Emitter completed event was not fired")
	}
}

func TestEmitterIntegration_OptimizationPipeline(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	config.OptimizationLevel = OptimizationAggressive
	config.EnableDeadCodeElim = true
	config.EnableVarOptimization = true
	
	emitter := NewEmitter(logger, eventBus, config)
	
	// Create code with potential optimization opportunities
	sourceType := domain.NewBasicType("OptimizeSource", reflect.Struct)
	destType := domain.NewBasicType("OptimizeDest", reflect.Struct)
	
	method, err := domain.NewMethod("ConvertWithRedundancy", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}
	
	results := &domain.ExecutionResults{
		PackageName: "optimizer_test",
		Methods: []*domain.MethodResult{
			{
				Method:      method,
				Code:        "func ConvertWithRedundancy(src OptimizeSource) OptimizeDest { return OptimizeDest{Field1: src.Field1, Field2: src.Field2} }",
				Imports:     []domain.Import{},
				Success:     true,
				Error:       nil,
				Metadata:    map[string]interface{}{"optimization": true},
				ProcessedAt: time.Now(),
				DurationMS:  2,
			},
		},
		Success:   true,
		Errors:    []*domain.ExecutionError{},
		TotalTime: 2 * time.Millisecond,
		Metadata:  map[string]interface{}{"test": "optimization"},
	}
	
	ctx := context.Background()
	generatedCode, err := emitter.GenerateCode(ctx, results)
	
	if err != nil {
		t.Fatalf("Code generation with optimization failed: %v", err)
	}
	
	// Verify optimization was applied
	if generatedCode.Metrics == nil {
		t.Error("Metrics should be available after optimization")
	}
	
	if generatedCode.Metrics.OptimizationTime <= 0 {
		t.Error("Optimization time should be recorded")
	}
	
	// Test standalone optimization
	optimizedCode, err := emitter.OptimizeOutput(ctx, generatedCode)
	
	if err != nil {
		t.Fatalf("Standalone optimization failed: %v", err)
	}
	
	if optimizedCode == nil {
		t.Error("Optimized code should not be nil")
	}
	
	t.Logf("Optimization applied successfully")
}

// TODO: Fix these tests with proper domain structure
/*
func TestEmitterIntegration_ConcurrentGeneration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	config.EnableConcurrentGen = true
	config.MaxConcurrentMethods = 3
	
	emitter := NewEmitter(logger, eventBus, config)
	
	// Create multiple methods for concurrent generation
	var methods []*domain.MethodResult
	for i := 0; i < 5; i++ {
		method := &domain.MethodResult{
			MethodName: fmt.Sprintf("ConvertMethod%d", i+1),
			Data: map[string]interface{}{
				"Field1": &domain.FieldResult{
					FieldID:      "Field1",
					Success:      true,
					Result:       "src.Field1",
					StrategyUsed: "direct",
					Duration:     time.Millisecond * time.Duration(i+1),
				},
				"Field2": &domain.FieldResult{
					FieldID:      "Field2",
					Success:      true,
					Result:       "src.Field2",
					StrategyUsed: "direct",
					Duration:     time.Millisecond * time.Duration(i+1),
				},
			},
		}
		methods = append(methods, method)
	}
	
	results := &domain.ExecutionResults{
		PackageName: "concurrent_test",
		Methods:     methods,
	}
	
	ctx := context.Background()
	startTime := time.Now()
	
	generatedCode, err := emitter.GenerateCode(ctx, results)
	
	duration := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Concurrent generation failed: %v", err)
	}
	
	if len(generatedCode.Methods) != 5 {
		t.Errorf("Expected 5 methods, got %d", len(generatedCode.Methods))
	}
	
	t.Logf("Concurrent generation of 5 methods took: %v", duration)
	
	// Compare with sequential generation
	config.EnableConcurrentGen = false
	sequentialEmitter := NewEmitter(logger, eventBus, config)
	
	startTimeSeq := time.Now()
	sequentialCode, err := sequentialEmitter.GenerateCode(ctx, results)
	durationSeq := time.Since(startTimeSeq)
	
	if err != nil {
		t.Fatalf("Sequential generation failed: %v", err)
	}
	
	if len(sequentialCode.Methods) != 5 {
		t.Errorf("Expected 5 methods in sequential, got %d", len(sequentialCode.Methods))
	}
	
	t.Logf("Sequential generation of 5 methods took: %v", durationSeq)
	t.Logf("Concurrent vs Sequential ratio: %f", float64(duration)/float64(durationSeq))
}

func TestEmitterIntegration_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	
	emitter := NewEmitter(logger, eventBus, config)
	eventHandler := NewEmitterEventHandler(emitter, eventBus, logger)
	
	// Track failure events
	var failureEvents []string
	failureHandler := events.NewFuncEventHandler("emitter.failed", func(ctx context.Context, event events.Event) error {
		failureEvents = append(failureEvents, event.Type())
		t.Logf("Failure event received: %v", event.Metadata())
		return nil
	})
	eventBus.Subscribe("emitter.failed", failureHandler)
	
	// Test with problematic data that might cause errors
	problematicResults := &domain.ExecutionResults{
		PackageName: "error_test",
		Methods: []*domain.MethodResult{
			{
				MethodName: "ConvertWithErrors",
				Data: map[string]interface{}{
					"SuccessField": &domain.FieldResult{
						FieldID:      "SuccessField",
						Success:      true,
						Result:       "src.SuccessField",
						StrategyUsed: "direct",
						Duration:     time.Millisecond,
					},
					"ErrorField": &domain.FieldResult{
						FieldID:      "ErrorField",
						Success:      false,
						Error:        &domain.ExecutionError{FieldID: "ErrorField", Error: "critical conversion error"},
						StrategyUsed: "converter",
						Duration:     15 * time.Millisecond,
						RetryCount:   3,
					},
				},
			},
		},
	}
	
	ctx := context.Background()
	generatedCode, err := emitter.GenerateCode(ctx, problematicResults)
	
	// Generation should succeed even with field errors
	if err != nil {
		t.Logf("Generation completed with errors (expected): %v", err)
	}
	
	if generatedCode == nil {
		t.Error("Generated code should not be nil even with field errors")
	}
	
	// Test error reporting through events
	err = eventHandler.PublishEmitterFailed(ctx, &domain.ExecutionError{Error: "test error"}, generatedCode)
	if err != nil {
		t.Errorf("Failed to publish failure event: %v", err)
	}
	
	// Give time for event processing
	time.Sleep(50 * time.Millisecond)
	
	if len(failureEvents) == 0 {
		t.Log("No failure events received (this might be expected depending on error handling)")
	}
}

func TestEmitterIntegration_CodeQuality(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultEmitterConfig()
	config.OptimizationLevel = OptimizationMaximal
	config.EnableSyntaxValidation = true
	
	emitter := NewEmitter(logger, eventBus, config)
	
	// Generate code with various complexity scenarios
	results := &domain.ExecutionResults{
		PackageName: "quality_test",
		Methods: []*domain.MethodResult{
			{
				MethodName: "ConvertQualityTest",
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
						Result:       "converter.ConvertString(src.ConvertedField)",
						StrategyUsed: "converter",
						Duration:     5 * time.Millisecond,
					},
					"ComplexField": &domain.FieldResult{
						FieldID:      "ComplexField",
						Success:      true,
						Result:       "processComplexData(src.ComplexField, options)",
						StrategyUsed: "expression",
						Duration:     10 * time.Millisecond,
					},
				},
			},
		},
	}
	
	ctx := context.Background()
	generatedCode, err := emitter.GenerateCode(ctx, results)
	
	if err != nil {
		t.Fatalf("Quality test generation failed: %v", err)
	}
	
	// Verify code quality aspects
	if generatedCode.Source == "" {
		t.Error("Generated source should not be empty")
	}
	
	// Check for basic Go code structure
	source := generatedCode.Source
	
	if !strings.Contains(source, "package quality_test") {
		t.Error("Generated code should contain correct package declaration")
	}
	
	if !strings.Contains(source, "func ConvertQualityTest") {
		t.Error("Generated code should contain method function")
	}
	
	// Check for proper formatting patterns
	lines := strings.Split(source, "\n")
	if len(lines) < 5 {
		t.Error("Generated code should have reasonable length")
	}
	
	// Verify imports are properly formatted
	if generatedCode.Imports != nil && len(generatedCode.Imports.Imports) > 0 {
		if generatedCode.Imports.Source == "" {
			t.Error("Import source should be formatted")
		}
	}
	
	t.Logf("Quality test generated %d lines of code", len(lines))
	t.Logf("Generated imports: %d", len(generatedCode.Imports.Imports))
	
	// Log a portion of the generated code for manual inspection
	if len(lines) > 20 {
		t.Logf("Generated code sample (first 20 lines):\n%s", strings.Join(lines[:20], "\n"))
	} else {
		t.Logf("Generated code:\n%s", source)
	}
}
*/