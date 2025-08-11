# CLI Commands

Complete reference for Convergen's command-line interface, including all options, flags, and usage patterns.

## Installation

Install Convergen as a global CLI tool:

```bash
go install github.com/reedom/convergen@latest
```

Verify installation:

```bash
convergen -version
```

## Basic Usage

```bash
convergen [flags] <input-file>
```

**Example:**

```bash
# Generate code from converter.go
convergen converter.go

# Output will be written to converter.gen.go
```

## Command-Line Flags

### `-out <path>`

Specify the output file path.

**Default:** `<input-file>.gen.go`

**Examples:**

```bash
# Custom output filename
convergen -out generated_converters.go converter.go

# Output to different directory
convergen -out ./generated/converters.go converter.go

# Output to stdout (use with -print)
convergen -out /dev/stdout converter.go
```

### `-dry`

Perform a dry run without writing any files.

**Use cases:**
- Preview generated code
- Validate syntax and annotations
- Debug conversion logic
- CI/CD validation

**Examples:**

```bash
# Preview what would be generated
convergen -dry converter.go

# Combine with -print to see output
convergen -dry -print converter.go
```

### `-print`

Print the generated code to stdout in addition to writing to file.

**Use cases:**
- Debugging generated output
- Piping to other tools
- Logging generation results

**Examples:**

```bash
# Print to console and write to file
convergen -print converter.go

# Print only (with -dry)
convergen -dry -print converter.go

# Pipe to less for paging
convergen -dry -print converter.go | less
```

### `-log`

Enable detailed logging during code generation.

**Log information includes:**
- File parsing progress
- Interface discovery
- Annotation processing
- Field matching decisions
- Error details and suggestions

**Examples:**

```bash
# Enable logging
convergen -log converter.go

# Log output goes to <output-file>.log
# E.g., converter.gen.go.log

# View log in real-time
convergen -log converter.go && tail -f converter.gen.go.log
```

### `-struct-literal`

**NEW in v9** - Force struct literal output for all generated functions.

**Use Cases:**
- Override project-level settings
- Ensure consistent struct literal style
- Performance optimization for simple conversions
- Clean, readable generated code

**Examples:**

```bash
# Force struct literal output
convergen -struct-literal converter.go

# Combine with other flags
convergen -struct-literal -print converter.go
```

**Generated Output:**

```go
func Convert(src *User) (dst *UserDTO) {
    return &UserDTO{
        ID:   src.ID,
        Name: src.Name,
        Email: src.Email,
    }
}
```

### `-no-struct-literal`

**NEW in v9** - Disable struct literal output, use traditional assignment style.

**Use Cases:**
- Force traditional assignment blocks
- Compatibility with complex conversion logic
- Override automatic struct literal detection
- Debug complex field mappings

**Examples:**

```bash
# Force assignment style
convergen -no-struct-literal converter.go

# For debugging complex mappings
convergen -no-struct-literal -log converter.go
```

**Generated Output:**

```go
func Convert(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = src.ID
    dst.Name = src.Name
    dst.Email = src.Email
    return
}
```

**Note:** Cannot be used together with `-struct-literal`.

### `-version`

Display version information and exit.

**Output includes:**
- Convergen version
- Go version used for building
- Build date and commit hash

**Example:**

```bash
convergen -version
# Output: convergen v9.0.0-beta.1 (built with go1.21.5)
```

### `-type <specification>`

**NEW in v9** - Specify generic type parameters for cross-package conversions.

**Format:** `<InterfaceName>[<SourceType>,<DestinationType>]`

**Use Cases:**
- Cross-package type conversions
- Generic interface instantiation
- Complex type parameter scenarios
- External package integration

**Examples:**

```bash
# Basic cross-package conversion
convergen -type 'UserConverter[models.User,dto.UserDTO]' \
          -imports 'models=./internal/models,dto=./api/dto' converter.go

# Multiple type parameters
convergen -type 'Mapper[domain.Order,api.OrderResponse]' \
          -imports 'domain=./pkg/domain,api=./pkg/api' mapper.go

# Complex nested types
convergen -type 'Converter[store.ProductData,view.ProductView]' \
          -imports 'store=./internal/store,view=./ui/view' product_converter.go
```

### `-imports <mappings>`

**NEW in v9** - Define package alias mappings for cross-package type resolution.

**Format:** `<alias>=<path>,<alias>=<path>,...`

**Requirements:**
- Must be used together with `-type` flag
- Paths must be valid Go import paths
- Aliases must match those used in type specifications

**Examples:**

```bash
# Single package mapping
convergen -imports 'models=./internal/models' converter.go

# Multiple package mappings
convergen -imports 'models=./internal/models,dto=./api/dto,common=./pkg/common' converter.go

# External package support
convergen -imports 'proto=./gen/proto,grpc=google.golang.org/grpc' grpc_converter.go
```

