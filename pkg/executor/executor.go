package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrExecutionPlanNil = errors.New("execution plan cannot be nil")
)

// Config defines configuration parameters for the execution engine.
type Config struct {
	// Worker pool settings
	MaxWorkers        int           `json:"max_workers"`
	MinWorkers        int           `json:"min_workers"`
	WorkerIdleTimeout time.Duration `json:"worker_idle_timeout"`

	// Resource limits
	MaxMemoryMB       int     `json:"max_memory_mb"`
	MemoryThreshold   float64 `json:"memory_threshold"` // Percentage for memory pressure detection
	MaxConcurrentJobs int     `json:"max_concurrent_jobs"`

	// Execution settings
	ExecutionTimeout time.Duration `json:"execution_timeout"`
	BatchTimeout     time.Duration `json:"batch_timeout"`
	FieldTimeout     time.Duration `json:"field_timeout"`
	RetryAttempts    int           `json:"retry_attempts"`
	RetryBackoffBase time.Duration `json:"retry_backoff_base"`
	RetryBackoffMax  time.Duration `json:"retry_backoff_max"`

	// Performance tuning
	EnablePipelining    bool    `json:"enable_pipelining"`
	PipelineBufferSize  int     `json:"pipeline_buffer_size"`
	EnableResourceReuse bool    `json:"enable_resource_reuse"`
	AdaptiveConcurrency bool    `json:"adaptive_concurrency"`
	ThroughputTarget    float64 `json:"throughput_target"` // Fields per second

	// Monitoring and debugging
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsInterval time.Duration `json:"metrics_interval"`
	EnableProfiling bool          `json:"enable_profiling"`
	EnableTracing   bool          `json:"enable_tracing"`
	DebugMode       bool          `json:"debug_mode"`
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxWorkers:          8,
		MinWorkers:          2,
		WorkerIdleTimeout:   30 * time.Second,
		MaxMemoryMB:         512,
		MemoryThreshold:     0.8,
		MaxConcurrentJobs:   16,
		ExecutionTimeout:    5 * time.Minute,
		BatchTimeout:        30 * time.Second,
		FieldTimeout:        10 * time.Second,
		RetryAttempts:       3,
		RetryBackoffBase:    100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
		EnablePipelining:    true,
		PipelineBufferSize:  100,
		EnableResourceReuse: true,
		AdaptiveConcurrency: true,
		ThroughputTarget:    1000.0,
		EnableMetrics:       true,
		MetricsInterval:     5 * time.Second,
		EnableProfiling:     false,
		EnableTracing:       false,
		DebugMode:           false,
	}
}

