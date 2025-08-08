# Struct Literal Implementation Tasks

See requirements.md for acceptance criteria and design.md for technical analysis.

## Phase 1: Foundation (v8.1.0) - 6-8 weeks

### TASK-001: Extend Domain Models
**Requirements**: CR-002, FR-001  
**Status**: [x] Complete
- [x] Add `DstVarStructLiteral` to `pkg/generator/model/enums.go`
- [x] Add `OutputStyle` enum with Auto/StructLiteral/Traditional values
- [x] Extend `Function` model with `OutputStyle`, `FallbackReason`, `CanUseStructLit` fields
- [x] Add `IsStructLiteral()` helper method to `DstVarStyle`
- [x] Update unit tests for model extensions

### TASK-002: Annotation System Extension
**Requirements**: FR-003, CR-002  
**Status**: [x] Complete
- [x] Add `:no-struct-literal` parsing to `pkg/option/option.go`
- [x] Extend `Options` struct with `NoStructLiteral` and `ForceStructLit` fields
- [x] Add annotation parsing logic for method and interface level
- [x] Update annotation parser unit tests
- [x] Add validation for conflicting annotations

### TASK-003: CLI Flag Integration
**Requirements**: FR-004, CR-003  
**Status**: [ ] Pending
- [ ] Add `--no-struct-literal` and `--struct-literal` flags to `cmd/convergen/main.go`
- [ ] Add `--verbose` flag for generation decision reporting
- [ ] Implement option passing from CLI to generator
- [ ] Add CLI flag validation and conflict detection
- [ ] Update help text and usage documentation

### TASK-004: Compatibility Detection Engine
**Requirements**: FR-005, NFR-001  
**Status**: [ ] Pending
- [ ] Implement `canUseStructLiteral()` method in generator
- [ ] Add `canAssignInStructLiteral()` assignment compatibility check
- [ ] Implement `getFallbackReason()` for documentation
- [ ] Add `validateStructLiteralCompatibility()` for pre-generation validation
- [ ] Create comprehensive compatibility test suite

### TASK-005: Core Generation Logic
**Requirements**: FR-001, FR-005, NFR-002  
**Status**: [ ] Pending
- [ ] Implement `determineOutputStyle()` priority system
- [ ] Extend `FuncToString()` with output style decision logic
- [ ] Implement `generateStructLiteralFunction()` method
- [ ] Add `formatStructLiteralField()` for field formatting
- [ ] Implement fallback decision reporting with comments

### TASK-006: Basic Testing Framework
**Requirements**: All FR requirements  
**Status**: [ ] Pending
- [ ] Create unit tests for new domain models
- [ ] Add compatibility detection tests
- [ ] Create struct literal generation tests
- [ ] Add CLI flag processing tests
- [ ] Implement annotation parsing tests

## Phase 2: Default Behavior (v8.2.0) - 4-6 weeks

### TASK-007: Default Behavior Implementation
**Requirements**: FR-001, FR-002  
**Status**: [ ] Pending
- [ ] Switch default generation to struct literal style
- [ ] Implement automatic compatibility checking in pipeline
- [ ] Add generated code comments explaining fallback decisions
- [ ] Update existing tests for new default behavior
- [ ] Verify backward compatibility with `--no-struct-literal`

### TASK-008: Enhanced Override System
**Requirements**: FR-003, FR-004  
**Status**: [ ] Pending
- [ ] Implement interface-level `:no-struct-literal` annotation processing
- [ ] Add priority resolution for CLI > method > interface > default
- [ ] Implement verbose mode reporting with `--verbose` flag
- [ ] Add override validation and conflict detection
- [ ] Update documentation for override system

### TASK-009: Error Handling Integration
**Requirements**: FR-006, NFR-001  
**Status**: [ ] Pending
- [ ] Implement error handling compatibility detection
- [ ] Add support for simple error handling in struct literals
- [ ] Implement automatic fallback for complex error handling
- [ ] Add error propagation tests for both output styles
- [ ] Verify error handling performance is maintained

### TASK-010: Type Conversion Support
**Requirements**: FR-007  
**Status**: [ ] Pending
- [ ] Verify `:typecast` works in struct literal fields
- [ ] Implement `:stringer` support in struct literal context
- [ ] Add `:map` annotation translation to struct literal fields
- [ ] Implement `:skip` field exclusion from struct literals
- [ ] Create comprehensive type conversion test suite

### TASK-011: Migration Analysis Tool
**Requirements**: FR-009  
**Status**: [ ] Pending
- [ ] Implement `--analyze-migration` flag and functionality
- [ ] Add compatibility analysis reporting
- [ ] Create migration recommendation engine
- [ ] Add output format comparison for existing projects
- [ ] Implement migration guide generation

### TASK-012: Integration Testing
**Requirements**: All requirements  
**Status**: [ ] Pending
- [ ] Create end-to-end tests for mixed compatibility scenarios
- [ ] Add CLI integration tests with all flag combinations
- [ ] Test annotation override priority system
- [ ] Verify generated code compilation and execution
- [ ] Add performance benchmarks for generated code

