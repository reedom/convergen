package coordinator

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// Mock component for testing
type mockComponent struct {
	name    string
	status  ComponentStatus
	metrics interface{}
	initErr error
	shutErr error
}

func (m *mockComponent) Name() string { return m.name }

func (m *mockComponent) Initialize(ctx context.Context, eventBus events.EventBus) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.status = StatusReady
	return nil
}

func (m *mockComponent) Shutdown(ctx context.Context) error {
	if m.shutErr != nil {
		return m.shutErr
	}
	m.status = StatusShutdown
	return nil
}

func (m *mockComponent) GetMetrics() interface{}    { return m.metrics }
func (m *mockComponent) GetStatus() ComponentStatus { return m.status }

func TestNewComponentManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	mgr := NewComponentManager(logger, config)

	if mgr == nil {
		t.Fatal("NewComponentManager returned nil")
	}

	// Verify it implements the interface
	var _ ComponentManager = mgr
}

func TestComponentManagerInitialize(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify components are initialized
	components := mgr.GetComponents()
	expectedComponents := []string{"parser", "planner", "executor", "emitter"}

	for _, expected := range expectedComponents {
		if _, exists := components[expected]; !exists {
			t.Errorf("Expected component %s not found", expected)
		}
	}

	// Verify component status
	for _, name := range expectedComponents {
		status := mgr.GetComponentStatus(name)
		if status != StatusReady {
			t.Errorf("Expected component %s to be ready, got %s", name, status)
		}
	}
}

func TestComponentManagerInitializeIdempotent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()

	// First initialization
	err1 := mgr.Initialize(ctx, config)
	if err1 != nil {
		t.Fatalf("First Initialize failed: %v", err1)
	}

	// Second initialization should be idempotent
	err2 := mgr.Initialize(ctx, config)
	if err2 != nil {
		t.Errorf("Second Initialize failed: %v", err2)
	}
}

func TestComponentManagerRegisterComponent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	// Create mock component
	mockComp := &mockComponent{
		name:   "test-component",
		status: StatusReady,
	}

	err := mgr.RegisterComponent("test-component", mockComp)
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	// Verify component is registered
	component, err := mgr.GetComponent("test-component")
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}

	if component != mockComp {
		t.Error("Retrieved component is not the same as registered")
	}

	// Verify status
	status := mgr.GetComponentStatus("test-component")
	if status != StatusReady {
		t.Errorf("Expected status %s, got %s", StatusReady, status)
	}
}

func TestComponentManagerRegisterAfterInitialize(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Try to register after initialization
	mockComp := &mockComponent{name: "late-component"}
	err = mgr.RegisterComponent("late-component", mockComp)

	if err == nil {
		t.Error("Expected error when registering component after initialization")
	}

	expectedMsg := "cannot register component after initialization"
	if err.Error() != expectedMsg+": late-component" {
		t.Errorf("Expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestComponentManagerGetNonExistentComponent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	_, err := mgr.GetComponent("non-existent")

	if err == nil {
		t.Error("Expected error for non-existent component")
	}

	expectedMsg := "component not found: non-existent"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestComponentManagerUpdateComponentStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	// Register a component
	mockComp := &mockComponent{name: "test-component", status: StatusReady}
	err := mgr.RegisterComponent("test-component", mockComp)
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	// Update status
	mgr.UpdateComponentStatus("test-component", StatusRunning)

	// Verify status was updated
	status := mgr.GetComponentStatus("test-component")
	if status != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, status)
	}
}

func TestComponentManagerShutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Shutdown
	err = mgr.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify all components are marked as shutdown
	expectedComponents := []string{"parser", "planner", "executor", "emitter"}
	for _, name := range expectedComponents {
		status := mgr.GetComponentStatus(name)
		if status != StatusShutdown {
			t.Errorf("Expected component %s to be shutdown, got %s", name, status)
		}
	}
}

func TestComponentManagerShutdownWithErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	// Register a component that will fail shutdown
	failingComp := &mockComponent{
		name:    "failing-component",
		status:  StatusReady,
		shutErr: errors.New("shutdown failed"),
	}

	err := mgr.RegisterComponent("failing-component", failingComp)
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	ctx := context.Background()
	err = mgr.Shutdown(ctx)

	// Should return error but continue shutting down other components
	if err == nil {
		t.Error("Expected error from shutdown with failing component")
	}

	if err != nil && err.Error() != "component shutdown errors: [component failing-component shutdown failed: shutdown failed]" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestComponentManagerShutdownNotInitialized(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Shutdown(ctx)

	// Should not error if not initialized
	if err != nil {
		t.Errorf("Shutdown of uninitialized manager failed: %v", err)
	}
}

