# Convergen Testing Framework Examples

This directory contains practical examples demonstrating how to use the Convergen behavior-driven testing framework. These examples serve as templates and learning resources for writing effective tests.

## Example Files

### `basic_usage_test.go`
Fundamental examples showing the core testing patterns:
- **ExampleBasicScenario**: Simple struct-to-struct conversion
- **ExampleAnnotationTesting**: Testing with Convergen annotations
- **ExampleBuiltInScenarios**: Using pre-built scenario helpers
- **ExampleErrorTesting**: Testing error conditions
- **ExampleComplexTypes**: Nested struct conversions
- **ExampleMultipleAssertions**: Different types of code assertions
- **ExampleBatchTesting**: Running multiple related tests
- **ExampleWithImports**: Testing scenarios requiring additional imports

### `advanced_patterns_test.go`
Advanced testing patterns and techniques:
- **ExampleCustomScenarioBuilder**: Creating reusable scenario builders
- **ExampleComplexAnnotations**: Multiple annotations working together
- **ExampleEdgeCases**: Boundary conditions and edge cases
- **ExamplePerformanceTesting**: Performance-focused test patterns
- **ExampleGenericTypes**: Testing with Go generics
- **ExampleComplexTypeRelationships**: Embedded types and interfaces
- **ExampleErrorRecoveryPatterns**: Graceful error handling
- **ExampleCustomAssertions**: Domain-specific assertion patterns
- **ExampleComprehensiveIntegration**: Full-featured integration test

## How to Use These Examples

### 1. Learning the Framework
Start with `basic_usage_test.go` to understand the fundamental concepts:
```bash
# Run basic examples
go test ./tests/examples -run ExampleBasic -v
```

### 2. Understanding Patterns
Review `advanced_patterns_test.go` for sophisticated testing techniques:
```bash
# Run advanced examples
go test ./tests/examples -run ExampleAdvanced -v
```

### 3. Copy and Adapt
Use these examples as templates for your own tests:
1. Copy the example that most closely matches your use case
2. Modify the types and interfaces for your scenario
3. Adjust the assertions to match your expectations
4. Add any custom converter functions needed

## Example Categories

### Basic Testing Patterns

**Simple Conversion**:
```go
scenario := helpers.NewInlineScenario("Name", "Description").
    WithTypes(sourceAndDestTypes).
    WithInterface(converterInterface).
    WithBehaviorTests().
    WithCodeChecks(assertions...)
```

**Annotation Testing**:
```go
scenario := helpers.StyleAnnotationScenario("return").WithBehaviorTests()
```

**Error Testing**:
```go
scenario := helpers.InvalidSyntaxScenario().ShouldFail("expected error")
```

### Advanced Patterns

**Custom Builders**:
```go
func MyAnnotationScenario(param string) helpers.InlineScenario {
    return helpers.NewInlineScenario(name, description).
        WithTypes(types).
        WithInterface(interface)
}
```

**Batch Testing**:
```go
scenarios := []helpers.TestScenario{
    scenario1, scenario2, scenario3,
}
runner.RunScenarios(scenarios)
```

**Complex Assertions**:
```go
helpers.MatchesRegex(`func\s+Convert.*\{`)
helpers.Contains("specific generated pattern")
helpers.AssertHasGeneratedFunction()
```

## Common Patterns by Use Case

### Testing New Annotations

When adding a new Convergen annotation:

1. **Create a scenario builder** (see `CustomMappingScenario` example)
2. **Test positive cases** with various parameters
3. **Test negative cases** with invalid syntax
4. **Add edge case tests** with boundary conditions

### Testing Type Conversions

For complex type structures:

1. **Start simple** with basic field mappings
2. **Add complexity gradually** with nested types
3. **Test edge cases** like empty structs or single fields
4. **Verify error handling** for incompatible types

### Integration Testing

For comprehensive feature testing:

1. **Combine multiple annotations** in single scenarios
2. **Use realistic type structures** from your domain
3. **Test complete workflows** end-to-end
4. **Verify all generated code patterns**

## Running Examples

### All Examples
```bash
go test ./tests/examples -v
```

### Specific Examples
```bash
# Run basic usage examples
go test ./tests/examples -run ExampleBasic -v

# Run advanced pattern examples
go test ./tests/examples -run ExampleAdvanced -v

# Run specific example
go test ./tests/examples -run ExampleCustomScenarioBuilder -v
```

### With Parallel Execution
```bash
go test ./tests/examples -parallel 4 -v
```

## Example Output

When you run the examples, you'll see output like:
```
=== RUN   ExampleBasicScenario
=== RUN   ExampleBasicScenario/BasicUserConversion
--- PASS: ExampleBasicScenario (0.05s)
    --- PASS: ExampleBasicScenario/BasicUserConversion (0.05s)
=== RUN   ExampleAnnotationTesting
=== RUN   ExampleAnnotationTesting/StyleAnnotation
--- PASS: ExampleAnnotationTesting (0.04s)
    --- PASS: ExampleAnnotationTesting/StyleAnnotation (0.04s)
```

## Customizing Examples

### Adapting for Your Domain

1. **Replace type definitions** with your domain models
2. **Update field names** to match your use case
3. **Modify annotations** to test your specific patterns
4. **Adjust assertions** to verify your expected output

### Adding New Examples

1. **Follow naming convention**: `ExampleYourFeature`
2. **Include clear documentation** explaining the purpose
3. **Use realistic scenarios** that others can relate to
4. **Add appropriate assertions** to verify behavior

### Creating Test Suites

Group related examples into test suites:
```go
func TestMyFeatureSuite(t *testing.T) {
    examples := []func(*testing.T){
        ExampleBasicCase,
        ExampleAdvancedCase,
        ExampleErrorCase,
    }
    
    for _, example := range examples {
        example(t)
    }
}
```

## Best Practices from Examples

### Type Design
- Use **realistic field names** and types
- Include **edge cases** like empty structs
- Test **complex hierarchies** with nested types
- Consider **generic types** when applicable

### Assertion Strategy
- **Start with basic assertions** (function existence)
- **Add specific pattern checks** for critical mappings
- **Include negative assertions** to verify exclusions
- **Use custom assertions** for domain-specific patterns

### Error Testing
- **Test syntax errors** for framework robustness
- **Verify error messages** are helpful and specific
- **Check error recovery** for partial failures
- **Test boundary conditions** that might cause errors

### Performance Considerations
- **Use parallel testing** for independent scenarios
- **Batch related tests** to reduce overhead
- **Profile complex scenarios** to ensure reasonable performance
- **Consider resource cleanup** to prevent memory leaks

## Contributing Examples

To add new examples:

1. **Create descriptive example functions** following the naming convention
2. **Include comprehensive documentation** explaining the use case
3. **Add to this README** with appropriate categorization
4. **Test thoroughly** to ensure examples work correctly
5. **Follow the established patterns** from existing examples

These examples provide a foundation for understanding and using the Convergen testing framework effectively. They demonstrate both basic usage and advanced techniques, serving as both learning resources and practical templates for real-world testing scenarios.