# Convergen Project Concept & Requirements

## Overview

Convergen is a Go code generator that creates type-to-type copy functions. It analyzes Go interfaces with special comment annotations and generates efficient copying functions between different struct types.

### Core Value Proposition
- **Automated Code Generation**: Eliminates manual boilerplate for type conversion functions
- **Annotation-Driven**: Uses intuitive comment-based configuration system
- **Type Safety**: Generates statically typed conversion functions with compile-time safety
- **Flexibility**: Supports complex mapping strategies and custom conversion logic
- **Performance**: Generates optimized code with minimal runtime overhead

## System Purpose

The system addresses the common Go development challenge of converting between similar struct types, particularly in scenarios involving:
- API layer to domain model conversion
- Database entities to business objects
- External service responses to internal representations
- Versioned API compatibility layers

## High-Level Architecture

Convergen implements a **pipeline-based architecture** with **event-driven coordination**. The system processes Go source files through four sequential stages, transforming annotated interface definitions into type-safe conversion functions.

**Core Pipeline**: Parser → Builder → Generator → Coordinator
**Supporting Systems**: Domain models, Executors, Emitters, Configuration, Events, Utilities

*For detailed architectural information, see `.claude/02_development_docs/01_architecture_design.md`*

## Annotation System

### Core Annotations
The system uses a rich comment-based annotation system for configuration:

- **`:match name|none`** - Controls field matching strategy
- **`:map <src> <dst>`** - Explicit field mapping between source and destination
- **`:conv <func> <src> [dst]`** - Custom converter function application
- **`:skip <pattern>`** - Skip destination fields matching pattern
- **`:typecast`** - Enable type casting for compatible types
- **`:stringer`** - Use String() methods for conversion
- **`:recv <var>`** - Generate receiver methods instead of standalone functions
- **`:style arg|return`** - Control function signature style

### Interface Marking
Interfaces can be marked for processing in two ways:
1. Named `Convergen` (convention-based)
2. Marked with `:convergen` annotation (explicit)

## Code Generation Process

### Processing Flow
1. **Interface Discovery**: Scan source files for marked interfaces
2. **Method Analysis**: Extract method signatures and analyze source/destination types
3. **Mapping Strategy**: Apply field mapping using name matching, explicit mapping, or custom converters
4. **Annotation Processing**: Handle special annotations for skipping, type casting, etc.
5. **Code Generation**: Produce Go functions with proper imports, error handling, and optimization

### Generated Code Characteristics
- **Type Safe**: All conversions are statically typed
- **Error Handling**: Appropriate error propagation and handling
- **Import Management**: Automatic import resolution and organization
- **Performance Optimized**: Minimal runtime overhead with direct field assignments
- **Readable**: Generated code follows Go conventions and is human-readable

## Domain Model

The system uses a sophisticated domain model with constructor patterns for type safety and event-driven communication for loose coupling between components.

*For detailed domain model and architectural patterns, see `.claude/02_development_docs/01_architecture_design.md`*

## Quality Standards

### Testing Strategy
- **Integration Tests**: Fixture-based testing in `tests/` directory
- **Unit Tests**: Comprehensive unit test coverage alongside source files
- **Coverage Target**: Maintains 67%+ test coverage (visible in README badge)
- **Test-Driven Development**: Tests serve as executable documentation

### Code Conventions
- **Go Standards**: Follows Effective Go principles and `gofmt` formatting
- **Naming**: Uses `MixedCaps` naming conventions
- **Error Handling**: Explicit error handling without panics for normal errors
- **Documentation**: Comments explain "why" not "what"
- **Operator Preference**: Uses `<` and `<=` instead of `>` and `>=` (number line model)

### Build and Development
- **Go Version**: Requires Go 1.23+
- **Module Structure**: Follows standard Go project layout with `pkg/` organization
- **Build System**: Makefile-based build system with comprehensive targets
- **Dependency Management**: Minimal external dependencies with security monitoring

## Module Information

- **Module Path**: `github.com/reedom/convergen/v8`
- **Entry Point**: `main.go`
- **Package Organization**: All source code in `pkg/` following standard Go layout
- **Version Compatibility**: Semantic versioning with backward compatibility guarantees

## Development Guidelines

The project follows established patterns for domain model usage, event system integration, and testing practices.

*For detailed development patterns and guidelines, see `CLAUDE.md` and `.claude/00_general_rules/02_coding_guidelines.md`*

## Development Workflow

The project follows Spec-Driven Development (SDD) principles for major changes and features. This involves:
1. Understanding requirements and analyzing existing codebase
2. Creating specification documents in `.spec/` directories
3. Seeking user approval before implementation
4. Systematic implementation following planned approach
5. Verification and validation of results

This ensures that all development work is deliberate, well-planned, and aligned with project goals while maintaining high code quality and architectural consistency.

