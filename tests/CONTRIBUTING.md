# Contributing to Convergen Testing Framework

This guide helps contributors understand how to add, modify, and extend the Convergen behavior-driven testing framework.

## Table of Contents

- [Getting Started](#getting-started)
- [Framework Architecture](#framework-architecture)
- [Adding New Tests](#adding-new-tests)
- [Creating Scenario Builders](#creating-scenario-builders)
- [Writing Assertions](#writing-assertions)
- [Error Testing](#error-testing)
- [Testing Guidelines](#testing-guidelines)
- [Code Review Process](#code-review-process)

## Getting Started

### Prerequisites

- Go 1.23+
- Understanding of Convergen annotation system
- Familiarity with Go testing patterns

### Running Tests

```bash
# Run all tests
go test ./tests -v

# Run specific test categories
go test ./tests -run TestAnnotationCoverage -v
go test ./tests -run TestErrorScenarios -v

# Run tests in parallel
go test ./tests -parallel 4 -v
```

### Project Structure

```
tests/
├── README.md                    # Framework documentation
├── CONTRIBUTING.md             # This file
├── behavior_test.go            # Main behavior-driven tests
├── error_test.go              # Error scenario tests
├── helpers/                   # Framework components
│   ├── scenario.go           # Core types and assertions
│   ├── inline_runner.go      # Test execution engine
│   └── inline_helpers.go     # Scenario builders
└── .spec/                    # SDD documentation
    ├── requirements.md
    ├── design.md
    └── tasks.md
```

## Framework Architecture

### Core Components

1. **TestScenario**: Defines what to test (types, interface, expectations)
2. **InlineScenarioRunner**: Executes tests (generates code, runs assertions)
3. **CodeAssertion**: Validates generated code patterns
4. **BehaviorTest**: Tests actual function behavior (future enhancement)

### Data Flow

```
TestScenario → InlineScenarioRunner → Temporary Files → 
Convergen Pipeline → Generated Code → Assertions → Results
```

### Key Design Principles

- **Zero Maintenance**: No static fixture files to maintain
- **Behavior Focus**: Test actual functionality, not code format
- **Isolation**: Each test runs in isolated temporary directory
- **Composability**: Reusable builders and assertions
- **Extensibility**: Easy to add new annotations and scenarios

## Adding New Tests

### 1. Simple Test Case

For testing a specific scenario without reusable builders:

```go
func TestMyFeature(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "MyFeatureName",
        "Description of what this tests",
    ).WithTypes(`
type Source struct {
    Field1 string
    Field2 int
}

type Dest struct {
    Field1 string
    Field2 int
}`).WithInterface(`
type Convergen interface {
    // Add annotations if needed
    Convert(*Source) *Dest
}`).WithBehaviorTests().
    WithCodeChecks(
        helpers.AssertHasGeneratedFunction(),
        helpers.Contains("expected pattern"),
    )

    runner.RunScenario(scenario)
}
```

### 2. Multiple Related Tests

For testing multiple variations of the same feature:

```go
func TestMyAnnotationVariations(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    testCases := []struct {
        name        string
        annotation  string
        expectation string
    }{
        {"BasicCase", ":myann basic", "basic pattern"},
        {"AdvancedCase", ":myann advanced", "advanced pattern"},
    }

    var scenarios []helpers.TestScenario
    for _, tc := range testCases {
        scenario := helpers.NewInlineScenario(tc.name, "Test "+tc.name).
            WithTypes(commonTypes).
            WithInterface(fmt.Sprintf(`
type Convergen interface {
    // %s
    Convert(*Source) *Dest
}`, tc.annotation)).
            WithBehaviorTests().
            WithCodeChecks(helpers.Contains(tc.expectation))
        
        scenarios = append(scenarios, scenario)
    }

    runner.RunScenarios(scenarios)
}
```

### 3. Error Testing

For testing error conditions:

```go
func TestMyErrorCondition(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "MyErrorCase",
        "Test specific error condition",
    ).WithTypes(`
// Define types that should cause an error
`).WithInterface(`
// Define interface that should cause an error
`).WithBehaviorTests().
    ShouldFail("expected error message")

    runner.RunScenario(scenario)
}
```

## Creating Scenario Builders

### When to Create Builders

Create reusable scenario builders when:
- Testing the same annotation with different parameters
- Need to test both positive and negative cases
- Multiple tests use similar type structures
- Want to provide convenience functions for common patterns

### Builder Implementation

Add to `helpers/inline_helpers.go`:

```go
// MyAnnotationScenario creates scenarios for testing :myann annotation
func MyAnnotationScenario(parameter string) InlineScenario {
    return NewInlineScenario(
        fmt.Sprintf("MyAnn_%s", parameter),
        fmt.Sprintf("Test :myann %s annotation", parameter),
    ).WithTypes(`
type Source struct {
    // Define appropriate source types
    Value string
}

type Dest struct {
    // Define appropriate destination types  
    Value string
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
    // :myann %s
    Convert(*Source) *Dest
}`, parameter))
}

// Advanced builder with multiple parameters
func ComplexAnnotationScenario(param1, param2 string) InlineScenario {
    return NewInlineScenario(
        fmt.Sprintf("Complex_%s_%s", param1, param2),
        fmt.Sprintf("Test complex annotation with %s and %s", param1, param2),
    ).WithTypes(fmt.Sprintf(`
type Source struct {
    Field1 string
    Field2 %s
}

type Dest struct {
    Field1 string
    Field2 %s
}`, param1, param2)).WithInterface(fmt.Sprintf(`
type Convergen interface {
    // :complex %s %s
    Convert(*Source) *Dest
}`, param1, param2))
}
```

### Builder Usage

```go
func TestMyAnnotation(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenarios := []helpers.TestScenario{
        helpers.MyAnnotationScenario("basic").WithBehaviorTests(),
        helpers.MyAnnotationScenario("advanced").WithBehaviorTests(),
        helpers.ComplexAnnotationScenario("string", "int").WithBehaviorTests(),
    }

    runner.RunScenarios(scenarios)
}
```

## Writing Assertions

### Built-in Assertions

Use existing assertions when possible:

```go
// Content checks
helpers.Contains("expected text")
helpers.NotContains("unwanted text")
helpers.MatchesRegex(`pattern`)
helpers.ExactMatch("exact content")

// Function checks
helpers.AssertHasGeneratedFunction()
helpers.AssertFunction("SpecificFunctionName")

// Compilation checks
helpers.CompilesSuccessfully()
```

### Custom Assertions

Add to `helpers/scenario.go` for reusable custom assertions:

```go
// AssertMyPattern checks for my specific pattern
func AssertMyPattern(expectedValue string) CodeAssertion {
    return CodeAssertion{
        Type:    AssertionRegex,
        Pattern: fmt.Sprintf(`MyPattern\s*:\s*%s`, regexp.QuoteMeta(expectedValue)),
        Message: fmt.Sprintf("Expected MyPattern with value: %s", expectedValue),
    }
}

// AssertComplexStructure validates complex code structures
func AssertComplexStructure() CodeAssertion {
    return CodeAssertion{
        Type:    AssertionRegex,
        Pattern: `func\s+\w+\([^)]*\)\s*[^{]*\{\s*(?:[^}]*\n)*[^}]*return\s+[^}]*\}`,
        Message: "Expected properly structured function with return statement",
    }
}
```

### Assertion Usage

```go
scenario := helpers.NewInlineScenario("Test", "Description").
    WithTypes(types).
    WithInterface(interface).
    WithCodeChecks(
        helpers.AssertMyPattern("expected"),
        helpers.AssertComplexStructure(),
        helpers.Contains("fallback check"),
    )
```

## Error Testing

### Error Categories

1. **Syntax Errors**: Invalid Go code syntax
2. **Annotation Errors**: Invalid or malformed annotations
3. **Type Errors**: Incompatible type conversions
4. **Function Errors**: Missing or invalid converter functions
5. **Interface Errors**: Invalid interface definitions

### Error Testing Patterns

```go
// Test syntax errors
scenario := helpers.InvalidSyntaxScenario().
    WithBehaviorTests().
    ShouldFail("failed to format source code")

// Test annotation errors  
scenario := helpers.NewInlineScenario("BadAnnotation", "Test bad annotation").
    WithTypes(validTypes).
    WithInterface(`
type Convergen interface {
    // :invalid_annotation_syntax
    Convert(*Source) *Dest
}`).WithBehaviorTests().
    ShouldFail("expected error message")

// Test type compatibility
scenario := helpers.TypeMismatchScenario().
    WithBehaviorTests().
    WithCodeChecks(helpers.AssertHasGeneratedFunction()) // Still generates code
```

### Error Message Validation

Be specific about expected error messages:

```go
// Good: Specific error message
.ShouldFail("function HashPassword not found")

// Bad: Generic error message  
.ShouldFail("error")

// Good: Pattern matching for variable parts
.ShouldFail("no assignment for dst.")
```

## Testing Guidelines

### Test Naming

```go
// Good: Descriptive and specific
func TestStyleAnnotationWithReturnStyle(t *testing.T)
func TestConvAnnotationWithMissingFunction(t *testing.T)
func TestNestedStructMapping(t *testing.T)

// Bad: Generic or unclear
func TestBasicCase(t *testing.T)
func TestScenario1(t *testing.T)
func TestStuff(t *testing.T)
```

### Test Organization

1. **Group Related Tests**: Put annotation tests together
2. **Use Subtests**: For variations of the same test
3. **Parallel Execution**: Use `t.Parallel()` for independent tests
4. **Clear Categories**: Separate success and error cases

### Test Data

```go
// Good: Realistic, meaningful data
type User struct {
    ID       uint64
    Username string
    Email    string
}

// Bad: Generic, meaningless data
type A struct {
    X int
    Y string
}
```

### Assertions

```go
// Good: Specific and meaningful
helpers.Contains("Email: src.Email")
helpers.AssertFunction("ConvertUser")
helpers.MatchesRegex(`Username:\s*src\.Username`)

// Bad: Too generic or brittle
helpers.Contains("src.")
helpers.Contains("func")
helpers.ExactMatch("entire generated file content")
```

## Code Review Process

### Before Submitting

1. **Run All Tests**: Ensure your changes don't break existing tests
2. **Add Documentation**: Update README.md if adding new features
3. **Follow Conventions**: Match existing code style and patterns
4. **Test Coverage**: Include both success and error scenarios
5. **Performance**: Consider impact on test execution time

### Review Checklist

#### For Test Files

- [ ] Tests have descriptive names
- [ ] Tests use `t.Parallel()` when appropriate
- [ ] Tests include `defer runner.Cleanup()`
- [ ] Assertions are specific and meaningful
- [ ] Error cases are tested with proper expected messages
- [ ] Test data is realistic and clear

#### For Scenario Builders

- [ ] Builder names follow convention (`XxxAnnotationScenario`)
- [ ] Builders are documented with clear descriptions
- [ ] Parameters are validated or documented
- [ ] Type definitions are minimal but realistic
- [ ] Interface definitions are syntactically correct

#### For Assertions

- [ ] Assertion names are descriptive (`AssertXxx`)
- [ ] Error messages are helpful for debugging
- [ ] Regex patterns are properly escaped
- [ ] Assertions are reusable across tests

### Common Review Comments

1. **"Add error test case"**: Include failure scenarios
2. **"Make assertion more specific"**: Avoid overly generic patterns
3. **"Use existing builder"**: Reuse existing scenario builders
4. **"Add documentation"**: Document new features and patterns
5. **"Consider edge cases"**: Test boundary conditions

## Advanced Patterns

### Complex Type Structures

```go
func TestNestedStructConversion(t *testing.T) {
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "NestedStructs",
        "Test conversion of nested struct hierarchies",
    ).WithTypes(`
type Address struct {
    Street string
    City   string
}

type User struct {
    Name    string
    Address Address
}

type UserModel struct {
    Name    string
    Address Address
}`).WithInterface(`
type Convergen interface {
    Convert(*User) *UserModel
}`).WithBehaviorTests().
    WithCodeChecks(
        helpers.AssertHasGeneratedFunction(),
        helpers.Contains("Address: src.Address"),
    )

    runner.RunScenario(scenario)
}
```

### Generic Type Testing

```go
func TestGenericTypes(t *testing.T) {
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "GenericConversion",
        "Test conversion involving generic types",
    ).WithTypes(`
type Container[T any] struct {
    Value T
    Count int
}

type StringContainer = Container[string]
type IntContainer = Container[int]`).WithInterface(`
type Convergen interface {
    Convert(*StringContainer) *IntContainer
}`).WithBehaviorTests().
    WithCodeChecks(helpers.AssertHasGeneratedFunction())

    runner.RunScenario(scenario)
}
```

### Performance Testing

```go
func BenchmarkScenarioExecution(b *testing.B) {
    for i := 0; i < b.N; i++ {
        runner := helpers.NewInlineScenarioRunner(&testing.T{})
        scenario := helpers.StyleAnnotationScenario("return").WithBehaviorTests()
        runner.RunScenario(scenario)
        runner.Cleanup()
    }
}
```

## Getting Help

- **Documentation**: Read `tests/README.md` for framework overview
- **Examples**: Look at existing tests in `behavior_test.go` and `error_test.go`
- **Issues**: Check existing issues or create new ones for questions
- **Discussions**: Use GitHub discussions for design questions

## Contribution Workflow

1. **Fork Repository**: Create your own fork
2. **Create Branch**: `git checkout -b feature/my-test-enhancement`
3. **Write Tests**: Add your test improvements
4. **Run Tests**: `go test ./tests -v`
5. **Update Documentation**: Update README.md if needed
6. **Commit Changes**: Use descriptive commit messages
7. **Create PR**: Submit pull request with clear description
8. **Address Feedback**: Respond to review comments
9. **Merge**: Maintainer will merge after approval

Thank you for contributing to the Convergen testing framework!