# Requirements

## Scope
- Context: Go code generator that creates type-to-type copy functions from annotated interfaces
- Out of Scope: Runtime type conversion, reflection-based solutions, cross-language code generation

## Stakeholders & Goals
- Go Developers: Eliminate manual boilerplate for type conversion functions with compile-time safety
- API Layer Engineers: Seamless conversion between domain models and DTOs
- Performance Engineers: Zero-cost abstractions with generated code as efficient as hand-written
- Maintainability Engineers: Clear, readable generated code that follows Go conventions

## Constraints
- Go 1.21+ compatibility requirement (generics support)
- Zero runtime dependencies for generated code
- Deterministic output across multiple runs
- Memory usage bounded during generation process
- Generation time <10 seconds for codebases with 100+ interfaces

## Functional Requirements (EARS)

### Core Processing Pipeline
- REQ-1.1: When a Go source file contains interfaces named `Convergen` or marked with `:convergen` annotation, the system shall discover and parse these interfaces in order to identify conversion specifications.
- REQ-1.2: When method signatures are encountered within conversion interfaces, the system shall extract source and destination type information including generic type parameters.
- REQ-1.3: When comment annotations are present on interface methods, the system shall parse and validate all annotation types (`:match`, `:map`, `:conv`, `:skip`, `:typecast`, `:stringer`, `:recv`, `:style`) in order to configure conversion behavior.
- REQ-1.4: When complex types including generics are referenced in method signatures, the system shall resolve full type information with constraints and underlying types.
- REQ-1.5: When all parsing is complete, the system shall generate idiomatic Go conversion functions that compile without errors.

### Type System Support
- REQ-2.1: Where generic types are encountered, the system shall support type parameters, constraints, and instantiation for concrete types.
- REQ-2.2: Where primitive types require conversion, the system shall handle direct assignment, type casting, and string conversion as appropriate.
- REQ-2.3: Where struct types are processed, the system shall support nested structs, pointers, slices, maps, and interface implementations.
- REQ-2.4: Where cross-package types are referenced, the system shall resolve types from imported packages and generate proper import statements.

### Field Mapping Strategies
- REQ-3.1: When no explicit mapping is provided, the system shall match fields by name between source and destination types.
- REQ-3.2: When `:map` annotations specify explicit field mappings, the system shall use these mappings instead of name matching.
- REQ-3.3: When `:conv` annotations specify custom converter functions, the system shall apply these functions to the appropriate fields.
- REQ-3.4: When `:skip` annotations specify field patterns, the system shall exclude matching destination fields from assignment.
- REQ-3.5: When `:typecast` annotation is present, the system shall allow type casting between compatible types.
- REQ-3.6: When `:stringer` annotation is present, the system shall use String() methods for field conversion.

### Code Generation Behavior
- REQ-4.1: Where simple field assignments are sufficient, the system shall generate struct literal syntax by default.
- REQ-4.2: Where complex processing is required (error handling, preprocessing), the system shall automatically fall back to assignment block style.
- REQ-4.3: When `:no-struct-literal` annotation is present, the system shall override default behavior and use assignment blocks.
- REQ-4.4: When `:recv` annotation specifies receiver variables, the system shall generate receiver methods instead of standalone functions.
- REQ-4.5: When `:style` annotation specifies function signature style, the system shall generate either argument-style or return-style functions.

### Error Handling Integration
- REQ-5.1: When converter functions return errors, the system shall propagate errors appropriately in generated code.
- REQ-5.2: When type conversion operations may fail, the system shall include proper error checking and propagation.
- REQ-5.3: When parsing or generation errors occur, the system shall provide clear error messages with file positions and context.
- REQ-5.4: When annotation validation fails, the system shall report specific annotation errors with suggested corrections.

### Concurrency and Performance
- REQ-6.1: Where multiple methods are processed simultaneously, the system shall utilize concurrent processing while maintaining deterministic output ordering.
- REQ-6.2: Where struct fields can be processed independently, the system shall process fields concurrently and assemble results in source order.
- REQ-6.3: Where caching opportunities exist, the system shall cache type resolution results to improve performance.
- REQ-6.4: When processing large interfaces, the system shall bound resource usage to prevent memory exhaustion.

### Output Stability and Reproducibility
- REQ-7.1: When identical input is processed multiple times, the system shall produce identical output code.
- REQ-7.2: When field ordering is present in source structs, the system shall preserve this ordering in generated assignments.
- REQ-7.3: When import statements are required, the system shall organize imports consistently and remove unused imports.
- REQ-7.4: When generated code is formatted, the system shall produce code that passes `go fmt` without changes.

### Integration and Build System
- REQ-8.1: When invoked via `go:generate` directives, the system shall integrate seamlessly with Go build tools.
- REQ-8.2: When CLI flags are provided, the system shall support global configuration options that override annotation behavior.
- REQ-8.3: When context cancellation is requested, the system shall respect context timeouts and cancellation throughout the pipeline.
- REQ-8.4: When multiple source files are processed, the system shall handle dependencies and cross-file type references correctly.

## Non-Functional Requirements

### Performance
- Compilation impact: Generated code compilation time increase <10% compared to hand-written equivalents
- Generation speed: Process 100+ interfaces in <10 seconds on modern hardware
- Memory efficiency: Peak memory usage during generation <500MB for large codebases
- CPU utilization: Leverage multiple CPU cores effectively for concurrent processing

### Reliability
- Error recovery: Graceful handling of malformed input with actionable error messages
- Resource cleanup: Proper cleanup of temporary resources and goroutines
- Deterministic behavior: Consistent output across different execution environments
- Fault tolerance: Continue processing valid interfaces when individual interfaces fail

### Security
- Input validation: Validate all annotation parameters to prevent code injection
- Safe code generation: Generated code follows secure coding practices
- Dependency management: No introduction of vulnerable dependencies
- File system safety: Proper path validation and access controls

### Observability
- Progress tracking: Report progress for long-running generation tasks
- Performance metrics: Track generation time and resource usage
- Error context: Rich error messages with file positions and stack traces
- Debug support: Optional verbose output for troubleshooting
