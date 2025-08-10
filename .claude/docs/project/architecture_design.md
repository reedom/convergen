# Convergen Architecture Design

## System Overview

Convergen implements a pipeline-based architecture for automated Go code generation. The system transforms annotated interface definitions into type-safe conversion functions through a series of coordinated processing stages.

## Architectural Principles

### Event-Driven Architecture
The system uses an event-driven architecture to coordinate between components:
- **Loose Coupling**: Components communicate through events rather than direct dependencies
- **Extensibility**: New components can be added by subscribing to relevant events
- **Error Handling**: Centralized error propagation through event system
- **Observability**: All component interactions are observable through event logging

### Pipeline Processing
The core processing follows a sequential pipeline pattern:
- **Single Responsibility**: Each stage has a clearly defined responsibility
- **Error Propagation**: Failures in any stage halt the pipeline gracefully
- **State Management**: Processing state is maintained through domain models
- **Validation Gates**: Each stage validates its inputs and outputs

### Domain-Driven Design
The architecture centers around a rich domain model:
- **Constructor Pattern**: All domain objects use constructor functions for validation
- **Type Safety**: Strong typing throughout the domain model
- **Immutability**: Domain objects are designed to be immutable where possible
- **Business Logic Encapsulation**: Domain logic is encapsulated within domain objects

## Core Processing Pipeline

### Stage 1: Parser (`pkg/parser/`)

**Responsibility**: Source file analysis and interface extraction

**Key Components**:
- **AST Parser**: Go abstract syntax tree analysis with concurrent processing and LRU caching
- **Interface Detector**: Identifies marked interfaces (`Convergen` or `:convergen`)
- **Annotation Parser**: Extracts and validates comment annotations
- **Type Analyzer**: Analyzes source and destination types including generic type parameters
- **Constraint Parser**: Parses and validates Go generic constraints (union, underlying, interface)
- **Generic Interface Parser**: Extracts generic interfaces with type parameter information

**Input**: Go source files
**Output**: Parsed interface definitions with annotations

**Domain Models**:
- `Interface`: Represents a parsed interface with methods
- `Method`: Individual method definition with annotations
- `Annotation`: Parsed comment annotations with validation
- `GenericInterface`: Generic interface with type parameters and constraints
- `ParsedConstraint`: Validated constraint information (any, comparable, union, underlying)
- `TypeParam`: Type parameter with constraint validation

### Stage 2: Builder (`pkg/builder/`)

**Responsibility**: Conversion logic modeling and field mapping strategy

**Key Components**:
- **Type Relationship Analyzer**: Determines type compatibility and conversion paths
- **Field Mapping Engine**: Applies various mapping strategies
- **Strategy Selector**: Chooses optimal mapping strategy for each field
- **Validation Engine**: Validates mapping completeness and correctness
- **Generic Field Mapper**: Handles field mapping for instantiated generic types
- **Type Instantiator**: Converts generic interfaces to concrete implementations

**Input**: Parsed interface definitions
**Output**: Conversion models with field mapping strategies

**Domain Models**:
- `Copier`: Central model representing a conversion function
- `FieldMapping`: Individual field conversion specification
- `MappingStrategy`: Enumeration of available mapping strategies

**Mapping Strategies**:
- **Name Matching**: Automatic field matching by name
- **Explicit Mapping**: User-specified field mappings via `:map`
- **Custom Converters**: User-provided conversion functions via `:conv`
- **Type Casting**: Automatic type conversion via `:typecast`
- **String Conversion**: String method usage via `:stringer`

### Stage 3: Generator (`pkg/generator/`)

**Responsibility**: Go code generation from conversion models

**Key Components**:
- **Code Template Engine**: Generates Go function templates
- **Import Manager**: Resolves and organizes import statements
- **Optimization Engine**: Applies code optimizations
- **Syntax Validator**: Ensures generated code is syntactically correct
- **Generic Code Generator**: Generates concrete implementations from generic templates
- **Template System**: Generic-aware templates with type substitution support

**Input**: Conversion models with mapping strategies
**Output**: Generated Go function representations

**Domain Models**:
- `Function`: Generated function representation
- `Import`: Import statement management
- `CodeBlock`: Structured code generation

### Stage 4: Coordinator (`pkg/coordinator/`)

**Responsibility**: Pipeline orchestration and component coordination

**Key Components**:
- **Pipeline Manager**: Orchestrates the processing pipeline
- **Event Bus**: Manages event-driven communication
- **Error Handler**: Centralizes error handling and recovery
- **Component Registry**: Manages component lifecycle

**Input**: Processing requests
**Output**: Coordinated pipeline execution

**Domain Models**:
- `PipelineConfig`: Pipeline configuration and settings
- `ProcessingContext`: Execution context and state
- `ComponentStatus`: Component health and status tracking

## Generics Architecture (`pkg/domain/`, `pkg/parser/`, `pkg/generator/`)

### Type System and Constraints

