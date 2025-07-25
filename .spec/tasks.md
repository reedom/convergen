# Convergen Rewrite Implementation Plan

This document outlines the comprehensive implementation plan for the Convergen rewrite. The plan follows the ground design decisions and implements a modern, concurrent, event-driven architecture.

## Implementation Strategy

### Phase 1: Foundation and Domain Models (Weeks 1-2)

#### Task 1.1: Core Infrastructure Setup
- [ ] **1.1.1**: Update `go.mod` with new dependencies
  ```go
  // New dependencies
  go.uber.org/zap         // Structured logging
  go.uber.org/fx          // Dependency injection (optional)
  golang.org/x/sync       // Advanced concurrency primitives
  ```
- [ ] **1.1.2**: Create new package structure
  ```
  pkg/
  ├── domain/
  ├── coordinator/
  ├── parser/
  ├── planner/
  ├── executor/
  ├── emitter/
  └── internal/
      ├── events/
      ├── concurrency/
      └── testing/
  ```

#### Task 1.2: Domain Models (`pkg/domain`)
- [ ] **1.2.1**: Define core types with generics
  ```go
  // Type system with generic support
  type Type interface {
    Name() string
    Kind() TypeKind
    Generic() bool
    Underlying() Type
  }
  
  type Field[T Type] struct {
    Name string
    Type T
    Tags reflect.StructTag
    Position int // For ordering
  }
  ```
- [ ] **1.2.2**: Implement conversion strategies
  ```go
  type ConversionStrategy interface {
    Name() string
    CanHandle(source, dest Type) bool
    GenerateCode(ctx context.Context, mapping FieldMapping) (Code, error)
    Dependencies() []string
  }
  ```
- [ ] **1.2.3**: Create immutable data structures
- [ ] **1.2.4**: Add comprehensive unit tests for domain models

#### Task 1.3: Event System (`pkg/internal/events`)
- [ ] **1.3.1**: Define event interfaces with context support
  ```go
  type Event interface {
    ID() string
    Type() string
    Timestamp() time.Time
    Context() context.Context
  }
  
  type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType string, handler EventHandler) error
    Close() error
  }
  ```
- [ ] **1.3.2**: Implement in-memory event bus
- [ ] **1.3.3**: Add event tracing and debugging capabilities
- [ ] **1.3.4**: Create event bus tests with concurrency verification

### Phase 2: Parser Rewrite (`pkg/parser`) (Weeks 3-4)

#### Task 2.1: AST Analysis Enhancement
- [ ] **2.1.1**: Rewrite interface discovery with generics support
- [ ] **2.1.2**: Implement annotation parsing with registry pattern
  ```go
  type AnnotationRegistry interface {
    Register(processor AnnotationProcessor) error
    Process(comments []string) ([]AnnotationConfig, error)
    Validate(configs []AnnotationConfig) error
  }
  ```
- [ ] **2.1.3**: Add comprehensive type resolution including generics
- [ ] **2.1.4**: Implement field ordering preservation

#### Task 2.2: Domain Model Construction
- [ ] **2.2.1**: Transform AST to domain models
- [ ] **2.2.2**: Validate domain model consistency
- [ ] **2.2.3**: Emit `ParseEvent` with context
- [ ] **2.2.4**: Add extensive parser tests with real-world Go code

#### Task 2.3: Error Handling Enhancement
- [ ] **2.3.1**: Implement rich error context with fmt.Errorf
- [ ] **2.3.2**: Add error aggregation across parsing phases
- [ ] **2.3.3**: Create parser-specific error types

### Phase 3: Planner Implementation (`pkg/planner`) (Weeks 5-6)

#### Task 3.1: Dependency Analysis
- [ ] **3.1.1**: Implement field dependency graph construction
  ```go
  type DependencyGraph interface {
    AddField(field FieldMapping) error
    AddDependency(from, to string) error
    TopologicalSort() ([][]FieldMapping, error) // Returns batches
    DetectCycles() ([]string, error)
  }
  ```
- [ ] **3.1.2**: Create topological sorting for field processing order
- [ ] **3.1.3**: Detect and handle circular dependencies

#### Task 3.2: Concurrency Planning
- [ ] **3.2.1**: Implement batch generation for concurrent processing
- [ ] **3.2.2**: Add resource limit calculation and enforcement
- [ ] **3.2.3**: Create execution plan optimization
- [ ] **3.2.4**: Emit `PlanEvent` with execution strategy

#### Task 3.3: Testing and Validation
- [ ] **3.3.1**: Add planner unit tests with various dependency scenarios
- [ ] **3.3.2**: Create integration tests with parser
- [ ] **3.3.3**: Add performance tests for large struct processing

### Phase 4: Executor Implementation (`pkg/executor`) (Weeks 7-8)

#### Task 4.1: Concurrent Execution Engine
- [ ] **4.1.1**: Implement worker pool with bounded resources
  ```go
  type WorkerPool interface {
    Submit(ctx context.Context, task Task) <-chan Result
    Close() error
    Stats() PoolStats
  }
  ```
- [ ] **4.1.2**: Create field processing workers with context support
- [ ] **4.1.3**: Implement result ordering and assembly
- [ ] **4.1.4**: Add timeout and cancellation handling

