# Installation

Convergen can be used in multiple ways to fit different development workflows. Choose the method that best fits your project setup.

## Prerequisites

- **Go 1.21 or later** (required)
- **Go modules** enabled (recommended)
- **Git** (for go:generate usage)

## Installation Methods

### Method 1: go:generate Integration (Recommended)

This is the most common and recommended approach for Go projects.

**Add to your Go file:**

```go
//go:build convergen

package mypackage

import (
    "github.com/myorg/myproject/internal/domain"
    "github.com/myorg/myproject/internal/storage"
)

//go:generate go run github.com/reedom/convergen@v9.0.0-beta.1
type Convergen interface {
    DomainToStorage(*domain.User) *storage.User
}
```

**Generate code:**

```bash
go generate ./...
```

**Advantages:**
- ✅ No global installation required
- ✅ Version pinning for reproducible builds
- ✅ Integrates with standard Go tooling
- ✅ Team-friendly (same version for everyone)
- ✅ Works in CI/CD out of the box

### Method 2: CLI Tool Installation

Install Convergen as a global command-line tool.

**Install globally:**

```bash
go install github.com/reedom/convergen@latest
```

**Use directly:**

```bash
# Generate from specific file
convergen input.go

# Custom output location
convergen -out generated.go input.go

# Preview without writing files
convergen -dry input.go

# Enable verbose logging
convergen -log input.go
```

**Advantages:**
- ✅ Direct command-line usage
- ✅ Scriptable for build systems
- ✅ No go:generate setup required
- ✅ Works with any text editor

**CLI Options:**

| Flag | Description | Default |
|------|-------------|---------|
| `-out <path>` | Output file path | `<input>.gen.go` |
| `-dry` | Preview without writing files | `false` |
| `-print` | Also print output to stdout | `false` |
| `-log` | Enable detailed logging | `false` |

### Method 3: Go Module Dependency

Add Convergen as a project dependency for programmatic usage.

**Add to your project:**

```bash
go get github.com/reedom/convergen@latest
```

**Use in custom tooling:**

```go
package main

import (
    "context"
    "log"
    
    "github.com/reedom/convergen/pkg/parser"
)

func main() {
    parser, err := parser.NewParser("input.go", "output.go")
    if err != nil {
        log.Fatal(err)
    }
    
    result, err := parser.Parse(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    // Use result for custom processing
}
```

**Advantages:**
- ✅ Programmatic API access
- ✅ Custom tooling integration
- ✅ Advanced configuration options
- ✅ Enterprise workflow integration

## Version Management

### Recommended Versioning Strategy

For production use, always pin to a specific version:

```go
// ✅ Recommended: Pin to specific version
//go:generate go run github.com/reedom/convergen@v9.0.0-beta.1

// ⚠️ Development only: Use latest
//go:generate go run github.com/reedom/convergen@latest

// ✅ Alternative: Pin to major version (gets patch updates)
//go:generate go run github.com/reedom/convergen@v9
```

### Version Compatibility

| Convergen Version | Go Version | Key Features |
|-------------------|------------|--------------|
| **v9.x** | Go 1.21+ | Concurrent parsing, enhanced performance |
| **v7.x** | Go 1.18+ | Basic generics support |
| **v6.x** | Go 1.16+ | Legacy compatibility |

### Checking Your Version

```bash
# For CLI installation
convergen -version

# For go:generate usage
go run github.com/reedom/convergen@v9.0.0-beta.1 -version

# Check installed version
go list -m github.com/reedom/convergen
```

## Project Setup

### File Organization

A typical project structure:

```
myproject/
├── go.mod
├── go.sum
├── internal/
│   ├── domain/
│   │   └── user.go
│   ├── storage/
│   │   └── user.go
│   └── converters/
│       ├── converter.go          # Your interface definitions
│       └── converter.gen.go      # Generated code (auto-created)
└── cmd/
    └── myapp/
        └── main.go
```

### Build Tags

Use build tags to exclude converter definitions from regular builds:

```go
//go:build convergen

package converters

// This file is only included during code generation
//go:generate go run github.com/reedom/convergen@v9.0.0-beta.1
type Convergen interface {
    // Your conversion methods
}
```

### Makefile Integration

```makefile
.PHONY: generate clean build

generate:
	go generate ./...

clean:
	find . -name "*.gen.go" -delete

build: generate
	go build ./...

test: generate
	go test ./...
```

## Verification

### Test Your Installation

Create a simple test file to verify everything works:

**Create `test_converter.go`:**

```go
//go:build convergen

package main

type User struct {
    ID   int
    Name string
}

type UserDTO struct {
    ID   int
    Name string
}

//go:generate go run github.com/reedom/convergen@v9.0.0-beta.1
type Convergen interface {
    UserToDTO(*User) *UserDTO
}
```

**Generate and test:**

```bash
# Generate code
go generate ./...

# Verify generated file exists
ls -la test_converter.gen.go

# Test compilation
go build -o /dev/null ./...
```

**Expected output:**

```go
// Code generated by github.com/reedom/convergen
// DO NOT EDIT.

package main

func UserToDTO(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = src.ID
    dst.Name = src.Name
    return
}
```

### Common Installation Issues

=== "Go Version Too Old"

    **Error**: `go: go.mod requires go >= 1.21`
    
    **Solution**:
    ```bash
    # Check your Go version
    go version
    
    # Update Go from https://golang.org/dl/
    # Or use version manager like g or gvm
    ```

=== "Module Not Found"

    **Error**: `cannot find module providing package`
    
    **Solution**:
    ```bash
    # Ensure Go modules are enabled
    go mod init myproject
    go mod tidy
    
    # Try the command again
    go generate ./...
    ```

=== "Permission Denied"

    **Error**: `permission denied: cannot write file`
    
    **Solution**:
    ```bash
    # Check file permissions
    ls -la *.gen.go
    
    # Remove readonly generated files
    chmod +w *.gen.go
    
    # Or clean and regenerate
    find . -name "*.gen.go" -delete
    go generate ./...
    ```

## Next Steps

Now that Convergen is installed:

1. **Follow the [Quick Start guide](quick-start.md)** to create your first conversion function
2. **Explore [Basic Examples](basic-examples.md)** to see common usage patterns
3. **Review [Best Practices](../guide/best-practices.md)** for team development guidelines

## CI/CD Integration

### GitHub Actions

```yaml
name: Generate and Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Generate code
      run: go generate ./...
    
    - name: Verify no changes
      run: git diff --exit-code
    
    - name: Test
      run: go test ./...
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go generate ./...
RUN go build -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/app /usr/local/bin/
ENTRYPOINT ["app"]
```

Ready to start generating? Continue to the [Quick Start guide](quick-start.md)!
