package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/emitter"
	"github.com/reedom/convergen/v8/pkg/executor"
	"github.com/reedom/convergen/v8/pkg/parser"
	"github.com/reedom/convergen/v8/pkg/planner"
	"go.uber.org/zap/zaptest"
)

// Integration tests that test the complete coordinator system with all components

func TestCoordinatorEndToEndIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	
	// Create coordinator
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	// Verify all subsystems are initialized
	verifyCoordinatorSubsystems(t, coord)
	
	// Test metrics and status
	initialMetrics := coord.GetMetrics()
	if initialMetrics.PipelineExecutions != 0 {
		t.Errorf("Expected 0 initial executions, got %d", initialMetrics.PipelineExecutions)
	}
	
	initialStatus := coord.GetStatus()
	if initialStatus.Stage != StageInitializing {
		t.Errorf("Expected initial stage %s, got %s", StageInitializing, initialStatus.Stage)
	}
	
	t.Logf("Coordinator initialized successfully with %d components", 
		len(initialStatus.ComponentStatus))
}

func TestCoordinatorComponentLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	
	coord := New(logger, config)
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Test component manager initialization
	ctx := context.Background()
	err := concreteCoord.componentMgr.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Component initialization failed: %v", err)
	}
	
	// Verify all components are ready
	expectedComponents := []string{"parser", "planner", "executor", "emitter"}
	components := concreteCoord.componentMgr.GetComponents()
	
	for _, name := range expectedComponents {
		if _, exists := components[name]; !exists {
			t.Errorf("Component %s not found", name)
		}
		
		status := concreteCoord.componentMgr.GetComponentStatus(name)
		if status != StatusReady {
			t.Errorf("Component %s not ready, status: %s", name, status)
		}
	}
	
	// Test shutdown
	err = coord.Shutdown(ctx)
	if err != nil {
		t.Errorf("Coordinator shutdown failed: %v", err)
	}
	
	// Verify components are shutdown
	for _, name := range expectedComponents {
		status := concreteCoord.componentMgr.GetComponentStatus(name)
		if status != StatusShutdown {
			t.Errorf("Component %s not shutdown, status: %s", name, status)
		}
	}
}

func TestCoordinatorErrorHandlingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	config.StopOnFirstError = false
	config.ErrorThreshold = 5
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Test error collection
	testErrors := []struct {
		component string
		errorMsg  string
		critical  bool
	}{
		{"parser", "syntax error in line 10", false},
		{"planner", "circular dependency detected", false},
		{"executor", "conversion failed", false},
		{"emitter", "code generation error", false},
		{"parser", "critical parse failure", true},
	}
	
	for _, testErr := range testErrors {
		if testErr.critical {
			concreteCoord.errorHandler.CollectCriticalError(testErr.component, 
				&testError{testErr.errorMsg})
		} else {
			concreteCoord.errorHandler.CollectError(testErr.component, 
				&testError{testErr.errorMsg})
		}
	}
	
	// Check error report
	errorReport := concreteCoord.errorHandler.GetErrors()
	
	if errorReport.TotalCount != 5 {
		t.Errorf("Expected 5 total errors, got %d", errorReport.TotalCount)
	}
	
	if errorReport.CriticalCount != 1 {
		t.Errorf("Expected 1 critical error, got %d", errorReport.CriticalCount)
	}
	
	// Should stop due to critical error
	if !concreteCoord.errorHandler.ShouldStop() {
		t.Error("Expected pipeline to stop due to critical error")
	}
	
	// Test error statistics
	stats := concreteCoord.errorHandler.GetErrorStats()
	if stats["parser_errors"] != 2 {
		t.Errorf("Expected 2 parser errors, got %d", stats["parser_errors"])
	}
}

func TestCoordinatorMetricsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	config.EnableMetrics = true
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Simulate pipeline events
	events := []struct {
		name     string
		duration time.Duration
		metadata map[string]interface{}
	}{
		{"pipeline_started", 0, map[string]interface{}{"sources": 3}},
		{"parser_completed", 100 * time.Millisecond, map[string]interface{}{"files": 3}},
		{"planner_completed", 50 * time.Millisecond, map[string]interface{}{"plans": 5}},
		{"executor_completed", 200 * time.Millisecond, map[string]interface{}{"methods": 10}},
		{"emitter_completed", 150 * time.Millisecond, map[string]interface{}{"lines": 500}},
		{"pipeline_completed", 500 * time.Millisecond, map[string]interface{}{"success": true}},
	}
	
	for _, event := range events {
		concreteCoord.metricsCollector.RecordEvent(event.name, event.duration, event.metadata)
	}
	
	// Record component metrics
	componentMetrics := map[string]interface{}{
		"parser": map[string]interface{}{
			"files_parsed": 3,
			"lines_parsed": 1500,
		},
		"executor": map[string]interface{}{
			"methods_processed": 10,
			"conversion_time":   200 * time.Millisecond,
		},
	}
	
	for component, metrics := range componentMetrics {
		concreteCoord.metricsCollector.RecordComponent(component, metrics)
	}
	
	// Get final metrics
	finalMetrics := coord.GetMetrics()
	
	if finalMetrics.PipelineExecutions != 1 {
		t.Errorf("Expected 1 pipeline execution, got %d", finalMetrics.PipelineExecutions)
	}
	
	if finalMetrics.SuccessRate != 1.0 {
		t.Errorf("Expected success rate 1.0, got %f", finalMetrics.SuccessRate)
	}
	
	if finalMetrics.EventCounts["pipeline_completed"] != 1 {
		t.Errorf("Expected 1 completion event, got %d", 
			finalMetrics.EventCounts["pipeline_completed"])
	}
	
	if len(finalMetrics.ComponentMetrics) != 2 {
		t.Errorf("Expected 2 component metrics, got %d", len(finalMetrics.ComponentMetrics))
	}
}

func TestCoordinatorResourceManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	config.WorkerPoolSize = 4
	config.BufferPoolSize = 8
	config.ChannelPoolSize = 4
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	concreteCoord := coord.(*ConcreteCoordinator)
	ctx := context.Background()
	
	// Test worker pool creation
	workerPool, err := concreteCoord.resourcePool.GetWorkerPool(ctx, 3)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}
	
	if workerPool.Size != 3 {
		t.Errorf("Expected worker pool size 3, got %d", workerPool.Size)
	}
	
	// Test buffer pool
	bufferPool := concreteCoord.resourcePool.GetBufferPool()
	if bufferPool == nil {
		t.Fatal("GetBufferPool returned nil")
	}
	
	buffer := bufferPool.GetBuffer()
	if buffer == nil {
		t.Error("GetBuffer returned nil")
	}
	bufferPool.PutBuffer(buffer)
	
	// Test channel pool
	channelPool := concreteCoord.resourcePool.GetChannelPool()
	if channelPool == nil {
		t.Fatal("GetChannelPool returned nil")
	}
	
	eventChan := channelPool.GetEventChannel()
	if eventChan == nil {
		t.Error("GetEventChannel returned nil")
	}
	channelPool.PutEventChannel(eventChan)
	
	// Test resource usage tracking
	usage := concreteCoord.resourcePool.GetResourceUsage()
	if usage == nil {
		t.Fatal("GetResourceUsage returned nil")
	}
	
	if usage.GoroutineCount < 0 {
		t.Errorf("Expected non-negative goroutine count, got %d", usage.GoroutineCount)
	}
}

func TestCoordinatorContextManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Test pipeline context creation
	parentCtx := context.Background()
	pipelineCtx, cancel := concreteCoord.contextMgr.CreatePipelineContext(
		parentCtx, 5*time.Second)
	defer cancel()
	
	// Verify context has timeout
	deadline, hasDeadline := pipelineCtx.Deadline()
	if !hasDeadline {
		t.Error("Expected pipeline context to have deadline")
	}
	
	if time.Until(deadline) > 5*time.Second {
		t.Error("Pipeline context deadline too far in future")
	}
	
	// Test component context creation
	componentCtx := concreteCoord.contextMgr.CreateComponentContext(pipelineCtx, "parser")
	
	// Verify component context has metadata
	if component, ok := GetComponentName(componentCtx); !ok || component != "parser" {
		t.Errorf("Expected component name 'parser', got %q", component)
	}
	
	if startTime, ok := GetStartTime(componentCtx); !ok {
		t.Error("Expected start time in component context")
	} else if time.Since(startTime) > time.Second {
		t.Error("Component context start time too old")
	}
	
	// Test context tracking
	activeCount := concreteCoord.contextMgr.GetActiveContextCount()
	if activeCount < 1 {
		t.Errorf("Expected at least 1 active context, got %d", activeCount)
	}
}

func TestCoordinatorConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	config.MaxConcurrency = 4
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Test concurrent metric recording
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 50; j++ {
				// Record metrics
				concreteCoord.metricsCollector.RecordEvent(
					"test_event", time.Millisecond, 
					map[string]interface{}{"id": id, "iteration": j})
				
				// Record errors
				concreteCoord.errorHandler.CollectError("component", 
					&testError{"concurrent error"})
				
				// Get status
				_ = coord.GetStatus()
				
				// Get metrics
				_ = coord.GetMetrics()
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify final state is consistent
	finalMetrics := coord.GetMetrics()
	errorReport := concreteCoord.errorHandler.GetErrors()
	
	if finalMetrics.EventCounts["test_event"] != 500 {
		t.Errorf("Expected 500 test events, got %d", 
			finalMetrics.EventCounts["test_event"])
	}
	
	if errorReport.TotalCount != 500 {
		t.Errorf("Expected 500 errors, got %d", errorReport.TotalCount)
	}
}

func TestCoordinatorShutdownIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	
	coord := New(logger, config)
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Initialize components
	ctx := context.Background()
	err := concreteCoord.componentMgr.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Component initialization failed: %v", err)
	}
	
	// Create some resources
	_, err = concreteCoord.resourcePool.GetWorkerPool(ctx, 2)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}
	
	// Record some activity
	concreteCoord.metricsCollector.RecordEvent("test", time.Millisecond, nil)
	concreteCoord.errorHandler.CollectError("test", &testError{"test error"})
	
	// Test graceful shutdown
	shutdownStart := time.Now()
	err = coord.Shutdown(ctx)
	shutdownDuration := time.Since(shutdownStart)
	
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	
	if shutdownDuration > 5*time.Second {
		t.Errorf("Shutdown took too long: %v", shutdownDuration)
	}
	
	// Verify components are shutdown
	components := []string{"parser", "planner", "executor", "emitter"}
	for _, name := range components {
		status := concreteCoord.componentMgr.GetComponentStatus(name)
		if status != StatusShutdown {
			t.Errorf("Component %s not shutdown, status: %s", name, status)
		}
	}
	
	// Test idempotent shutdown
	err = coord.Shutdown(ctx)
	if err != nil {
		t.Errorf("Second shutdown failed: %v", err)
	}
}

// Helper functions for integration tests

func createIntegrationTestConfig() *Config {
	return &Config{
		ParserConfig: &parser.ParserConfig{
			BuildTag:              "convergen",
			MaxConcurrentWorkers:  4,
			TypeResolutionTimeout: 30 * time.Second,
			CacheSize:             1000,
			EnableProgress:        false,
		},
		PlannerConfig:  planner.DefaultPlannerConfig(),
		ExecutorConfig: &executor.ExecutorConfig{
			MaxWorkers:        4,
			MinWorkers:        1,
			MaxConcurrentJobs: 10,
			ExecutionTimeout:  30 * time.Second,
			RetryAttempts:     3,
		},
		EmitterConfig:  emitter.DefaultEmitterConfig(),
		
		MaxConcurrency:     2,
		EventBufferSize:    100,
		ComponentTimeout:   5 * time.Second,
		ErrorThreshold:     10,
		EnableMetrics:      true,
		LogLevel:          "info",
		
		WorkerPoolSize:  4,
		BufferPoolSize:  8,
		ChannelPoolSize: 4,
		
		StopOnFirstError:     false,
		RetryTransientErrors: true,
		MaxRetries:           2,
		RetryDelay:           50 * time.Millisecond,
		
		EnableProfiling:    false,
		EnableEventTracing: false,
	}
}

func verifyCoordinatorSubsystems(t *testing.T, coord Coordinator) {
	concreteCoord := coord.(*ConcreteCoordinator)
	
	if concreteCoord.componentMgr == nil {
		t.Error("ComponentManager not initialized")
	}
	
	if concreteCoord.eventOrchestrator == nil {
		t.Error("EventOrchestrator not initialized")
	}
	
	if concreteCoord.resourcePool == nil {
		t.Error("ResourcePool not initialized")
	}
	
	if concreteCoord.errorHandler == nil {
		t.Error("ErrorHandler not initialized")
	}
	
	if concreteCoord.metricsCollector == nil {
		t.Error("MetricsCollector not initialized")
	}
	
	if concreteCoord.contextMgr == nil {
		t.Error("ContextManager not initialized")
	}
	
	if concreteCoord.eventBus == nil {
		t.Error("EventBus not initialized")
	}
}

