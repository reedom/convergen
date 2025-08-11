# Project Structure

## Root Directory Organization

```
convergen/
├── .kiro/                    # Kiro steering documents and project intelligence
├── build/                    # Compiled binaries and build artifacts
├── docs/                     # Source documentation (MkDocs format)
├── pkg/                      # Core application packages (library code)
├── site/                     # Generated documentation website
├── tests/                    # Integration and behavior-driven tests
├── main.go                   # Application entry point
├── go.mod/go.sum            # Go module definition and dependencies
├── Makefile                  # Build automation and development commands
├── mkdocs.yml               # Documentation site configuration
├── CLAUDE.md                # Development guidelines and conventions
└── README.md                # Project overview and quick start
```

### Key Directory Purposes

#### `/pkg/` - Core Library Architecture
**Purpose**: Modular, reusable packages following Go best practices
**Pattern**: Domain-driven design with clear separation of concerns
**Access**: Internal packages for implementation, public APIs for integration

#### `/docs/` - Source Documentation
**Purpose**: Human-readable documentation in Markdown format
**Pattern**: Hierarchical organization matching user journey
**Build**: Processed by MkDocs into static site under `/site/`

#### `/tests/` - Comprehensive Testing Suite
**Purpose**: Integration tests, behavior validation, and test utilities
**Pattern**: Behavior-driven testing focusing on functionality over implementation
**Scope**: End-to-end workflow validation and edge case coverage

## Core Package Architecture

### Processing Pipeline Packages

#### `/pkg/parser/` - AST Analysis & Type Resolution
```
parser/
├── parser.go                 # Main parser orchestration
├── ast_parser.go            # Go AST parsing and analysis
├── interface_analyzer.go    # Interface method analysis
├── cross_package_resolver.go # Cross-package type resolution
├── type_resolver.go         # Type system integration
├── constraint_parser.go     # Generic constraint handling
├── error_handler.go         # Parser error management
└── cache.go                 # AST parsing result caching
```

**Responsibilities:**
- Parse Go source files into AST representations
- Analyze interface definitions and method signatures
- Resolve types across package boundaries
- Handle generic type constraints and parameters
- Cache parsing results for performance optimization

#### `/pkg/builder/` - Field Mapping & Conversion Logic
```
builder/
├── handler.go               # Main building orchestration
├── method.go               # Method-level conversion building
├── assignment.go           # Assignment statement generation
├── generic_field_mapper.go # Generic-aware field mapping
├── postprocess.go          # Post-processing optimizations
└── model/                  # Builder domain models
    ├── struct.go           # Struct representation models
    ├── method.go           # Method models with metadata
    ├── copier.go           # Field copying strategies
    └── node.go             # AST node representations
```

**Responsibilities:**
- Build field mapping strategies from annotations
- Handle generic type substitution and compatibility
- Generate assignment logic for conversions
- Optimize conversion paths for performance
- Manage complex type relationships

#### `/pkg/generator/` - Code Generation Engine
```
generator/
├── generator.go            # Main generation orchestration
├── function.go            # Function generation logic
├── assignment.go          # Assignment code generation
├── generic_generator.go   # Generic-specific generation
├── generic_templates.go   # Template definitions for generics
└── model/                 # Generator domain models
    ├── function.go        # Function model representations
    ├── assignment.go      # Assignment models
    ├── code.go           # Code structure models
    └── enums.go          # Enumeration and constant models
```

**Responsibilities:**
- Generate Go source code from conversion models
- Handle generic function instantiation
- Produce optimized, readable code output
- Manage code templates and formatting
- Ensure type safety in generated code

#### `/pkg/emitter/` - Optimization & Output
```
emitter/
├── emitter.go              # Main emission orchestration
├── code_generator.go       # Final code generation
├── optimizer.go            # Code optimization passes
├── import_manager.go       # Import statement management
├── format_manager.go       # Code formatting and style
├── output_strategy.go      # File output strategies
└── generation_strategies.go # Code generation patterns
```