### Cross-Package Usage Pattern

**Complete Example:**

```bash
# Generate cross-package converter
convergen -type 'UserConverter[models.User,dto.UserDTO]' \
          -imports 'models=./internal/models,dto=./api/dto' \
          -out user_conversions.gen.go \
          user_converter.go
```

**Input file structure:**

```go
//go:build convergen

package converters

// Type will be resolved using -type and -imports flags
type UserConverter[S, D any] interface {
    // :typecast
    Convert(S) D
}
```

**Error Handling:**
- Invalid package paths result in clear error messages
- Missing type resolution provides helpful suggestions  
- Type compatibility validation with constraint checking

### `-help`

Display help information and exit.

**Example:**

```bash
convergen -help
```

## Exit Codes

Convergen uses standard exit codes to indicate success or failure:

| Exit Code | Meaning | Description |
|-----------|---------|-------------|
| `0` | Success | Code generated successfully |
| `1` | General Error | Invalid arguments, file not found, etc. |
| `2` | Parse Error | Syntax errors in input file |
| `3` | Generation Error | Errors during code generation |
| `4` | Write Error | Cannot write output file |

**Example usage in scripts:**

```bash
#!/bin/bash
convergen converter.go
if [ $? -eq 0 ]; then
    echo "Code generation successful"
    go build ./...
else
    echo "Code generation failed"
    exit 1
fi
```

## File Handling

### Input File Requirements

- **Extension:** Must be a `.go` file
- **Syntax:** Valid Go syntax
- **Interfaces:** Must contain Convergen interfaces
- **Imports:** All referenced types must be importable

### Output File Behavior

- **Default naming:** `<input>.gen.go`
- **Overwrite:** Existing files are overwritten
- **Permissions:** Inherits permissions from input file
- **Build tags:** Includes generation metadata

**Generated file header:**

```go
// Code generated by github.com/reedom/convergen
// DO NOT EDIT.

package mypackage

// ... generated code
```

## Common Usage Patterns

### Development Workflow

```bash
# 1. Write converter interface
vim converter.go

# 2. Test generation with struct literals
convergen -struct-literal -dry -print converter.go

# 3. Generate code
convergen -struct-literal converter.go

# 4. Test compilation
go build ./...

# 5. Run tests
go test ./...
```

### Cross-Package Development Workflow

```bash
# 1. Set up cross-package converter
vim cross_package_converter.go

# 2. Test with type resolution
convergen -type 'Converter[models.User,dto.UserDTO]' \
          -imports 'models=./internal/models,dto=./api/dto' \
          -dry -print cross_package_converter.go

# 3. Generate final code
convergen -type 'Converter[models.User,dto.UserDTO]' \
          -imports 'models=./internal/models,dto=./api/dto' \
          -struct-literal \
          cross_package_converter.go

# 4. Verify imports and compilation
go build ./...
```

### CI/CD Integration

```bash
#!/bin/bash
# generate.sh - CI script

set -e

echo "Generating code..."
for file in $(find . -name "*_converter.go"); do
    echo "Processing $file"
    convergen -log "$file"
    
    if [ $? -ne 0 ]; then
        echo "Generation failed for $file"
        exit 1
    fi
done

echo "Verifying no uncommitted changes..."
git diff --exit-code '*.gen.go'

echo "Testing compilation..."
go build ./...

echo "Running tests..."
go test ./...

echo "Code generation completed successfully"
```

### Makefile Integration

```makefile
.PHONY: generate clean test build

# File patterns
CONVERTER_FILES := $(shell find . -name "*_converter.go")
GENERATED_FILES := $(CONVERTER_FILES:.go=.gen.go)

generate: $(GENERATED_FILES)

%.gen.go: %.go
	convergen -log $<

clean:
	find . -name "*.gen.go" -delete
	find . -name "*.gen.go.log" -delete

test: generate
	go test ./...

build: generate
	go build ./...

# Force regeneration
regen:
	$(MAKE) clean
	$(MAKE) generate
```

## Debugging and Troubleshooting

### Enable Verbose Logging

```bash
# Generate with detailed logging
convergen -log converter.go

# View log file
cat converter.gen.go.log
```

**Log file contents:**

```
2024-01-15 10:30:00 INFO Starting code generation for converter.go
2024-01-15 10:30:00 INFO Found interface: Convergen
2024-01-15 10:30:00 INFO Processing method: UserToDTO
2024-01-15 10:30:00 INFO Field match: ID -> ID (int -> int)
2024-01-15 10:30:00 INFO Field match: Name -> Name (string -> string)
2024-01-15 10:30:00 INFO Skipping field: Password (annotation :skip)
2024-01-15 10:30:00 INFO Custom mapping: FirstName + " " + LastName -> FullName
2024-01-15 10:30:00 INFO Generated function: UserToDTO
2024-01-15 10:30:01 INFO Code generation completed successfully
```

