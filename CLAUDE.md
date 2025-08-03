# CLAUDE.md

## Quick Reference

- **SDD Workflow**:     See `.claude/00_general_rules/01_sdd_workflow_concepts.md`
- **Code Conventions**: See `.claude/00_general_rules/02_coding_guidelines.md`
- **Project Overview**: See `.claude/01_project/01_convergen_concept_requirements.md`
- **Architecture**:     See `.claude/02_development_docs/01_architecture_design.md`
- **Test Strategy**:    See `.claude/02_development_docs/02_test_strategy.md`
- **Logging Strategy**: See `.claude/02_development_docs/03_logging_strategy.md`

## Key Development Commands

### Build & Development
- `make build` - Build the CLI command to `build/convergen`
- `make test` - Run all tests (both integration tests and package tests)
- `make coverage` - Generate test coverage report
- `go run main.go <input-file>` - Run convergen directly on a file
- `go run github.com/reedom/convergen@v8.0.3` - Run as go:generate command

### Code Quality & Linting
- `make lint` - Run comprehensive linting with golangci-lint (no Docker required)
- `make lint-fix` - Run linter and automatically fix issues where possible
- `make lint-all` - Run all linting checks (comprehensive analysis)
- `make lint-security` - Run security-focused linting with gosec
- `make lint-complexity` - Check code complexity and maintainability
- `make lint-deps` - Check for dependency issues and vulnerabilities
- `make lint-docker` - Run linter using Docker (fallback option)
- `make install-linters` - Install all recommended linting tools

### Code Quality Workflow (REQUIRED)
- **ALWAYS run unit tests and linting after modifying packages**: This ensures consistent code quality
- **Required workflow after package modification**: 
  1. `go test ./pkg/package-name/...` - Run unit tests for the modified package
  2. `make fmt` - Format all Go code (includes gofmt + goimports)
  3. `make lint` - Run comprehensive linting checks (includes formatting)
  4. Commit only after all tests pass and linting is clean

### Code Formatting & Linting Commands
- `make fmt` - Format all Go code (includes gofmt + goimports)
- `make lint` - Run comprehensive linting checks (includes formatting validation)
- `make fmt-check` - Check if code is properly formatted (CI-friendly)
- **Package-specific linting**: `golangci-lint run ./pkg/package-name/...`
- **Format specific packages**: `go fmt ./pkg/package-name/...`
- **Format entire project**: `go fmt ./...`

### Testing Individual Packages (REQUIRED)
- **ALWAYS test modified packages before committing**: Ensures functionality remains intact
- `go test ./pkg/package-name/...` - Test specific package (REQUIRED after modifications)
- `go test -v ./pkg/package-name/...` - Run tests with verbose output for debugging
- `go test -run TestSpecificTest ./pkg/package-name/...` - Run specific test by name
- `go test github.com/reedom/convergen/v8/tests` - Run behavior-driven integration tests
- `go test github.com/reedom/convergen/v8/pkg/...` - Run all package tests
- **Package-specific linting**: `golangci-lint run ./pkg/package-name/...`

### Behavior-Driven Testing Framework
- **New Testing Approach**: Replaced file comparison with behavior-driven testing
- `go test ./tests -v` - Run comprehensive behavior-driven tests
- `go test ./tests -run TestAnnotationCoverage -v` - Test annotation coverage
- `go test ./tests -run TestErrorScenarios -v` - Test error conditions
- `go test ./tests/examples -v` - Run framework examples
- **Framework Benefits**: Zero maintenance overhead, tests actual functionality
- **Documentation**: See `tests/README.md` and `tests/CONTRIBUTING.md`

## Current Module Information

- **Module Path**: `github.com/reedom/convergen/v8`
- **Go Version**: 1.21+
- **Entry Point**: `main.go`
- **Package Layout**: Standard Go project layout with `pkg/` organization

## Key Package Overview

- **`pkg/domain/`** - Core domain models and types (use constructors!)
- **`pkg/parser/`** - **Enhanced AST parsing with concurrent processing** (see Enhanced Parser Features below)
- **`pkg/builder/`** - Type conversion logic and field mapping
- **`pkg/executor/`** - Field mapping strategy execution
- **`pkg/generator/`** - Go code generation from models
- **`pkg/emitter/`** - Final code emission with optimization
- **`pkg/coordinator/`** - Pipeline orchestration with events
- **`pkg/config/`** - Configuration management
- **`pkg/util/`** - AST utilities and type checking
- **`pkg/internal/events/`** - Event-driven communication

