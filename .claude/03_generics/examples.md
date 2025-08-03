# Generics Support Examples

## Overview

This document provides comprehensive examples of how Go generics will be supported in Convergen, demonstrating the syntax, capabilities, and generated code for various generic scenarios.

## Basic Generic Interfaces

### Example 1: Simple Generic Converter

**Input Interface**:
```go
//go:generate convergen

package examples

// BasicConverter provides generic type conversion
type BasicConverter[T any] interface {
    // Convert transforms a value of type T 
    // :recv conv
    Convert(src T) (T, error)
}
```

**Usage**:
```go
// Generate for specific types
//go:generate convergen -type BasicConverter[User]
//go:generate convergen -type BasicConverter[Product]
```

**Generated Code**:
```go
// Generated for BasicConverter[User]
func (conv *basicConverter) Convert(src User) (User, error) {
    var dst User
    dst.ID = src.ID
    dst.Name = src.Name
    dst.Email = src.Email
    dst.CreatedAt = src.CreatedAt
    return dst, nil
}

// Generated for BasicConverter[Product]  
func (conv *basicConverter) Convert(src Product) (Product, error) {
    var dst Product
    dst.ID = src.ID
    dst.Name = src.Name
    dst.Price = src.Price
    dst.CategoryID = src.CategoryID
    return dst, nil
}
```

### Example 2: Multi-Type Generic Mapper

**Input Interface**:
```go
//go:generate convergen

// TypeMapper converts between two different types
type TypeMapper[From any, To any] interface {
    // Map converts from source type to destination type
    // :recv mapper
    Map(src From) (To, error)
    
    // MapSlice converts slices between types
    // :recv mapper  
    MapSlice(src []From) ([]To, error)
}
```

**Usage**:
```go
//go:generate convergen -type TypeMapper[User,UserDTO]
//go:generate convergen -type TypeMapper[Product,ProductResponse]
```

**Generated Code**:
```go
// Generated for TypeMapper[User, UserDTO]
func (mapper *typeMapper) Map(src User) (UserDTO, error) {
    var dst UserDTO
    dst.ID = src.ID
    dst.FullName = src.Name // Custom field mapping
    dst.EmailAddress = src.Email
    return dst, nil
}

func (mapper *typeMapper) MapSlice(src []User) ([]UserDTO, error) {
    if src == nil {
        return nil, nil
    }
    dst := make([]UserDTO, len(src))
    for i, item := range src {
        mapped, err := mapper.Map(item)
        if err != nil {
            return nil, fmt.Errorf("failed to map item at index %d: %w", i, err)
        }
        dst[i] = mapped
    }
    return dst, nil
}
```

## Constrained Generics

### Example 3: Numeric Type Converter

**Input Interface**:
```go
//go:generate convergen

// NumericConverter handles numeric type conversions
type NumericConverter[T ~int | ~int32 | ~int64 | ~float32 | ~float64] interface {
    // ConvertToString converts numeric values to strings
    // :recv conv
    ConvertToString(src T) (string, error)
    
    // ConvertFromString parses strings to numeric values
    // :recv conv
    ConvertFromString(src string) (T, error)
}
```

**Generated Code**:
```go
// Generated for NumericConverter[int64]
func (conv *numericConverter) ConvertToString(src int64) (string, error) {
    return strconv.FormatInt(src, 10), nil
}

func (conv *numericConverter) ConvertFromString(src string) (int64, error) {
    result, err := strconv.ParseInt(src, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("failed to parse int64: %w", err)
    }
    return result, nil
}

// Generated for NumericConverter[float64]
func (conv *numericConverter) ConvertToString(src float64) (string, error) {
    return strconv.FormatFloat(src, 'f', -1, 64), nil
}

func (conv *numericConverter) ConvertFromString(src string) (float64, error) {
    result, err := strconv.ParseFloat(src, 64)
    if err != nil {
        return 0.0, fmt.Errorf("failed to parse float64: %w", err)
    }
    return result, nil
}
```

### Example 4: Comparable Types

**Input Interface**:
```go
//go:generate convergen

// ComparableConverter works with comparable types for caching/deduplication
type ComparableConverter[T comparable] interface {
    // ConvertWithCache converts with deduplication
    // :recv conv
    ConvertWithCache(src T) (T, error)
    
    // ConvertUnique converts only unique values
    // :recv conv
    ConvertUnique(src []T) ([]T, error)
}
```

