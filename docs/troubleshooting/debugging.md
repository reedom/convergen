# Debugging Guide

Advanced debugging techniques for Convergen issues, including detailed error analysis and troubleshooting workflows.

## Coming Soon

This section will cover:

- Enabling debug logging and verbose output
- Understanding error messages and stack traces
- Debugging generated code issues
- Parser and compilation problems
- Runtime behavior analysis

*This page is currently being developed with comprehensive debugging techniques.*

## Quick Debug Commands

### Enable Verbose Logging

```bash
# Generate with detailed logging
convergen -log converter.go

# View log file
cat converter.gen.go.log
```

### Preview Generated Code

```bash
# See what would be generated
convergen -dry -print converter.go
```

### Common Debug Patterns

#### Compilation Issues

Check generated code and imports:

```bash
# Test compilation
go build -o /dev/null ./...

# Check for missing imports
go list -e -json ./...
```

#### Runtime Issues

Add nil checks and validation:

```go
func SafeConvert(src *User) *UserDTO {
    if src == nil {
        return nil
    }
    return UserToDTO(src)  // Generated function
}
```

For more troubleshooting help, see [Common Issues](common-issues.md).