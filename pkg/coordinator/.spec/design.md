# Coordinator Package Design

This document outlines the detailed design for the `pkg/coordinator` package. The coordinator serves as the central orchestrator for the entire Convergen pipeline, managing the event-driven flow between all components.

## Architecture Overview

The coordinator follows a layered architecture with clear separation between orchestration logic and component management:

```
┌─────────────────────────────────────────────────────────┐
│                    Public API Layer                    │
├─────────────────────────────────────────────────────────┤
│                 Pipeline Orchestrator                  │
├─────────────────────────────────────────────────────────┤
│   Event Bus    │    Resource Pool   │   Error Handler  │
├─────────────────────────────────────────────────────────┤
│  Component Mgr │   Context Mgr     │   Metrics Collector│
├─────────────────────────────────────────────────────────┤
│              Foundation Services                        │
└─────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Pipeline Coordinator (`coordinator.go`)

The main coordinator interface that orchestrates the entire pipeline:

```go
type Coordinator interface {
    // Generate code from source files
    Generate(ctx context.Context, sources []string, config *Config) (*GenerationResult, error)
    
    // Generate from in-memory source code
    GenerateFromSource(ctx context.Context, source string, config *Config) (*GenerationResult, error)
    
    // Get coordinator metrics
    GetMetrics() *CoordinatorMetrics
    
    // Shutdown gracefully
    Shutdown(ctx context.Context) error
}

type ConcreteCoordinator struct {
    config         *Config
    logger         *zap.Logger
    eventBus       events.EventBus
    componentMgr   ComponentManager
    resourcePool   ResourcePool
    errorHandler   ErrorHandler
    metricsCollector MetricsCollector
    contextMgr     ContextManager
}
```

### 2. Component Manager (`component_manager.go`)

Manages the lifecycle of all pipeline components:

```go
type ComponentManager interface {
    // Initialize all pipeline components
    Initialize(ctx context.Context, config *Config) error
    
    // Register component with event handlers
    RegisterComponent(name string, component PipelineComponent) error
    
    // Get component by name
    GetComponent(name string) (PipelineComponent, error)
    
    // Shutdown all components
    Shutdown(ctx context.Context) error
}

type PipelineComponent interface {
    Name() string
    Initialize(ctx context.Context, eventBus events.EventBus) error
    Shutdown(ctx context.Context) error
    GetMetrics() interface{}
}
```

### 3. Event Orchestrator (`event_orchestrator.go`)

Manages event-driven coordination between components:

```go
type EventOrchestrator interface {
    // Start pipeline execution
    StartPipeline(ctx context.Context, input *PipelineInput) error
    
    // Handle pipeline events
    HandleEvent(ctx context.Context, event events.Event) error
    
    // Get pipeline status
    GetStatus() *PipelineStatus
    
    // Cancel pipeline execution
    Cancel() error
}

type PipelineInput struct {
    Sources    []string
    SourceCode string
    Config     *Config
    Context    context.Context
}

type PipelineStatus struct {
    Stage         PipelineStage
    Progress      float64
    ComponentStatus map[string]ComponentStatus
    StartTime     time.Time
    ElapsedTime   time.Duration
    Errors        []error
}
```

### 4. Resource Pool Manager (`resource_pool.go`)

Manages shared resources across the pipeline:

```go
type ResourcePool interface {
    // Get worker pool for concurrent processing
    GetWorkerPool(ctx context.Context, size int) (*WorkerPool, error)
    
    // Get memory buffer pool
    GetBufferPool() *BufferPool
    
    // Get channel pool for event communication
    GetChannelPool() *ChannelPool
    
    // Release resources
    Release(ctx context.Context) error
}

type WorkerPool struct {
    Size     int
    Workers  chan struct{}
    Tasks    chan func()
    Done     chan struct{}
    Error    chan error
}
```

### 5. Error Handler (`error_handler.go`)

Aggregates and manages errors across the pipeline:

```go
type ErrorHandler interface {
    // Collect error from any component
    CollectError(component string, err error)
    
    // Get aggregated errors
    GetErrors() *ErrorReport
    
    // Check if pipeline should stop
    ShouldStop() bool
    
    // Reset error state
    Reset()
}

type ErrorReport struct {
    Errors        []ComponentError
    Critical      []error
    Warnings      []error
    TotalCount    int
    CriticalCount int
    WarningCount  int
}

type ComponentError struct {
    Component string
    Stage     PipelineStage
    Error     error
    Timestamp time.Time
    Context   map[string]interface{}
}
```

### 6. Metrics Collector (`metrics_collector.go`)

Collects and aggregates metrics from all pipeline components:

```go
type MetricsCollector interface {
    // Record pipeline event
    RecordEvent(event string, duration time.Duration, metadata map[string]interface{})
    
    // Record component metrics
    RecordComponent(component string, metrics interface{})
    
    // Get aggregated metrics
    GetMetrics() *CoordinatorMetrics
    
    // Reset metrics
    Reset()
}

