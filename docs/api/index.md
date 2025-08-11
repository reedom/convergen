# API Reference

Complete reference for Convergen's command-line interface, go:generate integration, and configuration options.

## What You'll Find Here

### 🖥️ **[CLI Commands](cli.md)**
Complete command-line interface reference:

- Available commands and options
- Input/output file handling
- Debugging and logging options
- Integration with build systems
- Exit codes and error handling

### 🔧 **[go:generate Integration](go-generate.md)**
Using Convergen with Go's code generation:

- Setting up go:generate directives
- Version pinning strategies
- Build tag integration
- Module and workspace compatibility
- Troubleshooting generation issues

### ⚙️ **[Configuration](configuration.md)**
Advanced configuration options:

- Global configuration files
- Per-project settings
- Environment variable support
- Parser configuration options
- Performance tuning parameters

## Quick Reference

### Basic Usage Patterns

=== "CLI Usage"

    ```bash
    # Generate from specific file
    convergen input.go
    
    # Custom output location
    convergen -out generated.go input.go
    
    # Dry run to preview
    convergen -dry input.go
    
    # Enable logging
    convergen -log input.go
    ```

=== "go:generate"

    ```go
    //go:build convergen
    
    package mypackage
    
    // Version pinned for reproducible builds
    //go:generate go run github.com/reedom/convergen@v9.0.0-beta.1
    
    type Convergen interface {
        Convert(*Source) *Destination
    }
    ```

=== "Build Integration"

    ```makefile
    # Makefile integration
    generate:
        go generate ./...
    
    # With specific packages
    generate-models:
        go generate ./internal/models/...
    ```

## Usage Modes

Convergen supports multiple usage patterns to fit different development workflows:

### 1. **go:generate Integration** (Recommended)

**Best for**: Most Go projects, reproducible builds, version control

- Integrates seamlessly with `go generate`
- Version pinning for consistent results
- Works with Go modules and workspaces
- Automatic dependency management

### 2. **CLI Tool**

**Best for**: Build scripts, CI/CD pipelines, non-Go toolchains

- Direct command-line usage
- Flexible input/output options
- Scriptable and automatable
- No Go generate dependency

### 3. **Go Module Dependency**

**Best for**: Programmatic usage, custom tooling, advanced integration

- Import as Go library
- Programmatic API access
- Custom parser configuration
- Integration with existing tools

## Command Overview

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `convergen <file>` | Generate from file | `-out`, `-dry`, `-print` |
| `convergen -help` | Show help | N/A |
| `go generate` | Generate via directive | N/A (uses file annotations) |

### Flag Reference

| Flag | Description | Default |
|------|-------------|---------|
| `-out <path>` | Output file path | `<input>.gen.go` |
| `-dry` | Dry run (no files written) | `false` |
| `-print` | Print to stdout | `false` |
| `-log` | Enable logging | `false` |

## Integration Examples

### CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Generate Code
  run: |
    go generate ./...
    git diff --exit-code # Fail if changes
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
```

### Docker Build

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

## Version Management

### Recommended Versioning Strategy

```go
// Pin to specific version for stability
//go:generate go run github.com/reedom/convergen@v9.0.0-beta.1

// Use latest for development (not recommended for production)
//go:generate go run github.com/reedom/convergen@latest

// Use major version for automatic patches
//go:generate go run github.com/reedom/convergen@v9
```

### Version Compatibility

| Version | Go Version | Features |
|---------|------------|----------|
| v9.x | Go 1.21+ | Concurrent parsing, generics |
| v7.x | Go 1.18+ | Basic generics support |
| v6.x | Go 1.16+ | Legacy compatibility |

## Performance Considerations

### Parser Configuration

The v9 parser engine supports multiple strategies:

- **LegacyParser**: Traditional synchronous (backward compatible)
- **ModernParser**: Concurrent processing (40-70% faster)
- **AdaptiveParser**: Automatic strategy selection

### Build Performance

Tips for faster generation:

1. **Use specific file patterns** instead of `./...`
2. **Pin versions** to avoid repeated downloads
3. **Cache Go modules** in CI/CD
4. **Parallel generation** for independent packages

## Troubleshooting

Common integration issues:

### Generation Not Running

```bash
# Check if go:generate is properly formatted
grep -r "go:generate" .

# Verify Go version compatibility
go version

# Test manual generation
go run github.com/reedom/convergen@latest your-file.go
```

### Build Integration Issues

```bash
# Clean generated files
find . -name "*.gen.go" -delete

# Regenerate everything
go generate ./...

# Verify output
ls -la **/*.gen.go
```

## Next Steps

- **Learn CLI details**: [CLI Commands](cli.md)
- **Master go:generate**: [go:generate Integration](go-generate.md)  
- **Advanced configuration**: [Configuration](configuration.md)
- **Troubleshooting**: [Common Issues](../troubleshooting/common-issues.md)
