# Parser Package Design

This document outlines the technical architecture and design decisions for the `pkg/parser` package.

## Architecture Overview

The parser follows a multi-stage pipeline architecture with clear separation of concerns:

1. **Source Analysis**: AST parsing and basic structure extraction
2. **Interface Discovery**: Identification of convergen-annotated interfaces  
3. **Annotation Processing**: Comment-based configuration parsing
4. **Type Resolution**: Comprehensive type analysis with generic support
5. **Domain Model Construction**: Transformation to domain entities
6. **Event Emission**: Publishing results to the generation pipeline

## Core Design Patterns

### Strategy Pattern Implementation

The parser implements a strategy pattern to support different parsing approaches:

```go
type ParseStrategy int

const (
    StrategyLegacy ParseStrategy = iota  // Traditional synchronous parsing
    StrategyModern                       // Concurrent processing with worker pools
    StrategyAuto                         // Adaptive strategy selection
)

type ConvergenParser interface {
    ParseSourceFile(ctx context.Context, sourcePath, destPath string) (*ParseResult, error)
    GetStrategy() ParseStrategy
    SetConfig(config *ParserConfig) error
}
```

**Design Rationale**: Multiple strategies allow optimization for different use cases while maintaining backward compatibility.

### Factory Pattern for Parser Creation

```go
type ParserFactory struct {
    defaultConfig *ParserConfig
}

func (pf *ParserFactory) CreateParser(strategy ParseStrategy) (ConvergenParser, error)
```

**Design Decision**: Factory pattern encapsulates parser creation logic and allows for consistent configuration management across different strategies.

### Worker Pool Pattern for Concurrency

```go
type PackageLoader struct {
    workerPool    chan struct{}
    cache         map[string]*CacheEntry
    cacheMutex    sync.RWMutex
    circuitBreaker *CircuitBreaker
}
```

**Design Rationale**: Bounded worker pools prevent resource exhaustion while providing controlled concurrency for package loading and method processing.

## Component Architecture

### Parser Coordinator

The main `Parser` struct orchestrates the parsing pipeline:

```go
type Parser struct {
    srcPath       string
    file          *ast.File
    fset          *token.FileSet
    pkg           *packages.Package
    opts          option.Options
    imports       util.ImportNames
    intfEntries   []*intfEntry
    packageLoader *PackageLoader
    config        *ParserConfig
}
```

**Design Principle**: Single Responsibility - the Parser coordinates but delegates specialized tasks to dedicated components.

### Configuration Management

Functional options pattern for flexible configuration:

```go
type ParserConfig struct {
    EnableConcurrentLoading   bool
    EnableMethodConcurrency   bool
    MaxConcurrentWorkers      int
    TypeResolutionTimeout     time.Duration
    CacheSize                 int
    EnablePerformanceMetrics  bool
}

func WithTimeout(timeout time.Duration) ConfigOption
func WithConcurrency(workers int) ConfigOption
```

**Design Benefit**: Immutable configuration with validation and sensible defaults.

### Error Handling Architecture

Multi-layer error handling with rich context:

```go
type ErrorHandler struct {
    classifier    *ErrorClassifier
    recovery      *ErrorRecovery
    contextLogger *zap.Logger
}

type ParseError struct {
    Code      string
    Message   string
    Phase     ParsePhase
    Severity  ErrorSeverity
    Context   map[string]interface{}
}
```

**Design Philosophy**: Fail-fast detection with graceful recovery and comprehensive error context.

## Type Resolution Design

### Generic Type Support

The type resolver handles Go 1.21+ generics through a multi-phase approach:

1. **Type Parameter Detection**: Identify generic type parameters in interfaces
2. **Constraint Analysis**: Parse and validate type constraints
3. **Instantiation Resolution**: Resolve concrete types from generic signatures
4. **Compatibility Checking**: Validate type compatibility across conversions

```go
type TypeResolver struct {
    typeInfo    *types.Info
    packageInfo *packages.Package
    cache       *TypeCache
    logger      *zap.Logger
}
```

**Design Challenge**: Go's type system complexity requires careful handling of:
- Type parameters and constraints
- Interface satisfaction
- Method set computation
- Import path resolution

### Caching Strategy

LRU cache with TTL for type resolution results:

```go
type TypeCache struct {
    cache      map[string]*CacheEntry
    accessTime map[string]time.Time
    mutex      sync.RWMutex
    maxSize    int
    ttl        time.Duration
}
```