#### Task 4.2: Error Collection and Reporting
- [ ] **4.2.1**: Implement concurrent error collection
- [ ] **4.2.2**: Add error context enrichment
- [ ] **4.2.3**: Create error aggregation strategies
- [ ] **4.2.4**: Emit `ExecuteEvent` with results and errors

#### Task 4.3: Resource Management
- [ ] **4.3.1**: Implement memory usage monitoring
- [ ] **4.3.2**: Add goroutine leak detection
- [ ] **4.3.3**: Create resource cleanup mechanisms

### Phase 5: Emitter Implementation (`pkg/emitter`) (Weeks 9-10)

#### Task 5.1: Code Generation Engine
- [ ] **5.1.1**: Implement adaptive output strategy
  ```go
  type OutputStrategy interface {
    ShouldUseCompositeLiteral(fields []FieldResult) bool
    GenerateConstructor(fields []FieldResult) (string, error)
    GenerateAssignments(fields []FieldResult) ([]string, error)
  }
  ```
- [ ] **5.1.2**: Create composite literal vs assignment block decision logic
- [ ] **5.1.3**: Implement stable output ordering enforcement
- [ ] **5.1.4**: Add code formatting and import management

#### Task 5.2: Output Optimization
- [ ] **5.2.1**: Implement dead code elimination
- [ ] **5.2.2**: Add variable name deduplication
- [ ] **5.2.3**: Create import grouping and sorting
- [ ] **5.2.4**: Add generated code validation

### Phase 6: Coordinator Implementation (`pkg/coordinator`) (Weeks 11-12)

#### Task 6.1: Pipeline Orchestration
- [ ] **6.1.1**: Implement event-driven pipeline coordinator
  ```go
  type Coordinator interface {
    Process(ctx context.Context, request GenerationRequest) (*GenerationResult, error)
    RegisterComponent(component PipelineComponent) error
    AddMiddleware(middleware Middleware) error
  }
  ```
- [ ] **6.1.2**: Add component lifecycle management
- [ ] **6.1.3**: Implement context propagation throughout pipeline
- [ ] **6.1.4**: Create pipeline configuration and tuning

#### Task 6.2: Integration and Testing
- [ ] **6.2.1**: Integrate all pipeline components
- [ ] **6.2.2**: Add end-to-end integration tests
- [ ] **6.2.3**: Create performance benchmarks
- [ ] **6.2.4**: Implement comprehensive error handling

### Phase 7: Testing Architecture Redesign (Weeks 13-14)

#### Task 7.1: Scenario-Based Testing Framework
- [ ] **7.1.1**: Create test scenario structure (inspired by goverter)
  ```
  tests/scenarios/
  ├── basic_conversion/
  ├── concurrent_processing/
  ├── error_handling/
  ├── performance/
  └── edge_cases/
  ```
- [ ] **7.1.2**: Implement scenario runner framework
- [ ] **7.1.3**: Add output comparison utilities
- [ ] **7.1.4**: Create performance regression detection

#### Task 7.2: Comprehensive Test Coverage
- [ ] **7.2.1**: Migrate existing tests to new architecture
- [ ] **7.2.2**: Add concurrency and race condition tests
- [ ] **7.2.3**: Create integration tests for all components
- [ ] **7.2.4**: Add property-based testing for edge cases

### Phase 8: Migration and Cleanup (Weeks 15-16)

#### Task 8.1: Legacy Code Migration
- [ ] **8.1.1**: Update `main.go` to use new coordinator
- [ ] **8.1.2**: Migrate configuration handling
- [ ] **8.1.3**: Update CLI interface and help text
- [ ] **8.1.4**: Remove legacy packages and code

#### Task 8.2: Documentation and Finalization
- [ ] **8.2.1**: Update README with new architecture information
- [ ] **8.2.2**: Create developer documentation
- [ ] **8.2.3**: Add performance benchmarks and comparisons
- [ ] **8.2.4**: Create migration guide for users

## Success Criteria

### Performance Targets
- [ ] **P1**: 50%+ improvement in generation time for large structs (>20 fields)
- [ ] **P2**: Stable memory usage regardless of struct size
- [ ] **P3**: CPU utilization scales with available cores

### Quality Targets  
- [ ] **Q1**: 95%+ test coverage across all packages
- [ ] **Q2**: Zero race conditions detected in concurrent tests
- [ ] **Q3**: Identical output across multiple runs (deterministic)
- [ ] **Q4**: All existing integration tests pass

### Maintainability Targets
- [ ] **M1**: New annotation types addable in <50 lines of code
- [ ] **M2**: All components unit testable in isolation
- [ ] **M3**: Clean dependency graph with no circular imports
- [ ] **M4**: Comprehensive error messages with actionable context

## Risk Mitigation

### Technical Risks
- **Risk**: Concurrency complexity leads to bugs
  - **Mitigation**: Extensive testing, gradual rollout, comprehensive logging
- **Risk**: Performance regression during migration
  - **Mitigation**: Benchmark tracking, performance gates in CI
- **Risk**: Breaking changes for existing users
  - **Mitigation**: Backward compatibility testing, migration tooling

### Schedule Risks
- **Risk**: Underestimated complexity in concurrent execution
  - **Mitigation**: Prototype early, parallel development tracks
- **Risk**: Testing framework development takes longer than expected
  - **Mitigation**: Start testing framework early, reuse existing patterns

This implementation plan provides a structured approach to rewriting Convergen with modern, concurrent, maintainable architecture while ensuring stability and performance improvements.
