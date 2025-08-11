# Migration Guide

Guide for upgrading between Convergen versions, including breaking changes and compatibility information.

## Migration to v8.1 - Major Feature Update

### 🔥 New Features in v8.1

**Struct Literal Generation (90% complete):**
- Automatic detection with intelligent fallback
- `:struct-literal` / `:no-struct-literal` annotations
- CLI flags: `--struct-literal` / `--no-struct-literal`
- Performance benefits for simple conversions

**Enhanced Generics Support (85% complete):**
- Cross-package type resolution with `go/packages`
- Generic constraint validation (any, comparable, union, interface)
- New CLI flags: `-type` and `-imports` for cross-package scenarios
- Type compatibility checking with detailed error reporting
- LRU caching and performance optimization

### Backward Compatibility

**✅ Fully Backward Compatible** - All existing v8.0.x code continues to work:
- Existing annotations unchanged
- Generated code maintains same API
- No breaking changes to interface definitions
- Default behavior unchanged (struct literals opt-in)

### Upgrade Benefits

**Performance Improvements:**
- Struct literal generation for cleaner, faster code
- Enhanced parsing performance with concurrent processing
- LRU caching for cross-package type resolution

**Developer Experience:**
- Better error messages with constraint validation
- Cross-package generics support
- Automatic fallback detection
- Rich CLI integration

### Migration Steps for v8.1 Features

#### 1. Adopting Struct Literals

**Step 1:** Test with existing code (no changes needed)
```bash
# Your existing code works unchanged
go generate ./...
```

**Step 2:** Enable struct literals globally (optional)
```bash
# Try struct literals for better performance
convergen --struct-literal your-converter.go
```

**Step 3:** Add selective control (optional)
```go
type Convergen interface {
    // Force struct literal for simple conversions
    // :struct-literal
    SimpleConvert(*User) *UserDTO
    
    // Keep assignment block for complex conversions  
    // :no-struct-literal
    ComplexConvert(*User) (*UserDTO, error)
}
```

#### 2. Adopting Cross-Package Generics

**Before (single package):**
```go
package converters

type Convergen interface {
    Convert(*User) *UserDTO
}
```

**After (cross-package generics):**
```go
//go:build convergen

package converters

type UserConverter[S, D any] interface {
    Convert(S) D
    ConvertSlice([]S) []D
}
```

**Generation command:**
```bash
convergen -type 'UserConverter[models.User,dto.UserDTO]' \
          -imports 'models=./internal/models,dto=./api/dto' \
          -struct-literal \
          user_converter.go
```

**go:generate integration:**
```go
//go:generate convergen -type UserConverter[models.User,dto.UserDTO] -imports models=./internal/models,dto=./api/dto $GOFILE
```

#### 3. Validating Migration

**Test compilation:**
```bash
go build ./...
```

**Test generated code:**
```bash
go test ./...
```

**Check performance (optional):**
```bash
# Before struct literals
go test -bench=BenchmarkConvert .

# After struct literals  
convergen --struct-literal converter.go
go test -bench=BenchmarkConvert .
```

### Common Migration Patterns

#### Pattern 1: Simple Domain-to-DTO Conversion

**Before:**
```go
type Convergen interface {
    UserToDTO(*domain.User) *dto.UserDTO
}
```

**After (with struct literals):**
```go
type Convergen interface {
    // :struct-literal
    // :typecast
    UserToDTO(*domain.User) *dto.UserDTO
}
```

**Generated code improvement:**
```go
// Before (assignment block)
func UserToDTO(src *domain.User) (dst *dto.UserDTO) {
    dst = &dto.UserDTO{}
    dst.ID = int64(src.ID)
    dst.Name = src.Name
    return
}

// After (struct literal)
func UserToDTO(src *domain.User) *dto.UserDTO {
    return &dto.UserDTO{
        ID:   int64(src.ID),
        Name: src.Name,
    }
}
```

#### Pattern 2: Cross-Package Generic Conversion

