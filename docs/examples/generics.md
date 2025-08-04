# Generics Support

Examples demonstrating Convergen's integration with Go generics, including type parameters, constraints, and collection transformations.

## Coming Soon

This section will cover:

- Generic conversion functions
- Type parameter constraints
- Repository pattern implementations
- Collection transformations
- Type-safe wrapper conversions

*Comprehensive generics examples are currently being developed.*

## Quick Examples

### Generic Collection Conversion

```go
type Convergen interface {
    // Generic slice conversion
    ConvertSlice[T, U any]([]T) []U
    
    // With constraints
    ConvertNumbers[T ~int | ~float64, U ~int64 | ~float32](T) U
}
```

### Repository Pattern

```go
type Repository[T any] struct {
    data []T
}

type Convergen interface {
    // Generic repository conversion
    RepoToDTO[T any, U any](*Repository[T]) *DTORepository[U]
}
```

### Type-Safe Wrappers

```go
type ID[T any] struct {
    Value T
}

type Convergen interface {
    // :map Value.String() StringValue
    IDToString[T fmt.Stringer](*ID[T]) *StringID
}
```

For more examples, see the complete [Examples section](index.md).