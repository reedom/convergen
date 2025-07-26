package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/emitter"
	"github.com/reedom/convergen/v8/pkg/executor"
	"github.com/reedom/convergen/v8/pkg/parser"
	"github.com/reedom/convergen/v8/pkg/planner"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNew(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	
	coord := New(logger, config)
	
	if coord == nil {
		t.Fatal("New() returned nil coordinator")
	}
	
	// Verify coordinator implements interface
	var _ Coordinator = coord
}

func TestNewWithNilConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	coord := New(logger, nil)
	
	if coord == nil {
		t.Fatal("New() returned nil coordinator with nil config")
	}
	
	// Should use default config
	concreteCoord := coord.(*ConcreteCoordinator)
	if concreteCoord.config == nil {
		t.Error("Expected default config to be used")
	}
}

func TestGenerateWithEmptySources(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	coord := New(logger, config)
	
	ctx := context.Background()
	result, err := coord.Generate(ctx, []string{}, config)
	
	if err == nil {
		t.Error("Expected error for empty sources")
	}
	
	if result != nil {
		t.Error("Expected nil result for empty sources")
	}
	
	expectedErr := "no source files provided"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestGenerateFromSourceWithEmptySource(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	coord := New(logger, config)
	
	ctx := context.Background()
	result, err := coord.GenerateFromSource(ctx, "", config)
	
	if err == nil {
		t.Error("Expected error for empty source code")
	}
	
	if result != nil {
		t.Error("Expected nil result for empty source code")
	}
	
	expectedErr := "no source code provided"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestGetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	coord := New(logger, config)
	
	metrics := coord.GetMetrics()
	
	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}
	
	// Verify initial metrics
	if metrics.PipelineExecutions != 0 {
		t.Errorf("Expected 0 pipeline executions, got %d", metrics.PipelineExecutions)
	}
	
	if metrics.SuccessRate != 0 {
		t.Errorf("Expected 0 success rate, got %f", metrics.SuccessRate)
	}
}

func TestGetStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	coord := New(logger, config)
	
	status := coord.GetStatus()
	
	if status == nil {
		t.Fatal("GetStatus() returned nil")
	}
	
	if status.Stage != StageInitializing {
		t.Errorf("Expected stage %s, got %s", StageInitializing, status.Stage)
	}
	
	if status.Progress != 0.0 {
		t.Errorf("Expected progress 0.0, got %f", status.Progress)
	}
	
	if status.ComponentStatus == nil {
		t.Error("Expected component status map to be initialized")
	}
}

func TestShutdownIdempotent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	coord := New(logger, config)
	
	ctx := context.Background()
	
	// First shutdown
	err1 := coord.Shutdown(ctx)
	if err1 != nil {
		t.Errorf("First shutdown failed: %v", err1)
	}
	
	// Second shutdown should be idempotent
	err2 := coord.Shutdown(ctx)
	if err2 != nil {
		t.Errorf("Second shutdown failed: %v", err2)
	}
}

func TestShutdownWithTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.ComponentTimeout = 10 * time.Millisecond // Very short timeout
	coord := New(logger, config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	
	err := coord.Shutdown(ctx)
	
	// Should complete even with short timeout
	if err != nil {
		t.Logf("Shutdown completed with error (expected due to short timeout): %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	
	// Test default values
	if config.MaxConcurrency != 4 {
		t.Errorf("Expected MaxConcurrency 4, got %d", config.MaxConcurrency)
	}
	
	if config.EventBufferSize != 1000 {
		t.Errorf("Expected EventBufferSize 1000, got %d", config.EventBufferSize)
	}
	
	if config.ComponentTimeout != 30*time.Second {
		t.Errorf("Expected ComponentTimeout 30s, got %v", config.ComponentTimeout)
	}
	
	if config.ErrorThreshold != 10 {
		t.Errorf("Expected ErrorThreshold 10, got %d", config.ErrorThreshold)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
	
	if config.WorkerPoolSize != 8 {
		t.Errorf("Expected WorkerPoolSize 8, got %d", config.WorkerPoolSize)
	}
	
	if config.RetryTransientErrors != true {
		t.Error("Expected RetryTransientErrors to be true")
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}
	
	// Test component configs are not nil
	if config.ParserConfig == nil {
		t.Error("Expected ParserConfig to be initialized")
	}
	
	if config.PlannerConfig == nil {
		t.Error("Expected PlannerConfig to be initialized")
	}
	
	if config.ExecutorConfig == nil {
		t.Error("Expected ExecutorConfig to be initialized")
	}
	
	if config.EmitterConfig == nil {
		t.Error("Expected EmitterConfig to be initialized")
	}
}

func TestPipelineStageString(t *testing.T) {
	tests := []struct {
		stage    PipelineStage
		expected string
	}{
		{StageInitializing, "initializing"},
		{StageParsing, "parsing"},
		{StagePlanning, "planning"},
		{StageExecuting, "executing"},
		{StageEmitting, "emitting"},
		{StageCompleted, "completed"},
		{StageFailed, "failed"},
	}
	
	for _, test := range tests {
		if test.stage.String() != test.expected {
			t.Errorf("Expected %s.String() = %q, got %q", test.stage, test.expected, test.stage.String())
		}
	}
}

func TestComponentStatusString(t *testing.T) {
	tests := []struct {
		status   ComponentStatus
		expected string
	}{
		{StatusInitializing, "initializing"},
		{StatusReady, "ready"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusShutdown, "shutdown"},
	}
	
	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected %s.String() = %q, got %q", test.status, test.expected, test.status.String())
		}
	}
}

// Test helpers

func createTestConfig() *Config {
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
		ErrorThreshold:     5,
		EnableMetrics:      true,
		LogLevel:          "debug",
		
		WorkerPoolSize:  4,
		BufferPoolSize:  16,
		ChannelPoolSize: 8,
		
		StopOnFirstError:     false,
		RetryTransientErrors: true,
		MaxRetries:           2,
		RetryDelay:           100 * time.Millisecond,
		
		EnableProfiling:    false,
		EnableEventTracing: false,
	}
}

func createTestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
}

// Benchmark tests

func BenchmarkCoordinatorCreation(b *testing.B) {
	logger := zap.NewNop()
	config := DefaultConfig()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		coord := New(logger, config)
		_ = coord.Shutdown(context.Background())
	}
}

func BenchmarkGetMetrics(b *testing.B) {
	logger := zap.NewNop()
	config := DefaultConfig()
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = coord.GetMetrics()
	}
}

func BenchmarkGetStatus(b *testing.B) {
	logger := zap.NewNop()
	config := DefaultConfig()
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = coord.GetStatus()
	}
}

// Test data structures

type testPipelineInput struct {
	sources    []string
	sourceCode string
	config     *Config
}

func (t *testPipelineInput) Sources() []string    { return t.sources }
func (t *testPipelineInput) SourceCode() string  { return t.sourceCode }
func (t *testPipelineInput) Config() *Config     { return t.config }

// Mock implementations for testing

// mockComponent is defined in component_manager_test.go

// Integration test setup

func TestCoordinatorIntegration(t *testing.T) {
	logger := createTestLogger(t)
	config := createTestConfig()
	coord := New(logger, config)
	defer coord.Shutdown(context.Background())
	
	// Test that all subsystems are initialized
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

func TestCoordinatorLifecycle(t *testing.T) {
	logger := createTestLogger(t)
	config := createTestConfig()
	
	// Create coordinator
	coord := New(logger, config)
	
	// Verify initial state
	status := coord.GetStatus()
	if status.Stage != StageInitializing {
		t.Errorf("Expected initial stage %s, got %s", StageInitializing, status.Stage)
	}
	
	// Get metrics
	metrics := coord.GetMetrics()
	if metrics.PipelineExecutions != 0 {
		t.Errorf("Expected 0 executions, got %d", metrics.PipelineExecutions)
	}
	
	// Shutdown
	ctx := context.Background()
	err := coord.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	
	// Verify idempotent shutdown
	err = coord.Shutdown(ctx)
	if err != nil {
		t.Errorf("Second shutdown failed: %v", err)
	}
}