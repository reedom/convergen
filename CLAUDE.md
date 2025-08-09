# CLAUDE.md

## Quick Reference

- **SDD Workflow**:     See `.claude/docs/guidelines/SDD.md`
- **Code Conventions**: See `.claude/docs/guidelines/coding_guidelines.md`
- **Architecture**:     See `.claude/docs/project/architecture_design.md`
- **Test Strategy**:    See `.claude/docs/project/test_strategy.md`

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

### Behavior-Driven Testing Framework
- **New Testing Approach**: Replaced file comparison with behavior-driven testing
- `go test ./tests -v` - Run comprehensive behavior-driven tests
- `go test ./tests -run TestAnnotationCoverage -v` - Test annotation coverage
- `go test ./tests -run TestErrorScenarios -v` - Test error conditions
- `go test ./tests/examples -v` - Run framework examples
- **Framework Benefits**: Zero maintenance overhead, tests actual functionality
- **Documentation**: See `tests/README.md`

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

### 💡 Pro Tips for Future Claude
- **Domain model issues?** → Always check constructor patterns in CLAUDE.md first
- **Test failures?** → Check test strategy doc for new domain model patterns
- **Architectural questions?** → Architecture design doc has comprehensive answers
- **Planning complex work?** → Use SDD workflow to structure the approach
- **Code style questions?** → Coding guidelines have the answers