## Phase 3: Polish & Advanced Features (v8.3.0) - 4-5 weeks

### TASK-013: Custom Converter Integration
**Requirements**: FR-008  
**Status**: [ ] Pending
- [ ] Implement simple `:conv` function support in struct literals
- [ ] Add automatic fallback detection for error-returning `:conv`
- [ ] Create hybrid generation for mixed simple/complex conversions
- [ ] Add converter compatibility analysis
- [ ] Test custom converter integration thoroughly

### TASK-014: Hybrid Generation Implementation
**Requirements**: FR-005, NFR-002  
**Status**: [ ] Pending
- [ ] Implement `generateHybridFunction()` for mixed scenarios
- [ ] Add `partitionAssignments()` for simple/complex separation
- [ ] Create struct literal base with imperative additions
- [ ] Add decision logic for hybrid vs pure traditional generation
- [ ] Test hybrid generation edge cases

### TASK-015: Code Quality Optimization
**Requirements**: NFR-002, NFR-004  
**Status**: [ ] Pending
- [ ] Optimize struct literal formatting and indentation
- [ ] Implement proper Go formatting standards compliance
- [ ] Add field alignment and readability improvements
- [ ] Optimize generation performance for large interfaces
- [ ] Benchmark and optimize memory usage during generation

### TASK-016: Enhanced Error Reporting
**Requirements**: NFR-003  
**Status**: [ ] Pending
- [ ] Improve fallback reason documentation in generated comments
- [ ] Add verbose mode with detailed generation decision explanations
- [ ] Implement warning system for suboptimal configurations
- [ ] Create troubleshooting guide for common issues
- [ ] Add suggestion system for optimization opportunities

### TASK-017: Advanced Compatibility Features
**Requirements**: FR-005, NFR-001  
**Status**: [ ] Pending
- [ ] Implement smart preprocess/postprocess detection
- [ ] Add partial struct literal with imperative pre/post processing
- [ ] Create compatibility scoring system for generation decisions
- [ ] Add performance analysis for different generation strategies
- [ ] Implement adaptive generation based on complexity metrics

## Phase 4: Documentation & Release (All versions) - Ongoing

### TASK-018: Core Documentation Updates
**Requirements**: All requirements  
**Status**: [ ] Pending
- [ ] Update annotation reference documentation with `:no-struct-literal`
- [ ] Add struct literal usage examples to guides
- [ ] Create migration guide for existing users
- [ ] Update CLI reference with new flags
- [ ] Add troubleshooting section for struct literal issues

### TASK-019: Example Creation
**Requirements**: NFR-003  
**Status**: [ ] Pending
- [ ] Create basic struct literal usage examples
- [ ] Add complex conversion examples with fallback scenarios
- [ ] Create migration examples showing before/after
- [ ] Add performance comparison examples
- [ ] Create framework integration examples

### TASK-020: Testing Completion
**Requirements**: All requirements  
**Status**: [ ] Pending
- [ ] Achieve 95%+ test coverage for struct literal functionality
- [ ] Add comprehensive edge case testing
- [ ] Create performance regression test suite
- [ ] Add compatibility testing with real-world projects
- [ ] Implement automated testing for all supported Go versions

### TASK-021: Release Preparation
**Requirements**: All requirements  
**Status**: [ ] Pending
- [ ] Complete integration testing with existing convergen test suite
- [ ] Verify all requirements met with acceptance testing
- [ ] Update CHANGELOG.md with new features and breaking changes
- [ ] Create release notes with migration instructions
- [ ] Prepare version tagging and release artifacts

## Completion Criteria

Each task is considered complete when:
- [ ] All listed subtasks are implemented and tested
- [ ] Unit tests pass with 95%+ coverage for new code
- [ ] Integration tests verify functionality works end-to-end
- [ ] Code review completed and approved
- [ ] Documentation updated for user-facing changes
- [ ] Performance benchmarks show no regression
- [ ] Backward compatibility verified with existing projects

## Dependencies

**Cross-Task Dependencies:**
- TASK-002 depends on TASK-001 (domain models must exist before annotation parsing)
- TASK-005 depends on TASK-001, TASK-004 (needs models and compatibility detection)
- TASK-007 depends on TASK-005 (default behavior needs core generation logic)
- TASK-013 depends on TASK-010 (custom converters need type conversion support)
- TASK-014 depends on TASK-005, TASK-013 (hybrid generation needs core + converters)

**External Dependencies:**
- Go version compatibility requirements (CR-001)
- Existing convergen annotation system (CR-002)
- Current CLI infrastructure (CR-003)
- Generated code formatting standards (CR-004)

## Risk Mitigation

**High Risk Tasks:**
- TASK-007: Default behavior switch (potential breaking changes)
- TASK-009: Error handling integration (complex compatibility logic)
- TASK-014: Hybrid generation (complex edge case handling)

**Mitigation Strategies:**
- Extensive testing with real-world projects before release
- Gradual rollout with opt-out mechanisms
- Comprehensive documentation and migration guides
- Performance benchmarking throughout development