**Responsibilities:**
- Optimize generated code for performance and readability
- Manage import statements and dependencies
- Handle code formatting and Go style compliance
- Coordinate file output and organization
- Apply final optimizations before output

### Coordination & Infrastructure Packages

#### `/pkg/coordinator/` - Pipeline Orchestration
```
coordinator/
├── coordinator.go          # Main orchestration logic
├── event_orchestrator.go   # Event-driven coordination
├── component_manager.go    # Component lifecycle management
├── resource_pool.go        # Resource pooling and management
├── metrics_collector.go    # Performance metrics collection
├── context_manager.go      # Execution context management
└── error_handler.go        # Centralized error handling
```

**Responsibilities:**
- Orchestrate the entire processing pipeline
- Manage component lifecycle and dependencies
- Handle resource allocation and concurrency
- Collect performance metrics and diagnostics
- Coordinate error handling across components

#### `/pkg/domain/` - Domain Models & Business Logic
```
domain/
├── types.go                     # Core type system models
├── method.go                    # Method definition models
├── field.go                     # Field mapping models
├── options.go                   # Configuration option models
├── errors.go                    # Domain-specific error types
├── type_compatibility_checker.go # Type compatibility analysis
├── generic_compatibility_matrix.go # Generic type compatibility
├── substitution_validator.go    # Type substitution validation
└── instantiator.go              # Generic instantiation logic
```

**Responsibilities:**
- Define immutable domain models for all business concepts
- Provide type safety and validation logic
- Handle generic type system complexity
- Manage conversion rules and constraints
- Ensure thread-safe access to domain data

#### `/pkg/executor/` - Execution Management
```
executor/
├── executor.go             # Main execution orchestration
├── field_executor.go       # Field-level conversion execution
├── batch_executor.go       # Batch processing coordination
├── resource_pool.go        # Resource allocation management
└── metrics.go              # Execution metrics collection
```

**Responsibilities:**
- Execute conversion operations efficiently
- Manage concurrent processing and resource allocation
- Handle batch operations for large-scale processing
- Collect execution metrics and performance data

### Utility & Support Packages

#### `/pkg/util/` - Common Utilities
```
util/
├── ast.go                  # AST manipulation utilities
├── types.go                # Type system utilities
└── import.go               # Import handling utilities
```

#### `/pkg/config/` - Configuration Management
```
config/
├── config.go               # Configuration parsing and validation
└── config_test.go          # Configuration testing
```

#### `/pkg/option/` - Annotation Processing
```
option/
├── option.go               # Annotation option definitions
├── pattern_matcher.go      # Pattern matching for annotations
├── field_converter.go      # Field conversion option handling
├── name_matcher.go         # Name matching utilities
└── literal_setter.go       # Literal value handling
```

## File Naming Conventions

### Go Source Files
- **Main components**: `{component}.go` (e.g., `parser.go`, `generator.go`)
- **Specialized functionality**: `{component}_{specialization}.go` (e.g., `generic_generator.go`)
- **Test files**: `{component}_test.go` for unit tests
- **Integration tests**: `{feature}_integration_test.go`
- **Documentation**: `doc.go` for package documentation

### Model Packages
- **Domain models**: Organized in `/model/` subdirectories
- **Model files**: Named after the primary entity (e.g., `struct.go`, `method.go`)
- **Utility models**: `util.go` for shared model utilities
- **Validation**: `{model}_validator.go` for model-specific validation

### Configuration Files
- **Go modules**: `go.mod` / `go.sum` for dependency management
- **Build automation**: `Makefile` for development commands
- **Documentation**: `mkdocs.yml` for documentation site configuration
- **Project metadata**: `CLAUDE.md`, `README.md`, `LICENSE`

## Code Organization Patterns

