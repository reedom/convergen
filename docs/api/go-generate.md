# go:generate Integration

Complete guide for using Convergen with Go's built-in code generation system.

## Overview

The `go:generate` integration is the **recommended approach** for most Go projects. It provides:

- ✅ **Version pinning** for reproducible builds
- ✅ **Standard Go tooling** integration
- ✅ **Team consistency** - same version for everyone
- ✅ **CI/CD friendly** - works out of the box

## Basic Setup

### 1. Add go:generate Directive

```go
//go:build convergen

package mypackage

import (
    "github.com/myorg/myproject/domain"
    "github.com/myorg/myproject/storage"
)

//go:generate go run github.com/reedom/convergen@v8.0.3
type Convergen interface {
    DomainToStorage(*domain.User) *storage.User
}
```

### 2. Generate Code

```bash
# Generate for current package
go generate

# Generate for all packages
go generate ./...

# Generate for specific packages
go generate ./internal/converters/...
```

## Version Management

### Recommended Versioning

```go
// ✅ Recommended: Pin to specific version for stability
//go:generate go run github.com/reedom/convergen@v8.0.3

// ✅ Alternative: Pin to major version (gets patches automatically)
//go:generate go run github.com/reedom/convergen@v8

// ⚠️ Development only: Use latest (not recommended for production)
//go:generate go run github.com/reedom/convergen@latest
```

### Version Updates

```bash
# Update to latest v8.x
find . -name "*.go" -exec sed -i 's/@v8\.[0-9]\+\.[0-9]\+/@v8.0.3/g' {} +

# Or update to latest
find . -name "*.go" -exec sed -i 's/@v8\.[0-9]\+\.[0-9]\+/@latest/g' {} +
```

## Build Tags

### Isolate Generator Code

```go
//go:build convergen

package converters

// This file is only included during code generation
// and won't interfere with regular builds

//go:generate go run github.com/reedom/convergen@v8.0.3
type Convergen interface {
    Convert(*User) *UserDTO
}
```

### Multiple Build Tags

```go
//go:build convergen && !release

package converters

// Only include in development builds, skip in release
//go:generate go run github.com/reedom/convergen@v8.0.3
type Convergen interface {
    // Development-specific converters
}
```

## Project Organization

### Single File per Domain

```
internal/
├── converters/
│   ├── user_converter.go      # User conversions
│   ├── user_converter.gen.go  # Generated
│   ├── order_converter.go     # Order conversions
│   └── order_converter.gen.go # Generated
```

**user_converter.go:**
```go
//go:build convergen

package converters

//go:generate go run github.com/reedom/convergen@v8.0.3
type UserConvergen interface {
    // :skip Password
    UserToAPI(*domain.User) *api.User
    
    // :typecast
    UserToStorage(*domain.User) *storage.User
}
```

### Multiple Interfaces per File

```go
//go:build convergen

package converters

//go:generate go run github.com/reedom/convergen@v8.0.3

// :convergen
type UserConvergen interface {
    UserToAPI(*domain.User) *api.User
}

// :convergen
type OrderConvergen interface {
    OrderToAPI(*domain.Order) *api.Order
}
```

### Package-Level Organization

```
internal/
├── user/
│   ├── converter.go        # User-specific
│   └── converter.gen.go
├── order/
│   ├── converter.go        # Order-specific
│   └── converter.gen.go
└── shared/
    ├── converter.go        # Cross-domain
    └── converter.gen.go
```

## Integration Patterns

### Makefile Integration

```makefile
.PHONY: generate clean build test

generate:
	go generate ./...

clean:
	find . -name "*.gen.go" -delete

build: generate
	go build ./...

test: generate
	go test ./...
```

### Just Integration

```just
# Generate code
generate:
    go generate ./...

# Clean generated files
clean:
    find . -name "*.gen.go" -delete

# Build with generation
build: generate
    go build ./...

# Test with generation
test: generate
    go test ./...
```

### Taskfile Integration

