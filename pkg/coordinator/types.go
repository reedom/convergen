package coordinator

import (
	"context"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/emitter"
	"github.com/reedom/convergen/v8/pkg/executor"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"github.com/reedom/convergen/v8/pkg/parser"
	"github.com/reedom/convergen/v8/pkg/planner"
)

// PipelineStage represents the current stage of pipeline execution
type PipelineStage string

const (
	StageInitializing PipelineStage = "initializing"
	StageParsing      PipelineStage = "parsing"
	StagePlanning     PipelineStage = "planning"
	StageExecuting    PipelineStage = "executing"
	StageEmitting     PipelineStage = "emitting"
	StageCompleted    PipelineStage = "completed"
	StageFailed       PipelineStage = "failed"
)

// String returns the string representation of the pipeline stage
func (s PipelineStage) String() string {
	return string(s)
}

// ComponentStatus represents the status of a pipeline component
type ComponentStatus string

const (
	StatusInitializing ComponentStatus = "initializing"
	StatusReady        ComponentStatus = "ready"
	StatusRunning      ComponentStatus = "running"
	StatusCompleted    ComponentStatus = "completed"
	StatusFailed       ComponentStatus = "failed"
	StatusShutdown     ComponentStatus = "shutdown"
)

// String returns the string representation of the component status
func (s ComponentStatus) String() string {
	return string(s)
}