// ExecutionResult represents the result of executing an execution plan.
type ExecutionResult struct {
	PlanID         string                 `json:"plan_id"`
	Success        bool                   `json:"success"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       time.Duration          `json:"duration"`
	Metrics        *ExecutionMetrics      `json:"metrics"`
	Results        map[string]interface{} `json:"results"`
	Errors         []ExecutionError       `json:"errors,omitempty"`
	PartialResults bool                   `json:"partial_results"`
}

// ExecutionError represents an error that occurred during execution.
type ExecutionError struct {
	FieldID   string                 `json:"field_id"`
	BatchID   string                 `json:"batch_id"`
	Error     string                 `json:"error"`
	ErrorType string                 `json:"error_type"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Executor defines the interface for the execution engine, which is responsible for
// running conversion tasks based on an execution plan, managing resources, and
// reporting progress through events.
type Executor interface {
	// ExecutePlan executes a complete execution plan
	ExecutePlan(ctx context.Context, plan *domain.ExecutionPlan) (*ExecutionResult, error)

	// ExecuteBatch executes a single batch of field mappings
	ExecuteBatch(ctx context.Context, batch *BatchExecution) (*BatchResult, error)

	// ExecuteField executes a single field mapping
	ExecuteField(ctx context.Context, field *FieldExecution) (*FieldResult, error)

	// GetMetrics returns current execution metrics
	GetMetrics() *ExecutionMetrics

	// GetStatus returns current executor status
	GetStatus() *Status

	// Shutdown gracefully shuts down the executor
	Shutdown(ctx context.Context) error
}

// ConcreteExecutor implements the Executor interface.
type ConcreteExecutor struct {
	config   *Config
	logger   *zap.Logger
	eventBus events.EventBus

	// Execution components
	batchExecutor BatchExecutor
	fieldExecutor FieldExecutor
	resourcePool  *ResourcePool
	metrics       *ExecutionMetrics

	// State management
	status   *Status
	shutdown chan struct{}
	wg       sync.WaitGroup
	mutex    sync.RWMutex
}

// NewExecutor creates a new execution engine.
func NewExecutor(logger *zap.Logger, eventBus events.EventBus, config *Config) Executor {
	if config == nil {
		config = DefaultConfig()
	}

	metrics := NewExecutionMetrics(config.EnableMetrics)
	resourcePool := NewResourcePool(config, logger, metrics)

	status := &Status{
		State:            StateIdle,
		StartTime:        time.Now(),
		ActiveBatches:    make(map[string]*BatchExecution),
		CompletedBatches: make(map[string]*BatchResult),
		QueuedBatches:    make([]*BatchExecution, 0),
	}

	executor := &ConcreteExecutor{
		config:       config,
		logger:       logger,
		eventBus:     eventBus,
		resourcePool: resourcePool,
		metrics:      metrics,
		status:       status,
		shutdown:     make(chan struct{}),
	}

	// Initialize sub-executors
	executor.batchExecutor = NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	executor.fieldExecutor = NewFieldExecutor(config, logger, eventBus, metrics)

	// Start background monitoring if enabled
	if config.EnableMetrics {
		executor.startMetricsCollection()
	}

	logger.Info("executor initialized",
		zap.Int("max_workers", config.MaxWorkers),
		zap.Int("max_memory_mb", config.MaxMemoryMB),
		zap.Duration("execution_timeout", config.ExecutionTimeout))

	return executor
}

// ExecutePlan executes a complete execution plan with comprehensive coordination.
func (e *ConcreteExecutor) ExecutePlan(ctx context.Context, plan *domain.ExecutionPlan) (*ExecutionResult, error) {
	if plan == nil {
		return nil, ErrExecutionPlanNil
	}

	startTime := time.Now()
	e.logger.Info("starting plan execution",
		zap.String("plan_id", plan.ID),
		zap.Int("methods", len(plan.Methods)),
		zap.Int("max_workers", plan.GlobalLimits.MaxWorkers))

	// Initialize execution result
	result := e.initializeExecutionResult(plan, startTime)

	// Setup execution environment
	e.setupExecutionEnvironment(ctx, plan, startTime)

	// Execute all methods
	methodResults, errors := e.executeAllMethods(ctx, plan.Methods)
	result.Results = methodResults
	result.Errors = errors

	// Finalize execution
	e.finalizeExecution(ctx, plan, result)

	return result, nil
}

// ExecuteBatch executes a single batch through the batch executor.
func (e *ConcreteExecutor) ExecuteBatch(ctx context.Context, batch *BatchExecution) (*BatchResult, error) {
	result, err := e.batchExecutor.ExecuteBatch(ctx, batch)
	if err != nil {
		return result, fmt.Errorf("batch execution failed: %w", err)
	}

	return result, nil
}

// ExecuteField executes a single field through the field executor.
func (e *ConcreteExecutor) ExecuteField(ctx context.Context, field *FieldExecution) (*FieldResult, error) {
	result, err := e.fieldExecutor.ExecuteField(ctx, field)
	if err != nil {
		return result, fmt.Errorf("field execution failed: %w", err)
	}

	return result, nil
}

// GetMetrics returns current execution metrics.
func (e *ConcreteExecutor) GetMetrics() *ExecutionMetrics {
	return e.metrics.GetSnapshot()
}

// GetStatus returns current executor status.
func (e *ConcreteExecutor) GetStatus() *Status {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Create a copy to avoid race conditions
	status := *e.status
	status.ActiveBatches = make(map[string]*BatchExecution)
	status.CompletedBatches = make(map[string]*BatchResult)
	status.QueuedBatches = make([]*BatchExecution, len(e.status.QueuedBatches))

	for k, v := range e.status.ActiveBatches {
		status.ActiveBatches[k] = v
	}

	for k, v := range e.status.CompletedBatches {
		status.CompletedBatches[k] = v
	}

	copy(status.QueuedBatches, e.status.QueuedBatches)

	return &status
}

// Shutdown gracefully shuts down the executor.
func (e *ConcreteExecutor) Shutdown(ctx context.Context) error {
	e.logger.Info("shutting down executor")

	// Signal shutdown
	close(e.shutdown)

	// Update status
	e.updateStatus(func(status *Status) {
		status.State = StateShuttingDown
	})

	// Shutdown components
	if err := e.batchExecutor.Shutdown(ctx); err != nil {
		e.logger.Warn("batch executor shutdown error", zap.Error(err))
	}

	if err := e.fieldExecutor.Shutdown(ctx); err != nil {
		e.logger.Warn("field executor shutdown error", zap.Error(err))
	}

	if err := e.resourcePool.Shutdown(ctx); err != nil {
		e.logger.Warn("resource pool shutdown error", zap.Error(err))
	}

	// Wait for background tasks
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info("executor shutdown completed")
		return nil
	case <-ctx.Done():
		e.logger.Warn("executor shutdown timed out")
		return fmt.Errorf("executor shutdown context cancelled: %w", ctx.Err())
	}
}