**Generated Code**:
```go
// Generated for ComparableConverter[string]
func (conv *comparableConverter) ConvertWithCache(src string) (string, error) {
    // Check cache first
    if cached, exists := conv.cache[src]; exists {
        return cached, nil
    }
    
    // Perform conversion
    var dst string
    dst = strings.ToUpper(src) // Example transformation
    
    // Cache result
    conv.cache[src] = dst
    return dst, nil
}

func (conv *comparableConverter) ConvertUnique(src []string) ([]string, error) {
    seen := make(map[string]bool)
    var unique []string
    
    for _, item := range src {
        if !seen[item] {
            converted, err := conv.ConvertWithCache(item)
            if err != nil {
                return nil, err
            }
            unique = append(unique, converted)
            seen[item] = true
        }
    }
    
    return unique, nil
}
```

## Advanced Generic Patterns

### Example 5: Generic Collections

**Input Interface**:
```go
//go:generate convergen

// CollectionConverter handles various collection types
type CollectionConverter[T any] interface {
    // ConvertSlice converts slices
    // :recv conv
    ConvertSlice(src []T) ([]T, error)
    
    // ConvertMap converts maps with string keys  
    // :recv conv
    ConvertMap(src map[string]T) (map[string]T, error)
    
    // ConvertPointer handles pointer conversions
    // :recv conv
    ConvertPointer(src *T) (*T, error)
}
```

**Generated Code**:
```go
// Generated for CollectionConverter[User]
func (conv *collectionConverter) ConvertSlice(src []User) ([]User, error) {
    if src == nil {
        return nil, nil
    }
    
    dst := make([]User, len(src))
    for i, item := range src {
        converted, err := conv.convertSingleUser(item)
        if err != nil {
            return nil, fmt.Errorf("failed to convert item at index %d: %w", i, err)
        }
        dst[i] = converted
    }
    return dst, nil
}

func (conv *collectionConverter) ConvertMap(src map[string]User) (map[string]User, error) {
    if src == nil {
        return nil, nil
    }
    
    dst := make(map[string]User, len(src))
    for key, value := range src {
        converted, err := conv.convertSingleUser(value)
        if err != nil {
            return nil, fmt.Errorf("failed to convert value for key %q: %w", key, err)
        }
        dst[key] = converted
    }
    return dst, nil
}

func (conv *collectionConverter) ConvertPointer(src *User) (*User, error) {
    if src == nil {
        return nil, nil
    }
    
    converted, err := conv.convertSingleUser(*src)
    if err != nil {
        return nil, err
    }
    return &converted, nil
}
```

### Example 6: Generic with Annotations

**Input Interface**:
```go
//go:generate convergen

// AdvancedConverter demonstrates annotations with generics
type AdvancedConverter[T any, U any] interface {
    // Convert with custom field mappings
    // :recv conv
    // :map ID UserID  
    // :map Name FullName
    // :skip InternalField
    Convert(src T) (U, error)
    
    // ConvertWithValidation includes validation step
    // :recv conv
    // :conv validateEmail Email EmailAddress
    ConvertWithValidation(src T) (U, error)
}
```

**Generated Code**:
```go
// Generated for AdvancedConverter[User, UserDTO]
func (conv *advancedConverter) Convert(src User) (UserDTO, error) {
    var dst UserDTO
    dst.UserID = src.ID          // :map annotation applied
    dst.FullName = src.Name      // :map annotation applied
    dst.Email = src.Email
    // InternalField skipped due to :skip annotation
    return dst, nil
}

func (conv *advancedConverter) ConvertWithValidation(src User) (UserDTO, error) {
    var dst UserDTO
    dst.UserID = src.ID
    dst.FullName = src.Name
    
    // Apply :conv annotation for email validation
    emailAddress, err := conv.validateEmail(src.Email)
    if err != nil {
        return UserDTO{}, fmt.Errorf("email validation failed: %w", err)
    }
    dst.EmailAddress = emailAddress
    
    return dst, nil
}
```

## Complex Constraint Examples

### Example 7: Multiple Constraints

