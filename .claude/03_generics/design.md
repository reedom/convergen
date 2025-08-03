# Generics Support Design Document

## Architecture Overview

This document outlines the technical design for implementing comprehensive Go generics support in Convergen, building upon the existing type system and code generation pipeline.

## Current State Analysis

### Existing Infrastructure ✅
- **Domain Types**: `domain.TypeParam`, `domain.GenericType`, `KindGeneric`
- **Type Resolution**: Basic type parameter parsing in `type_resolver.go`
- **Parser Foundation**: AST parsing with `go/types` integration
- **Code Generation**: Template-based generation system

### Missing Components ❌
- **Interface Type Parameter Extraction**: Generic interface parsing
- **Constraint Resolution**: Complex constraint parsing and validation
- **Type Instantiation**: Converting generic types to concrete implementations
- **Generic Method Processing**: Method signature type substitution
- **Code Generation Enhancement**: Generic-aware template rendering

## Design Principles

1. **Incremental Enhancement**: Build on existing architecture without breaking changes
2. **Type Safety**: Compile-time type checking and validation
3. **Zero-Cost Abstractions**: No runtime overhead for generic code
4. **Backward Compatibility**: All existing functionality remains unchanged
5. **Maintainable Code**: Clear separation of concerns and well-tested components

## Component Design

### 1. Enhanced Type System (`pkg/domain/`)

#### 1.1 Enhanced TypeParam Structure
```go
// Enhanced TypeParam with comprehensive constraint support
type TypeParam struct {
    Name        string              `json:"name"`
    Constraint  Type               `json:"constraint"`
    Index       int                `json:"index"`
    
    // Enhanced constraint support
    UnionTypes  []Type             `json:"union_types,omitempty"`     // T ~int | ~string
    IsComparable bool              `json:"comparable,omitempty"`      // T comparable
    Underlying  *UnderlyingConstraint `json:"underlying,omitempty"`   // T ~string
    IsAny       bool               `json:"any,omitempty"`             // T any
}

// UnderlyingConstraint represents underlying type constraints (~string, ~int)
type UnderlyingConstraint struct {
    Type      Type   `json:"type"`
    Package   string `json:"package,omitempty"`
}
```

#### 1.2 Generic Interface Support
```go
// Enhanced InterfaceInfo with type parameters
type InterfaceInfo struct {
    Object      types.Object       // Existing
    Interface   *types.Interface   // Existing  
    Methods     []types.Object     // Existing
    Options     *InterfaceOptions  // Existing
    Annotations []*Annotation      // Existing
    Marker      string            // Existing
    Position    token.Pos         // Existing
    
    // NEW: Generic support
    TypeParams   []TypeParam       `json:"type_params"`
    IsGeneric    bool             `json:"is_generic"`
    Instantiations map[string]*InstantiatedInterface `json:"instantiations,omitempty"`
}

// InstantiatedInterface represents a concrete instantiation of a generic interface
type InstantiatedInterface struct {
    GenericInterface *InterfaceInfo
    TypeArguments    []Type
    ConcreteTypes    map[string]Type  // Type parameter name -> concrete type
    Methods          []*Method        // Instantiated methods
    GeneratedCode    string          // Generated implementation
}
```

#### 1.3 Type Instantiation Engine
```go
// TypeInstantiator handles generic type instantiation
type TypeInstantiator struct {
    typeBuilder  *TypeBuilder
    cache        map[string]*InstantiatedInterface
    logger       *zap.Logger
}

// InstantiateInterface creates a concrete interface from generic + type arguments
func (ti *TypeInstantiator) InstantiateInterface(
    genericInterface *InterfaceInfo,
    typeArgs []Type,
) (*InstantiatedInterface, error)

// SubstituteType replaces type parameters with concrete types in a given type
func (ti *TypeInstantiator) SubstituteType(
    genericType Type,
    typeParams []TypeParam,
    typeArgs []Type,
) (Type, error)
```

### 2. Enhanced Parser (`pkg/parser/`)

#### 2.1 Interface Analysis Enhancement
```go
// Enhanced analyzeInterface to extract type parameters
func (p *ASTParser) analyzeInterface(
    ctx context.Context,
    pkg *packages.Package,
    file *ast.File,
    obj types.Object,
    iface *types.Interface,
) (*InterfaceInfo, error) {
    // Extract type parameters from named interface types
    typeParams, err := p.extractInterfaceTypeParams(ctx, obj)
    if err != nil {
        return nil, fmt.Errorf("failed to extract type parameters: %w", err)
    }
    
    // Existing logic...
    interfaceInfo := &InterfaceInfo{
        // ... existing fields
        TypeParams: typeParams,
        IsGeneric:  len(typeParams) > 0,
        Instantiations: make(map[string]*InstantiatedInterface),
    }
    
    return interfaceInfo, nil
}

// extractInterfaceTypeParams extracts type parameters from interface declarations
func (p *ASTParser) extractInterfaceTypeParams(
    ctx context.Context,
    obj types.Object,
) ([]TypeParam, error)
```

