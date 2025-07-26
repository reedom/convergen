# Convergen Test Strategy

## Overview

Convergen employs a comprehensive testing strategy with multiple test types to ensure code quality, reliability, and maintainability. The testing approach follows Go best practices and supports the event-driven, pipeline-based architecture.

## Test Types and Structure

### Integration Tests (`tests/`)
**Purpose**: End-to-end testing of the complete code generation pipeline
**Location**: `tests/` directory
**Approach**: Fixture-based testing with real Go source files

**Characteristics**:
- Tests the complete pipeline from input parsing to code generation
- Uses actual Go source files as test fixtures
- Validates generated code for correctness and compilation
- Covers annotation processing and edge cases
- Command: `go test github.com/reedom/convergen/v8/tests`

### Unit Tests (`*_test.go`)
**Purpose**: Component-level testing of individual packages and functions
**Location**: Alongside source files (`*_test.go`)
**Approach**: Isolated testing of specific functionality

**Characteristics**:
- Test individual functions and methods in isolation
- Mock external dependencies where necessary
- Focus on edge cases and error conditions
- Validate domain model constructor patterns
- Command: `go test ./pkg/...` or `go test ./pkg/builder/...`

### Domain Model Testing
**Purpose**: Validate domain model integrity and constructor patterns
**Focus Areas**:
- Constructor function validation
- Type safety enforcement
- Event-driven communication patterns
- Error handling and propagation

## Test Coverage Strategy

### Coverage Targets
- **Overall Coverage**: 67%+ (visible in README badge)
- **Critical Paths**: 90%+ coverage for core pipeline components
- **Domain Models**: 100% coverage for constructor patterns
- **Error Handling**: Comprehensive error path testing

### Coverage Measurement
- `make coverage` - Generate comprehensive coverage report
- Automated coverage tracking through CI/CD
- Regular coverage monitoring and improvement

## Testing Patterns and Best Practices

### Domain Model Testing Patterns
```go
func TestMethodConstruction(t *testing.T) {
    //  Use constructors in tests
    sourceType := domain.NewBasicType("User", reflect.Struct)
    destType := domain.NewBasicType("UserDTO", reflect.Struct)
    
    method, err := domain.NewMethod("ConvertUser", sourceType, destType)
    assert.NoError(t, err)
    assert.NotNil(t, method)
    
    // L NEVER use direct struct literals in tests
    // method := &domain.Method{Name: "ConvertUser", ...}
}
```

### Event System Testing
```go
func TestEventHandling(t *testing.T) {
    handler := &mockHandler{}
    event := events.NewTestEvent("test-data")
    
    //  Use proper event handler interface
    err := handler.Handle(ctx, event)
    assert.NoError(t, err)
    
    // L NOT: err := handler(ctx, event)
}
```

### Result Structure Testing
```go
func TestMethodResult(t *testing.T) {
    method, _ := domain.NewMethod("TestMethod", sourceType, destType)
    
    //  Use new MethodResult structure
    result := &domain.MethodResult{
        Method:      method,          // NOT MethodName
        Code:        "generated code", // NOT Result field
        Success:     true,
        Error:       nil,
        ProcessedAt: time.Now(),
        DurationMS:  5,
    }
    
    // Test the result structure
    assert.Equal(t, method, result.Method)
    assert.True(t, result.Success)
}
```

## Test Migration Guidelines

### Legacy Test Updates
When updating tests for the new domain model:

1. **Replace Old Patterns**:
   - `MethodName` ’ `Method` (with proper Method object)
   - `Success/Result/StrategyUsed` ’ `Code/Error/ProcessedAt/DurationMS`

2. **Use Domain Constructors**:
   - Always use `domain.NewMethod()`, `domain.NewBasicType()`
   - Import `reflect` package when working with `BasicType`

3. **Handle Complex Migrations**:
   - Consider temporary commenting with TODO markers
   - Focus on core functionality first
   - Migrate incrementally to maintain test stability

## Continuous Integration Testing

### Automated Testing
- All tests run on every commit
- Multiple Go versions tested (1.23+)
- Cross-platform testing (Linux, macOS, Windows)
- Performance regression testing

### Quality Gates
- All tests must pass before merge
- Coverage thresholds must be maintained
- Linting and formatting checks required
- Security vulnerability scanning

## Test Commands Reference

```bash
# Run all tests
make test

# Run integration tests only
go test github.com/reedom/convergen/v8/tests

# Run package tests
go test ./pkg/...

# Run specific package tests
go test ./pkg/builder/...

# Run tests with verbose output
go test -v ./pkg/builder/...

# Run specific test by name
go test -run TestSpecificTest ./pkg/builder/...

# Generate coverage report
make coverage

# Run linting
make lint
```

## Mock and Test Utilities

### Domain Model Mocks
- Mock domain objects for isolated testing
- Event system mocks for component testing
- Configuration mocks for different scenarios

### Test Fixtures
- Real Go source files for integration testing
- Annotation examples covering all use cases
- Error condition test cases

## Performance Testing

### Benchmarks
- Code generation performance benchmarks
- Memory usage profiling
- Scalability testing with large inputs

### Performance Targets
- Sub-second generation for typical use cases
- Linear scaling with input size
- Minimal memory footprint

This testing strategy ensures comprehensive validation of Convergen's functionality while maintaining high code quality and supporting the project's architectural patterns.