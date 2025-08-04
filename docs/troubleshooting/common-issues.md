# Common Issues

Solutions to frequently encountered problems when using Convergen, organized by category with step-by-step troubleshooting guides.

## Installation Issues

### Go Version Compatibility

**Problem:** `go: convergen requires go >= 1.21`

**Solution:**

```bash
# Check your Go version
go version

# Update Go if needed (visit https://golang.org/dl/)
# Or use a Go version manager like gvm
```

**For older Go versions:**

| Convergen Version | Go Version | Use Command |
|-------------------|------------|-------------|
| v8.x | Go 1.21+ | `@latest` |
| v7.x | Go 1.18+ | `@v7.2.1` |
| v6.x | Go 1.16+ | `@v6.3.0` |

### Module Download Issues

**Problem:** `cannot find module providing package github.com/reedom/convergen`

**Solutions:**

```bash
# Ensure Go modules are enabled
go mod init myproject

# Try direct download
go mod download github.com/reedom/convergen

# Clear module cache if corrupted
go clean -modcache
go mod download

# Check GOPROXY settings
go env GOPROXY
```

## Generation Issues {#generation-problems}

### Interface Not Found

**Problem:** `no converter interfaces found in converter.go`

**Common Causes & Solutions:**

=== "Wrong Interface Name"

    **Issue:** Interface not named "Convergen"
    
    **Solution:**
    ```go
    // ❌ Wrong - not recognized
    type MyConverter interface {
        Convert(*User) *UserDTO
    }
    
    // ✅ Correct - auto-recognized
    type Convergen interface {
        Convert(*User) *UserDTO
    }
    
    // ✅ Alternative - use annotation
    // :convergen
    type MyConverter interface {
        Convert(*User) *UserDTO
    }
    ```

=== "Unexported Interface"

    **Issue:** Interface name starts with lowercase
    
    **Solution:**
    ```go
    // ❌ Wrong - unexported
    type convergen interface {
        Convert(*User) *UserDTO
    }
    
    // ✅ Correct - exported
    type Convergen interface {
        Convert(*User) *UserDTO
    }
    ```

=== "Missing Build Tag"

    **Issue:** Interface in file without proper build tag
    
    **Solution:**
    ```go
    //go:build convergen
    
    package mypackage
    
    // Now the interface will be found
    type Convergen interface {
        Convert(*User) *UserDTO
    }
    ```

### Type Resolution Errors

**Problem:** `cannot resolve type "User" in converter.go:15`

**Diagnostic Steps:**

```bash
# 1. Check if type is imported
grep -n "import" converter.go

# 2. Verify type accessibility
go doc github.com/myorg/myproject/domain.User

# 3. Check if type is exported
grep -n "type User" domain/*.go
```

**Common Solutions:**

=== "Missing Import"

    ```go
    // ❌ Missing import
    type Convergen interface {
        Convert(*User) *UserDTO  // User not found
    }
    
    // ✅ Add import
    import "github.com/myorg/myproject/domain"
    
    type Convergen interface {
        Convert(*domain.User) *UserDTO
    }
    ```

=== "Unexported Type"

    ```go
    // ❌ In domain package - unexported
    type user struct {
        ID   int
        Name string
    }
    
    // ✅ Export the type
    type User struct {
        ID   int
        Name string
    }
    ```

=== "Circular Import"

    **Problem:** Converter package imports domain, domain imports converter
    
    **Solution:** Move converters to separate package
    ```
    project/
    ├── domain/
    │   └── user.go
    ├── api/
    │   └── user.go
    └── converters/        # Separate package
        ├── converter.go
        └── converter.gen.go
    ```

### Annotation Syntax Errors

**Problem:** `invalid annotation syntax at converter.go:12`

**Common Syntax Issues:**

