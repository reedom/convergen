# Design

## Architecture Overview
- Style: Event-driven pipeline with staged processing
- Key Components: Parser (AST analysis) → Builder (mapping logic) → Generator (code creation) → Coordinator (orchestration) → Emitter (output)
- Data Flow: Source files → Domain models → Execution plans → Generated functions → Output files

## Interfaces & Data Contracts

### Pipeline Events
- ParseEvent: Methods parsed from source interfaces with annotations
- PlanEvent: Execution strategies for field mapping and concurrency
- ExecuteEvent: Generated assignments and function code
- EmitEvent: Final formatted output with imports

### Domain Models (pkg/domain)
- Type: Immutable type representations with generics support (constructor pattern required)
- Method: Complete conversion method specifications with source/dest types
- FieldMapping: Individual field conversion rules and strategies
- ExecutionPlan: Concurrency coordination and resource management

### Core Interfaces
```go
// Parser: Source → Domain Models
type ConvergenParser interface {
    ParseSourceFile(ctx context.Context, sourcePath, destPath string) (*ParseResult, error)
}

// Builder: Domain Models → Assignment Logic
type AssignmentHandler interface {
    Handle(lhs, rhs Node, args []Node) (Assignment, error)
}

// Generator: Assignment Logic → Code
type CodeGenerator interface {
    Generate(method Method) (Function, error)
}

// Coordinator: Pipeline Orchestration
type PipelineCoordinator interface {
    Generate(ctx context.Context, sources []string) (*Result, error)
}
```

## Error Handling & Resilience
- Timeouts: Context-based cancellation throughout pipeline with 30s default timeout
- Circuit breaker: Parser implements exponential backoff for package loading failures
- Error aggregation: Centralized error collection with rich context (file position, method name, field)
- Graceful degradation: Continue processing valid methods when individual methods fail
- Resource bounds: Goroutine pools limited to runtime.NumCPU() * 2, memory usage monitored

## Security & Compliance
- Input validation: All annotation parameters sanitized against code injection
- Generated code safety: No eval(), reflection, or unsafe operations in generated code
- File system isolation: Generated files written only to specified output directories
- Import resolution: Only standard library and explicitly imported packages allowed

## Rationale & Alternatives

### ADR-001: Event-Driven Architecture
- **Decision**: Use event system for component coordination instead of direct method calls
- **Reason**: Enables loose coupling, extensibility, and observability of pipeline flow
- **Alternatives**: Direct method composition (rejected: tight coupling), dependency injection (rejected: complexity)

### ADR-002: Domain Model Constructor Pattern
- **Decision**: Require NewMethod(), NewType() constructors instead of struct literals
- **Reason**: Ensures validation, immutability, and prevents invalid states
- **Alternatives**: Builder pattern (rejected: verbosity), validation methods (rejected: runtime errors)

### ADR-003: Concurrent Field Processing
- **Decision**: Process struct fields in parallel with ordered result assembly
- **Reason**: Significant performance improvement (40-70%) for complex structs
- **Alternatives**: Sequential processing (rejected: performance), fully async (rejected: ordering complexity)

### ADR-004: Struct Literal Default Output
- **Decision**: Generate struct literals by default, fallback to assignments for complexity
- **Reason**: More idiomatic Go code, better readability, compiler optimizations
- **Alternatives**: Always assignments (rejected: verbosity), always literals (rejected: complexity handling)

### ADR-005: Strategy Pattern for Parser
- **Decision**: Support LegacyParser, ModernParser, and AdaptiveParser strategies
- **Reason**: Backward compatibility while enabling performance optimizations
- **Alternatives**: Single parser (rejected: performance trade-offs), config flags (rejected: complexity)

### ADR-006: Chain of Responsibility for Assignment Generation
- **Decision**: Use handler chain (Skip → Literal → Converter → NameMapper → StructMatch)
- **Reason**: Extensible, testable, clear priority ordering
- **Alternatives**: Giant switch statement (rejected: maintainability), strategy per field type (rejected: complexity)

## Implementation Patterns

### Component Communication
All components communicate through events published to a central event bus. Each component subscribes to relevant events and publishes results for downstream consumption.

### Resource Management
Bounded goroutine pools prevent resource exhaustion. Worker pools are sized based on available CPU cores with configurable limits for memory-constrained environments.

### Type System Integration
Full Go generics support through type parameter extraction, constraint resolution, and concrete type instantiation. Type caching reduces redundant analysis.

### Output Generation Strategies
Adaptive code generation chooses between struct literal and assignment block styles based on complexity analysis. Error-returning converters automatically trigger assignment block mode.

### Testing Architecture
Behavior-driven integration tests in `tests/` directory verify end-to-end functionality. Unit tests alongside source provide component-level coverage. Domain model constructor pattern enables reliable test setup.