type CoordinatorMetrics struct {
    PipelineExecutions int64
    TotalDuration      time.Duration
    AverageDuration    time.Duration
    SuccessRate        float64
    ComponentMetrics   map[string]interface{}
    ResourceUsage      *ResourceUsage
    EventCounts        map[string]int64
}
```

## Pipeline Flow Control

### Event-Driven Pipeline

The coordinator orchestrates the pipeline through a series of events:

```
1. StartEvent → Parser
2. ParseCompleteEvent → Planner  
3. PlanCompleteEvent → Executor
4. ExecuteCompleteEvent → Emitter
5. EmitCompleteEvent → Completion
```

### Pipeline Stages

```go
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
```

### Configuration System

```go
type Config struct {
    // Component configurations
    ParserConfig   *parser.Config
    PlannerConfig  *planner.Config
    ExecutorConfig *executor.Config
    EmitterConfig  *emitter.Config
    
    // Coordinator-specific settings
    MaxConcurrency     int
    EventBufferSize    int
    ComponentTimeout   time.Duration
    ErrorThreshold     int
    EnableMetrics      bool
    LogLevel          string
    
    // Resource management
    WorkerPoolSize     int
    BufferPoolSize     int
    ChannelPoolSize    int
    
    // Pipeline behavior
    StopOnFirstError   bool
    RetryTransientErrors bool
    MaxRetries         int
    RetryDelay         time.Duration
}
```

## Error Handling Strategy

### Error Categories

1. **Critical Errors**: Stop pipeline execution immediately
   - Component initialization failures
   - Configuration validation errors
   - Context cancellation

2. **Component Errors**: Errors from specific pipeline stages
   - Parser errors (syntax issues, type resolution failures)
   - Planner errors (dependency cycles, impossible mappings)
   - Executor errors (conversion failures)
   - Emitter errors (code generation issues)

3. **Transient Errors**: Temporary failures that can be retried
   - Resource allocation failures
   - Network timeouts
   - Temporary file system issues

### Recovery Mechanisms

```go
type RecoveryStrategy interface {
    // Determine if error is recoverable
    IsRecoverable(err error) bool
    
    // Attempt error recovery
    Recover(ctx context.Context, err error) error
    
    // Get recovery delay
    GetRetryDelay(attempt int) time.Duration
}
```

## Context Management

### Context Propagation

```go
type ContextManager interface {
    // Create pipeline context with timeout
    CreatePipelineContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
    
    // Create component context
    CreateComponentContext(parent context.Context, component string) context.Context
    
    // Check if context should cancel pipeline
    ShouldCancel(ctx context.Context) bool
    
    // Propagate cancellation to all components
    CancelAll()
}
```

## Integration Architecture

### Component Integration

The coordinator integrates with existing components through standardized interfaces:

```go
// Parser integration
type ParserIntegration struct {
    parser parser.Parser
    eventHandler events.EventHandler
}

// Planner integration  
type PlannerIntegration struct {
    planner planner.Planner
    eventHandler events.EventHandler
}

// Executor integration
type ExecutorIntegration struct {
    executor executor.Executor
    eventHandler events.EventHandler
}

// Emitter integration
type EmitterIntegration struct {
    emitter emitter.Emitter
    eventHandler events.EventHandler
}
```

### Extension Points

The coordinator provides extension points for custom components:

```go
type ComponentRegistry interface {
    // Register custom component
    RegisterCustomComponent(name string, factory ComponentFactory) error
    
    // Register middleware
    RegisterMiddleware(middleware PipelineMiddleware) error
    
    // Register event interceptor
    RegisterInterceptor(interceptor EventInterceptor) error
}

type ComponentFactory func(config interface{}) (PipelineComponent, error)

type PipelineMiddleware interface {
    Process(ctx context.Context, input interface{}, next func(interface{}) interface{}) interface{}
}
```

## Testing Strategy

### Unit Testing

Each coordinator component must be unit testable:

```go
// Mock implementations for testing
type MockEventBus struct{}
type MockComponent struct{}
type MockResourcePool struct{}

// Test utilities
func NewTestCoordinator(config *Config) *ConcreteCoordinator
func CreateTestPipeline() *Pipeline
func SetupTestComponents() map[string]PipelineComponent
```

### Integration Testing

Integration tests validate the complete pipeline:

```go
func TestCompletePipeline(t *testing.T)
func TestErrorHandling(t *testing.T)  
func TestConcurrency(t *testing.T)
func TestResourceManagement(t *testing.T)
```

## Performance Considerations

### Optimization Strategies

1. **Event Batching**: Batch related events to reduce overhead
2. **Resource Reuse**: Pool and reuse expensive resources
3. **Concurrent Processing**: Execute independent stages concurrently
4. **Memory Management**: Minimize allocations and prevent leaks

### Monitoring Points

- Pipeline execution time
- Component initialization time  
- Event processing latency
- Memory usage patterns
- Error rates and types
- Resource utilization

This design ensures the coordinator provides robust, performant, and maintainable orchestration of the entire Convergen pipeline while maintaining clear separation of concerns and extensibility.