#### 2.2 Constraint Parser
```go
// ConstraintParser handles parsing of Go type constraints
type ConstraintParser struct {
    typeResolver *TypeResolver
    logger       *zap.Logger
}

// ParseConstraint parses constraint expressions like "~int | ~string | comparable"
func (cp *ConstraintParser) ParseConstraint(
    ctx context.Context,
    constraint types.Type,
) (*ParsedConstraint, error)

// ParsedConstraint represents a fully parsed type constraint
type ParsedConstraint struct {
    Type         Type
    IsComparable bool
    UnionTypes   []Type
    Underlying   *UnderlyingConstraint
    IsAny        bool
}
```

#### 2.3 Generic Method Processor
```go
// Enhanced method processing for generic methods
func (p *ASTParser) processGenericMethod(
    ctx context.Context,
    pkg *packages.Package,
    file *ast.File,
    methodObj types.Object,
    interfaceTypeParams []TypeParam,
    options *domain.InterfaceOptions,
) (*domain.Method, error) {
    // Extract method signature with type parameter substitution
    signature, err := p.extractGenericMethodSignature(methodObj, interfaceTypeParams)
    if err != nil {
        return nil, fmt.Errorf("failed to extract generic method signature: %w", err)
    }
    
    // Process method with generic context
    return p.buildGenericMethod(ctx, signature, interfaceTypeParams, options)
}
```

### 3. Code Generation Enhancement (`pkg/generator/`, `pkg/emitter/`)

#### 3.1 Generic Template System
```go
// GenericCodeGenerator handles code generation for generic interfaces
type GenericCodeGenerator struct {
    templateEngine    *TemplateEngine
    typeInstantiator *TypeInstantiator
    fieldMapper      *FieldMapper
    logger           *zap.Logger
}

// GenerateGenericImplementation generates concrete implementations for generic interfaces
func (gcg *GenericCodeGenerator) GenerateGenericImplementation(
    ctx context.Context,
    instantiatedInterface *InstantiatedInterface,
) (string, error)
```

#### 3.2 Type-Aware Template Rendering
```go
// Enhanced template data with generic type information
type GenericTemplateData struct {
    *TemplateData                    // Existing template data
    
    // Generic-specific data
    TypeParameters    []TypeParam    `json:"type_parameters"`
    TypeArguments     []Type        `json:"type_arguments"`
    TypeSubstitutions map[string]Type `json:"type_substitutions"`
    IsGeneric         bool          `json:"is_generic"`
}

// Template functions for generic code generation
var GenericTemplateFuncs = template.FuncMap{
    "substituteType":     substituteTypeInTemplate,
    "isGenericType":      isGenericTypeInTemplate,
    "formatTypeParam":    formatTypeParamInTemplate,
    "generateTypeSwitch": generateTypeSwitchInTemplate,
}
```

#### 3.3 Generic Field Mapping
```go
// Enhanced field mapping for generic types
func (fm *FieldMapper) MapGenericFields(
    srcType Type,
    dstType Type,
    typeSubstitutions map[string]Type,
    options *FieldMappingOptions,
) (*FieldMapping, error) {
    // Substitute generic types with concrete types
    concreteSrcType := fm.substituteGenerics(srcType, typeSubstitutions)
    concreteDstType := fm.substituteGenerics(dstType, typeSubstitutions)
    
    // Use existing field mapping logic with concrete types
    return fm.MapFields(concreteSrcType, concreteDstType, options)
}
```

## Data Flow Design

### 1. Generic Interface Processing Pipeline
```
Generic Interface Declaration
         ↓
Interface Analysis (extract TypeParams)
         ↓
Constraint Parsing & Validation
         ↓
Method Processing (with type context)
         ↓
Type Instantiation (for concrete types)
         ↓
Code Generation (type-specific implementations)
```

### 2. Type Parameter Resolution Flow
```
Go AST Type Parameter Node
         ↓
types.TypeParam Extraction
         ↓
Constraint Type Resolution
         ↓
Domain TypeParam Creation
         ↓
Validation & Caching
```

