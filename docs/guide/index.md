# User Guide

This comprehensive guide covers everything you need to master Convergen, from basic annotations to advanced usage patterns.

## What You'll Find Here

### 📋 **[Annotations Reference](annotations.md)**
Complete reference for all Convergen annotations:

- Field matching strategies (`:match`, `:case`, `:getter`)
- Type conversion helpers (`:typecast`, `:stringer`)
- Custom mappings (`:map`, `:conv`, `:literal`)
- Function styles (`:style`, `:recv`, `:reverse`)
- Processing hooks (`:preprocess`, `:postprocess`)

### 🚀 **[Advanced Usage](advanced-usage.md)**
Complex scenarios and advanced techniques:

- Error handling patterns
- Generics support and type parameters
- Complex nested struct conversions
- Multiple interface definitions
- Integration with existing codebases

### ⚡ **[Performance](performance.md)**
Optimization techniques and performance considerations:

- Parser performance improvements in v8
- Generated code optimization
- Memory allocation patterns
- Benchmarking your conversions
- Large-scale project considerations

### 🎯 **[Best Practices](best-practices.md)**
Recommended patterns and conventions:

- Project organization strategies
- Annotation guidelines
- Testing generated code
- Maintenance and evolution
- Team collaboration patterns

## Quick Navigation

Looking for something specific?

=== "Annotation Syntax"

    Need to understand annotation format and usage? Check the [Annotations Reference](annotations.md).

=== "Complex Scenarios"

    Working with generics, errors, or nested structs? See [Advanced Usage](advanced-usage.md).

=== "Performance Issues"

    Want to optimize generation or runtime performance? Read the [Performance guide](performance.md).

=== "Project Guidelines"

    Setting up team conventions? Review [Best Practices](best-practices.md).

## Learning Path

We recommend following this progression:

1. **Start with [Annotations Reference](annotations.md)** - Master the core syntax
2. **Explore [Advanced Usage](advanced-usage.md)** - Handle complex scenarios  
3. **Review [Best Practices](best-practices.md)** - Adopt proven patterns
4. **Optimize with [Performance](performance.md)** - Fine-tune for scale

## Quick Reference

### Common Annotation Patterns

```go
type Convergen interface {
    // Basic field matching
    // :match name
    Convert(*Source) *Destination
    
    // With type conversion
    // :typecast
    // :stringer  
    ConvertWithTypes(*Source) *Destination
    
    // Custom field mapping
    // :map SourceField DestField
    // :conv CustomConverter Field
    ConvertCustom(*Source) *Destination
    
    // Error handling
    // :conv ValidateAndConvert Field
    ConvertWithError(*Source) (*Destination, error)
}
```

### Interface-Level vs Method-Level Annotations

- **Interface-level**: Apply to all methods in the interface
- **Method-level**: Override interface defaults for specific methods

```go
// Interface-level: affects all methods
// :typecast
// :case:off
type Convergen interface {
    // Method-level: overrides for this method only
    // :case
    // :match none
    SpecialConvert(*Source) *Destination
    
    // Uses interface defaults
    RegularConvert(*Source) *Destination
}
```

## Need Help?

- **Can't find what you're looking for?** Check our [Examples section](../examples/index.md)
- **Having issues?** Visit [Troubleshooting](../troubleshooting/index.md)
- **Want to contribute?** See our [API Reference](../api/index.md) for development info

Ready to dive deeper? Start with the [Annotations Reference](annotations.md)!