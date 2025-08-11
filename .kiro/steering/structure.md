# Project Structure

## Root Directory Organization

### Core Project Structure
```
convergen/
├── main.go                    # CLI application entry point
├── go.mod                     # Go module definition (github.com/reedom/convergen/v8)
├── go.sum                     # Go module checksums
├── doc.go                     # Package-level documentation
├── Makefile                   # Comprehensive build and quality automation
├── .editorconfig              # Code formatting standards
├── mkdocs.yml                 # Documentation site configuration
│
├── pkg/                       # Core implementation packages (see detailed breakdown below)
├── tests/                     # Behavior-driven integration testing framework
├── docs/                      # Comprehensive documentation (MkDocs site)
├── site/                      # Generated documentation website
├── build/                     # Build artifacts (gitignored, created by make build)
│
├── README.md                  # Project overview and quick start
├── LICENSE                    # Apache 2.0 license
├── CLAUDE.md                  # AI assistant development guidance
├── TASKS.md                   # Implementation roadmap and task tracking
└── .kiro/                     # Kiro spec-driven development framework
    ├── specs/convergen/       # SDD specifications (requirements, design, tasks)
    └── steering/              # Project knowledge documents (product, tech, structure)
```

## Core Package Architecture (`pkg/`)

### Pipeline Components (Primary Flow)
```
pkg/
├── coordinator/               # 🎯 Central orchestration and lifecycle management
│   ├── coordinator.go         # Main orchestrator interface and implementation
│   ├── component_manager.go   # Pipeline component lifecycle management
│   ├── resource_pool.go       # Concurrent processing resource allocation
│   ├── event_orchestrator.go  # Pipeline event coordination
│   ├── error_handler.go       # Centralized error aggregation
│   └── metrics_collector.go   # Performance and resource usage tracking
│
├── parser/                    # 🔍 Multi-strategy AST parsing with concurrent processing
│   ├── parser.go              # Main parser interface and adaptive strategy selection
│   ├── ast_parser.go          # Enhanced AST parsing with 40-70% performance improvement
│   ├── interface_analyzer.go  # Interface discovery and method extraction
│   ├── cross_package_resolver.go    # Cross-package type resolution engine
│   ├── cross_package_type_loader.go # LRU-cached type loading system
│   ├── constraint_parser.go   # Generic constraint parsing and validation
│   ├── type_resolver.go       # Type instantiation and substitution
│   └── method_processor.go    # Concurrent method processing
│
├── builder/                   # 🏗️ Field mapping and conversion strategy execution
│   ├── handler.go             # Field mapping strategy handler interface
│   ├── generic_field_mapper.go      # Generic type field mapping engine
│   ├── generic_mapping_context.go   # Context for generic type mappings
│   ├── assignment.go          # Assignment generation logic
│   └── model/                 # Field mapping domain models
│
├── generator/                 # ⚡ Template-based code generation with optimization
│   ├── generator.go           # Main code generator interface
│   ├── generic_generator.go   # Generic-aware code generation
│   ├── generic_templates.go   # Template definitions for generic code patterns
│   ├── function.go            # Function generation and signature handling
│   └── model/                 # Code generation domain models
│
├── emitter/                   # 📤 Code formatting, import management, and output
│   ├── emitter.go             # Main emitter interface and orchestration
│   ├── code_generator.go      # Final code assembly and generation
│   ├── format_manager.go      # Go formatting and code style compliance
│   ├── import_manager.go      # Import organization and optimization
│   └── generation_strategies.go     # Struct literal vs assignment block strategies
│
└── executor/                  # 🔄 Field mapping strategy execution with concurrency
    ├── executor.go            # Main execution interface
    ├── batch_executor.go      # Concurrent batch processing
    ├── field_executor.go      # Individual field mapping execution
    └── resource_pool.go       # Resource pooling for concurrent operations
```

