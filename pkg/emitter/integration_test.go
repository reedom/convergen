package emitter

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// TestEmitterIntegration tests the complete emitter pipeline.
func TestEmitterIntegration_CompleteWorkflow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultConfig()
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
	_ = eventBus.Subscribe("emitter.started", eventHandler)

	completedHandler := events.NewFuncEventHandler("emitter.completed", func(ctx context.Context, event events.Event) error {
		eventsReceived = append(eventsReceived, event.Type())
		return nil
	})
	_ = eventBus.Subscribe("emitter.completed", completedHandler)

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

	if generatedCode.Metrics.MethodsGenerated != 1 {
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
	config := DefaultConfig()

	emitter := NewEmitter(logger, eventBus, config)
	eventHandler := NewEventHandler(emitter, eventBus, logger)

	// Register event handlers
	err := eventHandler.RegisterEventHandlers()
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Track all emitter events with thread safety
	var allEvents []string

	var eventsMu sync.Mutex

	eventTypes := []string{
		EventEmitterStarted,
		EventEmitterCompleted,
		EventCodeGenerationStarted,
		EventMethodGenerated,
		EventStrategySelected,
	}

	for _, eventType := range eventTypes {
		handler := events.NewFuncEventHandler(eventType, func(ctx context.Context, event events.Event) error {
			eventsMu.Lock()
			allEvents = append(allEvents, event.Type())
			eventsMu.Unlock()
			t.Logf("Received event: %s with metadata: %v", event.Type(), event.Metadata())

			return nil
		})
		_ = eventBus.Subscribe(eventType, handler)
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

	// Verify events were processed with thread safety
	eventsMu.Lock()
	eventCount := len(allEvents)
	eventsCopy := make([]string, len(allEvents))
	copy(eventsCopy, allEvents)
	eventsMu.Unlock()

	if eventCount == 0 {
		t.Error("No events were processed")
	}

	t.Logf("Total events processed: %d", eventCount)
	t.Logf("Events: %v", eventsCopy)

	// Check that emitter started and completed events were fired
	hasStarted := false
	hasCompleted := false

	for _, event := range eventsCopy {
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
	config := DefaultConfig()
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

// Note: Legacy concurrent generation tests removed - comprehensive concurrent testing
// is now provided by race_test.go with proper domain model usage.
