# Parser Package Requirements

This document outlines the requirements for the `pkg/parser` package, which is responsible for analyzing Go source code and transforming it into domain models for the generation pipeline.

## Functional Requirements

### Interface Discovery and Analysis

*   **REQ-1: Interface Discovery**: MUST locate and parse interfaces named `Convergen` or annotated with `// :convergen`
*   **REQ-2: Generic Interface Support**: MUST handle generic interfaces with type parameters and constraints
*   **REQ-3: Multiple Interface Support**: MUST process multiple converter interfaces within a single source file
*   **REQ-4: Package-Level Analysis**: MUST analyze imports and package-level type definitions

### Method Signature Processing

*   **REQ-5: Method Parsing**: MUST extract method signatures including names, parameters, and return types
*   **REQ-6: Generic Method Support**: MUST handle methods with generic type parameters
*   **REQ-7: Complex Type Resolution**: MUST resolve complex types including:
    *   Nested structs and embedded types
    *   Generic types with type parameters
    *   Interface types and implementations
    *   Pointer and slice types
    *   Map types with complex key/value types

### Annotation Processing System

*   **REQ-8: Extensible Annotation Parsing**: MUST support a registry-based annotation system allowing:
    *   Registration of new annotation processors
    *   Validation of annotation syntax and semantics
    *   Composition of multiple annotations per method
*   **REQ-9: Standard Annotation Support**: MUST parse all existing annotations:
    *   `:match`, `:style`, `:recv`, `:reverse`
    *   `:case`, `:getter`, `:stringer`, `:typecast`
    *   `:skip`, `:map`, `:conv`, `:literal`
    *   `:preprocess`, `:postprocess`
*   **REQ-10: Annotation Validation**: MUST validate annotation parameters and detect conflicts

### Type System Integration

*   **REQ-11: Comprehensive Type Resolution**: MUST resolve all referenced types with full type information including:
    *   Type identity and underlying types  
    *   Generic type instantiation
    *   Method sets and interface satisfaction
    *   Type constraints and bounds
*   **REQ-12: Cross-Package Type Support**: MUST resolve types across package boundaries
*   **REQ-13: Type Caching**: MUST cache resolved type information for performance

### Source Code Processing

*   **REQ-14: Field Order Preservation**: MUST capture and preserve exact field declaration order from source structs
*   **REQ-15: Source Location Tracking**: MUST maintain source file locations for error reporting
*   **REQ-16: Comment Association**: MUST correctly associate comments with their target elements
*   **REQ-17: Base Code Generation**: MUST produce clean source code with converter interfaces removed

### Error Handling and Validation

*   **REQ-18: Rich Error Context**: MUST provide detailed error messages with:
    *   Source file locations (line, column)
    *   Context about what was being parsed
    *   Suggestions for fixing common errors
*   **REQ-19: Error Aggregation**: MUST collect multiple parsing errors and report them together
*   **REQ-20: Validation Integration**: MUST validate parsed models for consistency and correctness

## Event Integration Requirements

*   **REQ-21: Event Emission**: MUST emit `ParseEvent` with parsed domain models and context
*   **REQ-22: Context Propagation**: MUST accept and propagate context.Context throughout parsing
*   **REQ-23: Cancellation Support**: MUST respect context cancellation during long-running parsing operations
*   **REQ-24: Progress Reporting**: MUST emit progress events for large source files

## Performance Requirements

*   **REQ-25: Concurrent Processing**: MUST support concurrent parsing of multiple interfaces/methods where possible
*   **REQ-26: Memory Efficiency**: MUST minimize memory usage during AST processing
*   **REQ-27: Incremental Parsing**: MUST support incremental re-parsing of modified source regions
*   **REQ-28: Parse Caching**: MUST cache parsing results to avoid redundant work

## Non-Functional Requirements

*   **REQ-29: AST Compatibility**: MUST work with standard Go AST packages and toolchain
*   **REQ-30: Go Version Support**: MUST support Go 1.21+ features including generics
*   **REQ-31: Large File Handling**: MUST handle source files with thousands of fields efficiently
*   **REQ-32: Thread Safety**: All parsing operations MUST be thread-safe for concurrent use
