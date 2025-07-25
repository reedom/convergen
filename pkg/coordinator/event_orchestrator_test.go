package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/internal/events"
	"go.uber.org/zap/zaptest"
)

func TestNewEventOrchestrator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	if orchestrator == nil {
		t.Fatal("NewEventOrchestrator returned nil")
	}
	
	// Verify it implements the interface
	var _ EventOrchestrator = orchestrator
}

func TestEventOrchestratorGetStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	status := orchestrator.GetStatus()
	
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}
	
	if status.Stage != StageInitializing {
		t.Errorf("Expected initial stage %s, got %s", StageInitializing, status.Stage)
	}
	
	if status.Progress != 0.0 {
		t.Errorf("Expected initial progress 0.0, got %f", status.Progress)
	}
	
	if status.ComponentStatus == nil {
		t.Error("Expected component status map to be initialized")
	}
}

func TestEventOrchestratorStartPipeline(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	// Create test input
	input := &PipelineInput{
		Sources: []string{"test.go"},
		Config:  config,
		Context: context.Background(),
		Metadata: map[string]interface{}{
			"generation_id": "test-pipeline-123",
			"start_time":    time.Now(),
		},
	}
	
	ctx := context.Background()
	err := orchestrator.StartPipeline(ctx, input)
	
	if err != nil {
		t.Fatalf("StartPipeline failed: %v", err)
	}
	
	// Verify status updated
	status := orchestrator.GetStatus()
	if status.PipelineID != "test-pipeline-123" {
		t.Errorf("Expected pipeline ID 'test-pipeline-123', got %q", status.PipelineID)
	}
	
	if status.CurrentInput != input {
		t.Error("Expected current input to be set")
	}
}

func TestEventOrchestratorStartPipelineAlreadyStarted(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	input := &PipelineInput{
		Sources: []string{"test.go"},
		Config:  config,
		Context: context.Background(),
		Metadata: map[string]interface{}{
			"generation_id": "test-pipeline",
		},
	}
	
	ctx := context.Background()
	
	// First start should succeed
	err := orchestrator.StartPipeline(ctx, input)
	if err != nil {
		t.Fatalf("First StartPipeline failed: %v", err)
	}
	
	// Second start should fail
	err = orchestrator.StartPipeline(ctx, input)
	if err == nil {
		t.Error("Expected error when starting already started pipeline")
	}
	
	expectedMsg := "pipeline already started"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestEventOrchestratorCancel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	// Start pipeline
	input := &PipelineInput{
		Sources: []string{"test.go"},
		Config:  config,
		Context: context.Background(),
		Metadata: map[string]interface{}{
			"generation_id": "test-pipeline",
		},
	}
	
	ctx := context.Background()
	err := orchestrator.StartPipeline(ctx, input)
	if err != nil {
		t.Fatalf("StartPipeline failed: %v", err)
	}
	
	// Cancel pipeline
	err = orchestrator.Cancel()
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	
	// Verify status updated
	status := orchestrator.GetStatus()
	if status.Stage != StageFailed {
		t.Errorf("Expected stage %s after cancel, got %s", StageFailed, status.Stage)
	}
	
	// Second cancel should be idempotent
	err = orchestrator.Cancel()
	if err != nil {
		t.Errorf("Second cancel failed: %v", err)
	}
}

func TestEventOrchestratorHandleEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	config.EventBufferSize = 10 // Small buffer for testing
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	// Create test event
	testEvent := events.NewEvent("test.event", map[string]interface{}{
		"test_data": "test_value",
	})
	
	ctx := context.Background()
	err := orchestrator.HandleEvent(ctx, testEvent)
	
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}
}

func TestEventOrchestratorHandleEventCancelled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	// Cancel orchestrator first
	err := orchestrator.Cancel()
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	
	// Create test event
	testEvent := events.NewEvent("test.event", map[string]interface{}{})
	
	ctx := context.Background()
	err = orchestrator.HandleEvent(ctx, testEvent)
	
	if err == nil {
		t.Error("Expected error when handling event on cancelled orchestrator")
	}
	
	expectedMsg := "pipeline cancelled"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestEventOrchestratorSetComponentManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	componentMgr := NewComponentManager(logger, config)
	
	orchestrator.SetComponentManager(componentMgr)
	
	// Verify component manager is set
	concreteOrchestrator := orchestrator.(*ConcreteEventOrchestrator)
	if concreteOrchestrator.componentMgr != componentMgr {
		t.Error("Component manager not set correctly")
	}
}

func TestEventOrchestratorSetErrorHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	errorHandler := NewErrorHandler(logger, config)
	
	orchestrator.SetErrorHandler(errorHandler)
	
	// Verify error handler is set
	concreteOrchestrator := orchestrator.(*ConcreteEventOrchestrator)
	if concreteOrchestrator.errorHandler != errorHandler {
		t.Error("Error handler not set correctly")
	}
}

func TestEventOrchestratorGetPipelineResults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	// Add some test results
	orchestrator.results[StageParsing] = "parse_results"
	orchestrator.results[StagePlanning] = "plan_results"
	
	results := orchestrator.GetPipelineResults()
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	if results[StageParsing] != "parse_results" {
		t.Errorf("Expected parse_results, got %v", results[StageParsing])
	}
	
	if results[StagePlanning] != "plan_results" {
		t.Errorf("Expected plan_results, got %v", results[StagePlanning])
	}
}

func TestEventOrchestratorEventHandlers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	// Test that event handlers are registered
	expectedHandlers := []string{
		"parser.completed",
		"parser.failed",
		"planner.completed",
		"planner.failed",
		"executor.completed",
		"executor.failed",
		"emitter.completed",
		"emitter.failed",
		"component.status.changed",
	}
	
	for _, eventType := range expectedHandlers {
		if _, exists := orchestrator.eventHandlers[eventType]; !exists {
			t.Errorf("Event handler for %q not registered", eventType)
		}
	}
}

func TestEventOrchestratorParseCompleteHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	// Create parse complete event
	event := events.NewEvent("parser.completed", map[string]interface{}{
		"results": "test_parse_results",
	})
	
	ctx := context.Background()
	err := orchestrator.handleParseComplete(ctx, event)
	
	if err != nil {
		t.Fatalf("handleParseComplete failed: %v", err)
	}
	
	// Verify stage updated
	if orchestrator.status.Stage != StagePlanning {
		t.Errorf("Expected stage %s, got %s", StagePlanning, orchestrator.status.Stage)
	}
	
	// Verify results stored
	if orchestrator.results[StageParsing] != "test_parse_results" {
		t.Errorf("Expected parse results to be stored")
	}
}

func TestEventOrchestratorParseFailedHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	errorHandler := NewErrorHandler(logger, config)
	orchestrator.SetErrorHandler(errorHandler)
	
	// Create parse failed event
	event := events.NewEvent("parser.failed", map[string]interface{}{
		"error": "parse error occurred",
	})
	
	ctx := context.Background()
	err := orchestrator.handleParseFailed(ctx, event)
	
	if err == nil {
		t.Error("Expected error from handleParseFailed")
	}
	
	// Verify stage updated
	if orchestrator.status.Stage != StageFailed {
		t.Errorf("Expected stage %s, got %s", StageFailed, orchestrator.status.Stage)
	}
	
	// Verify error collected
	errorReport := errorHandler.GetErrors()
	if errorReport.TotalCount != 1 {
		t.Errorf("Expected 1 error, got %d", errorReport.TotalCount)
	}
}

func TestEventOrchestratorComponentStatusHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	// Create component status change event
	event := events.NewEvent("component.status.changed", map[string]interface{}{
		"component": "parser",
		"status":    StatusRunning,
	})
	
	ctx := context.Background()
	err := orchestrator.handleComponentStatusChanged(ctx, event)
	
	if err != nil {
		t.Fatalf("handleComponentStatusChanged failed: %v", err)
	}
	
	// Verify status updated
	if orchestrator.status.ComponentStatus["parser"] != StatusRunning {
		t.Errorf("Expected parser status %s, got %s", 
			StatusRunning, orchestrator.status.ComponentStatus["parser"])
	}
}

func TestEventOrchestratorStageProgress(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	tests := []struct {
		stage            PipelineStage
		expectedProgress float64
	}{
		{StageInitializing, 0.0},
		{StageParsing, 0.2},
		{StagePlanning, 0.4},
		{StageExecuting, 0.6},
		{StageEmitting, 0.8},
		{StageCompleted, 1.0},
		{StageFailed, 1.0},
	}
	
	for _, test := range tests {
		orchestrator.updateStage(test.stage)
		
		if orchestrator.status.Progress != test.expectedProgress {
			t.Errorf("Stage %s: expected progress %f, got %f", 
				test.stage, test.expectedProgress, orchestrator.status.Progress)
		}
	}
}

