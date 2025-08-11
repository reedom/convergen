# Technology Stack

## Architecture

### High-Level System Design
**Pipeline Architecture**: Linear processing pipeline with strategic concurrent optimizations for performance-critical stages.

```
CLI Input → Adaptive Parser → Field Mapper → Code Generator → File Emitter
     ↓           ↓                ↓             ↓             ↓
Config Mgmt → Concurrent AST → Type Resolution → Template Engine → Format/Output
     ↓           ↓                ↓             ↓             ↓
Validation  → Cross-Package  → Strategy Exec  → Import Mgmt  → Error Handling
            → Type Loading   → Resource Pools → Optimization
```

### Core Components
- **Coordinator**: Central orchestrator managing pipeline lifecycle and resource allocation
- **Parser**: Multi-strategy AST parser with concurrent package loading (40-70% performance improvement)
- **Builder**: Field mapping strategy engine with type compatibility checking
- **Generator**: Template-based code generation with struct literal optimization
- **Emitter**: Code formatting, import management, and file output

## Backend Technology

### Language & Runtime
- **Go**: 1.21+ (specified in go.mod)
- **Module**: `github.com/reedom/convergen/v9`
- **Entry Point**: `main.go` (CLI application)
- **Architecture Pattern**: Package-based modular design with `pkg/` organization

### Core Dependencies
```go
// Production dependencies
golang.org/x/tools v0.13.0    // Go AST parsing and type checking

// Indirect dependencies
golang.org/x/mod v0.12.0       // Go module utilities
golang.org/x/sys v0.12.0       // System calls and OS interface
```

### Package Structure
```
pkg/
├── coordinator/     # Pipeline orchestration and lifecycle management
├── parser/         # Multi-strategy AST parsing with concurrent processing
├── builder/        # Field mapping and type conversion logic
├── generator/      # Code generation with template engine
├── emitter/        # Output formatting and file emission
├── executor/       # Field mapping strategy execution
├── domain/         # Core domain models (immutable with constructors)
├── config/         # Configuration management
├── util/           # AST utilities and type checking helpers
├── option/         # Annotation processing and validation
├── planner/        # Dependency analysis and optimization
├── runner/         # CLI execution framework
└── internal/events/ # Event-driven communication
```

## Development Environment

### Required Tools
- **Go**: 1.21+ with modules enabled
- **golangci-lint**: Comprehensive code linting (installed via `make install-linters`)
- **goimports**: Enhanced import formatting with local package prioritization

### Recommended Development Tools
- **gosec**: Security-focused static analysis
- **govulncheck**: Vulnerability scanning for dependencies
- **gocyclo**: Cyclomatic complexity analysis (threshold: 15)
- **gocognit**: Cognitive complexity analysis (threshold: 20)

### Code Quality Standards
```bash
# Formatting
indent_style = tab (Go files)
indent_size = 4 (Go files)
charset = utf-8
end_of_line = lf
local_import_prefix = github.com/reedom/convergen
```

## Common Commands

### Build & Development
```bash
make build                    # Build CLI to build/convergen
make test                     # Run all tests (behavior-driven + unit tests)
make coverage                 # Generate comprehensive test coverage report
go run main.go <input-file>   # Execute convergen directly on source files
```

### Code Quality & Linting
```bash
make lint                     # Run comprehensive linting with golangci-lint
make lint-fix                 # Auto-fix linting issues where possible
make lint-all                 # Run complete linting suite (fmt + lint + security + complexity + deps)
make lint-security            # Security-focused linting with gosec
make lint-complexity          # Check cyclomatic and cognitive complexity
make lint-deps                # Dependency vulnerability and tidiness check
make fmt                      # Format code with goimports (local package priority)
make install-linters          # Install all recommended linting tools
```

### Testing Framework
```bash
go test ./tests -v                                   # Behavior-driven integration tests
go test ./tests -run TestAnnotationCoverage -v      # Annotation system coverage
go test ./tests -run TestErrorScenarios -v          # Error handling scenarios
go test ./tests/examples -v                         # Framework usage examples
go test github.com/reedom/convergen/v9/pkg/...      # Unit tests for all packages
```

### Production Usage
```bash
# As go:generate tool (recommended)
//go:generate go run github.com/reedom/convergen@latest

# As installed CLI
go install github.com/reedom/convergen@latest
convergen --input=interfaces.go --output=generated.go
```

## Environment Variables

### Build Environment
```bash
GOOS=linux|darwin|windows     # Target operating system
GOARCH=amd64|arm64            # Target architecture
CGO_ENABLED=0                 # Disable CGO for static binaries
```

### Development Configuration
```bash
GOLANGCI_LINT_CACHE           # golangci-lint cache directory
GOPATH                        # Go workspace (if not using modules)
GOCACHE                       # Go build cache location
```

### Runtime Configuration
```bash
GOMAXPROCS                    # Maximum CPU cores for concurrent processing
```

## Performance Characteristics

### Benchmarks (Real-world Measurements)
- **Parser Performance**: 40-70% improvement with concurrent processing
- **Memory Usage**: <100MB typical usage for large codebases
- **Processing Latency**: Sub-second for typical interface processing
- **Concurrent Scaling**: Adaptive concurrency based on available CPU cores
- **Cross-Package Resolution**: Efficient LRU caching reduces repeated type lookups

### Resource Management
- **Bounded Concurrency**: Configurable worker pools prevent resource exhaustion
- **Memory Optimization**: Object pooling and explicit cleanup for temporary objects
- **Cache Strategy**: LRU caching for type resolution and AST parsing results
- **Deterministic Output**: Consistent results despite concurrent processing

## Testing Strategy

### Behavior-Driven Testing Framework
- **Philosophy**: Test functionality, not implementation details
- **Inline Code Generation**: Generate and test code at runtime
- **Zero Maintenance**: No fixture files or expected output management
- **Comprehensive Coverage**: All 18 annotation types and error scenarios
- **Cross-Package Testing**: Generic instantiation across module boundaries

### Coverage Standards
- **Target Coverage**: 95%+ for core packages
- **Integration Testing**: End-to-end pipeline validation
- **Performance Regression**: Automated benchmarking prevents performance degradation
- **Race Condition Detection**: Concurrent processing validation with race detector

## Security Considerations

### Static Analysis
- **gosec**: Security-focused linting with SARIF output
- **govulncheck**: Continuous dependency vulnerability scanning
- **golangci-lint**: Comprehensive code quality analysis

### Code Generation Safety
- **Input Validation**: Safe handling of malformed interfaces and annotations
- **Code Injection Prevention**: Proper escaping and validation of annotation parameters
- **Type Safety**: Compile-time validation of generated conversion functions
- **Error Handling**: Graceful failure with detailed error context
