# Convergen Rewrite Requirements

This document outlines the requirements for the complete rewrite of Convergen. The goal is to create a modern, fine-grained architecture that is effective, easy to maintain and extend, and runs fast.

## Core Design Goals

*   **Performance**: Utilize concurrent processing for field-level operations while maintaining stable output ordering
*   **Maintainability**: Clean, minimal architecture with clear separation of concerns
*   **Extensibility**: Easy addition of new annotation types and conversion strategies
*   **Robustness**: Comprehensive error handling and cancellation support
*   **Stability**: Deterministic, reproducible output generation

## Functional Requirements

### Core Generation Features

*   **REQ-1: Interface Discovery**: MUST find and parse interfaces named `Convergen` or annotated with `// :convergen`
*   **REQ-2: Method Parsing**: MUST parse method signatures and extract source/destination type information
*   **REQ-3: Annotation Processing**: MUST support all existing annotations with extensible annotation system:
    *   `:match <name|none>`: Field matching algorithm
    *   `:style <return|arg>`: Destination variable style  
    *   `:recv <var>`: Receiver specification
    *   `:reverse`: Copy direction reversal
    *   `:case` / `:case:off`: Case sensitivity toggle
    *   `:getter` / `:getter:off`: Getter method usage
    *   `:stringer` / `:stringer:off`: String() method usage
    *   `:typecast` / `:typecast:off`: Type casting toggle
    *   `:skip <pattern>`: Field skipping with regex support
    *   `:map <src> <dst>`: Explicit field mapping
    *   `:conv <func> <src> [dst]`: Custom converter functions
    *   `:literal <dst> <literal>`: Literal value assignment
    *   `:preprocess <func>`: Pre-processing functions
    *   `:postprocess <func>`: Post-processing functions
*   **REQ-4: Type Resolution**: MUST resolve all types, including generics, with full type information
*   **REQ-5: Code Generation**: MUST generate idiomatic Go code with stable output ordering

### Output Generation Requirements

*   **REQ-6: Stable Output**: Generated code MUST be deterministic and reproducible across runs
*   **REQ-7: Field Ordering**: MUST preserve exact source code field order in generated assignments
*   **REQ-8: Construction Style**: MUST support both composite literal (`Type{Field: value}`) and assignment block (`v.Field = value`) styles based on complexity
*   **REQ-9: Error Handling**: Generated code MUST include proper error handling and propagation

### Concurrency Requirements

*   **REQ-10: Field-Level Concurrency**: MUST process struct fields concurrently to improve performance
*   **REQ-11: Order Preservation**: Concurrent processing MUST maintain deterministic output ordering
*   **REQ-12: Resource Management**: MUST limit goroutine usage and memory consumption

### Modern Go Features

*   **REQ-13: Context Support**: MUST accept context.Context for cancellation and timeouts
*   **REQ-14: Generics Usage**: MUST utilize generics where appropriate for type safety and performance
*   **REQ-15: Error Wrapping**: MUST use fmt.Errorf and error wrapping for better error context
*   **REQ-16: Structured Logging**: MUST use zap for structured, efficient logging

## Non-Functional Requirements

*   **REQ-17: Performance**: Field processing MUST be parallelizable with minimal overhead
*   **REQ-18: Memory Efficiency**: MUST minimize memory allocations during processing
*   **REQ-19: Extensibility**: New annotation types MUST be addable without modifying core logic
*   **REQ-20: Testability**: All components MUST be unit testable with clear interfaces
*   **REQ-21: Maintainability**: Code MUST follow clean architecture principles with minimal coupling

## Architecture Requirements

*   **REQ-22: Event-Driven Pipeline**: Internal processing MUST use event-driven architecture for coordination
*   **REQ-23: Domain Separation**: Clear separation between parsing, planning, execution, and emission phases
*   **REQ-24: Interface Segregation**: Components MUST depend only on interfaces they actually use
*   **REQ-25: Dependency Injection**: Components MUST be configurable and testable through dependency injection
