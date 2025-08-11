# Technology Stack

## Architecture Overview

Convergen follows a **modular, pipeline-based architecture** with concurrent processing capabilities. The system is built around a domain-driven design with immutable models and event-driven coordination.

### High-Level Architecture
```
Input (Annotated Interfaces)
    ↓
Parser (AST Analysis + Cross-Package Resolution)
    ↓
Builder (Field Mapping + Type Analysis)
    ↓
Generator (Code Template Processing)
    ↓
Emitter (Optimization + Output)
    ↓
Generated Go Code (Zero Dependencies)
```

### Core Design Principles
- **Domain-Driven Design**: Immutable domain models as single source of truth
- **Event-Driven Architecture**: Loose coupling through event orchestration
- **Concurrent Processing**: Multi-core utilization with resource pooling
- **Pipeline Pattern**: Clear separation of concerns across processing stages
- **Fail-Fast Strategy**: Early error detection with comprehensive context

## Language & Runtime

### Go Language Requirements
- **Minimum Version**: Go 1.21.0
- **Recommended Toolchain**: go1.24.5 (as configured)
- **Module Path**: `github.com/reedom/convergen/v8`
- **Build Target**: Cross-platform (Linux, macOS, Windows)

### Language Features Used
- **Generics**: Full support for type parameters and constraints
- **AST Processing**: `go/ast`, `go/parser`, `go/types` packages
- **Concurrent Processing**: Goroutines with `golang.org/x/sync` coordination
- **Module System**: Go modules with semantic versioning

## Core Dependencies

### Production Dependencies
```go
// Essential Dependencies
github.com/google/go-cmp v0.6.0           // Deep comparison for testing/validation
github.com/matoous/go-nanoid v1.5.0       // Unique ID generation
go.uber.org/zap v1.27.0                   // Structured logging with performance
golang.org/x/sync v0.16.0                 // Extended concurrency primitives
golang.org/x/text v0.27.0                 // Text processing and encoding
golang.org/x/tools v0.34.0                // Go toolchain integration

// Testing Dependencies
github.com/stretchr/testify v1.8.1        // Comprehensive testing framework
```

### Dependency Strategy
- **Minimal Dependencies**: Only essential, well-maintained packages
- **No Runtime Dependencies**: Generated code requires zero external libraries
- **Security First**: All dependencies monitored for vulnerabilities via `govulncheck`
- **Performance Focus**: Dependencies chosen for minimal overhead and high performance

## Development Environment

### Required Tools

#### Core Development Tools
```bash
# Go toolchain
go version          # Go 1.23.0+
go mod              # Module management
go generate         # Code generation workflow

# Code Quality Tools
golangci-lint       # Comprehensive linting
goimports          # Import organization and formatting
gofmt              # Code formatting
```

#### Quality Assurance Tools
```bash
# Security & Analysis
gosec              # Security vulnerability scanning
govulncheck        # Go vulnerability database checking
gocyclo            # Cyclomatic complexity analysis
gocognit           # Cognitive complexity measurement

# Testing & Coverage
go test            # Native testing framework
go tool cover      # Coverage analysis
```

#### Documentation Tools
```bash
# Documentation Generation
mkdocs             # Documentation site generator
mkdocs-material    # Material design theme
```

### Development Workflow Tools

#### Build System
- **Make**: Primary build orchestration via `Makefile`
- **Go Build**: Native compilation with optimizations
- **Cross-compilation**: Support for multiple platforms

#### Linting & Quality
```bash
make lint          # Comprehensive linting with golangci-lint
make lint-fix      # Auto-fix linting issues where possible
make lint-security # Security-focused analysis
make lint-complexity # Complexity analysis
make lint-deps     # Dependency vulnerability checking
make lint-all      # Complete quality check suite
```

## Common Development Commands

### Build & Development
```bash
# Primary development commands
make build                    # Build CLI to build/convergen
make test                     # Run all tests (integration + unit)
make coverage                 # Generate coverage report

# Direct execution
go run main.go <input-file>   # Run convergen on specific file
go generate                   # Execute go:generate workflows
```