## Enhanced Parser Features (pkg/parser/)

The parser package has been significantly enhanced with production-ready concurrent processing capabilities:

### Parser Strategies
- **LegacyParser**: Traditional synchronous parsing (backward compatible)
- **ModernParser**: Concurrent processing with worker pools (40-70% performance improvement)
- **AdaptiveParser**: Automatically selects optimal strategy based on input complexity

### Usage Patterns
```go
// Basic usage (backward compatible)
parser, err := parser.NewParser(sourcePath, destPath)

// High-performance concurrent processing
config := parser.NewConcurrentParserConfig()
modernParser := parser.NewModernParser(config)
result, err := modernParser.ParseSourceFile(ctx, sourcePath, destPath)

// Adaptive strategy selection
factory := parser.NewParserFactory(nil)
adaptiveParser, err := factory.CreateParser(parser.StrategyAuto)
```

### Key Components
- **`config.go`** - Centralized configuration with functional options (`WithTimeout()`, `WithConcurrency()`, etc.)
- **`unified_interface.go`** - Strategy pattern implementation with factory
- **`package_loader.go`** - Concurrent package loading with worker pools
- **`concurrent_method.go`** - Concurrent method processing with error recovery
- **`error_handler.go`** - Rich contextual error system with categorization
- **`error_recovery.go`** - Circuit breaker pattern and retry logic
- **`error_classification.go`** - Pattern-based error classification and suggestions

### Performance & Reliability
- **Concurrent Processing**: 40-70% improvement for complex scenarios
- **Circuit Breaker**: Fault tolerance with exponential backoff
- **Error Recovery**: Panic recovery and graceful degradation  
- **Comprehensive Metrics**: Performance monitoring and cache hit rates
- **Type Caching**: Intelligent caching with memory management

## Essential Development Patterns

### Domain Model Usage (CRITICAL)
The project underwent major domain model restructuring. Always follow these patterns:

**✅ Use Constructors (Required)**:
```go
// Use constructors for all domain objects
sourceType := domain.NewBasicType("User", reflect.Struct)
method, err := domain.NewMethod("ConvertUser", sourceType, destType)

// ❌ NEVER use direct struct literals
// method := &domain.Method{Name: "ConvertUser", ...}
```

**✅ Event System Pattern**:
```go
// EventHandler interface calls
err := handler.Handle(ctx, event)

// Event publishing
err := eventBus.Publish(event)  // Single parameter only

// ❌ NOT: err := handler(ctx, event)
// ❌ NOT: eventBus.Publish(ctx, event)
```

**✅ Result Structures**:
```go
// New MethodResult structure
result := &domain.MethodResult{
    Method:      method,          // NOT MethodName
    Code:        "generated code", // NOT Result field
    Success:     true,
    Error:       nil,
    ProcessedAt: time.Now(),
    DurationMS:  5,
}
```

### Test Development Guidelines
- Replace legacy field patterns: `MethodName` → `Method` (with proper Method object)
- Replace: `Success/Result/StrategyUsed` → `Code/Error/ProcessedAt/DurationMS`
- Use proper domain constructors in test setup
- Import `reflect` package when working with `BasicType`

### Build Tags and Multiple Main Functions
- Use `//go:build tools` and `// +build tools` for verification utilities
- Prevents conflicts with main package compilation

## Annotation System Reference

- `:match name|none` - Field matching strategy
- `:map <src> <dst>` - Explicit field mapping  
- `:conv <func> <src> [dst]` - Custom converter functions
- `:skip <pattern>` - Skip destination fields
- `:typecast` - Allow type casting
- `:stringer` - Use String() methods
- `:recv <var>` - Generate receiver methods
- `:style arg|return` - Function signature style

## Project Context

**What it does**: Convergen generates Go type-to-type copy functions from annotated interfaces
**How it works**: 4-stage pipeline (Parser → Builder → Generator → Coordinator) with event-driven architecture
**Testing**: Integration tests in `tests/`, unit tests alongside source, 67%+ coverage target
**Recent Changes**: Major domain model restructuring completed, use constructor patterns throughout

## Spec-Driven Development (SDD) Workflow

For complex changes, follow SDD workflow documented in `.claude/00_general_rules/01_sdd_workflow_concepts.md`:
1. Understand requirements and analyze codebase
2. Create `.spec/` directory with requirements.md, design.md, tasks.md
3. Seek user approval before implementation
4. Implement systematically following the plan
5. Verify and validate results

---

## 📚 Documentation Reference Guide

