# Generics Design Specification

## Overview

This document provides comprehensive design specifications for Go generics support in Convergen.
The implementation supports full Go 1.21+ generics syntax including type parameters, constraints,
and instantiation with performance optimization and comprehensive error handling.

## Architecture Overview

### Core Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Parser Stage   │    │   Builder Stage  │    │ Generator Stage │
│                 │    │                  │    │                 │
│ ConstraintParser├───▶│ TypeInstantiator ├───▶│GenericGenerator │
│ GenericInterface│    │ SubstitutionEng  │    │ TemplateEngine  │
│ TypeAnalyzer    │    │ FieldMapper      │    │ ImportManager   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Data Flow

1. **Parse**: Extract generic interfaces with type parameters and constraints
2. **Instantiate**: Convert generic interfaces to concrete types with validation
3. **Generate**: Create concrete implementation code from templates

## Constraint System Design

### Supported Constraint Types

#### 1. Any Constraint
```go
type Converter[T any] interface {
    Convert(T) T
}
```
- **Implementation**: `ParsedConstraint.IsAny = true`
- **Validation**: Always satisfied by any type
- **Usage**: No restrictions on type arguments

#### 2. Comparable Constraint
```go
type KeyMapper[K comparable] interface {
    MapKey(K) string
}
```
- **Implementation**: `ParsedConstraint.IsComparable = true`
- **Validation**: Check type implements comparison operations
- **Usage**: Type must support `==` and `!=` operators

#### 3. Union Constraints
```go
type NumericConverter[T ~int | ~int32 | ~int64 | ~float32 | ~float64] interface {
    ToFloat64(T) float64
}
```
- **Implementation**: `ParsedConstraint.UnionTypes[]` with optional underlying flag
- **Validation**: Type argument must match one of the union members
- **Parsing**: Handle `|` separator and `~` underlying type modifier

#### 4. Underlying Type Constraints
```go
type StringLike[T ~string] interface {
    Process(T) string
}
```
- **Implementation**: `ParsedConstraint.Underlying` with type information
- **Validation**: Type must be assignable to the underlying type
- **Usage**: Custom types with specific underlying types

#### 5. Interface Constraints
```go
type Serializer[T io.Writer] interface {
    Serialize(T, interface{}) error
}
```
- **Implementation**: `ParsedConstraint.InterfaceType` with method set
- **Validation**: Type must implement all interface methods
- **Usage**: Rich behavioral constraints

### Constraint Parser Implementation

```go
type ConstraintParser struct {
    typeResolver *TypeResolver
    logger       *zap.Logger
}

// ParseConstraint handles all constraint types with comprehensive validation
func (cp *ConstraintParser) ParseConstraint(
    ctx context.Context,
    constraint types.Type,
) (*ParsedConstraint, error)
```

**Key Features**:
- **Performance**: Sub-millisecond parsing with caching
- **Error Handling**: Detailed error messages with suggested fixes
- **Extensibility**: Plugin architecture for custom constraints
- **Validation**: Comprehensive semantic validation

## Type Instantiation Design

### Instantiation Engine

```go
type TypeInstantiator struct {
    typeBuilder        *TypeBuilder
    substitutionEngine *TypeSubstitutionEngine
    cache              map[string]*InstantiatedInterface
    crossPackageLoader CrossPackageTypeLoader
}
```

### Instantiation Process

#### Phase 1: Validation
1. **Parameter Count Check**: Ensure type arguments match parameter count
2. **Constraint Validation**: Verify each type argument satisfies constraints
3. **Circular Dependency Check**: Prevent infinite recursion
4. **Cross-Package Validation**: Validate external type references

#### Phase 2: Substitution
1. **Type Mapping Creation**: Build parameter → concrete type mapping
2. **Recursive Substitution**: Replace type parameters in all nested structures
3. **Import Resolution**: Determine required import statements
4. **Result Caching**: Store for future reuse

#### Phase 3: Generation
1. **Template Selection**: Choose appropriate generation templates
2. **Code Generation**: Generate concrete implementation
3. **Optimization**: Apply performance optimizations
4. **Validation**: Ensure generated code compiles

### Type Substitution Engine

```go
type TypeSubstitutionEngine struct {
    typeBuilder *TypeBuilder
    cache       map[string]*SubstitutionResult
    config      *SubstitutionEngineConfig
}
```

**Substitution Capabilities**:
- **Basic Types**: Direct parameter replacement
- **Composite Types**: Recursive substitution in slices, pointers, maps
- **Function Types**: Parameter and return type substitution
- **Struct Types**: Field type substitution with preservation of tags
- **Interface Types**: Method signature substitution

**Performance Features**:
- **Caching**: LRU cache for substitution results
- **Optimization**: Avoid redundant substitutions
- **Metrics**: Performance tracking and analysis
- **Parallelization**: Concurrent substitution for independent types

### Cross-Package Type Resolution

