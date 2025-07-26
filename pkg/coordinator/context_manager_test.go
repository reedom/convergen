package coordinator

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewContextManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	mgr := NewContextManager(logger, config)

	if mgr == nil {
		t.Fatal("NewContextManager returned nil")
	}

	// Verify it implements the interface
	var _ ContextManager = mgr
}

func TestContextManagerCreatePipelineContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	parentCtx := context.Background()
	timeout := 5 * time.Second

	ctx, cancel := mgr.CreatePipelineContext(parentCtx, timeout)
	defer cancel()

	if ctx == nil {
		t.Fatal("CreatePipelineContext returned nil context")
	}

	// Verify context has deadline
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected pipeline context to have deadline")
	}

	expectedDeadline := time.Now().Add(timeout)
	if deadline.After(expectedDeadline.Add(time.Second)) ||
		deadline.Before(expectedDeadline.Add(-time.Second)) {
		t.Errorf("Pipeline context deadline not within expected range")
	}

	// Verify pipeline metadata exists
	if pipelineID, ok := GetPipelineID(ctx); !ok {
		t.Error("Expected pipeline ID in context")
	} else if pipelineID == "" {
		t.Error("Expected non-empty pipeline ID")
	}

	if startTime, ok := GetStartTime(ctx); !ok {
		t.Error("Expected start time in context")
	} else if time.Since(startTime) > time.Second {
		t.Error("Start time too old")
	}
}

func TestContextManagerCreateComponentContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	parentCtx := context.Background()
	componentName := "test-component"

	ctx := mgr.CreateComponentContext(parentCtx, componentName)

	if ctx == nil {
		t.Fatal("CreateComponentContext returned nil")
	}

	// Verify component metadata
	if component, ok := GetComponentName(ctx); !ok {
		t.Error("Expected component name in context")
	} else if component != componentName {
		t.Errorf("Expected component name %q, got %q", componentName, component)
	}

	if startTime, ok := GetStartTime(ctx); !ok {
		t.Error("Expected start time in component context")
	} else if time.Since(startTime) > time.Second {
		t.Error("Component start time too old")
	}
}

func TestContextManagerShouldCancel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	// Test active context
	activeCtx := context.Background()
	if mgr.ShouldCancel(activeCtx) {
		t.Error("Expected active context not to be cancelled")
	}

	// Test cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if !mgr.ShouldCancel(cancelledCtx) {
		t.Error("Expected cancelled context to be cancelled")
	}

	// Test timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer timeoutCancel()

	time.Sleep(time.Millisecond) // Ensure timeout

	if !mgr.ShouldCancel(timeoutCtx) {
		t.Error("Expected timeout context to be cancelled")
	}
}

func TestContextManagerCancelAll(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	// Create some contexts
	parentCtx := context.Background()

	ctx1, cancel1 := mgr.CreatePipelineContext(parentCtx, time.Minute)
	defer cancel1()

	ctx2 := mgr.CreateComponentContext(parentCtx, "component1")
	ctx3 := mgr.CreateComponentContext(parentCtx, "component2")

	// Track contexts
	mgr.TrackContext(ctx2, "component1")
	mgr.TrackContext(ctx3, "component2")

	initialCount := mgr.GetActiveContextCount()
	if initialCount == 0 {
		t.Error("Expected some active contexts before cancel all")
	}

	// Cancel all
	mgr.CancelAll()

	// Verify contexts are cancelled
	if !mgr.ShouldCancel(ctx1) {
		t.Error("Expected ctx1 to be cancelled after CancelAll")
	}

	finalCount := mgr.GetActiveContextCount()
	if finalCount != 0 {
		t.Errorf("Expected 0 active contexts after cancel all, got %d", finalCount)
	}
}

func TestContextManagerAddGetMetadata(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	ctx := context.Background()

	// Add metadata
	key := "test_key"
	value := "test_value"

	newCtx := mgr.AddMetadata(ctx, key, value)

	// Get metadata
	retrievedValue, exists := mgr.GetMetadata(newCtx, key)
	if !exists {
		t.Error("Expected metadata to exist")
	}

	if retrievedValue != value {
		t.Errorf("Expected metadata value %q, got %q", value, retrievedValue)
	}

	// Test non-existent metadata
	_, exists = mgr.GetMetadata(newCtx, "non_existent")
	if exists {
		t.Error("Expected non-existent metadata to not exist")
	}
}

func TestContextManagerTrackContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	ctx := context.Background()
	description := "test context"

	initialCount := mgr.GetActiveContextCount()

	mgr.TrackContext(ctx, description)

	newCount := mgr.GetActiveContextCount()
	if newCount != initialCount+1 {
		t.Errorf("Expected active context count to increase by 1, got %d", newCount-initialCount)
	}
}

func TestContextManagerContextCleanup(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config).(*ConcreteContextManager)

	parentCtx := context.Background()

	// Create pipeline context
	_, cancel := mgr.CreatePipelineContext(parentCtx, time.Minute)

	initialCount := mgr.GetActiveContextCount()
	if initialCount == 0 {
		t.Error("Expected at least 1 active context")
	}

	// Cancel should clean up
	cancel()

	finalCount := mgr.GetActiveContextCount()
	if finalCount >= initialCount {
		t.Errorf("Expected active context count to decrease, got %d -> %d",
			initialCount, finalCount)
	}
}

func TestContextManagerGetContextInfo(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config).(*ConcreteContextManager)

	// Create context with metadata
	parentCtx := context.Background()
	ctx, cancel := mgr.CreatePipelineContext(parentCtx, time.Minute)
	defer cancel()

	ctx = mgr.AddMetadata(ctx, "test_key", "test_value")
	mgr.TrackContext(ctx, "test pipeline")

	// Get context info
	info := mgr.GetContextInfo(ctx)

	if info == nil {
		t.Fatal("GetContextInfo returned nil")
	}

	if description, ok := info["description"].(string); !ok || description != "test pipeline" {
		t.Errorf("Expected description 'test pipeline', got %v", info["description"])
	}

	if status, ok := info["status"].(string); !ok || status != "active" {
		t.Errorf("Expected status 'active', got %v", info["status"])
	}

	if _, ok := info["pipeline_id"]; !ok {
		t.Error("Expected pipeline_id in context info")
	}

	if _, ok := info["start_time"]; !ok {
		t.Error("Expected start_time in context info")
	}

	if _, ok := info["elapsed_time"]; !ok {
		t.Error("Expected elapsed_time in context info")
	}
}

func TestContextManagerGetAllContextInfo(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config).(*ConcreteContextManager)

	// Create multiple contexts
	parentCtx := context.Background()

	_, cancel1 := mgr.CreatePipelineContext(parentCtx, time.Minute)
	defer cancel1()

	ctx2 := mgr.CreateComponentContext(parentCtx, "component1")
	mgr.TrackContext(ctx2, "component context")

	// Get all context info
	allInfo := mgr.GetAllContextInfo()

	if allInfo == nil {
		t.Fatal("GetAllContextInfo returned nil")
	}

	if activeCount, ok := allInfo["active_count"]; !ok {
		t.Error("Expected active_count in all context info")
	} else if count, ok := activeCount.(int); !ok || count < 1 {
		t.Errorf("Expected at least 1 active context, got %v", activeCount)
	}

	if contexts, ok := allInfo["contexts"]; !ok {
		t.Error("Expected contexts array in all context info")
	} else if contextList, ok := contexts.([]map[string]interface{}); !ok {
		t.Error("Expected contexts to be array of maps")
	} else if len(contextList) < 1 {
		t.Errorf("Expected at least 1 context in list, got %d", len(contextList))
	}
}

// Test utility functions

func TestGetPipelineID(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextKeyPipelineID, "test-pipeline-123")

	pipelineID, ok := GetPipelineID(ctx)
	if !ok {
		t.Error("Expected to find pipeline ID")
	}

	if pipelineID != "test-pipeline-123" {
		t.Errorf("Expected pipeline ID 'test-pipeline-123', got %q", pipelineID)
	}

	// Test context without pipeline ID
	emptyCtx := context.Background()
	_, ok = GetPipelineID(emptyCtx)
	if ok {
		t.Error("Expected not to find pipeline ID in empty context")
	}
}

