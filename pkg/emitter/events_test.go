package emitter

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

func TestEmitterEventHandler_RegisterEventHandlers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	emitter := NewEmitter(logger, eventBus, DefaultConfig())

	handler := NewEventHandler(emitter, eventBus, logger)

	err := handler.RegisterEventHandlers()
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Verify handlers can handle the expected event types
	if !handler.CanHandle("executor.completed") {
		t.Error("Handler should be able to handle executor.completed events")
	}

	if !handler.CanHandle("planner.method_planned") {
		t.Error("Handler should be able to handle planner.method_planned events")
	}

	if handler.CanHandle("unknown.event") {
		t.Error("Handler should not handle unknown event types")
	}
}

func TestEmitterEventHandler_PublishEvents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	emitter := NewEmitter(logger, eventBus, DefaultConfig())

	handler := NewEventHandler(emitter, eventBus, logger)
	ctx := context.Background()

	// Test publishing emitter started event
	err := handler.PublishEmitterStarted(ctx, "testpkg", 3)
	if err != nil {
		t.Errorf("Failed to publish emitter started event: %v", err)
	}

	// Test publishing strategy selected event
	err = handler.PublishStrategySelected(ctx, "TestMethod", StrategyCompositeLiteral, "optimal_performance")
	if err != nil {
		t.Errorf("Failed to publish strategy selected event: %v", err)
	}

	// Test publishing method generated event
	methodCode := &MethodCode{
		Name:      "TestMethod",
		Signature: "func TestMethod() error",
		Body:      "return nil",
	}

	err = handler.PublishMethodGenerated(ctx, methodCode, time.Millisecond*100)
	if err != nil {
		t.Errorf("Failed to publish method generated event: %v", err)
	}

	// Test publishing emitter completed event
	code := &GeneratedCode{
		PackageName: "testpkg",
		Methods:     []*MethodCode{methodCode},
	}
	metrics := &Metrics{
		TotalMethods: 1,
		TotalLines:   10,
	}

	err = handler.PublishEmitterCompleted(ctx, code, metrics)
	if err != nil {
		t.Errorf("Failed to publish emitter completed event: %v", err)
	}

	// Test publishing emitter failed event
	testErr := &domain.ExecutionError{
		Type:      "test_error",
		Message:   "test error",
		Component: "emitter",
		Field:     "test",
		Timestamp: time.Now(),
	}

	err = handler.PublishEmitterFailed(ctx, testErr, nil)
	if err != nil {
		t.Errorf("Failed to publish emitter failed event: %v", err)
	}
}

func TestEmitterEventHandler_HandleExecutorCompleted(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	emitter := NewEmitter(logger, eventBus, DefaultConfig())

	handler := NewEventHandler(emitter, eventBus, logger)
	ctx := context.Background()

	// Create executor completed event
	results := &domain.ExecutionResults{
		PackageName: "testpkg",
		Methods: []*domain.MethodResult{
			{
				Method: &domain.Method{
					Name: "TestMethod",
				},
				Success: true,
				Metadata: map[string]interface{}{
					"fields": map[string]interface{}{
						"Field1": map[string]interface{}{
							"field_id": "Field1",
							"result":   "src.Field1",
							"strategy": "direct",
						},
					},
				},
			},
		},
	}

	event := events.NewBaseEvent("executor.completed", ctx)
	event.WithMetadata("execution_results", results)

	err := handler.Handle(ctx, event)
	if err != nil {
		t.Errorf("Failed to handle executor completed event: %v", err)
	}

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)
}

func TestEmitterEventHandler_HandleMethodPlanned(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	emitter := NewEmitter(logger, eventBus, DefaultConfig())

	handler := NewEventHandler(emitter, eventBus, logger)
	ctx := context.Background()

	// Create method planned event
	event := events.NewBaseEvent("planner.method_planned", ctx)
	event.WithMetadata("method_name", "TestMethod")

	err := handler.Handle(ctx, event)
	if err != nil {
		t.Errorf("Failed to handle method planned event: %v", err)
	}
}