**Input Interface**:
```go
//go:generate convergen

// MultiConstraintConverter demonstrates complex constraints
type MultiConstraintConverter[
    K comparable,                    // Key must be comparable
    V ~string | ~int | ~float64,    // Value must be one of these underlying types
    M ~map[K]V,                     // Map type with constrained key/value
] interface {
    // ConvertMap converts between map types
    // :recv conv
    ConvertMap(src M) (M, error)
    
    // ExtractKeys gets all keys from the map
    // :recv conv
    ExtractKeys(src M) ([]K, error)
    
    // ExtractValues gets all values from the map
    // :recv conv  
    ExtractValues(src M) ([]V, error)
}
```

**Generated Code**:
```go
// Generated for MultiConstraintConverter[string, int, map[string]int]
func (conv *multiConstraintConverter) ConvertMap(src map[string]int) (map[string]int, error) {
    if src == nil {
        return nil, nil
    }
    
    dst := make(map[string]int, len(src))
    for key, value := range src {
        // Apply any necessary transformations
        convertedKey := conv.processKey(key)
        convertedValue := conv.processValue(value)
        dst[convertedKey] = convertedValue
    }
    return dst, nil
}

func (conv *multiConstraintConverter) ExtractKeys(src map[string]int) ([]string, error) {
    if src == nil {
        return nil, nil
    }
    
    keys := make([]string, 0, len(src))
    for key := range src {
        keys = append(keys, key)
    }
    return keys, nil
}

func (conv *multiConstraintConverter) ExtractValues(src map[string]int) ([]int, error) {
    if src == nil {
        return nil, nil
    }
    
    values := make([]int, 0, len(src))
    for _, value := range src {
        values = append(values, value)
    }
    return values, nil
}
```

### Example 8: Interface Constraints

**Input Interface**:
```go
//go:generate convergen

// Serializable defines a serialization interface
type Serializable interface {
    Serialize() ([]byte, error)
    Deserialize([]byte) error
}

// SerializableConverter works with types that implement Serializable
type SerializableConverter[T Serializable] interface {
    // ConvertViaSerialization converts using serialization round-trip
    // :recv conv
    ConvertViaSerialization(src T) (T, error)
    
    // ConvertBatch processes multiple serializable items
    // :recv conv
    ConvertBatch(src []T) ([]T, error)
}
```

**Generated Code**:
```go
// Generated for SerializableConverter[SerializableUser] 
// where SerializableUser implements Serializable
func (conv *serializableConverter) ConvertViaSerialization(src SerializableUser) (SerializableUser, error) {
    // Serialize source
    data, err := src.Serialize()
    if err != nil {
        return SerializableUser{}, fmt.Errorf("serialization failed: %w", err)
    }
    
    // Create new instance and deserialize
    var dst SerializableUser
    if err := dst.Deserialize(data); err != nil {
        return SerializableUser{}, fmt.Errorf("deserialization failed: %w", err)
    }
    
    return dst, nil
}

func (conv *serializableConverter) ConvertBatch(src []SerializableUser) ([]SerializableUser, error) {
    if src == nil {
        return nil, nil
    }
    
    dst := make([]SerializableUser, len(src))
    for i, item := range src {
        converted, err := conv.ConvertViaSerialization(item)
        if err != nil {
            return nil, fmt.Errorf("failed to convert item at index %d: %w", i, err)
        }
        dst[i] = converted
    }
    return dst, nil
}
```

## Error Handling Examples

### Example 9: Generic Error Handling

**Input Interface**:
```go
//go:generate convergen

// Result represents a result that can be success or error
type Result[T any] interface {
    IsSuccess() bool
    GetValue() T
    GetError() error
}

// ResultConverter handles Result type conversions
type ResultConverter[T any, E error] interface {
    // ConvertResult converts results with error handling
    // :recv conv
    ConvertResult(src Result[T]) (Result[T], error)
    
    // ConvertWithFallback provides fallback on error
    // :recv conv
    ConvertWithFallback(src T, fallback T) (T, error)
}
```