// Config defines configuration for the coordinator
type Config struct {
	// Component configurations
	ParserConfig   *parser.Config   `json:"parser_config,omitempty"`
	PlannerConfig  *planner.Config  `json:"planner_config,omitempty"`
	ExecutorConfig *executor.Config `json:"executor_config,omitempty"`
	EmitterConfig  *emitter.EmitterConfig `json:"emitter_config,omitempty"`

	// Coordinator-specific settings
	MaxConcurrency     int           `json:"max_concurrency"`
	EventBufferSize    int           `json:"event_buffer_size"`
	ComponentTimeout   time.Duration `json:"component_timeout"`
	ErrorThreshold     int           `json:"error_threshold"`
	EnableMetrics      bool          `json:"enable_metrics"`
	LogLevel          string        `json:"log_level"`

	// Resource management
	WorkerPoolSize  int `json:"worker_pool_size"`
	BufferPoolSize  int `json:"buffer_pool_size"`
	ChannelPoolSize int `json:"channel_pool_size"`

	// Pipeline behavior
	StopOnFirstError     bool          `json:"stop_on_first_error"`
	RetryTransientErrors bool          `json:"retry_transient_errors"`
	MaxRetries           int           `json:"max_retries"`
	RetryDelay           time.Duration `json:"retry_delay"`

	// Performance tuning
	EnableProfiling      bool `json:"enable_profiling"`
	ProfileOutputDir     string `json:"profile_output_dir"`
	EnableEventTracing   bool `json:"enable_event_tracing"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		ParserConfig:   parser.DefaultConfig(),
		PlannerConfig:  planner.DefaultConfig(),
		ExecutorConfig: executor.DefaultConfig(),
		EmitterConfig:  emitter.DefaultEmitterConfig(),

		MaxConcurrency:   4,
		EventBufferSize:  1000,
		ComponentTimeout: 30 * time.Second,
		ErrorThreshold:   10,
		EnableMetrics:    true,
		LogLevel:        "info",

		WorkerPoolSize:  8,
		BufferPoolSize:  32,
		ChannelPoolSize: 16,

		StopOnFirstError:     false,
		RetryTransientErrors: true,
		MaxRetries:           3,
		RetryDelay:           time.Second,

		EnableProfiling:    false,
		EnableEventTracing: false,
	}
}

// PipelineInput represents input to the pipeline
type PipelineInput struct {
	Sources    []string          `json:"sources"`
	SourceCode string           `json:"source_code,omitempty"`
	Config     *Config          `json:"config"`
	Context    context.Context  `json:"-"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// GenerationResult represents the output of the pipeline
type GenerationResult struct {
	// Generated code and metadata
	Code        string                    `json:"code"`
	Imports     []string                  `json:"imports"`
	Methods     []*domain.MethodResult    `json:"methods"`
	Metadata    *GenerationMetadata       `json:"metadata"`

	// Execution information
	Duration    time.Duration             `json:"duration"`
	Metrics     *CoordinatorMetrics       `json:"metrics"`
	Errors      *ErrorReport              `json:"errors,omitempty"`
	Warnings    []string                  `json:"warnings,omitempty"`

	// Pipeline status
	Status      PipelineStage             `json:"status"`
	Components  map[string]ComponentStatus `json:"components"`
}

// GenerationMetadata contains metadata about the generation process
type GenerationMetadata struct {
	Timestamp        time.Time              `json:"timestamp"`
	CoordinatorVersion string               `json:"coordinator_version"`
	PipelineID       string                 `json:"pipeline_id"`
	InputSources     []string               `json:"input_sources"`
	ComponentVersions map[string]string     `json:"component_versions"`
	ProcessingStages  []StageMetadata       `json:"processing_stages"`
}

// StageMetadata contains metadata about a specific pipeline stage
type StageMetadata struct {
	Stage     PipelineStage `json:"stage"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	ErrorCount int          `json:"error_count"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PipelineStatus represents the current status of pipeline execution
type PipelineStatus struct {
	Stage           PipelineStage                    `json:"stage"`
	Progress        float64                          `json:"progress"`
	ComponentStatus map[string]ComponentStatus       `json:"component_status"`
	StartTime       time.Time                        `json:"start_time"`
	ElapsedTime     time.Duration                    `json:"elapsed_time"`
	Errors          []error                          `json:"errors,omitempty"`
	CurrentInput    *PipelineInput                   `json:"current_input,omitempty"`
	PipelineID      string                           `json:"pipeline_id"`
}

// ComponentError represents an error from a specific component
type ComponentError struct {
	Component string                 `json:"component"`
	Stage     PipelineStage          `json:"stage"`
	Error     error                  `json:"error"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Retryable bool                   `json:"retryable"`
	Attempt   int                    `json:"attempt"`
}

// ErrorReport aggregates errors from the pipeline
type ErrorReport struct {
	Errors        []ComponentError `json:"errors"`
	Critical      []error          `json:"critical"`
	Warnings      []error          `json:"warnings"`
	TotalCount    int              `json:"total_count"`
	CriticalCount int              `json:"critical_count"`
	WarningCount  int              `json:"warning_count"`
	FirstError    *ComponentError  `json:"first_error,omitempty"`
	LastError     *ComponentError  `json:"last_error,omitempty"`
}

// CoordinatorMetrics tracks coordinator performance and statistics
type CoordinatorMetrics struct {
	// Execution statistics
	PipelineExecutions int64         `json:"pipeline_executions"`
	TotalDuration      time.Duration `json:"total_duration"`
	AverageDuration    time.Duration `json:"average_duration"`
	SuccessRate        float64       `json:"success_rate"`

	// Component metrics
	ComponentMetrics map[string]interface{} `json:"component_metrics"`
	
	// Resource usage
	ResourceUsage    *ResourceUsage         `json:"resource_usage"`
	
	// Event statistics
	EventCounts      map[string]int64       `json:"event_counts"`
	EventProcessingTime map[string]time.Duration `json:"event_processing_time"`

	// Error statistics
	ErrorCounts      map[string]int64       `json:"error_counts"`
	RetryStats       *RetryStatistics       `json:"retry_stats"`

	// Performance metrics
	Throughput       float64                `json:"throughput"` // pipelines/second
	Latency          *LatencyMetrics        `json:"latency"`
	ConcurrencyLevel float64                `json:"concurrency_level"`
}

// ResourceUsage tracks resource consumption
type ResourceUsage struct {
	PeakMemoryUsage    int64         `json:"peak_memory_usage"`
	CurrentMemoryUsage int64         `json:"current_memory_usage"`
	GoroutineCount     int           `json:"goroutine_count"`
	CPUUsage           float64       `json:"cpu_usage"`
	GCStats            *GCStatistics `json:"gc_stats"`
}

// GCStatistics tracks garbage collection statistics
type GCStatistics struct {
	NumGC        uint32        `json:"num_gc"`
	PauseTotalNs uint64        `json:"pause_total_ns"`
	PauseNs      []uint64      `json:"pause_ns"`
	LastGC       time.Time     `json:"last_gc"`
}

// RetryStatistics tracks retry behavior
type RetryStatistics struct {
	TotalRetries    int64              `json:"total_retries"`
	SuccessfulRetries int64            `json:"successful_retries"`
	FailedRetries   int64              `json:"failed_retries"`
	RetrysByComponent map[string]int64 `json:"retries_by_component"`
	AverageRetryDelay time.Duration    `json:"average_retry_delay"`
}

// LatencyMetrics tracks latency distribution
type LatencyMetrics struct {
	P50  time.Duration `json:"p50"`
	P90  time.Duration `json:"p90"`
	P95  time.Duration `json:"p95"`
	P99  time.Duration `json:"p99"`
	Min  time.Duration `json:"min"`
	Max  time.Duration `json:"max"`
	Mean time.Duration `json:"mean"`
}

// PipelineComponent interface for components that can be integrated into the pipeline
type PipelineComponent interface {
	// Name returns the component name
	Name() string
	
	// Initialize the component with event bus
	Initialize(ctx context.Context, eventBus events.EventBus) error
	
	// Shutdown the component gracefully
	Shutdown(ctx context.Context) error
	
	// GetMetrics returns component-specific metrics
	GetMetrics() interface{}
	
	// GetStatus returns current component status
	GetStatus() ComponentStatus
}

// ComponentFactory creates pipeline components
type ComponentFactory func(config interface{}) (PipelineComponent, error)

// PipelineMiddleware processes requests between pipeline stages
type PipelineMiddleware interface {
	Process(ctx context.Context, input interface{}, next func(interface{}) interface{}) interface{}
}

// EventInterceptor can intercept and modify events in the pipeline
type EventInterceptor interface {
	Intercept(ctx context.Context, event events.Event) (events.Event, error)
}

// RecoveryStrategy determines how to handle component failures
type RecoveryStrategy interface {
	// IsRecoverable determines if error can be recovered from
	IsRecoverable(err error) bool
	
	// Recover attempts to recover from the error
	Recover(ctx context.Context, err error) error
	
	// GetRetryDelay returns delay before retry attempt
	GetRetryDelay(attempt int) time.Duration
}

// WorkerPool manages a pool of worker goroutines
type WorkerPool struct {
	Size     int            `json:"size"`
	Workers  chan struct{}  `json:"-"`
	Tasks    chan func()    `json:"-"`
	Done     chan struct{}  `json:"-"`
	Error    chan error     `json:"-"`
	Active   int            `json:"active"`
	Processed int64         `json:"processed"`
}

// BufferPool manages reusable byte buffers
type BufferPool struct {
	pool    chan []byte
	size    int
	bufSize int
	created int64
	reused  int64
}

// ChannelPool manages reusable channels
type ChannelPool struct {
	eventChans chan chan events.Event
	size       int
	created    int64
	reused     int64
}