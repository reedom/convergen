# Struct Literal Output Requirements

## 1. Overview

This specification defines the requirements for implementing struct literal output as the default behavior in Convergen's code generation pipeline, replacing traditional assignment-based output while maintaining full backward compatibility.

## 2. Functional Requirements

### FR-001: Default Struct Literal Generation
**Priority**: Must Have  
**Description**: The system SHALL generate struct literal syntax by default for all compatible conversion methods  
**Acceptance Criteria**:
- Generated functions use `return &Type{field: value}` syntax by default
- No annotation required for struct literal output
- Output is syntactically correct Go code

### FR-002: Backward Compatibility
**Priority**: Must Have  
**Description**: The system SHALL maintain 100% backward compatibility with existing generated code  
**Acceptance Criteria**:
- Existing projects generate identical code when using compatibility flags
- All existing annotations continue to work without changes
- Generated function signatures remain unchanged

### FR-003: Annotation-Based Override
**Priority**: Must Have  
**Description**: WHEN `:no-struct-literal` annotation is present, the system SHALL use traditional assignment style  
**Acceptance Criteria**:
- `:no-struct-literal` at method level overrides default behavior
- `:no-struct-literal` at interface level affects all methods in interface
- Override behavior is clearly documented in generated comments

### FR-004: CLI Global Override
**Priority**: Must Have  
**Description**: WHEN `--no-struct-literal` flag is provided, the system SHALL disable struct literal output globally  
**Acceptance Criteria**:
- `--no-struct-literal` flag disables struct literal for entire execution
- `--struct-literal` flag explicitly enables (for overriding project settings)
- CLI flags take highest priority over annotations

### FR-005: Automatic Fallback Detection
**Priority**: Must Have  
**Description**: WHEN struct literal output is incompatible with method requirements, the system SHALL automatically use traditional assignment style  
**Acceptance Criteria**:
- Automatic fallback for `:preprocess`/`:postprocess` annotations
- Automatic fallback for error-returning `:conv` functions
- Automatic fallback for `:style arg` methods
- Clear documentation in generated comments explaining fallback

### FR-006: Error Handling Integration
**Priority**: Must Have  
**Description**: WHEN error handling is required, the system SHALL choose appropriate output format  
**Acceptance Criteria**:
- Simple conversions with errors use struct literal with error handling
- Complex conversions with errors fall back to traditional assignment
- Error propagation works correctly in both formats

### FR-007: Type Conversion Support
**Priority**: Must Have  
**Description**: The system SHALL support all existing type conversion annotations with struct literal output  
**Acceptance Criteria**:
- `:typecast` works within struct literal fields
- `:stringer` works within struct literal fields  
- `:map` annotations translate to correct struct literal field assignments
- `:skip` annotations exclude fields from struct literal

### FR-008: Custom Converter Integration
**Priority**: Should Have  
**Description**: WHEN simple `:conv` functions are used, the system SHALL include them in struct literal output  
**Acceptance Criteria**:
- Non-error-returning `:conv` functions work in struct literals
- Error-returning `:conv` functions trigger fallback to traditional assignment
- Function calls are correctly placed within struct literal syntax

### FR-009: Migration Analysis
**Priority**: Should Have  
**Description**: The system SHALL provide tooling to analyze existing code compatibility with struct literal output  
**Acceptance Criteria**:
- `--analyze-migration` flag reports compatibility status
- Analysis identifies methods that would change output format
- Analysis suggests override annotations where needed

### FR-010: Verbose Output
**Priority**: Could Have  
**Description**: WHEN `--verbose` flag is provided, the system SHALL report generation decisions  
**Acceptance Criteria**:
- Reports which methods use struct literal vs traditional assignment
- Explains reason for fallback decisions
- Shows annotation processing decisions

## 3. Non-Functional Requirements

### NFR-001: Performance
**Priority**: Must Have  
**Description**: Generated struct literal code SHALL perform at least as well as traditional assignment code  
**Acceptance Criteria**:
- No performance regression in generated code execution
- Generation time increase <10% for large interfaces
- Memory allocation efficiency maintained or improved

### NFR-002: Code Quality
**Priority**: Must Have  
**Description**: Generated struct literal code SHALL be readable and maintainable  
**Acceptance Criteria**:
- Proper indentation and formatting
- Field assignments clearly readable
- Generated code passes `go fmt` without changes

### NFR-003: Developer Experience
**Priority**: Should Have  
**Description**: The feature SHALL improve developer experience with generated code  
**Acceptance Criteria**:
- Generated code is more idiomatic Go
- Easier to understand data flow
- Better IDE support for generated code

### NFR-004: Compilation Speed
**Priority**: Should Have  
**Description**: Generated struct literal code SHALL not negatively impact compilation performance  
**Acceptance Criteria**:
- No significant increase in compilation time
- Generated code complexity remains reasonable
- Go compiler optimizations apply effectively

## 4. Constraint Requirements

### CR-001: Go Version Compatibility
**Priority**: Must Have  
**Description**: Generated code SHALL be compatible with Go versions supported by Convergen  
**Acceptance Criteria**:
- Generated code compiles on Go 1.21+
- No use of unsupported language features
- Maintains existing Go version support policy

### CR-002: Annotation Syntax Consistency
**Priority**: Must Have  
**Description**: New annotations SHALL follow existing Convergen annotation patterns  
**Acceptance Criteria**:
- `:no-struct-literal` follows existing colon-prefix pattern
- Annotation parsing uses existing infrastructure
- Documentation follows existing annotation reference format

### CR-003: CLI Interface Consistency
**Priority**: Must Have  
**Description**: New CLI flags SHALL follow existing Convergen CLI patterns  
**Acceptance Criteria**:
- Flag naming follows existing conventions
- Help text format consistent with existing flags
- Flag processing uses existing CLI infrastructure

### CR-004: Generated Code Format
**Priority**: Must Have  
**Description**: Generated struct literal code SHALL follow Go formatting standards  
**Acceptance Criteria**:
- Code passes `gofmt` without changes
- Indentation follows Go conventions
- Line length and wrapping follow Go standards

## 5. Priority Matrix

### Must Have (Release Blockers)
- FR-001: Default struct literal generation
- FR-002: Backward compatibility  
- FR-003: Annotation override system
- FR-004: CLI global override
- FR-005: Automatic fallback detection
- FR-006: Error handling integration
- FR-007: Type conversion support
- NFR-001: Performance requirements
- NFR-002: Code quality standards
- CR-001: Go version compatibility
- CR-002: Annotation syntax consistency
- CR-003: CLI interface consistency
- CR-004: Generated code format

### Should Have (Important for User Experience)
- FR-008: Custom converter integration
- FR-009: Migration analysis tooling
- NFR-003: Developer experience improvements
- NFR-004: Compilation speed maintenance

### Could Have (Nice to Have)
- FR-010: Verbose output and reporting

### Won't Have (Out of Scope)
- Custom struct literal formatting rules
- IDE integration enhancements
- Performance profiling tools
- Framework-specific templates

## 6. Acceptance Testing Strategy

**Unit Tests**: Each requirement SHALL have corresponding unit tests  
**Integration Tests**: End-to-end testing of struct literal generation pipeline  
**Compatibility Tests**: Verify backward compatibility with existing projects  
**Performance Tests**: Benchmark generated code execution and generation speed  
**Migration Tests**: Validate migration tooling accuracy and completeness

Implementation tracked in tasks.md, technical analysis in design.md.