# Convergen Testing Guide

Quick reference guide for using the Convergen behavior-driven testing framework.

## Quick Start

### 1. Basic Test Setup

```go
func TestMyConversion(t *testing.T) {
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    scenario := helpers.NewInlineScenario("TestName", "Description").
        WithTypes(`/* Go types */`).
        WithInterface(`/* Converter interface */`).
        WithBehaviorTests().
        WithCodeChecks(/* assertions */)

    runner.RunScenario(scenario)
}
```

### 2. Using Built-in Scenarios

```go
scenarios := []helpers.TestScenario{
    helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
    helpers.MatchAnnotationScenario("name").WithBehaviorTests(),
    helpers.ConvertAnnotationScenario("HashFunc", "src", "dst").WithBehaviorTests(),
}
runner.RunScenarios(scenarios)
```

### 3. Error Testing

```go
scenario := helpers.MissingConverterFunctionScenario().
    WithBehaviorTests().
    ShouldFail("function not found")
```

## Common Commands

```bash
# Run all tests
go test ./tests -v

# Run specific test categories
go test ./tests -run TestAnnotationCoverage -v
go test ./tests -run TestErrorScenarios -v

# Run examples
go test ./tests/examples -v

# Run with parallelism
go test ./tests -parallel 4 -v
```

## Built-in Scenario Builders

| Builder | Purpose | Example |
|---------|---------|---------|
| `StyleAnnotationScenario(style)` | Test `:style` annotations | `StyleAnnotationScenario("return")` |
| `MatchAnnotationScenario(algorithm)` | Test `:match` annotations | `MatchAnnotationScenario("name")` |
| `ConvertAnnotationScenario(func, src, dst)` | Test `:conv` annotations | `ConvertAnnotationScenario("Hash", "Password", "Hashed")` |
| `LiteralAnnotationScenario(field, value)` | Test `:literal` annotations | `LiteralAnnotationScenario("Status", `"active"`)` |
| `SkipAnnotationScenario(pattern)` | Test `:skip` annotations | `SkipAnnotationScenario("Password")` |
| `TypecastAnnotationScenario()` | Test `:typecast` annotations | `TypecastAnnotationScenario()` |
| `StringerAnnotationScenario()` | Test `:stringer` annotations | `StringerAnnotationScenario()` |
| `RecvAnnotationScenario(var)` | Test `:recv` annotations | `RecvAnnotationScenario("c")` |
| `MapAnnotationScenario(src, dst)` | Test `:map` basic field mapping | `MapAnnotationScenario("FirstName", "FullName")` |
| `MapTemplatedArgumentsScenario()` | Test `:map` with templated args ($1, $2) | `MapTemplatedArgumentsScenario()` |
| `MapMethodChainScenario()` | Test `:map` with method chains/getters | `MapMethodChainScenario()` |
| `MapNestedFieldScenario()` | Test `:map` with nested field paths | `MapNestedFieldScenario()` |

## Error Scenario Builders

| Builder | Purpose |
|---------|---------|
| `InvalidSyntaxScenario()` | Test syntax error handling |
| `TypeMismatchScenario()` | Test type compatibility errors |
| `MissingConverterFunctionScenario()` | Test missing converter functions |
| `EmptyInterfaceScenario()` | Test empty converter interfaces |
| `InvalidReturnTypeScenario()` | Test undefined return types |

## Common Assertions

| Assertion | Purpose | Example |
|-----------|---------|---------|
| `helpers.AssertHasGeneratedFunction()` | Verify function was generated | Required for all scenarios |
| `helpers.Contains("pattern")` | Check for specific text | `Contains("src.Name")` |
| `helpers.NotContains("pattern")` | Verify text absence | `NotContains("Password")` |
| `helpers.MatchesRegex("regex")` | Pattern matching | `MatchesRegex(`func\\s+\\w+`)` |
| `helpers.CompilesSuccessfully()` | Verify compilation | Basic syntax validation |

## Framework Benefits

- **Zero Maintenance**: No static fixture files to maintain
- **Behavior Focus**: Tests actual functionality, not code format
- **Comprehensive Coverage**: Easy to achieve 100% annotation coverage
- **Error Testing**: Robust error condition validation
- **Performance**: Efficient parallel execution
- **Documentation**: Self-documenting test scenarios

## Documentation Links

- **Framework Overview**: [`tests/README.md`](./README.md)
- **Contributor Guide**: [`tests/CONTRIBUTING.md`](./CONTRIBUTING.md)
- **Examples**: [`tests/examples/`](./examples/)
- **Project Integration**: [`CLAUDE.md`](../CLAUDE.md)

## Getting Help

1. **Check Examples**: Look at [`tests/examples/`](./examples/) for patterns
2. **Read Documentation**: Review [`tests/README.md`](./README.md) for details
3. **Review Tests**: Examine existing tests in [`behavior_test.go`](./behavior_test.go)
4. **Ask Questions**: Create issues for framework questions

## Migration from Legacy Tests

The new framework replaces file comparison with behavior-driven testing:

### Before (Legacy)
```go
expected := readFixtureFile("path/to/expected.go")
actual := generateCode(input)
assert.Equal(t, expected, actual)
```

### After (Behavior-Driven)
```go
scenario := helpers.NewInlineScenario("Test", "Description").
    WithTypes(inlineTypes).
    WithInterface(inlineInterface).
    WithBehaviorTests()
runner.RunScenario(scenario)
```

## Debugging Test Failures

### Enhanced Debugging Features

When tests fail, you now get detailed debugging information:

#### Basic Debugging (Default)
```go
scenario := helpers.NewInlineScenario("Test", "Description").
    WithTypes(types).
    WithInterface(interface).
    WithCodeChecks(helpers.Contains("expected_pattern"))
// Failure shows: assertion details + hint to use verbose debugging
```

#### Verbose Debugging 
```go
scenario := helpers.NewInlineScenario("Test", "Description").
    WithTypes(types).
    WithInterface(interface).
    WithCodeChecks(helpers.Contains("expected_pattern")).
    WithVerboseDebugging() // Shows full source + generated code
```

#### Debug Helper Functions
```go
// Enable debugging for any scenario
debugScenario := helpers.WithDebug(existingScenario)

// Create debug-enabled scenario from scratch
scenario := helpers.DebugScenario("Test", "Description")
```

### What You Get on Failure

#### Code Generation Failures
- ❌ Clear error message with scenario name
- 📄 Source file content that was generated
- 🔧 Generated code (first 1000 chars)
- 🚨 Separate source creation vs. code generation errors
- 📍 File paths for manual inspection

#### Assertion Failures
- ❌ Failed assertion details (type, pattern, message)
- 📊 Full generated code (in verbose mode)
- 💡 Helpful hint to enable verbose debugging
- ✅ Successful assertions logged (in verbose mode)

### Debugging Workflow

1. **Run test normally** - get basic failure info
2. **Add `.WithVerboseDebugging()`** - see full generated code
3. **Use `helpers.WithDebug(scenario)`** for quick debugging
4. **Check source file** at the logged path for manual inspection

## Best Practices

1. **Use `t.Parallel()`** for independent tests
2. **Always defer `runner.Cleanup()`** for resource management
3. **Use specific assertions** rather than generic patterns
4. **Include error scenarios** for comprehensive coverage
5. **Group related tests** using batch execution
6. **Follow naming conventions** for clarity and organization
7. **Enable verbose debugging** when investigating failures
8. **Use debug helpers** for quick troubleshooting