// Test event handling chain

func TestEventOrchestratorPipelineFlow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	ctx := context.Background()
	
	// Test complete pipeline flow
	stages := []struct {
		eventType string
		eventData map[string]interface{}
		expectedStage PipelineStage
	}{
		{
			"parser.completed",
			map[string]interface{}{"results": "parse_results"},
			StagePlanning,
		},
		{
			"planner.completed", 
			map[string]interface{}{"results": "plan_results"},
			StageExecuting,
		},
		{
			"executor.completed",
			map[string]interface{}{"results": "execute_results"},
			StageEmitting,
		},
		{
			"emitter.completed",
			map[string]interface{}{"results": "emit_results"},
			StageCompleted,
		},
	}
	
	for i, stage := range stages {
		event := events.NewEvent(stage.eventType, stage.eventData)
		
		handler, exists := orchestrator.eventHandlers[stage.eventType]
		if !exists {
			t.Fatalf("Handler for %s not found", stage.eventType)
		}
		
		err := handler(ctx, event)
		if err != nil {
			t.Fatalf("Handler for %s failed: %v", stage.eventType, err)
		}
		
		// Verify stage progression
		if orchestrator.status.Stage != stage.expectedStage {
			t.Errorf("Step %d: expected stage %s, got %s", 
				i, stage.expectedStage, orchestrator.status.Stage)
		}
		
		// Verify results stored
		var expectedResultStage PipelineStage
		switch stage.eventType {
		case "parser.completed":
			expectedResultStage = StageParsing
		case "planner.completed":
			expectedResultStage = StagePlanning
		case "executor.completed":
			expectedResultStage = StageExecuting
		case "emitter.completed":
			expectedResultStage = StageEmitting
		}
		
		if result, exists := orchestrator.results[expectedResultStage]; !exists {
			t.Errorf("Expected result for stage %s not found", expectedResultStage)
		} else if result != stage.eventData["results"] {
			t.Errorf("Expected result %v, got %v", stage.eventData["results"], result)
		}
	}
}

// Concurrent access tests

func TestEventOrchestratorConcurrentEventHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	config.EventBufferSize = 1000 // Large buffer for concurrent test
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	ctx := context.Background()
	done := make(chan bool, 10)
	
	// Concurrent event handling
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 50; j++ {
				event := events.NewEvent("test.event", map[string]interface{}{
					"id":        id,
					"iteration": j,
				})
				
				err := orchestrator.HandleEvent(ctx, event)
				if err != nil {
					t.Errorf("HandleEvent failed: %v", err)
				}
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify orchestrator is still functional
	status := orchestrator.GetStatus()
	if status == nil {
		t.Error("GetStatus failed after concurrent event handling")
	}
}

func TestEventOrchestratorConcurrentStatusAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	done := make(chan bool, 10)
	
	// Concurrent status updates and reads
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				if j%2 == 0 {
					// Update stage
					orchestrator.updateStage(StageParsing)
				} else {
					// Read status
					_ = orchestrator.GetStatus()
				}
			}
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify final state is consistent
	finalStatus := orchestrator.GetStatus()
	if finalStatus == nil {
		t.Error("Final status is nil")
	}
}

// Benchmark tests

func BenchmarkEventOrchestratorHandleEvent(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	event := events.NewEvent("test.event", map[string]interface{}{
		"test": "data",
	})
	
	ctx := context.Background()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := orchestrator.HandleEvent(ctx, event)
		if err != nil {
			b.Fatalf("HandleEvent failed: %v", err)
		}
	}
}

func BenchmarkEventOrchestratorGetStatus(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = orchestrator.GetStatus()
	}
}

func BenchmarkEventOrchestratorUpdateStage(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	config := createTestConfig()
	
	orchestrator := NewEventOrchestrator(logger, eventBus, config).(*ConcreteEventOrchestrator)
	
	stages := []PipelineStage{
		StageParsing, StagePlanning, StageExecuting, StageEmitting, StageCompleted,
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		stage := stages[i%len(stages)]
		orchestrator.updateStage(stage)
	}
}