// Test error type for integration tests
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

// Performance integration tests

func TestCoordinatorPerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	config.EnableMetrics = true
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	// Measure coordinator creation time
	start := time.Now()
	coord2 := New(logger, config)
	creationTime := time.Since(start)
	coord2.Shutdown(context.Background())
	
	if creationTime > 100*time.Millisecond {
		t.Errorf("Coordinator creation took too long: %v", creationTime)
	}
	
	// Measure metrics collection performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = coord.GetMetrics()
	}
	metricsTime := time.Since(start)
	
	avgMetricsTime := metricsTime / 1000
	if avgMetricsTime > time.Millisecond {
		t.Errorf("Average metrics collection time too high: %v", avgMetricsTime)
	}
	
	// Measure status collection performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = coord.GetStatus()
	}
	statusTime := time.Since(start)
	
	avgStatusTime := statusTime / 1000
	if avgStatusTime > time.Millisecond {
		t.Errorf("Average status collection time too high: %v", avgStatusTime)
	}
	
	t.Logf("Performance metrics - Creation: %v, Metrics: %v/call, Status: %v/call",
		creationTime, avgMetricsTime, avgStatusTime)
}

func TestCoordinatorMemoryUsageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory integration test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	
	// Create multiple coordinators to test memory usage
	coords := make([]Coordinator, 10)
	
	for i := 0; i < 10; i++ {
		coords[i] = New(logger, config)
	}
	
	// Get resource usage
	concreteCoord := coords[0].(*ConcreteCoordinator)
	usage := concreteCoord.resourcePool.GetResourceUsage()
	
	// Verify reasonable memory usage
	if usage.CurrentMemoryUsage < 0 {
		t.Error("Expected non-negative memory usage")
	}
	
	if usage.GoroutineCount < 0 {
		t.Error("Expected non-negative goroutine count")
	}
	
	// Cleanup
	for _, coord := range coords {
		coord.Shutdown(context.Background())
	}
	
	t.Logf("Memory usage - Current: %d bytes, Goroutines: %d", 
		usage.CurrentMemoryUsage, usage.GoroutineCount)
}

// Stress tests

func TestCoordinatorStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	logger := zaptest.NewLogger(t)
	config := createIntegrationTestConfig()
	config.MaxConcurrency = 8
	config.EventBufferSize = 10000
	
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	concreteCoord := coord.(*ConcreteCoordinator)
	
	// Stress test with high concurrency
	numGoroutines := 50
	operationsPerGoroutine := 100
	done := make(chan bool, numGoroutines)
	
	start := time.Now()
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of operations
				switch j % 5 {
				case 0:
					concreteCoord.metricsCollector.RecordEvent("stress_event", 
						time.Microsecond, map[string]interface{}{"id": id})
				case 1:
					concreteCoord.errorHandler.CollectError("stress_component", 
						&testError{"stress error"})
				case 2:
					_ = coord.GetMetrics()
				case 3:
					_ = coord.GetStatus()
				case 4:
					usage := concreteCoord.resourcePool.GetResourceUsage()
					_ = usage
				}
			}
		}(i)
	}
	
	// Wait for completion
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	duration := time.Since(start)
	totalOps := numGoroutines * operationsPerGoroutine
	opsPerSecond := float64(totalOps) / duration.Seconds()
	
	t.Logf("Stress test completed - %d ops in %v (%.0f ops/sec)", 
		totalOps, duration, opsPerSecond)
	
	// Verify system is still functional
	finalMetrics := coord.GetMetrics()
	if finalMetrics == nil {
		t.Error("Final metrics collection failed")
	}
	
	finalStatus := coord.GetStatus()
	if finalStatus == nil {
		t.Error("Final status collection failed")
	}
	
	// Should have processed many events
	if finalMetrics.EventCounts["stress_event"] != int64(numGoroutines*operationsPerGoroutine/5) {
		t.Errorf("Expected %d stress events, got %d", 
			numGoroutines*operationsPerGoroutine/5, 
			finalMetrics.EventCounts["stress_event"])
	}
}