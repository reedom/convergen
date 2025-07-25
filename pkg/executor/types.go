package executor

import (
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// ExecutorState represents the current state of the executor
type ExecutorState string

const (
	ExecutorStateIdle         ExecutorState = "idle"
	ExecutorStateExecuting    ExecutorState = "executing"
	ExecutorStateShuttingDown ExecutorState = "shutting_down"
	ExecutorStateError        ExecutorState = "error"
)

// ExecutorStatus provides comprehensive status information about the executor
type ExecutorStatus struct {
	State                ExecutorState                `json:"state"`
	StartTime            time.Time                   `json:"start_time"`
	CurrentPlan          *domain.ExecutionPlan       `json:"current_plan,omitempty"`
	PlanStartTime        *time.Time                  `json:"plan_start_time,omitempty"`
	LastCompletedPlan    *domain.ExecutionPlan       `json:"last_completed_plan,omitempty"`
	LastCompletionTime   *time.Time                  `json:"last_completion_time,omitempty"`
	ActiveBatches        map[string]*BatchExecution  `json:"active_batches"`
	CompletedBatches     map[string]*BatchResult     `json:"completed_batches"`
	QueuedBatches        []*BatchExecution           `json:"queued_batches"`
	TotalPlansExecuted   int                         `json:"total_plans_executed"`
	TotalBatchesExecuted int                         `json:"total_batches_executed"`
	TotalFieldsProcessed int64                       `json:"total_fields_processed"`
	TotalErrors          int                         `json:"total_errors"`
}

// BatchExecution represents a batch being executed
type BatchExecution struct {
	ID            string                    `json:"id"`
	Mappings      []*domain.FieldMapping    `json:"mappings"`
	MethodName    string                    `json:"method_name"`
	BatchIndex    int                       `json:"batch_index"`
	DependsOn     []string                  `json:"depends_on"`
	Configuration *ExecutorConfig           `json:"configuration"`
	StartTime     time.Time                 `json:"start_time"`
	Context       map[string]interface{}    `json:"context,omitempty"`
}

// BatchResult represents the result of executing a batch
type BatchResult struct {
	BatchID       string                    `json:"batch_id"`
	Success       bool                      `json:"success"`
	StartTime     time.Time                 `json:"start_time"`
	EndTime       time.Time                 `json:"end_time"`
	Duration      time.Duration             `json:"duration"`
	FieldResults  map[string]interface{}    `json:"field_results"`
	Errors        []ExecutionError          `json:"errors,omitempty"`
	Metrics       *BatchMetrics             `json:"metrics"`
	WorkersUsed   int                       `json:"workers_used"`
	MemoryUsedMB  int                       `json:"memory_used_mb"`
}

// FieldExecution represents a field mapping being executed
type FieldExecution struct {
	ID            string                    `json:"id"`
	Mapping       *domain.FieldMapping      `json:"mapping"`
	BatchID       string                    `json:"batch_id"`
	MethodName    string                    `json:"method_name"`
	Configuration *ExecutorConfig           `json:"configuration"`
	Context       map[string]interface{}    `json:"context,omitempty"`
	StartTime     time.Time                 `json:"start_time"`
	Timeout       time.Duration             `json:"timeout"`
}

// FieldResult represents the result of executing a field mapping
type FieldResult struct {
	FieldID       string                    `json:"field_id"`
	Success       bool                      `json:"success"`
	StartTime     time.Time                 `json:"start_time"`
	EndTime       time.Time                 `json:"end_time"`
	Duration      time.Duration             `json:"duration"`
	Result        interface{}               `json:"result"`
	Error         *ExecutionError           `json:"error,omitempty"`
	Metrics       *FieldMetrics             `json:"metrics"`
	RetryCount    int                       `json:"retry_count"`
	StrategyUsed  string                    `json:"strategy_used"`
}

// MethodResult represents the result of executing all batches for a method
type MethodResult struct {
	MethodName string                    `json:"method_name"`
	Success    bool                      `json:"success"`
	StartTime  time.Time                 `json:"start_time"`
	EndTime    time.Time                 `json:"end_time"`
	Duration   time.Duration             `json:"duration"`
	Data       map[string]interface{}    `json:"data"`
	Errors     []ExecutionError          `json:"errors,omitempty"`
}

// ResourceMetrics tracks resource usage during execution
type ResourceMetrics struct {
	WorkersActive      int       `json:"workers_active"`
	WorkersIdle        int       `json:"workers_idle"`
	WorkersTotal       int       `json:"workers_total"`
	MemoryUsedMB       int       `json:"memory_used_mb"`
	MemoryAvailableMB  int       `json:"memory_available_mb"`
	MemoryPressure     float64   `json:"memory_pressure"`
	CPUUsagePercent    float64   `json:"cpu_usage_percent"`
	QueuedJobs         int       `json:"queued_jobs"`
	CompletedJobs      int       `json:"completed_jobs"`
	LastUpdated        time.Time `json:"last_updated"`
}

// BatchMetrics contains detailed metrics for batch execution
type BatchMetrics struct {
	FieldsProcessed      int           `json:"fields_processed"`
	FieldsSuccessful     int           `json:"fields_successful"`
	FieldsFailed         int           `json:"fields_failed"`
	ThroughputPerSecond  float64       `json:"throughput_per_second"`
	AverageFieldDuration time.Duration `json:"average_field_duration"`
	MaxFieldDuration     time.Duration `json:"max_field_duration"`
	MinFieldDuration     time.Duration `json:"min_field_duration"`
	ConcurrencyAchieved  int           `json:"concurrency_achieved"`
	ResourceEfficiency   float64       `json:"resource_efficiency"`
	RetryCount           int           `json:"retry_count"`
}

// FieldMetrics contains detailed metrics for individual field execution
type FieldMetrics struct {
	ExecutionTime       time.Duration `json:"execution_time"`
	MemoryAllocated     int           `json:"memory_allocated"`
	CPUTime            time.Duration `json:"cpu_time"`
	IOOperations        int           `json:"io_operations"`
	CacheHits           int           `json:"cache_hits"`
	CacheMisses         int           `json:"cache_misses"`
	StrategyTime        time.Duration `json:"strategy_time"`
	ValidationTime      time.Duration `json:"validation_time"`
	TransformationTime  time.Duration `json:"transformation_time"`
}

// WorkerMetrics tracks individual worker performance
type WorkerMetrics struct {
	WorkerID         string        `json:"worker_id"`
	TasksCompleted   int           `json:"tasks_completed"`
	TasksFailed      int           `json:"tasks_failed"`
	TotalActiveTime  time.Duration `json:"total_active_time"`
	TotalIdleTime    time.Duration `json:"total_idle_time"`
	AverageTaskTime  time.Duration `json:"average_task_time"`
	LastTaskTime     time.Time     `json:"last_task_time"`
	CurrentMemoryMB  int           `json:"current_memory_mb"`
	PeakMemoryMB     int           `json:"peak_memory_mb"`
}

// QueueMetrics tracks job queue performance
type QueueMetrics struct {
	QueueSize          int           `json:"queue_size"`
	EnqueueRate        float64       `json:"enqueue_rate"`        // Jobs per second
	DequeueRate        float64       `json:"dequeue_rate"`        // Jobs per second
	AverageWaitTime    time.Duration `json:"average_wait_time"`
	MaxWaitTime        time.Duration `json:"max_wait_time"`
	QueueFullEvents    int           `json:"queue_full_events"`
	QueueEmptyTime     time.Duration `json:"queue_empty_time"`
	ThroughputTarget   float64       `json:"throughput_target"`
	ThroughputActual   float64       `json:"throughput_actual"`
}

// ExecutionEvent represents events emitted during execution
type ExecutionEvent struct {
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	ExecutorID  string                 `json:"executor_id"`
	PlanID      string                 `json:"plan_id,omitempty"`
	BatchID     string                 `json:"batch_id,omitempty"`
	FieldID     string                 `json:"field_id,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Severity    string                 `json:"severity"`
}

// Event type constants
const (
	EventExecutionStarted    = "execution.started"
	EventExecutionCompleted  = "execution.completed"
	EventExecutionFailed     = "execution.failed"
	EventBatchStarted        = "execution.batch.started"
	EventBatchCompleted      = "execution.batch.completed"
	EventBatchFailed         = "execution.batch.failed"
	EventFieldStarted        = "execution.field.started"
	EventFieldCompleted      = "execution.field.completed"
	EventFieldFailed         = "execution.field.failed"
	EventResourcePressure    = "execution.resource.pressure"
	EventResourceThreshold   = "execution.resource.threshold"
	EventWorkerStarted       = "execution.worker.started"
	EventWorkerStopped       = "execution.worker.stopped"
	EventWorkerIdle          = "execution.worker.idle"
	EventQueueFull           = "execution.queue.full"
	EventQueueEmpty          = "execution.queue.empty"
	EventRetryAttempt        = "execution.retry.attempt"
	EventRetryExhausted      = "execution.retry.exhausted"
)

// Severity levels for events
const (
	SeverityDebug   = "debug"
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
	SeverityCritical = "critical"
)

// WorkerState represents the state of a worker
type WorkerState string

const (
	WorkerStateIdle       WorkerState = "idle"
	WorkerStateActive     WorkerState = "active"
	WorkerStateShutdown   WorkerState = "shutdown"
	WorkerStateError      WorkerState = "error"
)

// Worker represents a worker in the resource pool
type Worker struct {
	ID           string        `json:"id"`
	State        WorkerState   `json:"state"`
	StartTime    time.Time     `json:"start_time"`
	LastActivity time.Time     `json:"last_activity"`
	TasksHandled int           `json:"tasks_handled"`
	CurrentTask  *FieldExecution `json:"current_task,omitempty"`
	Metrics      *WorkerMetrics  `json:"metrics"`
	Channel      chan *FieldExecution `json:"-"`
	Done         chan struct{}   `json:"-"`
}

// JobQueue represents a queue for managing execution jobs
type JobQueue struct {
	Capacity    int                    `json:"capacity"`
	Size        int                    `json:"size"`
	Jobs        chan *FieldExecution   `json:"-"`
	Metrics     *QueueMetrics          `json:"metrics"`
	Closed      bool                   `json:"closed"`
}

// ResourceLimits defines resource constraints for execution
type ResourceLimits struct {
	MaxWorkers      int     `json:"max_workers"`
	MaxMemoryMB     int     `json:"max_memory_mb"`
	MaxConcurrentJobs int   `json:"max_concurrent_jobs"`
	MemoryThreshold float64 `json:"memory_threshold"`
	CPUThreshold    float64 `json:"cpu_threshold"`
}

// PerformanceProfile contains performance tuning parameters
type PerformanceProfile struct {
	Name                string        `json:"name"`
	WorkerPoolSize      int           `json:"worker_pool_size"`
	BatchSize           int           `json:"batch_size"`
	ConcurrencyLevel    int           `json:"concurrency_level"`
	BufferSize          int           `json:"buffer_size"`
	TimeoutMultiplier   float64       `json:"timeout_multiplier"`
	RetryStrategy       string        `json:"retry_strategy"`
	MemoryOptimization  bool          `json:"memory_optimization"`
	Description         string        `json:"description"`
}

// Predefined performance profiles
var (
	HighThroughputProfile = &PerformanceProfile{
		Name:               "high_throughput",
		WorkerPoolSize:     16,
		BatchSize:          100,
		ConcurrencyLevel:   32,
		BufferSize:         500,
		TimeoutMultiplier:  2.0,
		RetryStrategy:      "exponential",
		MemoryOptimization: false,
		Description:        "Optimized for maximum throughput",
	}

	LowLatencyProfile = &PerformanceProfile{
		Name:               "low_latency",
		WorkerPoolSize:     8,
		BatchSize:          10,
		ConcurrencyLevel:   4,
		BufferSize:         50,
		TimeoutMultiplier:  0.5,
		RetryStrategy:      "immediate",
		MemoryOptimization: true,
		Description:        "Optimized for minimum latency",
	}

	BalancedProfile = &PerformanceProfile{
		Name:               "balanced",
		WorkerPoolSize:     8,
		BatchSize:          50,
		ConcurrencyLevel:   8,
		BufferSize:         200,
		TimeoutMultiplier:  1.0,
		RetryStrategy:      "linear",
		MemoryOptimization: true,
		Description:        "Balanced performance and resource usage",
	}

	MemoryOptimizedProfile = &PerformanceProfile{
		Name:               "memory_optimized",
		WorkerPoolSize:     4,
		BatchSize:          20,
		ConcurrencyLevel:   2,
		BufferSize:         100,
		TimeoutMultiplier:  1.5,
		RetryStrategy:      "exponential",
		MemoryOptimization: true,
		Description:        "Minimizes memory usage",
	}
)