### Domain and Support Packages
```
pkg/
├── domain/                    # 🏛️ Core domain models (immutable with constructors)
│   ├── types.go               # Primary domain type definitions
│   ├── method.go              # Method model with signature handling
│   ├── field.go               # Field mapping and conversion models
│   ├── options.go             # Annotation and configuration models
│   ├── instantiator.go        # Generic type instantiation engine (845+ lines)
│   ├── type_compatibility_checker.go  # Type compatibility validation (890+ lines)
│   ├── generic_compatibility_matrix.go # Generic type compatibility (663+ lines)
│   ├── field_mapping_validator.go     # Field mapping validation (497+ lines)
│   ├── substitution_validator.go      # Type substitution validation (566+ lines)
│   └── generic_error_context.go       # Rich error context (787+ lines)
│
├── config/                    # ⚙️ Configuration management
│   └── config.go              # CLI and annotation configuration handling
│
├── option/                    # 📝 Annotation processing and validation (18 annotation types)
│   ├── option.go              # Main annotation processing interface
│   ├── field_converter.go     # Custom converter function handling
│   ├── pattern_matcher.go     # Field pattern matching for :skip, :map annotations
│   ├── ident_matcher.go       # Identifier matching for field names
│   └── literal_setter.go      # Literal value assignment for :literal annotations
│
├── util/                      # 🔧 AST utilities and type checking helpers
│   ├── ast.go                 # AST manipulation and analysis utilities
│   ├── types.go               # Go type system utilities and helpers
│   └── import.go              # Import path resolution and management
│
├── planner/                   # 📋 Dependency analysis and optimization
│   ├── planner.go             # Main planning interface
│   ├── dependency_graph.go    # Cross-package dependency analysis
│   └── optimizer.go           # Code generation optimization strategies
│
├── runner/                    # 🏃 CLI execution framework
│   └── runner.go              # Command-line interface execution and coordination
│
├── logger/                    # 📊 Logging infrastructure
│   └── logger.go              # Structured logging with configurable levels
│
└── internal/                  # 🔒 Internal packages (not exported)
    └── events/                # Event-driven communication system
        ├── events.go          # Event type definitions and handling
        └── pipeline.go        # Pipeline event coordination
```

## Testing Structure (`tests/`)

### Behavior-Driven Testing Framework
```
tests/
├── README.md                  # Testing framework documentation
├── TESTING_GUIDE.md           # Comprehensive testing guidelines
│
├── behavior_test.go           # Core behavior validation tests
├── struct_literal_test.go     # Struct literal generation testing
├── error_test.go              # Error handling and edge case scenarios
├── generics_cross_package_test.go    # Cross-package generic resolution
├── generics_advanced_patterns_test.go # Advanced generic usage patterns
│
├── examples/                  # Framework usage examples with inline testing
│   ├── README.md              # Example documentation
│   ├── basic_usage_test.go    # Basic annotation and conversion examples
│   ├── advanced_patterns_test.go     # Complex conversion scenarios
│   └── debug_example_test.go  # Debugging and troubleshooting examples
│
└── helpers/                   # Testing infrastructure and utilities
    ├── scenario.go            # Test scenario definition and execution
    ├── inline_helpers.go      # Inline code generation testing helpers
    └── inline_runner.go       # Runtime code compilation and execution
```

## Documentation Structure (`docs/`)

### Comprehensive Documentation Site
```
docs/
├── index.md                   # Documentation homepage
├── assets/                    # Documentation assets and styling
│   ├── extra.css              # Custom styling for documentation site
│   ├── extra.js               # Custom JavaScript for enhanced navigation
│   ├── logo.svg               # Project logo and branding
│   └── favicon.ico            # Site icon
│
├── getting-started/           # New user onboarding
│   ├── index.md               # Getting started overview
│   ├── installation.md        # Installation instructions
│   ├── quick-start.md         # Quick start tutorial
│   └── basic-examples.md      # Basic usage examples
│
├── guide/                     # In-depth usage guides
│   ├── index.md               # Guide overview
│   ├── annotations.md         # Complete annotation reference (18 types)
│   ├── advanced-usage.md      # Advanced features and patterns
│   ├── performance.md         # Performance optimization guide
│   └── best-practices.md      # Development best practices
│
├── examples/                  # Real-world usage examples
│   ├── index.md               # Examples overview
│   ├── generics.md            # Go generics integration examples
│   ├── real-world.md          # Production use case examples
│   └── integrations.md        # Build system and tool integrations
│
├── api/                       # API and CLI reference
│   ├── index.md               # API overview
│   ├── cli.md                 # Command-line interface reference
│   ├── configuration.md       # Configuration options
│   └── go-generate.md         # go:generate integration guide
│
└── troubleshooting/           # Problem resolution
    ├── index.md               # Troubleshooting overview
    ├── common-issues.md       # Frequently encountered problems
    ├── debugging.md           # Debugging techniques and tools
    └── migration.md           # Migration guides for version upgrades
```

