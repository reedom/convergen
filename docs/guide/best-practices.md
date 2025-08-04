# Best Practices

Recommended patterns and conventions for using Convergen effectively in team environments.

## Coming Soon

This section will cover:

- Project organization strategies
- Annotation guidelines
- Testing generated code
- Maintenance and evolution
- Team collaboration patterns

*This page is currently being developed. Check back soon for comprehensive best practices.*

## Quick Guidelines

### Project Organization

```
project/
├── internal/
│   ├── domain/
│   ├── storage/
│   └── converters/
│       ├── user_converter.go
│       └── user_converter.gen.go
```

### Naming Conventions

- Use `*_converter.go` for input files
- Generated files are `*_converter.gen.go`
- Use descriptive interface names with `:convergen` annotation

### Version Control

Recommended to track generated files for build consistency.

For more detailed examples, see our [Examples section](../examples/index.md).