### 3. Code Generation Flow
```
Generic Interface + Type Arguments
         ↓
Type Instantiation
         ↓
Method Signature Substitution
         ↓
Field Mapping (with concrete types)
         ↓
Template Rendering
         ↓
Generated Implementation Code
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Goal**: Basic generic interface parsing and type parameter extraction

**Components**:
- Enhanced `domain.TypeParam` structure
- Interface type parameter extraction in `analyzeInterface`
- Basic constraint parsing (any, comparable)
- Updated `InterfaceInfo` with generic support

**Deliverables**:
- Generic interfaces can be parsed and analyzed
- Type parameters are extracted and stored
- Basic constraints are validated
- Existing functionality remains unchanged

### Phase 2: Type Instantiation (Week 3-4)
**Goal**: Type instantiation engine and method processing

**Components**:
- `TypeInstantiator` implementation
- Generic method signature processing
- Type substitution algorithms
- Instantiated interface caching

**Deliverables**:
- Generic interfaces can be instantiated with concrete types
- Method signatures are correctly substituted
- Type safety is maintained throughout the process

### Phase 3: Code Generation (Week 5-6)
**Goal**: Generate concrete implementations from generic templates

**Components**:
- Generic template system
- Enhanced field mapping for generic types
- Type-aware code generation
- Integration with existing emitter

**Deliverables**:
- Complete generic-to-concrete code generation
- All existing annotations work with generic methods
- Generated code is type-safe and performant

### Phase 4: Advanced Features (Week 7-8)
**Goal**: Complex constraints and optimization

**Components**:
- Union constraint support (`T ~int | ~string`)
- Underlying type constraints (`T ~string`)
- Performance optimization and caching
- Comprehensive testing and documentation

**Deliverables**:
- Full constraint system support
- Performance meets requirements (NFR-1)
- Complete test coverage and documentation

## Error Handling Strategy

### 1. Parse-Time Errors
- **Invalid Syntax**: Clear error messages for malformed generic syntax
- **Constraint Violations**: Detailed feedback on constraint mismatches
- **Type Parameter Conflicts**: Detection and reporting of name collisions

### 2. Type Instantiation Errors
- **Constraint Violations**: Runtime checking of type argument compatibility
- **Circular Dependencies**: Detection of circular generic type references
- **Overflow Protection**: Limits on recursive generic instantiation

### 3. Code Generation Errors
- **Template Errors**: Robust error handling in template rendering
- **Type Substitution Failures**: Graceful handling of substitution edge cases
- **Integration Errors**: Clear error propagation through the pipeline

## Performance Considerations

### 1. Compilation Performance
- **Caching Strategy**: Aggressive caching of parsed type parameters and instantiated interfaces
- **Parallel Processing**: Concurrent processing of independent generic instantiations
- **Memory Management**: Efficient memory usage for type parameter storage

### 2. Runtime Performance
- **Zero-Cost Abstractions**: Generated code has no runtime overhead vs hand-written
- **Code Size Optimization**: Minimize generated code duplication
- **Template Optimization**: Efficient template rendering and caching

### 3. Memory Usage
- **Type Parameter Caching**: LRU cache for frequently used type instantiations
- **Garbage Collection**: Proper cleanup of temporary type objects
- **Memory Profiling**: Built-in profiling for memory usage analysis

## Testing Strategy

### 1. Unit Testing
- **Component Testing**: Each component tested in isolation with mock dependencies
- **Type Parameter Parsing**: Comprehensive test cases for all constraint types
- **Type Instantiation**: Edge cases and error conditions thoroughly tested

### 2. Integration Testing
- **End-to-End Scenarios**: Complete generic interface to generated code workflows
- **Backward Compatibility**: All existing tests continue to pass unchanged
- **Performance Benchmarks**: Automated performance regression testing

### 3. Property-Based Testing
- **Generic Type Generation**: Property-based testing for type parameter combinations
- **Constraint Validation**: Automated testing of constraint satisfaction
- **Code Generation Equivalence**: Verify generated code matches expected semantics

## Migration and Compatibility

### 1. Backward Compatibility
- **Existing Interfaces**: All non-generic interfaces work exactly as before
- **Generated Code**: Existing generated code remains valid and functional
- **API Stability**: No breaking changes to annotation syntax or configuration

### 2. Migration Path
- **Incremental Adoption**: Users can adopt generics gradually
- **Mixed Codebases**: Generic and non-generic interfaces can coexist
- **Tooling Support**: Clear migration guides and tooling assistance

### 3. Version Strategy
- **Feature Flags**: Optional generic support with feature toggles
- **Deprecation Policy**: Clear communication about future changes
- **Long-term Support**: Commitment to maintaining backward compatibility