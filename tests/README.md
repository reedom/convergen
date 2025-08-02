# Convergen Behavior-Driven Testing Framework

A comprehensive behavior-driven testing framework for Convergen that tests actual conversion functionality through inline code generation and runtime execution.

## Overview

This framework replaces traditional file comparison testing with a robust, maintainable approach that focuses on testing actual conversion behavior rather than generated code format. It provides zero maintenance overhead by eliminating static fixture files and brittle file comparisons.

## Key Features

- **Inline Code Generation**: Define types and interfaces directly within test scenarios
- **Runtime Behavior Testing**: Compile and execute generated functions to test actual behavior
- **Comprehensive Annotation Coverage**: Focused test scenarios for all Convergen annotations
- **Zero Maintenance**: No static fixture files to maintain or update
- **Error Testing**: Comprehensive error condition testing with proper validation
- **Flexible Assertions**: Rich set of assertion helpers for code validation

## Quick Start

### Basic Test Scenario

```go
func TestMyScenario(t *testing.T) {
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "BasicConversion", 
        "Test basic struct-to-struct conversion",
    ).WithTypes(`
type User struct {
    ID   uint64
    Name string
}

type UserModel struct {
    ID   uint64
    Name string
}`).WithInterface(`
type Convergen interface {
    Convert(*User) *UserModel
}`).WithBehaviorTests().
    WithCodeChecks(
        helpers.AssertHasGeneratedFunction(),
        helpers.Contains("src.Name"),
    )

    runner.RunScenario(scenario)
}
```

### Annotation Testing

```go
func TestStyleAnnotation(t *testing.T) {
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.StyleAnnotationScenario("return").
        WithBehaviorTests().
        WithCodeChecks(helpers.AssertHasGeneratedFunction())

    runner.RunScenario(scenario)
}
```

### Error Testing

```go
func TestErrorCondition(t *testing.T) {
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.MissingConverterFunctionScenario().
        WithBehaviorTests().
        ShouldFail("function NonExistentFunction not found")

    runner.RunScenario(scenario)
}
```

## Framework Components

### Core Types

#### TestScenario
```go
type TestScenario struct {
    Name        string          // Test scenario name
    Description string          // Test description
    Category    string          // Test category for organization
    
    // Inline Code Definition
    SourceTypes string          // Go struct definitions
    Interface   string          // Converter interface definition
    Imports     []string        // Additional imports needed
    
    // Behavior Tests
    BehaviorTests []BehaviorTest
    
    // Code Expectations
    ShouldSucceed bool
    ExpectedError string
    CodeChecks    []CodeAssertion
    
    // Test metadata
    ShouldSkip bool
    SkipReason string
}
```

#### BehaviorTest
```go
type BehaviorTest struct {
    Name        string      // Test name
    Description string      // Test description
    TestFunc    string      // Name of generated function to test
    Input       interface{} // Input value for the function
    Expected    interface{} // Expected output value
    ShouldError bool       // Whether this test should produce an error
}
```

#### CodeAssertion
```go
type CodeAssertion struct {
    Type    AssertionType // contains, not_contains, regex, exact, compiles
    Pattern string        // Pattern to match
    Message string        // Custom assertion message
}
```

### Test Runner

#### InlineScenarioRunner
The `InlineScenarioRunner` handles the complete test execution workflow:

1. **Temporary Directory Creation**: Each test runs in an isolated temporary directory
2. **Source File Generation**: Creates Go files with inline types and interfaces
3. **Code Generation**: Runs Convergen pipeline to generate conversion code
4. **Code Compilation**: Compiles the generated code to verify syntax
5. **Behavior Testing**: Executes actual conversion functions with test data
6. **Assertion Validation**: Runs code assertions against generated code
7. **Cleanup**: Automatically removes temporary files

```go
// Create and use runner
runner := helpers.NewInlineScenarioRunner(t)
defer runner.Cleanup()

// Run single scenario
runner.RunScenario(scenario)

// Run multiple scenarios
runner.RunScenarios([]helpers.TestScenario{scenario1, scenario2})
```