=== "Missing Parameters"

    ```go
    // ❌ Wrong - missing destination
    // :map Name
    Convert(*User) *UserDTO
    
    // ✅ Correct - include destination  
    // :map Name FullName
    Convert(*User) *UserDTO
    ```

=== "Incorrect Format"

    ```go
    // ❌ Wrong - missing colon
    // skip Password
    Convert(*User) *UserDTO
    
    // ✅ Correct - starts with colon
    // :skip Password
    Convert(*User) *UserDTO
    ```

=== "Invalid Field Path"

    ```go
    // ❌ Wrong - field doesn't exist
    // :map NonExistentField Name
    Convert(*User) *UserDTO
    
    // ✅ Correct - valid field path
    // :map FirstName Name
    Convert(*User) *UserDTO
    ```

## Compilation Issues

### Generated Code Won't Compile

**Problem:** `undefined: SomeType` in generated code

**Diagnostic Commands:**

```bash
# Check generated code
cat converter.gen.go

# Test compilation
go build -o /dev/null ./...

# Check for missing imports
go list -e -json ./... | jq '.Error'
```

**Common Solutions:**

=== "Missing Imports in Generated Code"

    **Issue:** Generated code lacks required imports
    
    **Solution:** Ensure all types are properly imported in converter file
    ```go
    // Add all necessary imports
    import (
        "time"
        "github.com/myorg/myproject/domain"
        "github.com/myorg/myproject/api"
        "github.com/myorg/myproject/storage"
    )
    ```

=== "Type Conversion Issues"

    **Issue:** Incompatible types in assignment
    
    **Solution:** Use type conversion annotations
    ```go
    // Enable automatic type casting
    // :typecast
    type Convergen interface {
        Convert(*User) *UserDTO
    }
    ```

### Custom Converter Function Issues

**Problem:** `undefined: customConverter` in generated code

**Solutions:**

=== "Function Not Accessible"

    ```go
    // ❌ Wrong - unexported function
    func customConverter(s string) string {
        return strings.ToUpper(s)
    }
    
    // ✅ Correct - export function or define in same file
    func CustomConverter(s string) string {
        return strings.ToUpper(s)
    }
    
    type Convergen interface {
        // :conv CustomConverter Name
        Convert(*User) *UserDTO
    }
    ```

=== "Wrong Function Signature"

    ```go
    // ❌ Wrong - doesn't match expected signature
    func CustomConverter() string {
        return "default"
    }
    
    // ✅ Correct - matches field type
    func CustomConverter(input string) string {
        return strings.ToUpper(input)
    }
    
    // ✅ With error handling
    func CustomConverter(input string) (string, error) {
        if input == "" {
            return "", errors.New("empty input")
        }
        return strings.ToUpper(input), nil
    }
    ```

## Runtime Issues

### Nil Pointer Dereference

**Problem:** `panic: runtime error: invalid memory address or nil pointer dereference`

**Solutions:**

=== "Nil Source Object"

    ```go
    // Add nil checks in your code
    func Convert(src *User) *UserDTO {
        if src == nil {
            return nil  // or return empty object
        }
        return UserToDTO(src)  // Generated function
    }
    ```

=== "Nil Nested Objects"

    ```go
    // Use safe navigation in custom converters
    func SafeGetAddress(user *User) string {
        if user == nil || user.Address == nil {
            return ""
        }
        return user.Address.Street
    }
    
    type Convergen interface {
        // :conv SafeGetAddress Address
        Convert(*User) *UserDTO
    }
    ```

### Type Assertion Failures

**Problem:** `panic: interface conversion: interface {} is nil, not string`

**Solution:** Add type safety in custom converters

```go
func SafeStringConvert(v interface{}) string {
    if v == nil {
        return ""
    }
    
    switch val := v.(type) {
    case string:
        return val
    case fmt.Stringer:
        return val.String()
    default:
        return fmt.Sprintf("%v", val)
    }
}
```

## Performance Issues {#performance-issues}

### Slow Generation

