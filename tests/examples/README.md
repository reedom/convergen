# Testing Framework Examples

This directory contains **educational examples** showing how to use the Convergen testing framework APIs. These are **not comprehensive tests** - for complete test coverage, see [`../behavior_test.go`](../behavior_test.go).

## Purpose

The examples here focus on **teaching the testing framework**, not testing Convergen itself:

- ✅ **Framework usage patterns** - How to write tests
- ✅ **API demonstrations** - How to use different features  
- ✅ **Best practices** - Recommended patterns and approaches
- ❌ **Not comprehensive Convergen testing** - That's in `behavior_test.go`

## Examples Overview

### [`basic_usage_test.go`](basic_usage_test.go)
**Learn the framework fundamentals:**
- `TestFrameworkBasics` - Essential components every test needs
- `TestCodeAssertions` - How to validate generated code
- `TestBuiltInScenarios` - Using pre-built scenario helpers
- `TestErrorScenarios` - Testing failure cases
- `TestBatchExecution` - Running multiple tests efficiently
- `TestWithImports` - Handling external dependencies

### [`advanced_patterns_test.go`](advanced_patterns_test.go)  
**Master advanced framework features:**
- `TestCustomScenarioBuilder` - Creating reusable scenario builders
- `TestVerboseDebugging` - Getting detailed test failure information
- `TestDebugHelper` - Quick debugging with helper functions
- `TestCustomAssertions` - Building domain-specific assertions
- `TestDataDrivenTests` - Table-driven test patterns
- `TestConditionalAssertions` - Advanced assertion logic
- `TestScenarioCategories` - Organizing tests with categories
- `TestPerformancePattern` - Patterns for testing varying complexity

### [`debug_example_test.go`](debug_example_test.go)
**Learn debugging techniques:**
- Basic vs verbose debugging modes
- Using debug helper functions
- Understanding failure output
- Troubleshooting test issues

## Quick Start

```go
func TestMyFirst(t *testing.T) {
    // 1. Create runner
    runner := helpers.NewInlineScenarioRunner(t)
    defer runner.Cleanup()

    // 2. Define scenario
    scenario := helpers.NewInlineScenario(
        "TestName", "Description",
    ).WithTypes(`type User struct { Name string }`).
      WithInterface(`type Convergen interface { Convert(*User) *User }`)

    // 3. Run test
    runner.RunScenario(scenario)
}
```

## Learning Path

1. **Start with [`basic_usage_test.go`](basic_usage_test.go)** - Essential patterns
2. **Try [`debug_example_test.go`](debug_example_test.go)** - Debugging skills  
3. **Explore [`advanced_patterns_test.go`](advanced_patterns_test.go)** - Advanced techniques
4. **Check [`../TESTING_GUIDE.md`](../TESTING_GUIDE.md)** - Quick reference
5. **Review [`../behavior_test.go`](../behavior_test.go)** - Complete test patterns

## Key Concepts

### Framework Components
- **`InlineScenarioRunner`** - Test execution engine
- **`InlineScenario`** - Individual test definition
- **Code Assertions** - Validation of generated code
- **Built-in Scenarios** - Pre-made test builders

### Testing Patterns
- **Scenario builders** - Reusable test factories
- **Batch execution** - Running multiple tests
- **Debug modes** - Detailed failure information
- **Custom assertions** - Domain-specific validation

### Best Practices
- Always `defer runner.Cleanup()`
- Use descriptive test names and descriptions
- Prefer built-in scenarios for common patterns
- Enable verbose debugging for failures
- Organize tests with categories

## Framework API Reference

### Creating Tests
```go
// Basic scenario
scenario := helpers.NewInlineScenario("Name", "Description")

// With types and interface
scenario.WithTypes(`type User struct { Name string }`).
         WithInterface(`type Convergen interface { Convert(*User) *User }`)

// With assertions
scenario.WithCodeChecks(
    helpers.AssertHasGeneratedFunction(),
    helpers.Contains("pattern"),
    helpers.CompilesSuccessfully(),
)

// With debugging
scenario.WithVerboseDebugging()
```

### Built-in Scenarios
```go
// Style annotations
helpers.StyleAnnotationScenario("return")
helpers.StyleAnnotationScenario("arg")

// Match strategies  
helpers.MatchAnnotationScenario("name")
helpers.MatchAnnotationScenario("none")

// Custom converters
helpers.ConvertAnnotationScenario("FuncName", "SrcField", "DstField")

// Error scenarios
scenario.ShouldFail("expected error message")
```

### Custom Builders
```go
func MyScenarioBuilder(param string) helpers.InlineScenario {
    return helpers.NewInlineScenario(
        fmt.Sprintf("Test_%s", param),
        fmt.Sprintf("Testing %s functionality", param),
    ).WithTypes(generateTypes(param)).
      WithInterface(generateInterface(param))
}
```

## Comparison with behavior_test.go

| Examples Directory | behavior_test.go |
|---|---|
| 🎓 **Educational** | 🧪 **Validation** |
| Framework usage patterns | Comprehensive Convergen testing |
| Simple, focused examples | Complete annotation coverage |
| Learning progression | Edge cases and error scenarios |
| API demonstrations | Production test suite |

## Running Examples

```bash
# All examples
go test ./tests/examples -v

# Specific tests
go test ./tests/examples -run TestFrameworkBasics -v
go test ./tests/examples -run TestVerboseDebugging -v

# With parallel execution
go test ./tests/examples -parallel 4 -v
```

## Need Help?

1. **Quick reference**: [`../TESTING_GUIDE.md`](../TESTING_GUIDE.md)
2. **Full documentation**: [`../README.md`](../README.md)  
3. **Project context**: [`../../CLAUDE.md`](../../CLAUDE.md)
4. **Complete test examples**: [`../behavior_test.go`](../behavior_test.go)