## Built-in Scenario Builders

### Annotation Scenarios

```go
// Style annotation testing
helpers.StyleAnnotationScenario("return")
helpers.StyleAnnotationScenario("arg")

// Match annotation testing
helpers.MatchAnnotationScenario("name")
helpers.MatchAnnotationScenario("none")

// Custom converter testing
helpers.ConvertAnnotationScenario("HashPassword", "Password", "HashedPassword")

// Field operations
helpers.LiteralAnnotationScenario("Status", `"active"`)
helpers.SkipAnnotationScenario("Password")
```

### Error Scenarios

```go
// Syntax and parsing errors
helpers.InvalidSyntaxScenario()
helpers.InvalidAnnotationScenario()

// Type and conversion errors
helpers.TypeMismatchScenario()
helpers.MissingConverterFunctionScenario()

// Interface and structure errors
helpers.EmptyInterfaceScenario()
helpers.InvalidReturnTypeScenario()
```

### Custom Scenarios

```go
// Builder pattern for custom scenarios
scenario := helpers.NewInlineScenario("CustomTest", "My custom test").
    WithTypes(sourceTypes).
    WithInterface(converterInterface).
    WithImports("time", "fmt").
    WithBehaviorTests(behaviorTests...).
    WithCodeChecks(assertions...).
    WithCategory("custom")
```

## Assertion Library

### Basic Assertions

```go
// Content assertions
helpers.Contains("expected text")
helpers.NotContains("unwanted text")
helpers.MatchesRegex(`func\s+\w+`)
helpers.ExactMatch("exact code match")

// Compilation assertions
helpers.CompilesSuccessfully()

// Function assertions
helpers.AssertHasGeneratedFunction()
helpers.AssertFunction("MyFunction")
```

### Error Assertions

```go
// Error message validation
helpers.AssertErrorContains("specific error text")
helpers.AssertErrorType("parse error")

// Error pattern matching
helpers.AssertParseError()
helpers.AssertTypeError()
helpers.AssertAnnotationError()
```

## Best Practices

### Test Organization

1. **Group by Category**: Organize tests by annotation type or functionality
2. **Use Descriptive Names**: Clear test names that describe what's being tested
3. **Single Responsibility**: Each scenario should test one specific aspect
4. **Error Coverage**: Include both success and failure scenarios

### Scenario Design

1. **Minimal Examples**: Use the simplest code that demonstrates the feature
2. **Clear Expectations**: Define specific assertions for what should be generated
3. **Realistic Data**: Use realistic type structures and field names
4. **Edge Cases**: Include boundary conditions and edge cases

### Performance Considerations

1. **Parallel Testing**: Use `t.Parallel()` for independent tests
2. **Resource Cleanup**: Always defer `runner.Cleanup()`
3. **Batch Operations**: Use `RunScenarios()` for multiple related tests
4. **Selective Testing**: Use test filtering for focused development

## Examples

### Testing Complex Annotations

```go
func TestComplexAnnotations(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "ComplexMapping",
        "Test complex field mapping with multiple annotations",
    ).WithTypes(`
type Source struct {
    FirstName string
    LastName  string
    Password  string
    Age       int
}

type Dest struct {
    FullName       string
    HashedPassword string
    Age            int
    Status         string
}`).WithInterface(`
type Convergen interface {
    // :map FirstName,LastName FullName
    // :conv HashPassword Password HashedPassword
    // :literal Status "active"
    Convert(*Source) *Dest
}

func HashPassword(password string) string {
    return "hashed_" + password
}`).WithBehaviorTests().
    WithCodeChecks(
        helpers.AssertHasGeneratedFunction(),
        helpers.Contains("HashPassword(src.Password)"),
        helpers.Contains(`Status: "active"`),
    )

    runner.RunScenario(scenario)
}
```

