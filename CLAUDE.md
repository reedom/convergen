# Claude Code Spec-Driven Development

Kiro-style Spec Driven Development implementation using claude code slash commands, hooks and agents.

## Project Context

### Paths
- **Steering**: `.kiro/steering/`
- **Specs**: `.kiro/specs/`
- **Commands**: `.claude/commands/`

### Steering vs Specification

**Steering** (`.kiro/steering/`) - Guide AI with project-wide rules and context
**Specs** (`.kiro/specs/`) - Formalize development process for individual features

### Active Specifications
- Check `.kiro/specs/` for active specifications
- Use `/kiro:spec-status [feature-name]` to check progress

## Development Guidelines
Think in English, generate responses in the language that your asks unless the user
specifically requests a different language.

## Workflow

### Phase 0: Steering (Optional)
`/kiro:steering` - Create/update steering documents
`/kiro:steering-custom` - Create custom steering for specialized contexts

**Note**: Optional for new features or small additions. Can proceed directly to spec-init.

### Phase 1: Specification Creation
1. `/kiro:spec-init [detailed description]` - Initialize spec with detailed project description
2. `/kiro:spec-requirements [feature]` - Generate requirements document
3. `/kiro:spec-design [feature]` - Interactive: "requirements.mdをレビューしましたか？ [y/N]"
4. `/kiro:spec-tasks [feature]` - Interactive: Confirms both requirements and design review

### Phase 2: Progress Tracking
`/kiro:spec-status [feature]` - Check current progress and phases

## Development Rules
1. **Consider steering**: Run `/kiro:steering` before major development (optional for new features)
2. **Follow 3-phase approval workflow**: Requirements → Design → Tasks → Implementation
3. **Approval required**: Each phase requires human review (interactive prompt or manual)
4. **No skipping phases**: Design requires approved requirements; Tasks require approved design
5. **Update task status**: Mark tasks as completed when working on them
6. **Keep steering current**: Run `/kiro:steering` after significant changes
7. **Check spec compliance**: Use `/kiro:spec-status` to verify alignment

## Steering Configuration

### Current Steering Files
Managed by `/kiro:steering` command. Updates here reflect command changes.

### Active Steering Files
- `product.md`: Always included - Product context and business objectives
- `tech.md`: Always included - Technology stack and architectural decisions
- `structure.md`: Always included - File organization and code patterns

### Custom Steering Files
<!-- Added by /kiro:steering-custom command -->
<!-- Format:
- `filename.md`: Mode - Pattern(s) - Description
  Mode: Always|Conditional|Manual
  Pattern: File patterns for Conditional mode
-->

### Inclusion Modes
- **Always**: Loaded in every interaction (default)
- **Conditional**: Loaded for specific file patterns (e.g., `"*.test.js"`)
- **Manual**: Reference with `@filename.md` syntax

## Quick Reference

- **Code Conventions**: See `.claude/docs/guidelines/coding_guidelines.md`
- **Architecture**: See `.claude/docs/project/architecture_design.md`
- **Test Strategy**: See `.claude/docs/project/test_strategy.md`

## Key Development Commands

### Build & Development
- `make build` - Build the CLI command to `build/convergen`
- `make test` - Run all tests (both integration tests and package tests)
- `make coverage` - Generate test coverage report
- `go run main.go <input-file>` - Run convergen directly on a file
- `go run github.com/reedom/convergen@v9` - Run as go:generate command

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

- **Module Path**: `github.com/reedom/convergen/v9`
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

## Code Conventions

### Comparison Operators

To maintain consistency and readability, all comparisons should follow the model of a number line. This means:

-   **Use `<` and `<=`**: Always prefer the "less than" and "less than or equal to" operators.
-   **Avoid `>` and `>=`**: Do not use the "greater than" and "greater than or equal to" operators. Re-order the expression to use `<` or `<=` instead.

### 💡 Pro Tips for Future Claude
- **Starting new work?** → Use Kiro SDD workflow: `/kiro:spec-init` → `/kiro:spec-requirements` → `/kiro:spec-design` → `/kiro:spec-tasks`
- **Domain model issues?** → Check constructor patterns and steering docs (`.kiro/steering/`)
- **Test failures?** → Check test strategy doc and behavior-driven testing approach (`tests/README.md`)
- **Architectural questions?** → Check steering docs and architecture design documents
- **Planning complex work?** → Always use SDD 3-phase workflow with human approval gates
- **Code style questions?** → Check coding guidelines and tech stack documentation
- **Check progress?** → Use `/kiro:spec-status` to verify spec compliance and current phase
- **Generate Japanese responses** → Think in English, but respond in Japanese as per development guidelines