**Constraint Parsing System**:
- **Union Constraints**: Parse `~int | ~string | ~float64` syntax
- **Underlying Constraints**: Handle `~string` underlying type constraints  
- **Interface Constraints**: Support custom interface constraints and `comparable`
- **Validation**: Comprehensive constraint satisfaction checking

**Constraint Types Supported**:
```go
// Basic constraints
type Orderable interface{ ~int | ~float64 | ~string }
type Comparable interface{ comparable }
type Any interface{ any }

// Complex constraints
type StringLike interface{ ~string }
type NumericUnion interface{ ~int | ~int32 | ~int64 | ~float32 | ~float64 }
```

### Type Instantiation Engine

**Core Components**:
- **TypeInstantiator**: Main instantiation engine with caching and validation
- **TypeSubstitutionEngine**: Handles type parameter replacement in complex types
- **CrossPackageTypeLoader**: Resolves types from external packages
- **ConstraintValidator**: Validates type arguments against constraints

**Instantiation Process**:
1. **Parse**: Extract generic interface with type parameters
2. **Validate**: Check type arguments satisfy constraints
3. **Substitute**: Replace type parameters with concrete types throughout type structure
4. **Cache**: Store instantiated results for performance
5. **Generate**: Create concrete implementation code

**Performance Features**:
- **Caching**: LRU cache for instantiated interfaces with configurable capacity
- **Cycle Detection**: Prevents infinite recursion in circular type dependencies
- **Cross-Package**: Support for resolving types from imported packages
- **Metrics Tracking**: Performance monitoring and optimization statistics

### Generic Code Generation

**Template Architecture**:
- **Generic Templates**: Specialized templates for generic type generation
- **Type Substitution**: Runtime type parameter replacement in generated code
- **Import Resolution**: Automatic import management for generic type dependencies
- **Optimization**: Dead code elimination and performance optimizations

**Code Generation Templates**:
```go
// Basic generic conversion template
func Convert{{.TypeParams}}(src {{.SourceType}}) ({{.DestType}}, error) {
    return {{.DestType}}{
        {{range .FieldMappings}}
        {{.DestField}}: {{.ConversionExpression}},
        {{end}}
    }, nil
}

// Complex conversion with error handling
func ConvertWithValidation{{.TypeParams}}(src {{.SourceType}}) ({{.DestType}}, error) {
    {{.ValidationCode}}
    // ... conversion logic
}
```

### Integration Points

**CLI Integration**:
- Support for `-type TypeMapper[T,U]` syntax
- Multiple generic type instantiation in single generation run
- Error handling for invalid constraint combinations

**Pipeline Integration**:
- **Parser Stage**: Extract generic interfaces and type parameters
- **Builder Stage**: Instantiate concrete types from generic specifications
- **Generator Stage**: Generate concrete implementation code
- **Emitter Stage**: Output with proper imports and formatting

## Supporting Architecture

### Domain Model (`pkg/domain/`)

**Core Types**:
- `Method`: Central method representation with constructors
- `MethodResult`: Processing results with metadata
- `BasicType`: Type system representation
- `StructType`: Struct-specific type information
- `ExecutionError`: Domain-specific error types
- `GenerationError`: Code generation error types

**Generics-Specific Types**:
- `TypeParam`: Type parameter with constraint information and validation
- `GenericInterface`: Generic interface with type parameters and methods
- `InstantiatedInterface`: Concrete interface created from generic + type arguments
- `TypeInstantiator`: Engine for converting generic types to concrete types
- `TypeSubstitutionEngine`: Handles type parameter replacement in complex type structures
- `ParsedConstraint`: Validated constraint with union/underlying/interface support
- `SubstitutionResult`: Result of type substitution with performance metrics
- `ValidationResult`: Constraint validation results with detailed violation information

**Constructor Pattern**:
```go
// Always use constructors for domain object creation
sourceType := domain.NewBasicType("User", reflect.Struct)
method, err := domain.NewMethod("ConvertUser", sourceType, destType)

// Avoid direct struct literals
// ❌ method := &domain.Method{Name: "ConvertUser", ...}
// ✅ method, err := domain.NewMethod("ConvertUser", sourceType, destType)
```

### Executor (`pkg/executor/`)

**Responsibility**: Field mapping strategy execution

**Key Components**:
- **Strategy Executor**: Executes specific mapping strategies
- **Field Processor**: Processes individual field conversions
- **Type Converter**: Handles type conversions and casting
- **Validation Engine**: Validates execution results

**Strategy Implementations**:
- **NameMatcher**: Implements name-based field matching
- **ExplicitMapper**: Handles `:map` annotations
- **ConverterApplier**: Applies `:conv` custom functions
- **TypeCaster**: Implements `:typecast` conversions
- **StringConverter**: Handles `:stringer` conversions

### Emitter (`pkg/emitter/`)

**Responsibility**: Final code generation and optimization

**Key Components**:
- **Code Generator**: Final Go code generation
- **Formatter**: Code formatting and style application
- **Optimizer**: Code optimization and dead code elimination
- **File Writer**: Output file management

