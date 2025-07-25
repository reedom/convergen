# Emitter Package Design

This document outlines the detailed design for the `pkg/emitter` package. The emitter is responsible for generating high-quality, stable Go code from execution results using an event-driven architecture with sophisticated optimization strategies.

## Architecture Overview

The emitter follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────┐
│                    Event Bus Integration                │
├─────────────────────────────────────────────────────────┤
│                  Emitter Controller                     │
├─────────────────────────────────────────────────────────┤
│  Code Generator  │  Output Strategy  │  Format Manager  │
├─────────────────────────────────────────────────────────┤
│   Import Mgr    │   Template Sys   │   Optimization    │
├─────────────────────────────────────────────────────────┤
│             Foundation Components                       │
└─────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Emitter Controller (`emitter.go`)

The main controller orchestrates the code generation pipeline:

```go
type Emitter interface {
    // Generate code from execution results
    GenerateCode(ctx context.Context, results *domain.ExecutionResults) (*GeneratedCode, error)
    
    // Generate single method
    GenerateMethod(ctx context.Context, method *domain.MethodResult) (*MethodCode, error)
    
    // Apply global optimizations
    OptimizeOutput(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error)
    
    // Get generation metrics
    GetMetrics() *EmitterMetrics
    
    // Shutdown gracefully
    Shutdown(ctx context.Context) error
}

type ConcreteEmitter struct {
    config       *EmitterConfig
    logger       *zap.Logger
    eventBus     events.EventBus
    codeGen      CodeGenerator
    outputStrat  OutputStrategy
    formatMgr    FormatManager
    importMgr    ImportManager
    templateSys  TemplateSystem
    optimizer    CodeOptimizer
    metrics      *EmitterMetrics
}
```

### 2. Code Generator (`code_generator.go`)

Handles the core code generation logic:

```go
type CodeGenerator interface {
    // Generate method implementation
    GenerateMethodCode(ctx context.Context, method *domain.MethodResult) (*MethodCode, error)
    
    // Generate field assignments
    GenerateFieldCode(ctx context.Context, field *domain.FieldResult) (*FieldCode, error)
    
    // Generate error handling
    GenerateErrorHandling(ctx context.Context, errors []domain.ExecutionError) (*ErrorCode, error)
}

type ConcreteCodeGenerator struct {
    strategies   map[string]GenerationStrategy
    templates    TemplateSystem
    validator    CodeValidator
    logger       *zap.Logger
}
```

### 3. Output Strategy (`output_strategy.go`)

Determines the optimal code generation approach:

```go
type OutputStrategy interface {
    // Determine construction strategy for a method
    SelectStrategy(ctx context.Context, method *domain.MethodResult) ConstructionStrategy
    
    // Analyze field complexity for strategy selection
    AnalyzeFieldComplexity(fields []*domain.FieldResult) *ComplexityMetrics
    
    // Decide between composite literal vs assignment blocks
    ShouldUseCompositeLiteral(method *domain.MethodResult) bool
    
    // Estimate performance characteristics of different strategies
    EstimatePerformance(method *domain.MethodResult) *PerformanceEstimate
}
```

**Strategy Selection Logic:**
- **Composite Literal**: ≤5 fields, no errors, simple types
- **Assignment Block**: Complex fields, error handling required
- **Mixed Approach**: Mix of simple and complex fields (≥3 fields)

### 4. Format Manager (`format_manager.go`)

Handles professional code formatting:

```go
type FormatManager interface {
    // Format complete generated code
    FormatCode(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error)
    
    // Apply standard Go formatting
    ApplyGoFormat(source string) (string, error)
    
    // Validate formatting compliance
    ValidateFormat(source string) error
    
    // Format import declarations
    FormatImports(imports *ImportDeclaration) (*ImportDeclaration, error)
}
```

### 5. Import Manager (`import_manager.go`)

Manages Go imports with conflict resolution:

```go
type ImportManager interface {
    // Analyze code and determine required imports
    AnalyzeImports(ctx context.Context, code *GeneratedCode) (*ImportAnalysis, error)
    
    // Generate import declarations from analysis
    GenerateImports(ctx context.Context, analysis *ImportAnalysis) (*ImportDeclaration, error)
    
    // Resolve import name conflicts
    ResolveConflicts(imports []*Import) ([]*Import, error)
    
    // Optimize import organization
    OptimizeImports(imports []*Import) ([]*Import, error)
}
```

### 6. Code Optimizer (`optimizer.go`)

Multi-level code optimization engine:

```go
type CodeOptimizer interface {
    // Apply all configured optimizations
    OptimizeCode(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error)
    
    // Optimize single method
    OptimizeMethodCode(method *MethodCode) error
    
    // Remove unused variables and unreachable code
    EliminateDeadCode(code *GeneratedCode) error
    
    // Optimize variable names and remove conflicts
    OptimizeVariableNames(code *GeneratedCode) error
    
    // Simplify complex expressions
    SimplifyExpressions(code *GeneratedCode) error
    
    // Remove redundant operations
    RemoveRedundancy(code *GeneratedCode) error
}
```