### Package Structure Principles
1. **Domain-Driven Design**: Packages organized around business concepts
2. **Dependency Direction**: Dependencies flow inward toward domain models
3. **Interface Segregation**: Small, focused interfaces for loose coupling
4. **Immutable Models**: Domain models are immutable after construction
5. **Clear Boundaries**: Well-defined package boundaries with minimal coupling

### Import Organization Standards
```go
// Standard library imports
import (
    "context"
    "fmt"
    "go/ast"
)

// Third-party dependencies
import (
    "github.com/google/go-cmp/cmp"
    "go.uber.org/zap"
)

// Local project imports
import (
    "github.com/reedom/convergen/v8/pkg/domain"
    "github.com/reedom/convergen/v8/pkg/util"
)
```

### Error Handling Patterns
- **Domain errors**: Defined in `/pkg/domain/errors.go`
- **Context preservation**: Errors include full context for debugging
- **Wrapped errors**: Use `fmt.Errorf()` with `%w` verb for error chains
- **Structured logging**: Use zap logger for consistent error reporting

## Testing Architecture

### Test Organization
```
tests/
├── behavior_test.go          # Core behavior validation
├── error_test.go             # Error condition testing
├── struct_literal_test.go    # Struct literal generation tests
├── generics_*_test.go        # Generic-specific test suites
├── examples/                 # Real-world example tests
│   ├── basic_usage_test.go   # Basic functionality examples
│   └── advanced_patterns_test.go # Advanced pattern examples
└── helpers/                  # Test utilities and helpers
    ├── scenario.go           # Test scenario framework
    └── inline_runner.go      # Inline test execution
```

### Testing Patterns
- **Behavior-driven**: Focus on what the system does, not how
- **Scenario-based**: Use scenario framework for complex test cases
- **Inline testing**: Test generation and execution in single test functions
- **Example-driven**: Real-world examples as primary test cases

## Build & Deployment Structure

### Build Artifacts
```
build/
├── convergen               # Compiled CLI binary
├── convergen-linux         # Linux cross-compiled binary
├── convergen-windows.exe   # Windows cross-compiled binary
└── coverage.out            # Test coverage reports
```

### Documentation Structure
```
site/                       # Generated documentation website
├── api/                    # API reference documentation
├── examples/               # Usage examples and tutorials
├── getting-started/        # Onboarding documentation
├── guide/                  # Comprehensive user guides
├── troubleshooting/        # Problem-solving guides
└── assets/                 # Static assets (CSS, JS, images)
```

## Key Architectural Principles

### 1. **Immutable Domain Models**
All domain models are immutable after construction, ensuring thread safety and preventing accidental mutations during concurrent processing.

### 2. **Event-Driven Coordination**
Components communicate through events rather than direct coupling, enabling flexible orchestration and easier testing.

### 3. **Resource Pooling**
CPU-intensive operations use resource pools to manage concurrency and prevent resource exhaustion.

### 4. **Fail-Fast Error Handling**
Errors are detected and reported as early as possible in the pipeline with comprehensive context.

### 5. **Separation of Concerns**
Each package has a single, well-defined responsibility with minimal dependencies on other packages.

### 6. **Performance-First Design**
Architecture optimized for high-performance processing with concurrent execution and efficient resource utilization.

### 7. **Extensible Pipeline**
Processing pipeline designed for extensibility with clear interfaces between stages.

## Development Workflow Integration

### IDE Integration
- **VS Code**: Full Go language server support with debugging
- **GoLand**: Native support for Go modules and testing
- **Vim/Neovim**: go.vim plugin with full feature support

### Git Workflow
- **Branch Strategy**: Feature branches with PR-based integration
- **Generated Code**: Formatted consistently for clean diffs
- **Documentation**: Synchronized with code changes through automation

### CI/CD Integration
- **Quality Gates**: Comprehensive linting and testing before merge
- **Cross-platform Builds**: Automated builds for Linux, macOS, Windows
- **Documentation Deployment**: Automatic documentation updates on main branch changes
