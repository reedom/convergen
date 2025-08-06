# Config Package Design

This document outlines the technical architecture and design decisions for the `pkg/config` package.

## Architecture Overview

The config package follows a simple, focused architecture for command-line configuration management:

1. **Command Line Parsing**: Flag-based argument processing with environment fallbacks
2. **Cross-Package Type Support**: Advanced generic interface instantiation configuration  
3. **Validation Layer**: Comprehensive input validation with rich error reporting
4. **Output Path Generation**: Intelligent default path generation with customization

## Core Design Patterns

### Configuration Object Pattern

The config package uses a single `Config` struct to hold all configuration state:

```go
type Config struct {
    // Core configuration
    Input   string
    Output  string
    Log     string
    DryRun  bool
    Prints  bool
    
    // Cross-package type support
    TypeSpec  string
    ImportMap map[string]string
}
```

**Design Rationale**: Centralized configuration state with clear separation between basic and advanced features.

### Validation Strategy Pattern

Multi-layer validation with specific error types:

```go
func (c *Config) validateCrossPackageConfig() error
func (c *Config) validateTypeSpec() error
func (c *Config) validatePackageAlias(alias string) error
func (c *Config) validateImportPath(importPath string) error
```

**Design Principle**: Early validation with descriptive error messages prevents downstream issues.

### Default Generation Pattern

Intelligent default value generation:

```go
// Default output path generation
ext := path.Ext(inputPath)
c.Output = inputPath[0:len(inputPath)-len(ext)] + ".gen" + ext

// Conditional log path generation
if *logs {
    ext := path.Ext(c.Output)
    c.Log = c.Output[0:len(c.Output)-len(ext)] + ".log"
}
```

**Design Benefit**: Reduces user configuration burden while maintaining full customization capability.

## Component Architecture

### Command Line Interface

Flag-based configuration with standard Go flag package:

```go
func (c *Config) ParseArgs() error {
    output := flag.String("out", "", "Set the output file path")
    logs := flag.Bool("log", false, "Write log messages to <output path>.log.")
    dryRun := flag.Bool("dry", false, "Perform a dry run without writing files.")
    prints := flag.Bool("print", false, "Print the resulting code to STDOUT as well.")
    
    // Advanced features
    typeSpec := flag.String("type", "", "Specify generic interface to instantiate")
    imports := flag.String("imports", "", "Package alias mappings")
}
```

**Design Philosophy**: Standard Unix conventions with clear flag naming and help text.

### Cross-Package Type System

Advanced configuration for generic interface instantiation:

```go
type Config struct {
    TypeSpec  string            // "TypeMapper[models.User,dto.UserDTO]"
    ImportMap map[string]string // {"models": "./internal/models", "dto": "./pkg/dto"}
}

func (c *Config) parseImportMap(importsStr string) (map[string]string, error)
func (c *Config) hasQualifiedTypes() bool
func (c *Config) GetPackageAlias(alias string) (string, bool)
```

**Design Innovation**: Enables complex cross-package type specifications while maintaining simple CLI syntax.

### Validation Architecture

Comprehensive validation with specific error handling:

```go
var (
    ErrInvalidImportMappingFormat    = errors.New("invalid import mapping format")
    ErrEmptyPackageAlias             = errors.New("empty package alias in mapping")
    ErrPackageAliasInvalidIdentifier = errors.New("package alias must be a valid Go identifier")
    ErrTypeSpecWithQualifiedTypes    = errors.New("type specification contains qualified types but no import mappings provided")
)
```

**Design Pattern**: Static error definitions for consistent error handling and testing.

## Key Design Features

### Input Source Priority

Configuration values resolved in priority order:
1. Command line flags (highest priority)
2. Environment variables (GOFILE for input)
3. Generated defaults (lowest priority)

**Design Rationale**: Follows Unix conventions while providing sensible automation.

### Regex-Based Validation

Go identifier validation using compiled regex:

```go
identifierRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
func (c *Config) isValidIdentifier(id string) bool {
    return id != "" && identifierRegex.MatchString(id)
}
```

**Design Trade-off**: Regex compilation cost vs. validation accuracy and Go language compliance.

### Qualified Type Detection

Intelligent parsing of type specifications:

```go
func (c *Config) hasQualifiedTypes() bool {
    if strings.Contains(c.TypeSpec, "[") && strings.Contains(c.TypeSpec, ".") {
        start := strings.Index(c.TypeSpec, "[")
        end := strings.LastIndex(c.TypeSpec, "]")
        if start != -1 && end != -1 && end > start {
            typeArgs := c.TypeSpec[start+1 : end]
            return strings.Contains(typeArgs, ".")
        }
    }
    return false
}
```

**Design Approach**: Simple string-based parsing for type argument detection without full AST parsing.

## Current Implementation Status

### ✅ **Core CLI Functionality**
**Status**: Fully implemented and tested
- Command line argument parsing with standard Go flag package
- Environment variable fallback for input detection
- Default output path generation with proper file extension handling
- Dry run and print mode support

### ✅ **Cross-Package Type Support**
**Status**: Fully implemented with comprehensive validation
- Generic interface type specification parsing
- Package alias to import path mapping
- Qualified type detection and validation
- Import mapping syntax support

### ✅ **Validation System**
**Status**: Production-ready with rich error reporting
- Package alias validation against Go identifier rules
- Import path format validation
- Type specification consistency checking
- Comprehensive error messages with context

### ✅ **Usage and Help System**
**Status**: Complete with examples and documentation
- Integrated usage information with flag.Usage
- Cross-package type examples in help text
- Clear syntax documentation for complex features

## Design Decisions

### Single Configuration Object

**Decision**: Use single Config struct rather than separate structs for different feature areas
**Rationale**: Simplifies API and reduces complexity for users
**Trade-off**: Larger struct vs. more complex interface with multiple types

### String-Based Type Parsing

**Decision**: Use string manipulation for type specification parsing rather than full AST parsing
**Rationale**: Sufficient for current needs, much simpler implementation, better performance
**Alternative**: Full AST parsing was considered but adds significant complexity

### Flag Package Integration

**Decision**: Use standard Go flag package rather than third-party CLI libraries
**Rationale**: 
- Zero external dependencies
- Standard Go ecosystem integration
- Familiar patterns for Go developers
**Trade-off**: Less advanced features vs. simplicity and compatibility

### Map-Based Import Configuration

**Decision**: Use map[string]string for package alias mappings
**Rationale**: Natural fit for alias->path relationships, efficient lookup
**Alternative**: Struct-based mapping was considered but adds unnecessary complexity

## Future Enhancement Opportunities

### Configuration File Support
- YAML/TOML configuration file support for complex projects
- Configuration file discovery and merging with CLI flags
- Environment-specific configuration profiles

### Advanced Type Parsing
- Full AST-based type specification parsing
- Support for more complex generic constraints
- Integration with go/types for enhanced validation

### Plugin Configuration
- Configuration for custom converter plugins
- Plugin-specific configuration sections
- Dynamic plugin discovery and configuration