```yaml
# Taskfile.yml
version: '3'

tasks:
  generate:
    desc: Generate code
    cmds:
      - go generate ./...

  clean:
    desc: Clean generated files
    cmds:
      - find . -name "*.gen.go" -delete

  build:
    desc: Build with generation
    deps: [generate]
    cmds:
      - go build ./...

  test:
    desc: Test with generation
    deps: [generate]
    cmds:
      - go test ./...
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build and Test

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
    
    - name: Verify no uncommitted changes
      run: git diff --exit-code
    
    - name: Build
      run: go build ./...
    
    - name: Test
      run: go test ./...
```

### GitLab CI

```yaml
stages:
  - generate
  - build
  - test

variables:
  GO_VERSION: "1.21"

before_script:
  - apt-get update -qq && apt-get install -y -qq git ca-certificates
  - wget -O- https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar -C /usr/local -xzf -
  - export PATH="/usr/local/go/bin:${PATH}"

generate:
  stage: generate
  script:
    - go generate ./...
    - git diff --exit-code # Ensure no changes

build:
  stage: build
  script:
    - go generate ./...
    - go build ./...

test:
  stage: test
  script:
    - go generate ./...
    - go test ./...
```

### Docker Integration

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate code
RUN go generate ./...

# Build application
RUN go build -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/app /usr/local/bin/
ENTRYPOINT ["app"]
```

## Advanced Configuration

### Conditional Generation

```go
//go:build convergen && debug

package converters

// Only generate debug converters in debug builds
//go:generate go run github.com/reedom/convergen@v8.0.3
type DebugConvergen interface {
    // :literal DebugInfo true
    AddDebugInfo(*User) *DebugUser
}
```

### Multiple Versions

```go
//go:build convergen

package converters

// Use different versions for different purposes
//go:generate go run github.com/reedom/convergen@v8.0.3

// Production converters
type ProdConvergen interface {
    UserToAPI(*domain.User) *api.User
}

// Could also pin to different version for experimental features
// //go:generate go run github.com/reedom/convergen@latest
// type ExperimentalConvergen interface {
//     // Experimental converters
// }
```

## Troubleshooting

### Generation Not Running

```bash
# Check if files have build tags
head -5 converter.go

# Verify go:generate directive format
grep -n "go:generate" converter.go

# Test manual generation
go run github.com/reedom/convergen@v8.0.3 converter.go
```

### Module Download Issues

```bash
# Clear module cache
go clean -modcache

# Retry download
go mod download github.com/reedom/convergen

# Check GOPROXY settings
go env GOPROXY
```

### Build Integration Problems

```bash
# Check Go version
go version

# Verify all files generated
find . -name "*.gen.go" -newer converter.go

# Force regeneration
rm -f *.gen.go && go generate ./...
```

## Best Practices

### Version Control

```bash
# Recommended: Track generated files
echo "*.gen.go.log" >> .gitignore

# Alternative: Ignore generated files (requires CI generation)
echo "*.gen.go" >> .gitignore
echo "*.gen.go.log" >> .gitignore
```

### File Naming

```bash
# Consistent naming patterns
*_converter.go      # Source files
*_converter.gen.go  # Generated files
```

### Error Prevention

```bash
# Pre-commit hook to ensure generation
#!/bin/bash
go generate ./...
if [ -n "$(git status --porcelain '*.gen.go')" ]; then
    echo "Generated files out of date. Run 'go generate ./...' and commit."
    exit 1
fi
```

## Migration from CLI

If you're currently using the CLI tool:

```bash
# Before (CLI)
convergen converter.go

# After (go:generate)
# Add to converter.go:
//go:generate go run github.com/reedom/convergen@v8.0.3

# Then run:
go generate
```

## Next Steps

- **Learn CLI usage**: [CLI Commands](cli.md)
- **Advanced configuration**: [Configuration](configuration.md)
- **Troubleshooting**: [Common Issues](../troubleshooting/common-issues.md)