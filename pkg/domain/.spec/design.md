# Domain Package Design

This document outlines the design of the `pkg/domain` package, which serves as the core business logic layer containing all essential data models and interfaces.

## Architecture Principles

### Immutability and Value Objects
All domain models are designed as immutable value objects to ensure thread safety and prevent unintended mutations during concurrent processing.

### Generic Type System
Leverage Go generics for type-safe interfaces while maintaining backwards compatibility and performance.

### Clear Abstractions
Provide clear abstractions that hide implementation complexity while exposing necessary functionality for other packages.

## Core Components

### Type System

```go
// Core type interface with generic support
type Type interface {
    Name() string
    Kind() TypeKind
    Generic() bool
    TypeParams() []TypeParam
    Underlying() Type
    String() string
    
    // Type relationship methods
    AssignableTo(other Type) bool
    Implements(iface Type) bool
    Comparable() bool
}

// Type kinds for dispatch and processing
type TypeKind int

const (
    KindBasic TypeKind = iota
    KindStruct
    KindSlice
    KindMap
    KindInterface
    KindPointer
    KindGeneric
    KindNamed
)

// Generic type parameter representation
type TypeParam struct {
    Name        string
    Constraint  Type
    Index       int
}
```

### Field and Struct Models

```go
// Field represents a struct field with metadata
type Field struct {
    Name      string
    Type      Type
    Tags      reflect.StructTag
    Position  int  // For ordering preservation
    Exported  bool
    Doc       string
}

// StructType represents a struct with ordered fields
type StructType struct {
    Name       string
    Fields     []Field
    TypeParams []TypeParam
    Package    string
}

// FieldSpec identifies a specific field access path
type FieldSpec struct {
    Path     []string  // e.g., ["User", "Address", "Street"]
    Type     Type
    IsMethod bool      // true for getter methods
    Receiver Type      // for method calls
}
```

### Conversion and Mapping Models

```go
// ConversionStrategy defines how to convert between field types
type ConversionStrategy interface {
    Name() string
    CanHandle(source, dest Type) bool
    GenerateCode(ctx context.Context, mapping FieldMapping) (Code, error)
    Dependencies() []string
    Priority() int  // For strategy selection
}

// FieldMapping represents a conversion between two fields
type FieldMapping struct {
    ID           string
    Source       FieldSpec
    Dest         FieldSpec
    Strategy     ConversionStrategy
    Config       MappingConfig
    Dependencies []string  // Field IDs this mapping depends on
}

// MappingConfig holds configuration for a specific mapping
type MappingConfig struct {
    Skip         bool
    Converter    *ConverterFunc
    Literal      *LiteralValue
    ErrorHandler ErrorHandlingStrategy
    Custom       map[string]any  // For strategy-specific config
}
```

### Method and Generation Models

```go
// Method represents a conversion method to be generated
type Method struct {
    Name        string
    SourceType  Type
    DestType    Type
    Config      MethodConfig
    Mappings    []FieldMapping
    Signature   MethodSignature
}

// MethodConfig holds method-level configuration
type MethodConfig struct {
    Style         StyleConfig      // return vs arg style
    Receiver      *ReceiverConfig  // optional receiver
    Reverse       bool
    CaseSensitive bool
    UseGetters    bool
    UseStringers  bool
    TypeCasting   bool
    PreProcess    []ManipulatorFunc
    PostProcess   []ManipulatorFunc
}

// MethodSignature describes the generated method signature
type MethodSignature struct {
    Name       string
    Receiver   *Receiver
    Params     []Parameter
    Results    []Parameter
    HasError   bool
}
```

### Execution Planning Models

```go
// ExecutionPlan defines how to execute field conversions concurrently
type ExecutionPlan struct {
    Method     *Method
    Batches    []ConcurrentBatch
    Resources  ResourceLimits
    Strategy   ExecutionStrategy
}

// ConcurrentBatch groups fields that can be processed in parallel
type ConcurrentBatch struct {
    ID       string
    Fields   []FieldMapping
    DependsOn []string  // Batch IDs this batch depends on
}

// ResourceLimits defines execution constraints
type ResourceLimits struct {
    MaxGoroutines int
    MaxMemoryMB   int
    TimeoutMS     int
    MaxConcurrentFields int
}

// ExecutionStrategy determines how to balance performance vs resources
type ExecutionStrategy int

const (
    StrategySequential ExecutionStrategy = iota
    StrategyBatched
    StrategyFullyConcurrent
    StrategyAdaptive
)
```

### Result and Code Models