**Problem:** Code generation takes too long

**Solutions:**

=== "Update to v8"

    ```bash
    # Update to latest version for performance improvements
    go get github.com/reedom/convergen@latest
    ```

=== "Optimize Project Structure"

    ```bash
    # Avoid deep package hierarchies
    # Minimize cross-package dependencies
    # Use specific imports instead of wildcard imports
    ```

=== "Use Concurrent Parser (v8)"

    **Automatic in v8, but you can configure:**
    ```go
    // For programmatic usage
    config := parser.NewConcurrentParserConfig()
    modernParser := parser.NewModernParser(config)
    ```

### Large Generated Files

**Problem:** Generated files are too large

**Solutions:**

=== "Split Interfaces"

    ```go
    // ❌ One large interface
    type Convergen interface {
        // 50+ methods here
    }
    
    // ✅ Split by domain
    // :convergen
    type UserConvergen interface {
        // User-related conversions
    }
    
    // :convergen
    type OrderConvergen interface {
        // Order-related conversions
    }
    ```

=== "Use Multiple Files"

    ```
    converters/
    ├── user_converter.go
    ├── user_converter.gen.go
    ├── order_converter.go
    └── order_converter.gen.go
    ```

## Integration Issues

### CI/CD Pipeline Failures

**Problem:** Generation works locally but fails in CI

**Common Solutions:**

=== "Go Version Mismatch"

    ```yaml
    # Ensure consistent Go version
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'  # Match your local version
    ```

=== "Missing Dependencies"

    ```yaml
    # Install Convergen in CI
    - name: Install Convergen
      run: go install github.com/reedom/convergen@v8.0.3
    ```

=== "File Permissions"

    ```bash
    # Make sure files are writable
    chmod +w *.gen.go
    
    # Or clean before generation
    find . -name "*.gen.go" -delete
    ```

### Docker Build Issues

**Problem:** Generation fails in Docker container

**Solution:**

```dockerfile
# Multi-stage build
FROM golang:1.21-alpine AS builder

# Install Convergen
RUN go install github.com/reedom/convergen@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate code
RUN find . -name "*_converter.go" -exec convergen {} \;

# Build app
RUN go build -o app .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/app /usr/local/bin/
ENTRYPOINT ["app"]
```

## Debugging Techniques

### Enable Verbose Logging

```bash
# Generate with detailed logs
convergen -log converter.go

# View the log
cat converter.gen.go.log
```

### Preview Generated Code

```bash
# See what would be generated
convergen -dry -print converter.go

# Save for analysis
convergen -dry -print converter.go > preview.go
```

### Step-by-Step Debugging

```bash
# 1. Verify file syntax
go fmt converter.go

# 2. Check interfaces
grep -A 10 "interface" converter.go

# 3. Test parsing
convergen -dry converter.go

# 4. Check generated code
convergen -dry -print converter.go

# 5. Test compilation
go build -o /dev/null ./...
```

## Getting Help

If you're still experiencing issues:

1. **Check the logs:** Enable `-log` flag for detailed information
2. **Search existing issues:** [GitHub Issues](https://github.com/reedom/convergen/issues)
3. **Create minimal reproduction:** Strip down to simplest failing case
4. **Provide environment details:** Go version, OS, Convergen version

### Issue Template

When reporting issues, include:

```markdown
**Environment:**
- Go version: `go version`
- Convergen version: `convergen -version`
- OS: `uname -a`

**Input file:**
```go
// Minimal example that reproduces the issue
```

**Expected behavior:**
What you expected to happen

**Actual behavior:**
What actually happened

**Error output:**
```
Complete error message and stack trace
```

**Additional context:**
Any other relevant information
```

## Next Steps

- **Advanced debugging**: [Debugging Guide](debugging.md)
- **Version migration**: [Migration Guide](migration.md)
- **Best practices**: [Best Practices](../guide/best-practices.md)