**Design Trade-off**: Memory usage vs. performance - cache size is configurable to balance these concerns.

## Concurrency Design

### Package Loading Concurrency

Concurrent package loading with bounded resources:

```go
func (pl *PackageLoader) LoadPackageConcurrent(ctx context.Context, sourcePath, destPath string) (*LoadResult, error) {
    // Worker pool limits concurrent operations
    select {
    case pl.workerPool <- struct{}{}:
        defer func() { <-pl.workerPool }()
    case <-ctx.Done():
        return nil, ctx.Err()
    }
    
    // Proceed with loading...
}
```

**Design Consideration**: Context cancellation and timeout handling prevent hanging operations.

### Method Processing Concurrency

Parallel method processing with error aggregation:

```go
func (p *Parser) parseMethodsConcurrent(entry *intfEntry) ([]*model.MethodEntry, error) {
    methodChan := make(chan methodResult, len(entry.intf.Methods.List))
    
    // Process methods in parallel
    for _, method := range entry.intf.Methods.List {
        go func(m *ast.Field) {
            result := p.processMethodSafe(entry, m)
            methodChan <- result
        }(method)
    }
    
    // Collect results with error aggregation
}
```

**Design Principle**: Fail-fast with error aggregation - collect all errors before failing.

## Event System Integration

### Domain Event Publishing

```go
type ParseEvent struct {
    Methods     []*domain.Method
    Interfaces  []*domain.Interface
    BaseCode    string
    ProcessedAt time.Time
}

func (p *Parser) publishParseEvent(ctx context.Context, methods []*domain.Method) error {
    event := &ParseEvent{
        Methods:     methods,
        ProcessedAt: time.Now(),
    }
    return p.eventBus.Publish(ctx, event)
}
```

**Design Pattern**: Observer pattern with domain events for loose coupling between parser and downstream components.

## Performance Optimization Design

### Adaptive Strategy Selection

The adaptive parser automatically selects the optimal strategy:

```go
func (ap *AdaptiveParser) determineStrategy(ctx context.Context, sourcePath string) ParseStrategy {
    complexity := ap.assessComplexity(sourcePath)
    
    if complexity < 0.3 {
        return StrategyLegacy  // Simple files don't need concurrency
    }
    
    if complexity > 0.7 {
        return StrategyModern  // Complex files benefit from concurrency
    }
    
    return StrategyModern  // Default to modern for medium complexity
}
```

**Design Heuristics**: File size, interface count, and method complexity drive strategy selection.

### Circuit Breaker Pattern

Fault tolerance for external operations:

```go
type CircuitBreaker struct {
    state         CircuitState
    failureCount  int64
    lastFailTime  time.Time
    timeout       time.Duration
    maxFailures   int64
}
```

**Design Purpose**: Prevent cascading failures in distributed parsing scenarios.

## Current Technical Issues

### Memory Management

**Issue**: Type cache can grow unbounded in long-running processes.
**Solution**: Implement cache eviction policies based on LRU and TTL.

### Error Context Preservation

**Issue**: Complex error chains can lose original context.
**Solution**: Rich error wrapping with structured context preservation.

### Concurrent Safety

**Issue**: Shared state access patterns need careful synchronization.
**Solution**: Read-write mutexes for cache access, immutable configuration.

## Design Decisions

### Strategy Pattern Implementation

**Decision**: Implement strategy pattern with three parser types rather than single unified parser.
**Rationale**: 
- Backward compatibility is critical for existing integrations (LegacyParser)
- Performance gains significant for complex scenarios (ModernParser: 40-70% improvement)
- Intelligent selection reduces complexity for users (AdaptiveParser)
**Trade-off**: Code complexity vs. performance and usability benefits.
**Implementation**: Factory pattern encapsulates complexity, unified interface simplifies usage.

### Configuration Approach

**Decision**: Functional options pattern for configuration.
**Rationale**: Provides flexibility while maintaining type safety and defaults.
**Alternative**: Builder pattern was considered but adds complexity.

### Error Handling Strategy

**Decision**: Rich error context with recovery mechanisms.
**Rationale**: Debugging complex parsing issues requires detailed context.
**Trade-off**: Performance overhead vs. debuggability.

### Concurrency Model

**Decision**: Worker pool pattern with bounded resources.
**Rationale**: Predictable resource usage while enabling performance gains.
**Alternative**: Unbounded goroutines were rejected due to resource exhaustion risk.