func TestComponentManagerGetComponentStatusNonExistent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	status := mgr.GetComponentStatus("non-existent")

	// Should return failed status for non-existent component
	if status != StatusFailed {
		t.Errorf("Expected %s for non-existent component, got %s", StatusFailed, status)
	}
}

// Test component adapters

func TestParserAdapter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config).(*ConcreteComponentManager)

	// Create parser adapter through factory
	factory := mgr.factories["parser"]
	component, err := factory(config.ParserConfig)
	if err != nil {
		t.Fatalf("Parser factory failed: %v", err)
	}

	adapter, ok := component.(*ParserAdapter)
	if !ok {
		t.Fatal("Expected ParserAdapter")
	}

	// Test adapter interface
	if adapter.Name() != "parser" {
		t.Errorf("Expected name 'parser', got %q", adapter.Name())
	}

	// Test initialization
	ctx := context.Background()
	eventBus := events.NewInMemoryEventBus(logger)
	err = adapter.Initialize(ctx, eventBus)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if adapter.GetStatus() != StatusReady {
		t.Errorf("Expected status %s, got %s", StatusReady, adapter.GetStatus())
	}

	// Test shutdown
	err = adapter.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if adapter.GetStatus() != StatusShutdown {
		t.Errorf("Expected status %s, got %s", StatusShutdown, adapter.GetStatus())
	}

	// Test metrics
	metrics := adapter.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be non-nil")
	}
}

func TestPlannerAdapter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config).(*ConcreteComponentManager)

	// Create planner adapter through factory
	factory := mgr.factories["planner"]
	component, err := factory(config.PlannerConfig)
	if err != nil {
		t.Fatalf("Planner factory failed: %v", err)
	}

	adapter, ok := component.(*PlannerAdapter)
	if !ok {
		t.Fatal("Expected PlannerAdapter")
	}

	if adapter.Name() != "planner" {
		t.Errorf("Expected name 'planner', got %q", adapter.Name())
	}
}

func TestExecutorAdapter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config).(*ConcreteComponentManager)

	// Create executor adapter through factory
	factory := mgr.factories["executor"]
	component, err := factory(config.ExecutorConfig)
	if err != nil {
		t.Fatalf("Executor factory failed: %v", err)
	}

	adapter, ok := component.(*ExecutorAdapter)
	if !ok {
		t.Fatal("Expected ExecutorAdapter")
	}

	if adapter.Name() != "executor" {
		t.Errorf("Expected name 'executor', got %q", adapter.Name())
	}
}

func TestEmitterAdapter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config).(*ConcreteComponentManager)

	// Create emitter adapter through factory
	factory := mgr.factories["emitter"]
	component, err := factory(config.EmitterConfig)
	if err != nil {
		t.Fatalf("Emitter factory failed: %v", err)
	}

	adapter, ok := component.(*EmitterAdapter)
	if !ok {
		t.Fatal("Expected EmitterAdapter")
	}

	if adapter.Name() != "emitter" {
		t.Errorf("Expected name 'emitter', got %q", adapter.Name())
	}
}

// Concurrent access tests

func TestComponentManagerConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test concurrent access to components
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Concurrent reads
			for j := 0; j < 100; j++ {
				_ = mgr.GetComponents()
				_ = mgr.GetComponentStatus("parser")

				component, err := mgr.GetComponent("parser")
				if err != nil {
					t.Errorf("GetComponent failed: %v", err)
				}
				if component == nil {
					t.Error("GetComponent returned nil")
				}
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cleanup
	err = mgr.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestComponentManagerConcurrentStatusUpdates(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	// Register test component
	mockComp := &mockComponent{name: "test", status: StatusReady}
	err := mgr.RegisterComponent("test", mockComp)
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	// Concurrent status updates
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				if j%2 == 0 {
					mgr.UpdateComponentStatus("test", StatusRunning)
				} else {
					mgr.UpdateComponentStatus("test", StatusReady)
				}
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final status should be either Running or Ready
	finalStatus := mgr.GetComponentStatus("test")
	if finalStatus != StatusRunning && finalStatus != StatusReady {
		t.Errorf("Expected final status to be Running or Ready, got %s", finalStatus)
	}
}

// Benchmark tests

func BenchmarkComponentManagerGetComponent(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)
	if err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := mgr.GetComponent("parser")
		if err != nil {
			b.Fatalf("GetComponent failed: %v", err)
		}
	}
}

func BenchmarkComponentManagerGetComponents(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)
	if err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = mgr.GetComponents()
	}
}

func BenchmarkComponentManagerUpdateStatus(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	mgr := NewComponentManager(logger, config)

	ctx := context.Background()
	err := mgr.Initialize(ctx, config)
	if err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			mgr.UpdateComponentStatus("parser", StatusRunning)
		} else {
			mgr.UpdateComponentStatus("parser", StatusReady)
		}
	}
}