## Code Organization Patterns

### Package Responsibility Boundaries
1. **coordinator**: Orchestration and lifecycle management - no domain logic
2. **parser**: Input processing and AST analysis - no code generation
3. **builder**: Field mapping strategy - no code generation or I/O
4. **generator**: Code generation templates - no file I/O
5. **emitter**: Output formatting and file operations - no domain logic
6. **domain**: Pure domain models - no external dependencies
7. **util**: Stateless utilities - no global state

### Dependency Flow Architecture
```
CLI (main.go) → coordinator → runner
                     ↓
            parser → builder → generator → emitter
                     ↓         ↓         ↓
                  domain ←  domain ←  domain
                     ↓         ↓         ↓
                   util      util      util
```

## File Naming Conventions

### Go Source Files
- **Interfaces**: `{domain}.go` (e.g., `parser.go`, `generator.go`)
- **Implementations**: `{feature}_{type}.go` (e.g., `ast_parser.go`, `resource_pool.go`)
- **Domain Models**: `{entity}.go` (e.g., `method.go`, `field.go`, `types.go`)
- **Test Files**: `{source}_test.go` (comprehensive test coverage for all public interfaces)
- **Documentation**: `doc.go` (package-level documentation for complex packages)

### Internal Organization
- **Model Subpackages**: Domain models grouped by complexity (e.g., `builder/model/`, `generator/model/`)
- **Internal Packages**: Non-exported utilities in `pkg/internal/`
- **Helper Packages**: Testing utilities in dedicated `helpers/` directories

## Import Organization

### Import Grouping (goimports with local prefix)
```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // Third-party dependencies  
    "golang.org/x/tools/go/packages"

    // Local packages (github.com/reedom/convergen priority)
    "github.com/reedom/convergen/v8/pkg/domain"
    "github.com/reedom/convergen/v8/pkg/util"
)
```

### Module Path Structure
- **Module Root**: `github.com/reedom/convergen/v8`
- **Package Imports**: `github.com/reedom/convergen/v8/pkg/{package}`
- **Internal Imports**: `github.com/reedom/convergen/v8/pkg/internal/{package}`

## Key Architectural Principles

### 1. Clean Pipeline Architecture
- **Linear Flow**: Parser → Builder → Generator → Emitter
- **Strategic Concurrency**: Applied only where beneficial (parsing, field processing)
- **Resource Management**: Bounded concurrency with configurable pools
- **Error Aggregation**: Centralized error handling with rich context

### 2. Immutable Domain Models
- **Constructor Patterns**: All domain objects created via validated constructors
- **Thread Safety**: Immutable models enable safe concurrent processing
- **Validation**: Input validation at construction time prevents invalid states
- **Zero External Dependencies**: Domain models have no external dependencies

### 3. Behavior-Driven Testing
- **Functionality Over Implementation**: Tests validate behavior, not internal structure
- **Inline Code Generation**: Generate and compile test code at runtime
- **Zero Maintenance Overhead**: No fixture files or expected output management
- **Comprehensive Coverage**: All 18 annotation types and error scenarios

### 4. Performance-First Design
- **Concurrent Processing**: 40-70% performance improvement with strategic concurrency
- **LRU Caching**: Intelligent caching for type resolution and AST parsing
- **Resource Pooling**: Object pools for expensive operations
- **Memory Optimization**: Explicit cleanup and bounded resource usage

### 5. Enterprise Reliability
- **Graceful Degradation**: Partial processing continues when individual methods fail
- **Rich Error Context**: Detailed error messages with file locations and suggestions
- **Deterministic Output**: Consistent results across different environments
- **Resource Cleanup**: Proper goroutine and resource lifecycle management

### 6. Go Ecosystem Integration
- **Standard Conventions**: Generated code follows Go formatting standards
- **Build Tool Integration**: Native go:generate and CLI support
- **Module System**: Full Go modules support with semantic versioning
- **Cross-Package Resolution**: Intelligent handling of external types and dependencies