func TestEmitterEvents_Creation(t *testing.T) {
	ctx := context.Background()

	// Test EmitterStartedEvent
	startedEvent := NewStartedEvent(ctx, "testpkg", 5)
	if startedEvent.Type() != EventEmitterStarted {
		t.Errorf("Expected event type %s, got %s", EventEmitterStarted, startedEvent.Type())
	}

	if startedEvent.PackageName != "testpkg" {
		t.Errorf("Expected package name 'testpkg', got '%s'", startedEvent.PackageName)
	}

	if startedEvent.MethodCount != 5 {
		t.Errorf("Expected method count 5, got %d", startedEvent.MethodCount)
	}

	// Test EmitterCompletedEvent
	code := &GeneratedCode{PackageName: "testpkg", Methods: []*MethodCode{}}
	metrics := &Metrics{TotalMethods: 3}

	completedEvent := NewCompletedEvent(ctx, code, metrics)
	if completedEvent.Type() != EventEmitterCompleted {
		t.Errorf("Expected event type %s, got %s", EventEmitterCompleted, completedEvent.Type())
	}

	// Test EmitterFailedEvent
	testErr := &domain.ExecutionError{
		Type:      "test_error",
		Message:   "test error",
		Component: "emitter",
		Timestamp: time.Now(),
	}

	failedEvent := NewFailedEvent(ctx, testErr, nil)
	if failedEvent.Type() != EventEmitterFailed {
		t.Errorf("Expected event type %s, got %s", EventEmitterFailed, failedEvent.Type())
	}

	if !errors.Is(failedEvent.Error, testErr) {
		t.Error("Failed event should contain the original error")
	}

	// Test CodeGenerationStartedEvent
	codeGenEvent := NewCodeGenerationStartedEvent(ctx, "TestMethod", StrategyCompositeLiteral)
	if codeGenEvent.Type() != EventCodeGenerationStarted {
		t.Errorf("Expected event type %s, got %s", EventCodeGenerationStarted, codeGenEvent.Type())
	}

	if codeGenEvent.MethodName != "TestMethod" {
		t.Errorf("Expected method name 'TestMethod', got '%s'", codeGenEvent.MethodName)
	}

	// Test MethodGeneratedEvent
	methodCode := &MethodCode{Name: "TestMethod"}
	duration := time.Millisecond * 150

	methodEvent := NewMethodGeneratedEvent(ctx, methodCode, duration)
	if methodEvent.Type() != EventMethodGenerated {
		t.Errorf("Expected event type %s, got %s", EventMethodGenerated, methodEvent.Type())
	}

	if methodEvent.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, methodEvent.Duration)
	}

	// Test StrategySelectedEvent
	strategyEvent := NewStrategySelectedEvent(ctx, "TestMethod", StrategyMixedApproach, "complexity_analysis")
	if strategyEvent.Type() != EventStrategySelected {
		t.Errorf("Expected event type %s, got %s", EventStrategySelected, strategyEvent.Type())
	}

	if strategyEvent.Strategy != StrategyMixedApproach {
		t.Errorf("Expected strategy %s, got %s", StrategyMixedApproach, strategyEvent.Strategy)
	}

	if strategyEvent.Reason != "complexity_analysis" {
		t.Errorf("Expected reason 'complexity_analysis', got '%s'", strategyEvent.Reason)
	}
}

func TestEventAwareEmitter_GenerateCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	innerEmitter := NewEmitter(logger, eventBus, DefaultConfig())

	eventAwareEmitter := NewEventAwareEmitter(innerEmitter, eventBus, logger)

	// Create test execution results
	results := &domain.ExecutionResults{
		PackageName: "testpkg",
		Methods: []*domain.MethodResult{
			{
				Method: &domain.Method{
					Name: "ConvertUser",
				},
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
		},
	}

	ctx := context.Background()
	code, err := eventAwareEmitter.GenerateCode(ctx, results)

	if err != nil {
		t.Fatalf("EventAwareEmitter GenerateCode failed: %v", err)
	}

	if code == nil {
		t.Fatal("Generated code is nil")
	}

	if len(code.Methods) != 1 {
		t.Errorf("Expected 1 method, got %d", len(code.Methods))
	}
}

func TestEventAwareEmitter_GenerateMethod(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	innerEmitter := NewEmitter(logger, eventBus, DefaultConfig())

	eventAwareEmitter := NewEventAwareEmitter(innerEmitter, eventBus, logger)

	// Create test method result
	method := &domain.MethodResult{
		Method: &domain.Method{
			Name: "ConvertSimple",
		},
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
	}

	ctx := context.Background()
	methodCode, err := eventAwareEmitter.GenerateMethod(ctx, method)

	if err != nil {
		t.Fatalf("EventAwareEmitter GenerateMethod failed: %v", err)
	}

	if methodCode == nil {
		t.Fatal("Generated method code is nil")
	}

	if methodCode.Name != "ConvertSimple" {
		t.Errorf("Expected method name 'ConvertSimple', got '%s'", methodCode.Name)
	}
}

func TestEventAwareEmitter_DelegatedMethods(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	innerEmitter := NewEmitter(logger, eventBus, DefaultConfig())

	eventAwareEmitter := NewEventAwareEmitter(innerEmitter, eventBus, logger)

	// Test OptimizeOutput delegation
	code := &GeneratedCode{
		PackageName: "testpkg",
		Methods:     []*MethodCode{},
		Metadata:    &GenerationMetadata{},
		Metrics:     NewGenerationMetrics(),
	}

	ctx := context.Background()

	optimized, err := eventAwareEmitter.OptimizeOutput(ctx, code)
	if err != nil {
		t.Errorf("OptimizeOutput delegation failed: %v", err)
	}

	if optimized == nil {
		t.Error("OptimizeOutput should return non-nil result")
	}

	// Test GetMetrics delegation
	metrics := eventAwareEmitter.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics should return non-nil result")
	}

	// Test Shutdown delegation
	err = eventAwareEmitter.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown delegation failed: %v", err)
	}
}

func TestEmitterEventHandler_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	emitter := NewEmitter(logger, eventBus, DefaultConfig())

	handler := NewEventHandler(emitter, eventBus, logger)
	ctx := context.Background()

	// Test handling event with missing metadata
	event := events.NewBaseEvent("executor.completed", ctx)
	// No execution_results metadata

	err := handler.Handle(ctx, event)
	if err == nil {
		t.Error("Expected error when handling event with missing metadata")
	}

	// Test handling unknown event type
	unknownEvent := events.NewBaseEvent("unknown.event", ctx)

	err = handler.Handle(ctx, unknownEvent)
	if err != nil {
		t.Errorf("Should not error on unknown event types: %v", err)
	}
}