// Helper methods

func (e *ConcreteExecutor) executeMethod(ctx context.Context, methodName string, methodPlan *domain.MethodPlan) *MethodResult {
	e.logger.Debug("executing method",
		zap.String("method", methodName),
		zap.Int("batches", len(methodPlan.Batches)))

	startTime := time.Now()
	methodResult := &MethodResult{
		MethodName: methodName,
		StartTime:  startTime,
		Data:       make(map[string]interface{}),
		Errors:     make([]ExecutionError, 0),
	}

	// Execute batches in dependency order
	for i, batch := range methodPlan.Batches {
		batchExecution := &BatchExecution{
			ID:            batch.ID,
			Mappings:      batch.Fields,
			MethodName:    methodName,
			BatchIndex:    i,
			DependsOn:     batch.DependsOn,
			Configuration: e.config,
			StartTime:     time.Now(),
		}

		batchResult, err := e.batchExecutor.ExecuteBatch(ctx, batchExecution)
		if err != nil {
			methodResult.Errors = append(methodResult.Errors, ExecutionError{
				BatchID:   batch.ID,
				Error:     err.Error(),
				ErrorType: "batch_execution",
				Timestamp: time.Now(),
				Retryable: true,
			})

			continue
		}

		// Merge batch results
		for k, v := range batchResult.FieldResults {
			methodResult.Data[k] = v
		}
	}

	methodResult.EndTime = time.Now()
	methodResult.Duration = methodResult.EndTime.Sub(methodResult.StartTime)
	methodResult.Success = len(methodResult.Errors) == 0

	return methodResult
}

func (e *ConcreteExecutor) updateStatus(updateFn func(*Status)) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	updateFn(e.status)
}

func (e *ConcreteExecutor) emitEvent(ctx context.Context, eventType string, data map[string]interface{}) error {
	event := events.NewEvent(eventType, data)
	if err := e.eventBus.Emit(ctx, event); err != nil {
		return fmt.Errorf("failed to emit executor event: %w", err)
	}

	return nil
}

func (e *ConcreteExecutor) startMetricsCollection() {
	if !e.config.EnableMetrics {
		return
	}

	e.wg.Add(1)

	go func() {
		defer e.wg.Done()

		ticker := time.NewTicker(e.config.MetricsInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.collectMetrics()
			case <-e.shutdown:
				return
			}
		}
	}()
}

func (e *ConcreteExecutor) collectMetrics() {
	// Update resource pool metrics
	e.metrics.UpdateResourceMetrics(e.resourcePool.GetMetrics())

	// Update executor status metrics
	status := e.GetStatus()
	e.metrics.RecordExecutorStatus(status)

	e.logger.Debug("metrics collected",
		zap.Int("active_batches", len(status.ActiveBatches)),
		zap.Int("completed_batches", len(status.CompletedBatches)),
		zap.String("state", string(status.State)))
}