```go
// GenerationResult represents the outcome of processing
type GenerationResult struct {
    Method      *Method
    Code        GeneratedCode
    Errors      []GenerationError
    Metrics     ProcessingMetrics
    Diagnostics []Diagnostic
}

// GeneratedCode represents the generated code structure
type GeneratedCode struct {
    Function    string
    Imports     []Import
    Comments    []Comment
    Metadata    CodeMetadata
}

// ProcessingMetrics track performance and resource usage
type ProcessingMetrics struct {
    TotalDurationMS   int64
    ConcurrentFields  int
    MaxGoroutines     int
    MemoryUsageMB     int
    CacheHitRate      float64
}
```

### Error and Validation Models

```go
// GenerationError provides rich error context
type GenerationError struct {
    Code     ErrorCode
    Message  string
    Phase    ProcessingPhase
    Method   string
    Field    string
    Source   SourceLocation
    Cause    error
    Context  map[string]any
}

// ErrorCode for categorizing errors
type ErrorCode int

const (
    ErrTypeResolution ErrorCode = iota
    ErrIncompatibleTypes
    ErrCircularDependency
    ErrInvalidAnnotation
    ErrConverterNotFound
    ErrCodeGeneration
)

// ProcessingPhase identifies where error occurred
type ProcessingPhase int

const (
    PhaseParsing ProcessingPhase = iota
    PhasePlanning
    PhaseExecution
    PhaseEmission
)
```

## Implementation Strategy

### Immutability Patterns

```go
// Builder pattern for complex object construction
type MethodBuilder struct {
    method *Method
}

func NewMethodBuilder(name string, src, dst Type) *MethodBuilder {
    return &MethodBuilder{
        method: &Method{
            Name: name,
            SourceType: src,
            DestType: dst,
            Mappings: make([]FieldMapping, 0),
        },
    }
}

func (b *MethodBuilder) AddMapping(mapping FieldMapping) *MethodBuilder {
    newMethod := *b.method  // Shallow copy
    newMethod.Mappings = append(newMethod.Mappings[:len(newMethod.Mappings):len(newMethod.Mappings)], mapping)
    return &MethodBuilder{method: &newMethod}
}

func (b *MethodBuilder) Build() *Method {
    // Return immutable copy
    result := *b.method
    result.Mappings = make([]FieldMapping, len(b.method.Mappings))
    copy(result.Mappings, b.method.Mappings)
    return &result
}
```

### Generic Interfaces

```go
// Generic repository pattern for type-safe storage
type Repository[T any] interface {
    Store(ctx context.Context, id string, item T) error
    Load(ctx context.Context, id string) (T, error)
    List(ctx context.Context, filter Filter[T]) ([]T, error)
    Delete(ctx context.Context, id string) error
}

// Type-safe field access with generics
type FieldAccessor[T any] interface {
    GetField(obj T, path []string) (any, error)
    SetField(obj T, path []string, value any) (T, error)
    HasField(obj T, path []string) bool
}
```

### Validation Framework

```go
// Validator interface for domain model validation
type Validator[T any] interface {
    Validate(ctx context.Context, item T) []ValidationError
    ValidateField(ctx context.Context, item T, field string) []ValidationError
}

// Composite validator for complex validation logic
type CompositeValidator[T any] struct {
    validators []Validator[T]
}

func (v *CompositeValidator[T]) Validate(ctx context.Context, item T) []ValidationError {
    var errors []ValidationError
    for _, validator := range v.validators {
        if errs := validator.Validate(ctx, item); len(errs) > 0 {
            errors = append(errors, errs...)
        }
    }
    return errors
}
```

### Cache Integration

```go
// Type information caching for performance
type TypeCache interface {
    GetType(name string) (Type, bool)
    PutType(name string, typ Type)
    InvalidateType(name string)
    Stats() CacheStats
}

// Method analysis caching
type MethodCache interface {
    GetAnalysis(method *Method) (*MethodAnalysis, bool)
    PutAnalysis(method *Method, analysis *MethodAnalysis)
    InvalidateMethod(methodID string)
}
```

## Extension Points

### Strategy Registration

```go
// Strategy registry for extensible conversion strategies
type StrategyRegistry interface {
    Register(strategy ConversionStrategy) error
    Unregister(name string) error
    Get(name string) (ConversionStrategy, bool)
    List() []ConversionStrategy
    FindBest(source, dest Type) (ConversionStrategy, error)
}
```

### Custom Type Support

```go
// CustomTypeHandler for special type handling
type CustomTypeHandler interface {
    Name() string
    CanHandle(typ Type) bool
    AnalyzeType(ctx context.Context, typ Type) (*TypeAnalysis, error)
    GenerateAccessCode(ctx context.Context, spec FieldSpec) (string, error)
}
```

This design provides a solid foundation for the domain layer, emphasizing immutability, type safety, and extensibility while supporting the concurrent processing requirements of the new architecture.