### 7. Event Integration (`events.go`)

Event-driven architecture integration:

```go
type EmitterEventHandler struct {
    emitter    Emitter
    eventBus   events.EventBus
    logger     *zap.Logger
}

// Event types
const (
    EventEmitterStarted        = "emitter.started"
    EventEmitterCompleted      = "emitter.completed" 
    EventEmitterFailed         = "emitter.failed"
    EventCodeGenerationStarted = "emitter.code_generation.started"
    EventMethodGenerated       = "emitter.method.generated"
    EventStrategySelected      = "emitter.strategy.selected"
)
```

## Generation Strategies

The emitter implements three core generation strategies:

### 1. Composite Literal Strategy

**Use Cases:** Simple methods with ≤5 fields, no error handling

```go
// Generated example
return &DestType{
    Name:  src.Name,
    Email: src.Email,
    ID:    src.ID,
}, nil
```

### 2. Assignment Block Strategy  

**Use Cases:** Complex methods, error handling required

```go
// Generated example
var dest DestType

dest.Name = src.Name
converted_Email, err := converter.Convert(src.Email)
if err != nil {
    return nil, fmt.Errorf("converting Email: %w", err)
}
dest.Email = converted_Email

return &dest, nil
```

### 3. Mixed Approach Strategy

**Use Cases:** Mix of simple and complex fields (≥3 fields)

```go
// Generated example
dest := &DestType{
    Name: src.Name,    // Simple fields in composite literal
    ID:   src.ID,
}

// Complex fields as assignments
converted_Email, err := converter.Convert(src.Email)
if err != nil {
    return nil, fmt.Errorf("converting Email: %w", err)  
}
dest.Email = converted_Email

return dest, nil
```

## Implementation Architecture

### Key Data Structures

```go
type GeneratedCode struct {
    PackageName string
    Imports     *ImportDeclaration
    Methods     []*MethodCode
    BaseCode    string
    Source      string
    Metadata    *GenerationMetadata
    Metrics     *GenerationMetrics
}

type MethodCode struct {
    Name          string
    Signature     string
    Body          string
    ErrorHandling string
    Documentation string
    Imports       []*Import
    Complexity    *ComplexityMetrics
    Strategy      ConstructionStrategy
    Fields        []*FieldCode
}

type EmitterConfig struct {
    // Output preferences
    PreferCompositeLiterals bool
    MaxFieldsForComposite   int
    IndentStyle            string
    LineWidth              int
    
    // Optimization settings
    OptimizationLevel     OptimizationLevel
    EnableDeadCodeElim    bool
    EnableVarOptimization bool
    EnableImportOpt       bool
    
    // Performance settings
    EnableConcurrentGen  bool
    MaxConcurrentMethods int
    GenerationTimeout    time.Duration
}
```

## Optimization Levels

```go
type OptimizationLevel int

const (
    OptimizationNone OptimizationLevel = iota
    OptimizationBasic      // Dead code elimination, basic variable optimization
    OptimizationAggressive // + Expression simplification, redundancy removal
    OptimizationMaximal    // + Advanced AST optimizations
)
```

## Event-Driven Integration

The emitter integrates with the pipeline through comprehensive event handling:

**Published Events:**
- `emitter.started` - Code generation begins
- `emitter.completed` - Generation completes successfully
- `emitter.failed` - Generation encounters errors
- `emitter.method.generated` - Individual method completion
- `emitter.strategy.selected` - Strategy selection with reasoning

**Consumed Events:**
- `executor.completed` - Triggers code generation from execution results
- `planner.method_planned` - Pre-planning optimization hints

## Testing Coverage

The emitter package includes comprehensive test coverage:

### Test Files
- **emitter_test.go** - Core emitter functionality (8 test cases)
- **events_test.go** - Event system integration (6 test cases)  
- **code_generator_test.go** - Code generation logic (7 test cases)
- **strategies_test.go** - Generation strategies (6 test cases)
- **integration_test.go** - End-to-end workflows (6 test cases)

### Coverage Areas
- ✅ Unit testing of all major components
- ✅ Integration testing of complete workflows
- ✅ Event-driven architecture validation
- ✅ Concurrent generation testing
- ✅ Error handling and edge cases
- ✅ Performance and optimization validation

## Implementation Status

**✅ COMPLETED** - All design components have been fully implemented:

- ✅ **Event-driven architecture** with comprehensive event types
- ✅ **Adaptive generation strategies** with intelligent selection
- ✅ **Multi-level optimization engine** with AST analysis
- ✅ **Professional code formatting** with gofmt/goimports
- ✅ **Sophisticated import management** with conflict resolution
- ✅ **Concurrent processing** with stability guarantees
- ✅ **Comprehensive testing** with 33+ test cases
- ✅ **Performance optimization** with configurable levels

The emitter package successfully delivers a sophisticated, high-performance code generation system that integrates seamlessly with the modern Convergen pipeline architecture.