### Testing Error Recovery

```go
func TestPartialSuccess(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario(
        "PartialFieldMapping",
        "Test partial success when some fields can't be mapped",
    ).WithTypes(`
type Source struct {
    ValidField   string
    ComplexField complex128 // Unmappable type
}

type Dest struct {
    ValidField   string
    MissingField string    // No source mapping
}`).WithInterface(`
type Convergen interface {
    Convert(*Source) *Dest
}`).WithBehaviorTests().
    WithCodeChecks(
        helpers.AssertHasGeneratedFunction(),
        helpers.Contains("src.ValidField"),
    )

    runner.RunScenario(scenario)
}
```

### Batch Testing

```go
func TestAllStyleAnnotations(t *testing.T) {
    t.Parallel()
    
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenarios := []helpers.TestScenario{
        helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
        helpers.StyleAnnotationScenario("arg").WithBehaviorTests(),
    }

    runner.RunScenarios(scenarios)
}
```

## Contributing

When adding new test scenarios:

1. **Follow Naming Conventions**: Use descriptive scenario names
2. **Add Documentation**: Include clear descriptions of what's being tested
3. **Include Assertions**: Add appropriate code assertions
4. **Test Both Success and Failure**: Include error scenarios when applicable
5. **Update Builder Functions**: Add new scenario builders for reusable patterns

### Adding New Scenario Builders

```go
// Add to inline_helpers.go
func MyNewAnnotationScenario(param string) InlineScenario {
    return NewInlineScenario(
        "MyAnnotation_" + param,
        "Test :myannotation " + param + " annotation",
    ).WithTypes(`
// Define appropriate types
`).WithInterface(fmt.Sprintf(`
type Convergen interface {
    // :myannotation %s
    Convert(*Source) *Dest
}`, param))
}
```

## Migration from Legacy Tests

The behavior-driven framework replaces the legacy file comparison approach:

### Before (Legacy)
```go
// Static fixture file required
// Brittle file path dependencies
// Manual fixture maintenance
expected := readFixtureFile("path/to/expected.go")
actual := generateCode(input)
assert.Equal(t, expected, actual)
```

### After (Behavior-Driven)
```go
// Inline code definition
// Zero maintenance overhead
// Behavior-focused testing
scenario := helpers.NewInlineScenario("Test", "Description").
    WithTypes(inlineTypes).
    WithInterface(inlineInterface).
    WithBehaviorTests()
runner.RunScenario(scenario)
```

## Troubleshooting

### Common Issues

1. **Test Failures**: Check that assertions match actual generated code patterns
2. **Compilation Errors**: Verify that inline type definitions are valid Go syntax
3. **Import Issues**: Add necessary imports using `WithImports()`
4. **Temporary File Issues**: Ensure `runner.Cleanup()` is called with `defer`

### Debugging Tips

1. **Enable Verbose Testing**: Run tests with `-v` flag for detailed output
2. **Check Generated Code**: Examine the actual generated code in assertion failures
3. **Validate Syntax**: Ensure inline code compiles independently
4. **Isolate Issues**: Run individual scenarios to isolate problems

## Performance

The behavior-driven framework is designed for efficiency:

- **Parallel Execution**: Tests run concurrently without interference
- **Temporary Isolation**: Each test uses isolated temporary directories
- **Minimal Overhead**: Direct API usage avoids external process spawning
- **Resource Management**: Automatic cleanup prevents resource leaks
- **Linear Scaling**: Performance scales linearly with test count

## Architecture

The framework follows a layered architecture:

1. **Test Definition Layer**: `TestScenario`, `BehaviorTest`, `CodeAssertion`
2. **Builder Layer**: Scenario builders and helper functions
3. **Execution Layer**: `InlineScenarioRunner` and test orchestration
4. **Integration Layer**: Convergen pipeline integration
5. **Assertion Layer**: Code validation and behavior testing

This architecture provides separation of concerns, testability, and extensibility for future enhancements.