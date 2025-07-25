# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Convergen is a Go code generator that creates type-to-type copy functions. It analyzes Go interfaces with special comment annotations and generates efficient copying functions between different struct types.

## Key Commands

### Build & Development
- `make build` - Build the CLI command to `build/convergen`
- `make test` - Run all tests (both integration tests and package tests)
- `make lint` - Run golangci-lint via Docker
- `make coverage` - Generate test coverage report
- `go run main.go <input-file>` - Run convergen directly on a file
- `go run github.com/reedom/convergen@v8.0.3` - Run as go:generate command

### Testing Individual Packages
- `go test github.com/reedom/convergen/v8/tests` - Run integration tests
- `go test github.com/reedom/convergen/v8/pkg/...` - Run all package tests
- `go test ./pkg/builder/...` - Test specific package

## Architecture

### Core Processing Pipeline
1. **Parser** (`pkg/parser/`) - Parses Go source files and extracts interface definitions with annotations
2. **Builder** (`pkg/builder/`) - Analyzes type relationships and builds conversion logic models
3. **Generator** (`pkg/generator/`) - Generates actual Go code from the conversion models
4. **Runner** (`pkg/runner/`) - Orchestrates the entire process

### Key Packages

- **`pkg/config/`** - Command-line argument parsing and configuration
- **`pkg/parser/`** - AST parsing, interface detection, and comment annotation parsing
- **`pkg/builder/`** - Core logic for building type conversion mappings and handling complex field assignments
- **`pkg/generator/`** - Code generation engine that produces the final Go functions
- **`pkg/option/`** - Handles various annotation options (`:map`, `:conv`, `:skip`, etc.)
- **`pkg/util/`** - AST utilities, type checking, and import management
- **`pkg/logger/`** - Logging utilities

### Important Models

- **`builder/model/copier.go`** - Central model representing a conversion function
- **`generator/model/function.go`** - Generated function representation
- **`parser/interface.go`** - Parsed interface and method definitions

## Code Generation Process

1. Parse source file and extract `Convergen` interfaces (or those marked with `:convergen`)
2. For each method in the interface, analyze source and destination types
3. Build field mapping using various strategies (name matching, explicit `:map`, `:conv`)
4. Handle special annotations (`:skip`, `:literal`, `:typecast`, `:stringer`, etc.)
5. Generate Go code with proper imports and error handling

## Annotation System

The project uses a rich comment-based annotation system:
- `:match name|none` - Field matching strategy
- `:map <src> <dst>` - Explicit field mapping  
- `:conv <func> <src> [dst]` - Custom converter functions
- `:skip <pattern>` - Skip destination fields
- `:typecast` - Allow type casting
- `:stringer` - Use String() methods
- `:recv <var>` - Generate receiver methods
- `:style arg|return` - Function signature style

## Testing

The project has comprehensive test coverage:
- Integration tests in `tests/` using fixture-based approach
- Unit tests alongside source files (`*_test.go`)
- Coverage target visible in README badge (currently 67.4%)

## Code Conventions

From CONTRIBUTING.md:
- Use `<` and `<=` operators instead of `>` and `>=` (number line model)
- Follow Effective Go principles
- Use `gofmt` for formatting
- Prefer `MixedCaps` naming
- Explicit error handling with no panics for normal errors
- Comments explain "why" not "what"

## Module Structure

- Go module: `github.com/reedom/convergen/v8`
- Go version: 1.23+
- Main entry point: `main.go`
- All source code in `pkg/` following standard Go project layout

## Spec-Driven Development (SDD) Workflow

Read ./CONCEPTS.md for the SDD workflow.
