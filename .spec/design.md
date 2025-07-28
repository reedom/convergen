# Convergen Rewrite Design Overview

This document provides a comprehensive design for the new Convergen architecture. It captures all ground design decisions and serves as the definitive guide for the rewrite.

## Ground Design Decisions

### 1. Architecture Philosophy
- **Minimal Clean Architecture**: Clear boundaries without over-engineering
- **Event-Driven Pipeline**: Internal coordination through events for robustness
- **Stable Output Ordering**: Deterministic, reproducible code generation
- **Field-Level Concurrency**: Parallel processing with ordered result assembly

### 2. Performance StrategyOk
- **Concurrent Field Processing**: Each struct field processed in parallel
- **Ordered Assembly**: Results combined in source code field order
- **Resource Management**: Bounded goroutine pools and memory usage
- **CPU Utilization**: Leverage multiple cores for generation speed

### 3. Code Generation Principles
- **Preserve Field Order**: Maintain exact source struct field ordering
- **Adaptive Construction**: Choose between composite literals and assignment blocks
- **Stable Output**: Identical results across multiple runs
- **Idiomatic Go**: Generate clean, readable Go code

### 4. Modern Go Integration
- **Context-Aware**: Support cancellation and timeouts throughout pipeline
- **Generic Types**: Use generics for type safety and performance
- **Error Wrapping**: Rich error context with fmt.Errorf
- **Structured Logging**: Efficient logging with zap

## Overall Architecture

### Event-Driven Pipeline Flow

```
Source File → [Parser] → ParseEvent → [Planner] → PlanEvent → [Executor] → ExecuteEvent → [Emitter] → Generated Code
                ↓                         ↓                        ↓                      ↓
            Domain Model             Execution Plan          Concurrent Results      Ordered Output
```

### Core Components

1. **Coordinator** (`pkg/coordinator`)
   - Pipeline orchestration with context
   - Event bus management
   - Error aggregation and reporting
   - Resource lifecycle management

2. **Domain Models** (`pkg/domain`) 
   - Core business entities (fields, mappings, conversions)
   - Type-safe representations
   - Immutable data structures
   - Generic interfaces where appropriate

3. **Parser** (`pkg/parser`)
   - AST analysis and annotation extraction  
   - Domain model construction
   - Type resolution with generics support
   - Emit: `ParseEvent{Methods: []domain.Method}`

4. **Planner** (`pkg/planner`)
   - Execution plan generation
   - Concurrency strategy determination
   - Field dependency analysis
   - Emit: `PlanEvent{ExecutionPlan: domain.ExecutionPlan}`

5. **Executor** (`pkg/executor`)
   - Concurrent field processing
   - Result ordering and assembly
   - Error collection and propagation
   - Emit: `ExecuteEvent{Results: []domain.FieldResult}`

6. **Emitter** (`pkg/emitter`)
   - Code generation from results
   - Output formatting and optimization
   - Stable ordering enforcement
   - Final file writing

### Data Flow Models

#### Domain Models (`pkg/domain`)

```go
// Core conversion specification
type Method struct {
    Name         string
    SourceType   Type
    DestType     Type
    Config       MethodConfig
    Fields       []FieldMapping
}

// Field-level conversion plan
type FieldMapping struct {
    Source      FieldSpec
    Dest        FieldSpec  
    Strategy    ConversionStrategy
    Dependencies []FieldMapping  // For ordering
}

// Execution coordination
type ExecutionPlan struct {
    Batches     []ConcurrentBatch
    Dependencies map[string][]string
    Resources   ResourceLimits
}
```

#### Event System

```go
// Internal pipeline events
type ParseEvent struct {
    Methods []domain.Method
    BaseCode string
    Context context.Context
}

type PlanEvent struct {
    Plan domain.ExecutionPlan
    Context context.Context
}

type ExecuteEvent struct {
    Results []domain.FieldResult
    Errors  []error
    Context context.Context
}
```

### Concurrency Design

#### Field-Level Parallelism

1. **Analysis Phase**: Identify field dependencies
2. **Batching Phase**: Group independent fields for concurrent processing
3. **Execution Phase**: Process each batch in parallel
4. **Assembly Phase**: Collect results in source order

#### Resource Management

```go
// Concurrent execution with bounded resources
type ConcurrentBatch struct {
    Fields    []FieldMapping
    Resources ResourceLimits
    Timeout   time.Duration
}

type ResourceLimits struct {
    MaxGoroutines int
    MaxMemoryMB   int
    Timeout       time.Duration
}
```

### Output Generation Strategy

#### Construction Style Selection

```go
// Adaptive output generation
type OutputStrategy interface {
    ShouldUseCompositeLiteral(fields []FieldResult) bool
    GenerateAssignment(field FieldResult) string
    PreserveFieldOrder(results []FieldResult) []FieldResult
}
```

**Decision Logic:**
- **Composite Literal**: Simple field assignments, no errors, <5 fields
- **Assignment Blocks**: Complex conversions, error handling, >5 fields
- **Mixed Approach**: Composite literal with assignment block for complex fields

#### Ordering Guarantees

1. **Parse-Time Ordering**: Capture source field declaration order
2. **Execution Ordering**: Process concurrently but collect in order
3. **Output Ordering**: Emit assignments in source field order
4. **Stability**: Identical output across runs

### Error Handling Strategy

#### Event-Driven Error Aggregation

```go
// Comprehensive error context
type GenerationError struct {
    Phase     string              // "parse", "plan", "execute", "emit"
    Method    string              // Method being processed
    Field     string              // Field being processed (if applicable)
    Cause     error               // Root cause
    Context   map[string]any      // Additional context
}

// Error collection across concurrent operations
type ErrorCollector interface {
    Collect(ctx context.Context, err GenerationError)
    HasErrors() bool
    Errors() []GenerationError
    Summary() string
}
```

### Extension Points

#### Annotation System

```go
// Extensible annotation processing
type AnnotationProcessor interface {
    Name() string
    Parse(comment string) (AnnotationConfig, error)
    Validate(config AnnotationConfig, context ValidationContext) error
}

// Registry for new annotation types
type AnnotationRegistry interface {
    Register(processor AnnotationProcessor)
    Process(comments []string) ([]AnnotationConfig, error)
}
```

#### Conversion Strategies

```go
// Pluggable conversion strategies
type ConversionStrategy interface {
    Name() string
    CanHandle(source, dest Type) bool
    GenerateCode(mapping FieldMapping) (Code, error)
    Dependencies() []string
}
```

### Testing Architecture

#### Scenario-Based Testing (inspired by goverter)

```
tests/
├── scenarios/
│   ├── basic_conversion/
│   │   ├── input.go          # Source with Convergen interface
│   │   ├── expected.go       # Expected generated output  
│   │   └── test.yaml         # Test configuration
│   ├── concurrent_fields/
│   ├── error_handling/
│   └── performance/
└── framework/
    ├── scenario_runner.go    # Test execution framework
    ├── output_comparer.go    # Generated code comparison
    └── performance_tracker.go # Performance regression detection
```

#### Component Testing Strategy

- **Unit Tests**: Each component in isolation with mocked dependencies
- **Integration Tests**: Pipeline segments with real data flow
- **Scenario Tests**: End-to-end generation with expected outputs
- **Concurrency Tests**: Race condition and ordering verification
- **Performance Tests**: Generation speed and memory usage tracking