// Helper methods for ExecutePlan refactoring

// initializeExecutionResult creates the initial execution result structure
func (e *ConcreteExecutor) initializeExecutionResult(plan *domain.ExecutionPlan, startTime time.Time) *ExecutionResult {
	return &ExecutionResult{
		PlanID:    plan.ID,
		StartTime: startTime,
		Results:   make(map[string]interface{}),
		Errors:    make([]ExecutionError, 0),
		Metrics:   e.metrics.Clone(),
	}
}

// setupExecutionEnvironment configures the execution environment for the plan
func (e *ConcreteExecutor) setupExecutionEnvironment(ctx context.Context, plan *domain.ExecutionPlan, startTime time.Time) {
	// Emit plan started event
	if err := e.emitEvent(ctx, "execution.plan.started", map[string]interface{}{
		"plan_id":    plan.ID,
		"methods":    len(plan.Methods),
		"start_time": startTime,
	}); err != nil {
		e.logger.Warn("failed to emit plan started event", zap.Error(err))
	}

	// Apply global resource limits
	e.resourcePool.SetLimits(plan.GlobalLimits.MaxWorkers, plan.GlobalLimits.MaxMemoryMB)

	// Update executor status
	e.updateStatus(func(status *Status) {
		status.State = StateExecuting
		status.CurrentPlan = plan
		status.PlanStartTime = &startTime
	})
}

// executeAllMethods executes all methods concurrently and returns results
func (e *ConcreteExecutor) executeAllMethods(ctx context.Context, methods map[string]*domain.MethodPlan) (map[string]interface{}, []ExecutionError) {
	var methodWg sync.WaitGroup
	methodResults := make(map[string]*MethodResult)
	var resultMutex sync.Mutex
	errorChannel := make(chan ExecutionError, len(methods)*10)

	for methodName, methodPlan := range methods {
		methodWg.Add(1)

		go func(name string, mPlan *domain.MethodPlan) {
			defer methodWg.Done()

			methodResult := e.executeMethod(ctx, name, mPlan)

			resultMutex.Lock()
			if methodResult != nil {
				methodResults[name] = methodResult
			}
			resultMutex.Unlock()
		}(methodName, methodPlan)
	}

	// Wait for all methods to complete
	methodWg.Wait()
	close(errorChannel)

	// Collect errors and convert results
	var errors []ExecutionError
	for execError := range errorChannel {
		errors = append(errors, execError)
	}

	results := make(map[string]interface{})
	for name, methodResult := range methodResults {
		if methodResult != nil {
			results[name] = methodResult.Data
		}
	}

	return results, errors
}

// finalizeExecution finalizes the execution result with metrics and events
func (e *ConcreteExecutor) finalizeExecution(ctx context.Context, plan *domain.ExecutionPlan, result *ExecutionResult) {
	// Finalize result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0
	result.PartialResults = len(result.Errors) > 0 && len(result.Results) > 0
	result.Metrics = e.metrics.GetSnapshot()

	// Update executor status
	e.updateStatus(func(status *Status) {
		status.State = StateIdle
		status.CurrentPlan = nil
		status.PlanStartTime = nil
		status.LastCompletedPlan = plan
		status.LastCompletionTime = &result.EndTime
	})

	// Emit plan completed event
	if err := e.emitEvent(ctx, "execution.plan.completed", map[string]interface{}{
		"plan_id":          plan.ID,
		"success":          result.Success,
		"duration_ms":      result.Duration.Milliseconds(),
		"fields_processed": result.Metrics.FieldsProcessed,
		"errors":           len(result.Errors),
		"end_time":         result.EndTime,
	}); err != nil {
		e.logger.Warn("failed to emit plan completed event", zap.Error(err))
	}

	e.logger.Info("plan execution completed",
		zap.String("plan_id", plan.ID),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
		zap.Int("errors", len(result.Errors)),
		zap.Int("methods_completed", len(result.Results)))

	// Record plan execution metrics
	e.metrics.RecordPlanExecution(result)
}