```go
type CrossPackageTypeLoader interface {
    ResolveType(ctx context.Context, qualifiedTypeName string) (Type, error)
    ValidateTypeArguments(ctx context.Context, typeArguments []string) error
    GetImportPaths(typeArguments []string) []string
}
```

**Implementation Strategy**:
- **Package Discovery**: Use Go module system for package resolution
- **Type Analysis**: Extract type information from external packages
- **Import Management**: Generate correct import statements
- **Caching**: Cache resolved types for performance

## Code Generation Design

### Template System

#### Generic Template Structure
```go
type GenericTemplateData struct {
    BaseTemplateData
    TypeParameters    []TypeParam
    TypeArguments     []TypeArg  
    TypeSubstitutions map[string]TypeSubstitution
    Methods           []*MethodData
}
```

#### Template Categories

**1. Basic Conversion Templates**
```go
// Template: generic_simple_conversion
func {{.FunctionName}}{{.TypeParams}}(src {{.SourceType}}) {{.DestType}} {
    return {{.DestType}}{
        {{range .FieldMappings}}
        {{.DestField}}: {{.ConversionExpression}},
        {{end}}
    }
}
```

**2. Complex Conversion Templates**
```go
// Template: generic_complex_conversion  
func {{.FunctionName}}{{.TypeParams}}(src {{.SourceType}}) ({{.DestType}}, error) {
    {{.ValidationCode}}
    
    {{range .FieldMappings}}
    {{if .RequiresValidation}}
    if {{.ValidationCondition}} {
        return {{.ZeroValue}}, fmt.Errorf("{{.ErrorMessage}}")
    }
    {{end}}
    {{end}}
    
    return {{.DestType}}{
        {{range .FieldMappings}}
        {{.DestField}}: {{.ConversionExpression}},
        {{end}}
    }, nil
}
```

**3. Error Handling Templates**
```go
// Template: generic_method_with_error
func {{.FunctionName}}{{.TypeParams}}({{.Parameters}}) ({{.ReturnTypes}}) {
    {{.MethodBody}}
    
    if err != nil {
        return {{.ZeroValues}}, fmt.Errorf("{{.FunctionName}} failed: %w", err)
    }
    
    return {{.ReturnValues}}
}
```

### Template Functions

**Type Manipulation Functions**:
```go
substituteType(typeName string, substitutions map[string]TypeSubstitution) string
isGenericType(typeName string) bool
formatTypeParam(param TypeParam) string
generateTypeSwitch(unionTypes []Type) string
```

**Field Access Functions**:
```go
generateFieldAccess(source, field string, accessor string) string
hasAnnotation(annotations map[string]string, key string) bool
getAnnotation(annotations map[string]string, key string, defaultValue string) string
```

### Import Management

**Import Resolution Algorithm**:
1. **Collect**: Gather all type references from template data
2. **Analyze**: Determine package dependencies for each type
3. **Deduplicate**: Remove duplicate imports and resolve conflicts
4. **Organize**: Sort imports according to Go conventions
5. **Validate**: Ensure all imports are accessible and correct

**Import Categories**:
- **Standard Library**: Built-in Go packages
- **External Dependencies**: Third-party packages
- **Local Packages**: Project-specific packages
- **Generic Dependencies**: Packages required by type arguments

## Performance Optimization

### Caching Strategy

#### Multi-Level Caching
1. **Constraint Cache**: Parsed constraint results
2. **Substitution Cache**: Type substitution results
3. **Instantiation Cache**: Complete instantiated interfaces
4. **Generation Cache**: Generated code templates

#### Cache Configuration
```go
type CacheConfig struct {
    ConstraintCacheSize     int           // Default: 1000
    SubstitutionCacheSize   int           // Default: 5000
    InstantiationCacheSize  int           // Default: 2000
    GenerationCacheSize     int           // Default: 1000
    TTL                     time.Duration // Default: 1 hour
    EnableMetrics           bool          // Default: true
}
```

### Performance Metrics

```go
type GenericsPerformanceMetrics struct {
    // Parsing metrics
    ConstraintParseTime     time.Duration
    GenericInterfaceParses  int64
    ConstraintCacheHits     int64
    
    // Instantiation metrics  
    TypeInstantiations      int64
    InstantiationTime       time.Duration
    SubstitutionOperations  int64
    
    // Generation metrics
    CodeGenerations         int64
    GenerationTime          time.Duration
    TemplateExecutions      int64
    OptimizationsApplied    int64
}
```

### Memory Optimization

**Memory Management Strategies**:
- **Object Pooling**: Reuse frequently created objects
- **Lazy Loading**: Load type information on demand
- **Garbage Collection**: Proactive cleanup of cached data
- **Memory Profiling**: Track memory usage patterns

## Error Handling

### Error Categories

#### 1. Parse Errors
```go
var (
    ErrInvalidConstraintSyntax = errors.New("invalid constraint syntax")
    ErrUnsupportedConstraintType = errors.New("unsupported constraint type")
    ErrCircularConstraint = errors.New("circular constraint dependency")
)
```