### Code Quality & Maintenance
```bash
# Formatting and quality
make fmt                      # Format all Go code with goimports
make fmt-check               # Verify code formatting
make lint                    # Run comprehensive linting
make lint-fix                # Auto-fix issues where possible

# Comprehensive analysis
make lint-all                # Complete quality suite
make lint-security           # Security vulnerability analysis
make lint-complexity         # Code complexity metrics
make lint-deps              # Dependency analysis
```

### Setup & Installation
```bash
# Tool installation
make install-linters         # Install all development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Testing Strategy

### Testing Framework Architecture
- **Behavior-Driven Testing**: Focus on actual functionality over file comparison
- **Integration Testing**: End-to-end workflow validation
- **Unit Testing**: Component-level verification
- **Performance Testing**: Benchmarking and regression detection

### Testing Commands
```bash
# Core testing
go test github.com/reedom/convergen/v8/tests    # Integration tests
go test github.com/reedom/convergen/v8/pkg/...  # Package unit tests

# Coverage analysis
make coverage                                     # Generate coverage report
go tool cover -html=coverage.out                # HTML coverage visualization
```

## Documentation System

### MkDocs Configuration
- **Site Generator**: MkDocs with Material theme
- **Hosting**: GitHub Pages at `https://reedom.github.io/convergen/`
- **Features**: Search, syntax highlighting, responsive design, dark mode
- **Content**: Comprehensive user guides, API references, examples

### Documentation Structure
```
docs/
├── getting-started/     # Installation, quick start, basic examples
├── guide/              # Comprehensive user guides and best practices
├── api/                # CLI and programmatic API references
├── examples/           # Real-world usage examples and integrations
└── troubleshooting/    # Common issues, debugging, migration guides
```

### Documentation Commands
```bash
# Local development
mkdocs serve            # Local development server
mkdocs build            # Generate static site
mkdocs gh-deploy        # Deploy to GitHub Pages
```

## Configuration Management

### Configuration Sources
1. **Command-line arguments**: Primary configuration interface
2. **Environment variables**: Secondary configuration for automation
3. **Configuration files**: Future extensibility (not currently implemented)

### Key Configuration Areas
- **Input/Output paths**: Source files and output destinations
- **Processing options**: Concurrency limits, memory constraints
- **Generation options**: Code style, optimization levels
- **Debugging options**: Verbose logging, intermediate file output

## Performance & Scalability

### Concurrent Architecture
- **Resource Pooling**: Configurable worker pools for CPU-intensive tasks
- **Batch Processing**: Efficient handling of multiple files
- **Memory Management**: Streaming processing for large codebases
- **Cache Strategy**: AST parsing results cached across operations

### Performance Characteristics
- **Build Performance**: 40-70% faster than previous versions
- **Memory Efficiency**: Minimal allocation overhead in generated code
- **Scalability**: Handles large codebases with thousands of conversion methods
- **Resource Usage**: Configurable limits for different deployment environments

## Integration Points

### Go Ecosystem Integration
- **go:generate**: Native integration with Go toolchain
- **Module System**: Full Go module compatibility
- **IDE Support**: Works with all major Go IDEs (VS Code, GoLand, etc.)
- **CI/CD Integration**: Compatible with all major CI systems

### External Tool Integration
- **Version Control**: Git-friendly generated code with consistent formatting
- **Build Systems**: Makefile, Go modules, Docker compatible
- **Documentation**: Automated API documentation generation
- **Quality Gates**: Integration with linting and security scanning tools

## Security Considerations

### Security Architecture
- **No External Network Access**: All processing is local
- **No Sensitive Data Storage**: Temporary files cleaned automatically
- **Input Validation**: Comprehensive validation of all inputs
- **Safe Code Generation**: Generated code follows security best practices

### Security Tools & Processes
```bash
gosec ./...                   # Security vulnerability scanning
govulncheck ./...             # Go vulnerability database check
make lint-deps               # Dependency security analysis
```

## Future Technology Considerations

### Planned Enhancements
- **Plugin Architecture**: Extensible processing pipeline
- **Configuration Files**: YAML/JSON configuration support
- **IDE Extensions**: Enhanced development experience
- **Performance Monitoring**: Runtime performance analytics
- **Multi-language Support**: Expansion beyond Go
