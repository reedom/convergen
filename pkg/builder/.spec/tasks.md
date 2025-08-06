# pkg/builder Implementation Tasks

*See design.md for technical analysis and requirements.md for acceptance criteria*

## TASK-001: Core Assignment Generation (Addresses FR-001, FR-002)
- [x] **TASK-001.1**: Implement `assignmentBuilder` struct
- [x] **TASK-001.2**: Implement `FunctionBuilder` for function generation  
- [x] **TASK-001.3**: Implement struct-to-struct assignment processing
- [x] **TASK-001.4**: Implement field accessibility validation

## TASK-002: Handler Framework (Addresses FR-003, NFR-003)
- [x] **TASK-002.1**: Define `AssignmentHandler` interface
- [x] **TASK-002.2**: Create base `next` handler struct
- [x] **TASK-002.3**: Implement `SkipHandler` (Addresses FR-015)
- [x] **TASK-002.4**: Implement `LiteralSetterHandler` (Addresses FR-006)
- [ ] **TASK-002.5**: Implement `ConverterHandler` (Addresses FR-004)
- [ ] **TASK-002.6**: Implement `NameMapperHandler` (Addresses FR-005)
- [ ] **TASK-002.7**: Implement `SliceHandler` (Addresses FR-007)
- [ ] **TASK-002.8**: Implement `StructFieldMatchHandler` (Addresses FR-010)

## TASK-003: Advanced Assignment Features (Addresses FR-008, FR-009)
- [x] **TASK-003.1**: Implement type casting with `castNode` method
- [x] **TASK-003.2**: Implement Stringer interface support
- [x] **TASK-003.3**: Implement slice-to-slice assignment logic
- [x] **TASK-003.4**: Implement nested struct processing with null checks

## TASK-004: Field Mapping Features (Addresses FR-004, FR-005)
- [x] **TASK-004.1**: Implement converter function application
- [x] **TASK-004.2**: Implement field name mapping
- [x] **TASK-004.3**: Implement templated name mapping with additional arguments
- [x] **TASK-004.4**: Implement field resolution with `resolveExpr`

## TASK-005: Additional Arguments & Processing (Addresses FR-011, FR-012, FR-013)
- [x] **TASK-005.1**: Implement additional arguments support
- [x] **TASK-005.2**: Implement pre/post processing with manipulators
- [x] **TASK-005.3**: Implement reverse conversion support
- [x] **TASK-005.4**: Implement templated expression resolution

## TASK-006: Generic Type Support (Addresses FR-014, NFR-004)
- [x] **TASK-006.1**: Implement `GenericFieldMapper` class
- [x] **TASK-006.2**: Implement type substitution engine integration
- [x] **TASK-006.3**: Implement generic mapping context
- [x] **TASK-006.4**: Implement field mapping strategies and optimization
- [x] **TASK-006.5**: Implement metrics collection for generic mapping

## TASK-007: Chain Integration & Migration (Addresses NFR-001)
- [ ] **TASK-007.1**: Replace direct `dispatch` calls with handler chain
- [ ] **TASK-007.2**: Migrate `matchStructFieldAndStruct` to handler chain
- [ ] **TASK-007.3**: Remove deprecated `dispatch` and field matching methods
- [ ] **TASK-007.4**: Optimize handler chain performance

## TASK-008: Error Handling & Validation (Addresses CR-002)
- [x] **TASK-008.1**: Implement null pointer safety checks
- [x] **TASK-008.2**: Implement field accessibility validation
- [x] **TASK-008.3**: Implement type compatibility validation
- [ ] **TASK-008.4**: Add comprehensive error recovery in handlers

## TASK-009: Code Quality & Testing (Addresses NFR-002, CR-001)
- [x] **TASK-009.1**: Create unit tests for handler framework
- [ ] **TASK-009.2**: Add integration tests for generic field mapping
- [ ] **TASK-009.3**: Add performance benchmarks for handler chains
- [ ] **TASK-009.4**: Validate Go 1.21+ compatibility
- [ ] **TASK-009.5**: Execute full test suite and fix regressions

## TASK-010: Performance Optimization (Addresses NFR-001, NFR-004)
- [x] **TASK-010.1**: Implement caching for generic field mapping
- [x] **TASK-010.2**: Implement optimization strategies for assignments  
- [ ] **TASK-010.3**: Profile handler chain execution performance
- [ ] **TASK-010.4**: Optimize memory allocation patterns