func TestGetComponentName(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextKeyComponent, "test-component")

	component, ok := GetComponentName(ctx)
	if !ok {
		t.Error("Expected to find component name")
	}

	if component != "test-component" {
		t.Errorf("Expected component name 'test-component', got %q", component)
	}

	// Test context without component name
	emptyCtx := context.Background()
	_, ok = GetComponentName(emptyCtx)
	if ok {
		t.Error("Expected not to find component name in empty context")
	}
}

func TestGetStartTime(t *testing.T) {
	now := time.Now()
	ctx := context.WithValue(context.Background(), contextKeyStartTime, now)

	startTime, ok := GetStartTime(ctx)
	if !ok {
		t.Error("Expected to find start time")
	}

	if !startTime.Equal(now) {
		t.Errorf("Expected start time %v, got %v", now, startTime)
	}

	// Test context without start time
	emptyCtx := context.Background()
	_, ok = GetStartTime(emptyCtx)
	if ok {
		t.Error("Expected not to find start time in empty context")
	}
}

func TestGetElapsedTime(t *testing.T) {
	startTime := time.Now().Add(-time.Second)
	ctx := context.WithValue(context.Background(), contextKeyStartTime, startTime)

	elapsed, ok := GetElapsedTime(ctx)
	if !ok {
		t.Error("Expected to find elapsed time")
	}

	if elapsed <= 0 {
		t.Errorf("Expected positive elapsed time, got %v", elapsed)
	}

	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected elapsed time around 1 second, got %v", elapsed)
	}
}

func TestIsContextActive(t *testing.T) {
	// Test active context
	activeCtx := context.Background()
	if !IsContextActive(activeCtx) {
		t.Error("Expected active context to be active")
	}

	// Test cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if IsContextActive(cancelledCtx) {
		t.Error("Expected cancelled context to be inactive")
	}

	// Test timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer timeoutCancel()

	time.Sleep(time.Millisecond) // Ensure timeout

	if IsContextActive(timeoutCtx) {
		t.Error("Expected timeout context to be inactive")
	}
}

// Concurrent access tests

func TestContextManagerConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	parentCtx := context.Background()
	done := make(chan bool, 10)

	// Concurrent context operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 50; j++ {
				// Create contexts
				ctx, cancel := mgr.CreatePipelineContext(parentCtx, time.Minute)

				componentCtx := mgr.CreateComponentContext(ctx, "component")

				// Add metadata
				ctx = mgr.AddMetadata(ctx, "key", "value")

				// Get metadata
				_, _ = mgr.GetMetadata(ctx, "key")

				// Track context
				mgr.TrackContext(componentCtx, "test")

				// Get count
				_ = mgr.GetActiveContextCount()

				// Cancel
				cancel()
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	finalCount := mgr.GetActiveContextCount()
	if finalCount < 0 {
		t.Errorf("Expected non-negative active context count, got %d", finalCount)
	}
}

// Benchmark tests

func BenchmarkContextManagerCreatePipelineContext(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	parentCtx := context.Background()
	timeout := time.Minute

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := mgr.CreatePipelineContext(parentCtx, timeout)
		_ = ctx
		cancel()
	}
}

func BenchmarkContextManagerCreateComponentContext(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	parentCtx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := mgr.CreateComponentContext(parentCtx, "component")
		_ = ctx
	}
}

func BenchmarkContextManagerAddGetMetadata(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	mgr := NewContextManager(logger, config)

	ctx := context.Background()
	ctx = mgr.AddMetadata(ctx, "key", "value")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mgr.GetMetadata(ctx, "key")
	}
}

func BenchmarkGetPipelineID(b *testing.B) {
	ctx := context.WithValue(context.Background(), contextKeyPipelineID, "test-pipeline")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = GetPipelineID(ctx)
	}
}

func BenchmarkIsContextActive(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = IsContextActive(ctx)
	}
}