**Optimization Features**:
- **Dead Code Elimination**: Removes unused imports and variables
- **Inline Optimization**: Inlines simple conversions
- **Import Deduplication**: Consolidates duplicate imports
- **Performance Optimization**: Generates efficient code patterns

### Event System (`pkg/internal/events/`)

**Event Types**:
- **Processing Events**: Pipeline stage completion events
- **Error Events**: Error occurrence and propagation
- **Status Events**: Component status changes
- **Metric Events**: Performance and timing metrics

**Event Pattern**:
```go
// Event publishing
err := eventBus.Publish(event)

// Event handling
err := handler.Handle(ctx, event)

// Event creation
event := events.NewEvent(eventType, data)
```

## Configuration Architecture (`pkg/config/`)

**Configuration Sources**:
- **Command Line Arguments**: CLI flag processing
- **Environment Variables**: Runtime environment configuration
- **Configuration Files**: Optional config file support
- **Default Values**: Sensible defaults for all options

**Configuration Categories**:
- **Processing Options**: Annotation processing behavior
- **Output Options**: Code generation preferences
- **Optimization Options**: Performance optimization settings
- **Debug Options**: Logging and debugging configuration

## Utility Architecture (`pkg/util/`)

**Utility Categories**:
- **AST Utilities**: Go AST manipulation and analysis
- **Type Checking**: Type compatibility and validation
- **Import Management**: Import path resolution and organization
- **File Utilities**: File system operations and path management

## Data Flow Architecture

### Request Processing Flow
1. **Input Validation**: Validate source files and configuration
2. **Interface Discovery**: Scan for marked interfaces
3. **Method Analysis**: Extract method signatures and annotations
4. **Type Analysis**: Analyze source and destination types
5. **Mapping Generation**: Generate field mapping strategies
6. **Code Generation**: Create Go function implementations
7. **Optimization**: Apply code optimizations
8. **Output Generation**: Write final generated code

### Event Flow
1. **Request Received**: Initial processing request event
2. **Stage Started**: Each pipeline stage start event
3. **Progress Updates**: Intermediate progress events
4. **Error Notifications**: Error occurrence events
5. **Stage Completed**: Pipeline stage completion events
6. **Result Generated**: Final result generation event

### Error Handling Flow
1. **Error Detection**: Component-level error detection
2. **Error Wrapping**: Domain-specific error wrapping
3. **Error Propagation**: Event-based error propagation
4. **Error Recovery**: Graceful degradation strategies
5. **Error Reporting**: User-friendly error reporting

## Dependency Architecture

### Internal Dependencies
- **Domain-First**: All packages depend on domain models
- **Utility Support**: Common utilities support all packages
- **Event Communication**: Event system enables loose coupling
- **Configuration Injection**: Configuration passed through contexts

### External Dependencies
- **Minimal Philosophy**: Prefer standard library over external dependencies
- **Go Standard Library**: Primary dependency on Go AST and reflection
- **Security Monitoring**: All dependencies monitored for vulnerabilities
- **Version Pinning**: Specific version dependencies for reproducibility

## Module Structure

```
github.com/reedom/convergen/v8/
├── main.go                 # CLI entry point
├── pkg/
│   ├── config/            # Configuration management
│   ├── parser/            # Stage 1: Source parsing
│   ├── builder/           # Stage 2: Logic building
│   ├── generator/         # Stage 3: Code generation
│   ├── coordinator/       # Stage 4: Orchestration
│   ├── executor/          # Field mapping execution
│   ├── emitter/           # Final code emission
│   ├── domain/            # Domain models
│   ├── option/            # Annotation options
│   ├── util/              # Utilities
│   ├── logger/            # Logging
│   └── internal/
│       └── events/        # Event system
└── tests/                 # Integration tests
```

## Design Patterns

### Constructor Pattern
All domain objects use constructor functions for validation and initialization:
```go
method, err := domain.NewMethod(name, sourceType, destType)
if err != nil {
    return fmt.Errorf("invalid method: %w", err)
}
```

### Event-Driven Pattern
Components communicate through events rather than direct calls:
```go
// Publisher
event := events.NewProcessingEvent(stage, data)
eventBus.Publish(event)

// Subscriber
func (h *Handler) Handle(ctx context.Context, event events.Event) error {
    // Handle event
    return nil
}
```

### Pipeline Pattern
Processing flows through sequential stages with validation:
```go
type Pipeline interface {
    Process(ctx context.Context, input Input) (Output, error)
}

type Stage interface {
    Execute(ctx context.Context, input StageInput) (StageOutput, error)
    Validate(input StageInput) error
}
```

### Strategy Pattern
Field mapping uses strategy pattern for different conversion approaches:
```go
type MappingStrategy interface {
    CanHandle(source, dest FieldType) bool
    Apply(source, dest Field) (Code, error)
}
```

This architecture ensures maintainable, extensible, and robust code generation while following Go best practices and design principles.