**Before (package-specific):**
```go
package userconverters

import (
    "myproject/internal/models"
    "myproject/api/dto"
)

type Convergen interface {
    Convert(*models.User) *dto.UserDTO
    ConvertSlice([]*models.User) []*dto.UserDTO
}
```

**After (generic):**
```go
//go:build convergen

package converters

//go:generate convergen -type UserConverter[models.User,dto.UserDTO] -imports models=./internal/models,dto=./api/dto $GOFILE

type UserConverter[S, D any] interface {
    // :struct-literal
    Convert(S) D
    ConvertSlice([]S) []D
}
```

#### Pattern 3: Error Handling with Struct Literals

**Automatic fallback for error scenarios:**
```go
type Convergen interface {
    // Simple conversion - uses struct literal
    // :struct-literal
    SimpleConvert(*User) *UserDTO
    
    // Error handling - automatically falls back to assignment block
    // :struct-literal  
    // :conv validateEmail Email
    SafeConvert(*User) (*UserDTO, error)
}
```

### Troubleshooting Migration

#### Struct Literal Issues

**Issue:** Struct literal forced but incompatible with complex conversion
```bash
Error: method ConvertUser: method is forced to use struct literal but has incompatible features: preprocess annotation requires imperative execution before struct creation
```

**Solution:** Remove `:struct-literal` annotation or use `:no-struct-literal`
```go
// Instead of forcing struct literal
// :struct-literal
// :preprocess validate
ConvertUser(*User) (*UserDTO, error)

// Use assignment block
// :no-struct-literal
// :preprocess validate  
ConvertUser(*User) (*UserDTO, error)
```

#### Cross-Package Type Resolution Issues

**Issue:** Cannot resolve package types
```bash
Error: cannot resolve package "models" at path "./internal/models"
```

**Solutions:**
1. Check package path: `ls ./internal/models`
2. Verify package compiles: `go build ./internal/models`
3. Update import mapping: `-imports 'models=./internal/models'`

**Issue:** Type compatibility warnings
```bash
Warning: Field mismatch detected between models.User and dto.Product
```

**Solution:** Add explicit field mappings
```go
type Converter[S, D any] interface {
    // :map Name ProductName
    // :skip Password
    Convert(S) D
}
```

## Migration from v7 to v8

### Performance Improvements

v8 introduces significant performance improvements:

- 40-70% faster parsing with concurrent processing
- Smart strategy selection
- Enterprise-grade reliability features

### Compatibility

v8 is **fully backward compatible** with v7 code:

- All existing annotations work unchanged
- Generated code maintains same API
- No breaking changes to interface definitions

### Requirements

- **Go 1.21+** (updated from Go 1.18+ in v7)

### Upgrade Steps

1. **Update Go version to 1.21+**
2. **Update Convergen version**:
   ```bash
   # For go:generate
   //go:generate go run github.com/reedom/convergen@v8.0.3
   
   # For CLI
   go install github.com/reedom/convergen@latest
   ```
3. **Regenerate code**:
   ```bash
   go generate ./...
   ```
4. **Test compilation**:
   ```bash
   go build ./...
   go test ./...
   ```

### New Features in v8

- Concurrent parser engine
- Enhanced error handling
- Improved performance metrics
- Better debugging capabilities

## Migration from v6 to v7

### Generics Support

v7 introduced basic generics support:

- Generic type parameters in interfaces
- Type constraints
- Collection transformations

### Breaking Changes

- Minimum Go version increased to 1.18
- Some internal APIs changed (affects only programmatic usage)

## Version Compatibility Matrix

| Convergen Version | Go Version | Key Features |
|-------------------|------------|--------------|
| **v8.x** | Go 1.21+ | Concurrent parsing, enhanced performance |
| **v7.x** | Go 1.18+ | Basic generics support |
| **v6.x** | Go 1.16+ | Legacy compatibility |

## Getting Help

For migration assistance:

- [Common Issues](common-issues.md) - Troubleshooting migration problems
- [GitHub Issues](https://github.com/reedom/convergen/issues) - Report migration bugs
- [Examples](../examples/index.md) - Updated examples for latest version

*Detailed migration scenarios and troubleshooting coming soon.*