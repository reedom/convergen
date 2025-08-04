# Migration Guide

Guide for upgrading between Convergen versions, including breaking changes and compatibility information.

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