### Common Error Patterns

=== "Interface Not Found"

    **Error:** `no converter interfaces found in converter.go`
    
    **Causes:**
    - Interface not named "Convergen"
    - Missing `:convergen` annotation for custom names
    - Interface not exported (lowercase name)
    
    **Solutions:**
    ```bash
    # Check interface names
    grep -n "type.*interface" converter.go
    
    # Look for :convergen annotations
    grep -n ":convergen" converter.go
    ```

=== "Type Resolution Error"

    **Error:** `cannot resolve type "User" in converter.go:15`
    
    **Causes:**
    - Missing import statements
    - Unexported types
    - Package path issues
    
    **Solutions:**
    ```bash
    # Check imports
    head -20 converter.go | grep import
    
    # Verify type accessibility
    go doc github.com/myorg/myproject/domain.User
    ```

=== "Annotation Syntax Error"

    **Error:** `invalid annotation syntax ":map ID" at converter.go:12`
    
    **Causes:**
    - Incorrect annotation format
    - Missing annotation parameters
    - Typos in annotation names
    
    **Solutions:**
    ```bash
    # Show line with error
    sed -n '12p' converter.go
    
    # Check annotation syntax
    grep -n ":" converter.go
    ```

### Preview Generated Code

```bash
# See what would be generated without writing files
convergen -dry -print converter.go

# Save preview to file for review
convergen -dry -print converter.go > preview.go

# Compare with existing generated code
convergen -dry -print converter.go | diff - converter.gen.go
```

### Performance Analysis

```bash
# Time the generation process
time convergen converter.go

# For multiple files
time find . -name "*_converter.go" -exec convergen {} \;

# Memory usage (Linux/macOS)
/usr/bin/time -v convergen converter.go
```

## Integration Examples

### Docker Usage

```dockerfile
FROM golang:1.21-alpine AS builder

# Install Convergen
RUN go install github.com/reedom/convergen@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate code
RUN find . -name "*_converter.go" -exec convergen {} \;

# Build application
RUN go build -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/app /usr/local/bin/
ENTRYPOINT ["app"]
```

### GitHub Actions

```yaml
name: Generate and Test

on: [push, pull_request]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install Convergen
      run: go install github.com/reedom/convergen@latest
    
    - name: Generate code
      run: |
        find . -name "*_converter.go" -exec convergen -log {} \;
    
    - name: Verify no changes
      run: git diff --exit-code
    
    - name: Test
      run: go test ./...
```

### Build Scripts

```bash
#!/bin/bash
# build.sh - Complete build pipeline

set -euo pipefail

# Configuration
CONVERGEN_VERSION="v9.0.0-beta.1"
LOG_LEVEL="INFO"

# Functions
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

install_convergen() {
    log "Installing Convergen $CONVERGEN_VERSION"
    go install "github.com/reedom/convergen@$CONVERGEN_VERSION"
}

generate_code() {
    log "Generating conversion code"
    
    local files
    files=$(find . -name "*_converter.go" -type f)
    
    if [ -z "$files" ]; then
        log "No converter files found"
        return 0
    fi
    
    local count=0
    while IFS= read -r file; do
        log "Processing $file"
        convergen -log "$file"
        ((count++))
    done <<< "$files"
    
    log "Generated code for $count files"
}

verify_build() {
    log "Verifying build"
    go mod tidy
    go build ./...
    go test ./...
}

main() {
    log "Starting build process"
    
    # Install tools
    install_convergen
    
    # Generate code
    generate_code
    
    # Verify everything works
    verify_build
    
    log "Build completed successfully"
}

# Run main function
main "$@"
```

## Best Practices

### File Organization

```bash
# Recommended structure
project/
├── internal/
│   ├── domain/
│   │   ├── user.go
│   │   └── order.go
│   ├── storage/
│   │   ├── user.go
│   │   └── order.go
│   └── converters/
│       ├── user_converter.go      # Input file
│       ├── user_converter.gen.go  # Generated (gitignore optional)
│       ├── order_converter.go     # Input file
│       └── order_converter.gen.go # Generated (gitignore optional)
```

### Naming Conventions

```bash
# Consistent naming patterns
*_converter.go      # Input files
*_converter.gen.go  # Generated files
*_converter.gen.go.log  # Log files (add to .gitignore)
```

### Version Control

```bash
# .gitignore options

# Option 1: Track generated files (recommended)
*.gen.go.log

# Option 2: Ignore generated files (requires CI generation)
*.gen.go
*.gen.go.log
```

## Next Steps

- **Learn go:generate integration**: [go:generate Integration](go-generate.md)
- **Advanced configuration**: [Configuration](configuration.md)
- **Troubleshooting**: [Common Issues](../troubleshooting/common-issues.md)