### When to Read Specific Documents

#### 🔧 **General Rules & Guidelines**

**`.claude/00_general_rules/01_sdd_workflow_concepts.md`**
- **Read when**: Starting major features, refactoring, or architectural changes
- **Purpose**: Spec-Driven Development methodology with EARS notation
- **Contains**: Workflow steps, requirements templates, planning guidelines
- **Use for**: Complex tasks requiring systematic planning and user approval

**`.claude/00_general_rules/02_coding_guidelines.md`**
- **Read when**: Writing new code, reviewing code, onboarding to project
- **Purpose**: Code conventions and Effective Go practices
- **Contains**: Operator preferences, naming conventions, formatting rules
- **Use for**: Ensuring code consistency and following project standards

#### 📋 **Project Understanding**

**`.claude/01_project/01_convergen_concept_requirements.md`**
- **Read when**: New to the project, explaining project to others, planning features
- **Purpose**: High-level project concept and requirements
- **Contains**: Value proposition, system purpose, annotation system overview
- **Use for**: Understanding what Convergen does and why it exists

#### 🏗️ **Development & Architecture**

**`.claude/02_development_docs/01_architecture_design.md`**
- **Read when**: Implementing complex features, debugging system issues, architectural decisions
- **Purpose**: Comprehensive technical architecture reference
- **Contains**: Pipeline stages, domain models, design patterns, data flow
- **Use for**: Understanding how components interact and system design decisions

**`.claude/02_development_docs/02_test_strategy.md`**
- **Read when**: Writing tests, debugging test failures, updating domain model tests
- **Purpose**: Testing methodology and patterns
- **Contains**: Test types, domain model testing patterns, coverage targets
- **Use for**: Following correct testing patterns, especially for domain model constructor usage

**`.claude/02_development_docs/03_logging_strategy.md`** *(if exists)*
- **Read when**: Adding logging, debugging issues, implementing observability
- **Purpose**: Logging standards and practices
- **Use for**: Consistent logging patterns across the system

### 🎯 Situation-Specific Reading Recommendations

#### **Starting New Feature Development**
1. **Always read**: This `CLAUDE.md` for development patterns
2. **For context**: `.claude/01_project/01_convergen_concept_requirements.md`
3. **For complex features**: `.claude/00_general_rules/01_sdd_workflow_concepts.md`
4. **For architecture changes**: `.claude/02_development_docs/01_architecture_design.md`

#### **Fixing Domain Model Issues**
1. **Start with**: This `CLAUDE.md` Essential Development Patterns section
2. **Deep dive**: `.claude/02_development_docs/01_architecture_design.md` Domain Model section
3. **For tests**: `.claude/02_development_docs/02_test_strategy.md` Domain Model Testing Patterns

#### **Debugging Pipeline Issues**
1. **Architecture understanding**: `.claude/02_development_docs/01_architecture_design.md`
2. **Event system patterns**: This `CLAUDE.md` Event System Pattern section
3. **Testing approach**: `.claude/02_development_docs/02_test_strategy.md`

#### **Code Quality & Standards**
1. **Conventions**: `.claude/00_general_rules/02_coding_guidelines.md`
2. **Testing patterns**: `.claude/02_development_docs/02_test_strategy.md`
3. **Architecture compliance**: `.claude/02_development_docs/01_architecture_design.md`

#### **Major Refactoring or System Changes**
1. **Planning**: `.claude/00_general_rules/01_sdd_workflow_concepts.md`
2. **Requirements**: `.claude/01_project/01_convergen_concept_requirements.md`
3. **Architecture**: `.claude/02_development_docs/01_architecture_design.md`
4. **Testing strategy**: `.claude/02_development_docs/02_test_strategy.md`

#### **Onboarding to Project**
**Recommended reading order**:
1. This `CLAUDE.md` (overview and patterns)
2. `.claude/01_project/01_convergen_concept_requirements.md` (project understanding)
3. `.claude/00_general_rules/02_coding_guidelines.md` (code standards)
4. `.claude/02_development_docs/01_architecture_design.md` (technical depth)
5. `.claude/02_development_docs/02_test_strategy.md` (testing approach)

### 💡 Pro Tips for Future Claude
- **Domain model issues?** → Always check constructor patterns in CLAUDE.md first
- **Test failures?** → Check test strategy doc for new domain model patterns
- **Architectural questions?** → Architecture design doc has comprehensive answers
- **Planning complex work?** → Use SDD workflow to structure the approach
- **Code style questions?** → Coding guidelines have the answers
