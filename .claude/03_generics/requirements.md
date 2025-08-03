# Generics Support Requirements

## Overview

This document defines the requirements for implementing comprehensive Go generics support in Convergen, enabling users to generate type-safe conversion functions for generic types and interfaces.

## Business Context

### Value Proposition
- **Type Safety**: Eliminate runtime type assertions and enable compile-time type checking
- **Code Reuse**: Single generic interface generates multiple type-specific implementations
- **Developer Experience**: Intuitive syntax that leverages Go's native generics features
- **Performance**: Zero-cost abstractions with compile-time type resolution

### User Stories

**As a Go developer**, I want to:
- Define generic conversion interfaces that work with any type
- Generate type-safe converters for specific type pairs (e.g., `User` ↔ `UserDTO`)
- Use constraint-based generics for specialized conversions (e.g., numeric types only)
- Maintain backward compatibility with existing non-generic code

## Functional Requirements

### FR-1: Generic Interface Parsing
**Priority**: Must Have
**Description**: Parse Go generic interface declarations with type parameters

**Acceptance Criteria**:
- Parse `type Converter[T any] interface { ... }`
- Extract type parameter names, constraints, and positions
- Support multiple type parameters: `type Mapper[T, U any] interface { ... }`
- Handle complex constraints: `[T ~int | ~string, U comparable]`

**Examples**:
```go
//go:generate convergen

// Basic generic interface
type Converter[T any] interface {
    // :recv conv
    Convert(src T) (T, error)
}

// Multiple type parameters
type Mapper[From any, To any] interface {
    // :recv mapper
    Map(src From) (To, error)
}

// Constrained generics
type NumericConverter[T ~int | ~float64] interface {
    // :recv conv
    ConvertToString(src T) (string, error)
}
```

### FR-2: Type Constraint Support
**Priority**: Must Have
**Description**: Support Go's type constraint syntax including unions and underlying types

**Acceptance Criteria**:
- `any` constraint (equivalent to `interface{}`)
- `comparable` constraint for equality operations
- Union constraints: `~int | ~string | ~float64`
- Underlying type constraints: `~string`, `~int`
- Interface constraints: custom interface types

### FR-3: Type Instantiation
**Priority**: Must Have
**Description**: Generate concrete type implementations from generic interfaces

**Acceptance Criteria**:
- Instantiate generic interfaces for specific type combinations
- Generate separate functions for each type pair
- Maintain type safety in generated code
- Support nested generic types: `[]T`, `map[string]T`, `*T`

**Generated Code Example**:
```go
// From: Converter[User] interface { Convert(src User) (User, error) }
// Generated:
func (conv *converter) Convert(src User) (User, error) {
    var dst User
    dst.ID = src.ID
    dst.Name = src.Name
    return dst, nil
}
```

### FR-4: Generic Method Processing
**Priority**: Must Have
**Description**: Process generic method signatures and generate appropriate implementations

**Acceptance Criteria**:
- Extract type parameters from method signatures
- Substitute concrete types in method implementations
- Handle generic return types and parameters
- Support variadic generic parameters: `...T`

### FR-5: Code Generation Enhancement
**Priority**: Must Have
**Description**: Enhance code generation to support generic type substitution

**Acceptance Criteria**:
- Generate type-specific method implementations
- Handle generic slice/map/pointer operations
- Maintain field mapping logic for generic types
- Support all existing annotations on generic methods

### FR-6: Annotation Compatibility
**Priority**: Must Have
**Description**: Ensure all existing annotations work with generic interfaces and methods

**Acceptance Criteria**:
- `:map`, `:conv`, `:skip` work with generic types
- `:style`, `:match` compatible with generic methods
- Type-specific annotations: `:typecast` handles generic constraints
- Error handling annotations work with generic error types

## Non-Functional Requirements

### NFR-1: Performance
- **Compile Time**: Generic code generation adds <10% to compilation time
- **Runtime**: Generated code has zero performance overhead vs hand-written code
- **Memory**: Type instantiation uses efficient caching to minimize memory usage

### NFR-2: Backward Compatibility
- **Existing Code**: All existing non-generic interfaces continue to work unchanged
- **API Stability**: No breaking changes to current annotation syntax
- **Generated Code**: Existing generated code remains valid and functional

### NFR-3: Error Handling
- **Type Errors**: Clear error messages for type constraint violations
- **Syntax Errors**: Helpful feedback for invalid generic syntax
- **Debug Support**: Generated code includes type information for debugging

### NFR-4: Maintainability
- **Code Quality**: Generic support follows established architecture patterns
- **Testing**: Comprehensive test coverage for all generic scenarios
- **Documentation**: Clear examples and usage patterns for developers

## Technical Constraints

### TC-1: Go Version Compatibility
- **Minimum**: Go 1.18+ (generics introduction)
- **Target**: Go 1.21+ (current project minimum)
- **Future**: Compatible with upcoming Go versions

### TC-2: Type System Limitations
- **Go Limitations**: Bound by Go's generic type system capabilities
- **Reflection**: Minimize runtime reflection for type operations
- **Code Size**: Reasonable limits on generated code size

### TC-3: Integration Requirements
- **Build System**: Compatible with `go:generate` and build tools
- **IDE Support**: Generated code works with Go language servers
- **Testing**: Integration with existing testing framework

## Edge Cases and Limitations

### EC-1: Complex Constraints
- **Union Types**: Limited to Go's union constraint syntax
- **Recursive Types**: Support basic recursive generic types
- **Type Inference**: Limited to explicit type instantiation

### EC-2: Nested Generics
- **Generic Fields**: Structs with generic field types
- **Generic Slices**: `[]T` where T is generic
- **Generic Maps**: `map[K]V` with generic key/value types

### EC-3: Method Conflicts
- **Name Collision**: Handle generic method name conflicts
- **Signature Overlap**: Detect and resolve overlapping method signatures
- **Interface Composition**: Generic interfaces embedding other interfaces

## Success Criteria

### Definition of Done
- [ ] Generic interfaces parsed and analyzed correctly
- [ ] Type constraints validated and enforced
- [ ] Code generation produces type-safe implementations
- [ ] All existing functionality remains unchanged
- [ ] Comprehensive test coverage (>90% for new code)
- [ ] Performance benchmarks meet requirements
- [ ] Documentation and examples complete

### Acceptance Tests
1. **Basic Generic Interface**: Simple `Converter[T any]` works end-to-end
2. **Constrained Generics**: `NumericConverter[T ~int | ~float64]` enforces constraints
3. **Multiple Type Params**: `Mapper[From, To any]` generates correct code
4. **Complex Types**: Generic slices, maps, pointers handled correctly
5. **Annotation Compatibility**: All annotations work with generic methods
6. **Error Handling**: Clear error messages for invalid generic syntax

## Future Considerations

### Phase 2 Enhancements
- **Type Inference**: Automatic type parameter inference
- **Generic Composition**: Advanced interface composition patterns
- **Performance Optimization**: Advanced caching and optimization strategies

### Integration Opportunities
- **IDE Integration**: Enhanced editor support for generic annotations
- **Build Tools**: Specialized build tools for large-scale generic code generation
- **Testing Tools**: Generic-aware testing utilities and benchmarks