**Generated Code**:
```go
// Generated for ResultConverter[User, ValidationError]
func (conv *resultConverter) ConvertResult(src Result[User]) (Result[User], error) {
    if !src.IsSuccess() {
        // Propagate error result
        return src, nil
    }
    
    // Convert the success value
    value := src.GetValue()
    converted, err := conv.convertUser(value)
    if err != nil {
        return createErrorResult[User](err), nil
    }
    
    return createSuccessResult(converted), nil
}

func (conv *resultConverter) ConvertWithFallback(src User, fallback User) (User, error) {
    converted, err := conv.convertUser(src)
    if err != nil {
        // Log error and return fallback
        conv.logger.Warn("conversion failed, using fallback", 
            zap.Error(err),
            zap.Any("fallback", fallback))
        return fallback, nil
    }
    return converted, nil
}
```

## Migration Examples

### Example 10: Migrating Existing Interfaces

**Before (Non-Generic)**:
```go
//go:generate convergen

// UserConverter - specific to User type
type UserConverter interface {
    // :recv conv
    ConvertUser(src User) (User, error)
}

// ProductConverter - specific to Product type  
type ProductConverter interface {
    // :recv conv
    ConvertProduct(src Product) (Product, error)
}
```

**After (Generic)**:
```go
//go:generate convergen

// UniversalConverter - works with any type
type UniversalConverter[T any] interface {
    // :recv conv
    Convert(src T) (T, error)
}

// Usage remains similar:
//go:generate convergen -type UniversalConverter[User]
//go:generate convergen -type UniversalConverter[Product]
```

**Migration Benefits**:
- **Code Reuse**: Single interface definition for multiple types
- **Type Safety**: Compile-time type checking
- **Maintainability**: Less code duplication
- **Consistency**: Uniform interface across all types

## Performance Considerations

### Example 11: Optimized Generic Code

**Input Interface**:
```go
//go:generate convergen

// OptimizedConverter demonstrates performance-optimized patterns
type OptimizedConverter[T any] interface {
    // Convert with pre-allocated destination
    // :recv conv
    // :preprocess preallocateDestination
    Convert(src T) (T, error)
    
    // ConvertBatch with batching optimization
    // :recv conv
    // :postprocess optimizeBatch
    ConvertBatch(src []T, batchSize int) ([]T, error)
}
```

**Generated Code** (optimized):
```go
// Generated with performance optimizations
func (conv *optimizedConverter) Convert(src User) (User, error) {
    // Pre-allocated destination (from :preprocess)
    dst := conv.preallocateDestination()
    
    // Optimized field assignments
    dst.ID = src.ID
    dst.Name = src.Name
    dst.Email = src.Email
    
    return dst, nil
}

func (conv *optimizedConverter) ConvertBatch(src []User, batchSize int) ([]User, error) {
    if src == nil {
        return nil, nil
    }
    
    // Pre-allocate full result slice
    dst := make([]User, 0, len(src))
    
    // Process in batches for memory efficiency
    for i := 0; i < len(src); i += batchSize {
        end := min(i+batchSize, len(src))
        batch := src[i:end]
        
        for _, item := range batch {
            converted, err := conv.Convert(item)
            if err != nil {
                return nil, fmt.Errorf("batch processing failed at index %d: %w", i, err)
            }
            dst = append(dst, converted)
        }
    }
    
    // Apply post-processing optimization
    return conv.optimizeBatch(dst), nil
}
```

## Testing Examples

### Example 12: Generic Interface Testing

**Test Code**:
```go
func TestGenericConverter(t *testing.T) {
    // Test with User type
    t.Run("User conversion", func(t *testing.T) {
        conv := &basicConverter{}
        user := User{ID: 1, Name: "John", Email: "john@example.com"}
        
        result, err := conv.Convert(user)
        require.NoError(t, err)
        assert.Equal(t, user.ID, result.ID)
        assert.Equal(t, user.Name, result.Name)
        assert.Equal(t, user.Email, result.Email)
    })
    
    // Test with Product type
    t.Run("Product conversion", func(t *testing.T) {
        conv := &basicConverter{}
        product := Product{ID: 1, Name: "Widget", Price: 9.99}
        
        result, err := conv.Convert(product)
        require.NoError(t, err)
        assert.Equal(t, product.ID, result.ID)
        assert.Equal(t, product.Name, result.Name)
        assert.Equal(t, product.Price, result.Price)
    })
}
```

This comprehensive set of examples demonstrates the power and flexibility of generics support in Convergen, showing how developers can write type-safe, reusable conversion interfaces that generate efficient, correct code for any type combination.