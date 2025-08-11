# Enhanced Nested Generic Type Field Mapping

This document describes the enhancements made to the GenericFieldMapper for handling deeply nested generic structures as per task 1.1.

## Overview

The enhanced GenericFieldMapper now supports:

1. **Deeply nested generic structures** like `Map[string, List[T]]` → `Map[string, Array[U]]`
2. **Recursive type parameter resolution** for complex scenarios
3. **Generic type aliases and constraints** support
4. **Performance optimization** for recursive operations

## Key Components

### 1. RecursiveTypeResolver

A new component that handles recursive type parameter resolution:

```go
type RecursiveTypeResolver struct {
    logger              *zap.Logger
    typeSubstitution    *domain.TypeSubstitutionEngine
    config              *RecursiveResolverConfig
    
    // Resolution tracking
    resolutionStack     []string
    visitedTypes        map[string]*ResolutionResult
    circularReferences  map[string]bool
    
    // Type alias support
    aliasRegistry       map[string]domain.Type
    aliasDepthTracker   map[string]int
    
    // Constraint validation
    constraintCache     map[string]*ConstraintValidationResult
    
    // Performance metrics
    metrics             *RecursiveResolutionMetrics
}
```

**Key Features:**
- **Cycle detection**: Prevents infinite recursion in circular type dependencies
- **Type alias resolution**: Supports complex type alias chains
- **Constraint validation**: Validates generic type constraints
- **Performance tracking**: Comprehensive metrics for optimization

### 2. Enhanced GenericFieldMapper

The GenericFieldMapper has been enhanced with:

```go
type GenericFieldMapper struct {
    // ... existing fields ...
    
    // Enhanced: Recursive type resolver for deeply nested generics
    recursiveResolver *RecursiveTypeResolver
}
```

**New Methods:**
- `generateNestedSliceFieldAssignment()`: Handles slice-to-slice conversions
- `generateNestedMapFieldAssignment()`: Handles map-to-map conversions  
- `generateNestedGenericFieldAssignment()`: Handles generic parameter conversions
- `typesConvertibleForMaps()`: Enhanced map type compatibility
- `typesConvertibleForGenerics()`: Enhanced generic type compatibility
- `isDeeplyNestedGeneric()`: Detects complex nested structures

### 3. Type Compatibility System

Enhanced type compatibility checking:

```go
// Enhanced compatibility checking
func (gfm *GenericFieldMapper) typesConvertible(srcType, dstType domain.Type) bool {
    // Basic type conversions (existing)
    // ... 
    
    // Enhanced: Map conversions for nested generics
    if srcType.Kind() == domain.KindMap && dstType.Kind() == domain.KindMap {
        return gfm.typesConvertibleForMaps(srcType, dstType)
    }
    
    // Enhanced: Generic type conversions
    if srcType.Kind() == domain.KindGeneric || dstType.Kind() == domain.KindGeneric {
        return gfm.typesConvertibleForGenerics(srcType, dstType)
    }
    
    // Enhanced: Named type conversions with generic support
    if srcType.Kind() == domain.KindNamed || dstType.Kind() == domain.KindNamed {
        return gfm.typesConvertibleForNamedTypes(srcType, dstType)
    }
    
    return false
}
```

## Usage Examples

### 1. Basic Nested Generic Conversion

```go
// Source: Container[List[string]]
// Dest:   Container[Array[string]]

mapper := NewGenericFieldMapper(...)
result, err := mapper.MapGenericFields(sourceType, destType, typeSubstitutions, options)
```

### 2. Complex Map-List Conversion

```go
// Source: Map[string, List[T]]
// Dest:   Map[string, Array[U]]

typeSubstitutions := map[string]domain.Type{
    "T": domain.NewBasicType("int", 0),
    "U": domain.NewBasicType("int", 0),
}

result, err := mapper.MapGenericFields(sourceType, destType, typeSubstitutions, options)
```

### 3. Type Alias Registration

```go
// Register common type aliases
mapper.RegisterTypeAlias("List", domain.NewSliceType(domain.NewBasicType("interface{}", 0), ""))
mapper.RegisterTypeAlias("Array", domain.NewSliceType(domain.NewBasicType("interface{}", 0), ""))
mapper.RegisterTypeAlias("Map", domain.NewBasicType("map[interface{}]interface{}", 0))
```

### 4. Deep Recursive Structures

```go
// Handles deeply nested structures automatically
// Source: Level1[Level2[Level3[T]]]
// Dest:   Level1[Level2[Level3[U]]]

result, err := mapper.MapGenericFields(deepSourceType, deepDestType, typeSubstitutions, options)
```

## Configuration Options

### RecursiveResolverConfig

```go
type RecursiveResolverConfig struct {
    MaxRecursionDepth     int           // Default: 50
    MaxTypeAliasDepth     int           // Default: 20
    ConstraintCacheSize   int           // Default: 1000
    ResolutionTimeout     time.Duration // Default: 30s
    EnableCircularCheck   bool          // Default: true
    EnableConstraintCache bool          // Default: true
    EnablePerformanceTrack bool         // Default: true
    DebugMode            bool          // Default: false
}
```

### GenericFieldMapperConfig

Existing configuration with performance enhancements:

```go
type GenericFieldMapperConfig struct {
    EnableCaching        bool          // Enhanced caching
    MaxCacheSize         int           // Larger cache for complex types
    EnableOptimization   bool          // Enhanced optimization
    MappingTimeout       time.Duration // Timeout for complex mappings
    EnableTypeValidation bool          // Enhanced validation
    DebugMode            bool          // Debug logging
    PerformanceMode      bool          // Performance optimizations
}
```

## Performance Optimizations

### 1. Intelligent Detection

The system uses heuristics to detect when recursive resolution is needed:

```go
func (gfm *GenericFieldMapper) isDeeplyNestedGeneric(typ domain.Type) bool {
    // Check bracket depth for nested generics like Map[K, List[V]]
    // Check for known complex patterns
    // Avoid unnecessary recursive resolution for simple types
}
```

### 2. Caching Strategy

- **Type substitution caching**: Avoids redundant substitutions
- **Resolution result caching**: Caches complex resolution results
- **Constraint validation caching**: Caches constraint checks

### 3. Circular Reference Detection

Prevents infinite recursion with comprehensive cycle detection:

```go
func (rtr *RecursiveTypeResolver) checkCircularReference(typ domain.Type, depth int) error {
    // Track resolution stack
    // Detect circular dependencies
    // Provide clear error messages
}
```

## Testing

### Unit Tests

- `TestGenericFieldMapper_EnhancedNestedGenerics`: Tests enhanced functionality
- `TestGenericFieldMapper_TypeAliasSupport`: Tests type alias registration
- `TestGenericFieldMapper_RecursiveTypeResolution`: Tests recursive scenarios
- `TestGenericFieldMapper_Performance`: Tests performance with large/deep structures

### Integration Tests

- `TestNestedGenericFieldMapping_MapListConversion`: Tests specific Map[string, List[T]] → Map[string, Array[U]] conversion
- `TestRecursiveTypeParameterResolution`: Tests recursive resolution capabilities
- Complex real-world scenarios with multiple nesting levels

## Metrics and Monitoring

### RecursiveResolutionMetrics

```go
type RecursiveResolutionMetrics struct {
    TotalResolutions          int64         // Total resolution attempts
    SuccessfulResolutions     int64         // Successful resolutions
    FailedResolutions         int64         // Failed resolutions
    AverageResolutionTime     time.Duration // Performance tracking
    MaxRecursionDepthReached  int           // Complexity tracking
    CircularReferencesDetected int64        // Safety tracking
    TypeAliasResolutions      int64         // Alias usage
    ConstraintValidations     int64         // Constraint checks
    CacheHits                 int64         // Cache efficiency
    CacheMisses               int64         // Cache efficiency
}
```

### Enhanced Field Mapping Metrics

Extended existing metrics with recursive resolution tracking:

```go
// Access recursive metrics
recursiveMetrics := mapper.GetRecursiveResolutionMetrics()
if recursiveMetrics != nil {
    log.Printf("Recursive resolutions: %d", recursiveMetrics.TotalResolutions)
    log.Printf("Average resolution time: %v", recursiveMetrics.AverageResolutionTime)
    log.Printf("Max recursion depth: %d", recursiveMetrics.MaxRecursionDepthReached)
}
```

## Error Handling

### Enhanced Error Types

```go
var (
    ErrRecursiveResolutionFailed  = errors.New("recursive type parameter resolution failed")
    ErrMaxRecursionDepthExceeded  = errors.New("maximum recursion depth exceeded")
    ErrCircularTypeReference      = errors.New("circular type reference detected")
    ErrTypeAliasResolutionFailed  = errors.New("type alias resolution failed")
    ErrConstraintValidationFailed = errors.New("type constraint validation failed")
)
```

### Error Context

All errors provide detailed context:
- Type names and structures involved
- Resolution path taken
- Recursion depth reached
- Constraint validation details

## Future Enhancements

### Planned Improvements

1. **Advanced Constraint Support**: More sophisticated constraint validation
2. **Performance Profiling**: Detailed performance analysis tools  
3. **Visual Type Mapping**: Tools for visualizing complex type mappings
4. **Configuration Templates**: Pre-configured setups for common scenarios
5. **Integration Hooks**: Better integration with the broader convergen pipeline

### Extensibility Points

The enhanced system provides several extension points:
- Custom type compatibility checkers
- Custom constraint validators  
- Custom resolution strategies
- Custom performance optimizers

## Migration Guide

### From Previous Version

Existing GenericFieldMapper usage remains compatible. New features are opt-in:

```go
// Existing usage (still works)
mapper := NewGenericFieldMapper(baseMapper, typeSubstitution, logger, config)
result, err := mapper.MapGenericFields(src, dst, substitutions, options)

// Enhanced usage (new features)
mapper.RegisterTypeAlias("List", sliceType)
metrics := mapper.GetRecursiveResolutionMetrics()
mapper.ClearRecursiveResolutionCache()
```

### Configuration Updates

Update configuration to take advantage of new features:

```go
config := DefaultGenericFieldMapperConfig()
config.EnableOptimization = true
config.PerformanceMode = true
config.DebugMode = true // For development

mapper := NewGenericFieldMapper(baseMapper, typeSubstitution, logger, config)
```

This enhancement provides a robust foundation for handling complex nested generic type conversions while maintaining backward compatibility and performance.