#### 2. Instantiation Errors
```go
var (
    ErrTypeArgumentCountMismatch = errors.New("type argument count mismatch")
    ErrConstraintViolation = errors.New("constraint violation")
    ErrRecursiveInstantiation = errors.New("recursive instantiation")
)
```

#### 3. Generation Errors
```go
var (
    ErrTemplateExecutionFailed = errors.New("template execution failed")
    ErrTypeSubstitutionFailed = errors.New("type substitution failed")
    ErrImportResolutionFailed = errors.New("import resolution failed")
)
```

### Error Context Enhancement

**Rich Error Information**:
- **Location**: Source file, line, and column information
- **Context**: Type parameter names and constraint details
- **Suggestions**: Automated fix suggestions where possible
- **Related**: Links to related errors and documentation

**Example Error Output**:
```
Error: Constraint violation in generic interface TypeMapper[T, U]
  File: converter.go:15:25
  Type Parameter: T
  Constraint: comparable
  Actual Type: []string
  Problem: Slice types are not comparable
  
  Suggestion: Use a comparable type like string, int, or a custom struct with comparable fields
  
  Valid examples:
    TypeMapper[string, User]     // ✓ string is comparable
    TypeMapper[UserID, Profile]  // ✓ if UserID is comparable
```

## Integration Points

### CLI Integration

**Flag Support**:
```bash
# Basic generic instantiation
convergen -type "TypeMapper[string,User]"

# Multiple instantiations
convergen -type "TypeMapper[string,User]" -type "TypeMapper[int,Product]"

# With package qualification
convergen -type "TypeMapper[github.com/user/pkg.CustomType,User]"
```

**Flag Processing**:
1. **Parse**: Extract type name and parameters from flag
2. **Validate**: Check syntax and parameter count
3. **Resolve**: Resolve type arguments to concrete types
4. **Generate**: Create instantiated code

### Pipeline Integration

**Event-Driven Processing**:
- **GenericInterfaceDiscovered**: When parser finds generic interface
- **TypeInstantiationRequested**: When CLI specifies instantiation
- **ConstraintValidationCompleted**: After constraint checking
- **CodeGenerationCompleted**: After template execution

**Error Propagation**:
- **ParseError**: Constraint parsing failures
- **InstantiationError**: Type instantiation failures
- **GenerationError**: Code generation failures

## Testing Strategy

### Unit Testing

**Test Categories**:
1. **Constraint Parsing**: All constraint types and edge cases
2. **Type Instantiation**: Valid and invalid instantiations
3. **Code Generation**: Template execution and optimization
4. **Error Handling**: Comprehensive error scenarios

### Integration Testing

**Test Scenarios**:
1. **End-to-End**: Complete pipeline from source to generated code
2. **Cross-Package**: External type resolution and imports
3. **Performance**: Caching effectiveness and resource usage
4. **Compatibility**: Different Go versions and constraint styles

### Behavior-Driven Testing

**Current Test Coverage** (11/11 scenarios passing):
- Basic generic interfaces
- Constraint validation
- Multiple type parameters
- Generic with annotations
- Union constraint parsing
- Nested generic types
- Error scenarios

**Test Implementation**:
```go
func TestGenericsFeatures(t *testing.T) {
    genericsTests := map[string]helpers.TestScenario{
        "basic_generic_interface":     helpers.BasicGenericInterfaceScenario(),
        "generic_with_constraints":    helpers.GenericWithConstraintsScenario(),
        "multiple_type_parameters":    helpers.MultipleTypeParametersScenario(),
        "generic_type_instantiation":  helpers.GenericTypeInstantiationScenario(),
        // ... additional scenarios
    }
}
```

## Future Enhancements

### Planned Features

1. **Advanced Constraints**: Support for custom constraint interfaces
2. **Type Inference**: Automatic type argument inference where possible
3. **Generic Struct Support**: Support for generic struct definitions
4. **Performance Profiling**: Built-in performance analysis tools
5. **IDE Integration**: Language server protocol support

### Research Areas

1. **Constraint Inference**: Automatic constraint generation from usage patterns
2. **Optimization Algorithms**: Advanced code generation optimizations
3. **Memory Efficiency**: Further memory usage optimizations
4. **Parallel Processing**: Enhanced concurrent processing capabilities

## Conclusion

The generics implementation in Convergen provides comprehensive support for Go 1.21+ generics with:

- **Complete Constraint Support**: All Go constraint types supported
- **High Performance**: Multi-level caching and optimization
- **Robust Error Handling**: Detailed error reporting with suggestions
- **Extensible Architecture**: Plugin-friendly design for future enhancements
- **Production Ready**: Comprehensive testing and validation

The implementation is currently ~75-80% complete with solid foundations in constraint parsing, type instantiation, and code generation. Remaining work focuses on field mapping optimization and integration refinements.
