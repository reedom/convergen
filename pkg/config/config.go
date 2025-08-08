// Package config provides configuration management for the convergen CLI tool.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	// ErrInvalidImportMappingFormat is returned when the import mapping format is invalid.
	ErrInvalidImportMappingFormat = errors.New("invalid import mapping format")
	// ErrEmptyPackageAlias is returned when a package alias is empty in mapping.
	ErrEmptyPackageAlias = errors.New("empty package alias in mapping")
	// ErrEmptyImportPath is returned when an import path is empty in mapping.
	ErrEmptyImportPath = errors.New("empty import path in mapping")
	// ErrPackageAliasCannotBeEmpty is returned when a package alias is empty.
	ErrPackageAliasCannotBeEmpty = errors.New("package alias cannot be empty")
	// ErrPackageAliasInvalidIdentifier is returned when a package alias is not a valid Go identifier.
	ErrPackageAliasInvalidIdentifier = errors.New("package alias must be a valid Go identifier")
	// ErrImportPathCannotBeEmpty is returned when an import path is empty.
	ErrImportPathCannotBeEmpty = errors.New("import path cannot be empty")
	// ErrImportPathCannotContainSpaces is returned when an import path contains spaces.
	ErrImportPathCannotContainSpaces = errors.New("import path cannot contain spaces")
	// ErrTypeSpecWithQualifiedTypes is returned when type specification contains qualified types but no import mappings provided.
	ErrTypeSpecWithQualifiedTypes = errors.New("type specification contains qualified types but no import mappings provided")
	// ErrInvalidInterfaceName is returned when interface name in type specification is invalid.
	ErrInvalidInterfaceName = errors.New("invalid interface name in type specification")
)

// Usage prints the usage of the tool.
func Usage() {
	var sb strings.Builder

	sb.WriteString("\nUsage: convergen [flags] <input path>\n\n")
	sb.WriteString("By default, the generated code is written to <input path>.gen.go\n\n")
	sb.WriteString("Cross-Package Type Support:\n")
	sb.WriteString("  -type TypeMapper[pkg.User,dto.UserDTO]  Specify generic interface with cross-package types\n")
	sb.WriteString("  -imports pkg=./internal/pkg,dto=./dto   Import mappings for package aliases\n\n")
	sb.WriteString("Examples:\n")
	sb.WriteString("  convergen -type Converter[User,UserDTO] input.go\n")
	sb.WriteString("  convergen -type TypeMapper[models.User,dto.UserDTO] -imports models=./internal/models,dto=./pkg/dto input.go\n\n")
	sb.WriteString("Flags:\n")
	_, _ = fmt.Fprint(os.Stderr, sb.String())

	flag.PrintDefaults()
}

// Config holds the configuration options for the convergen tool.
type Config struct {
	// Input is the path of the input file.
	Input string
	// Output is the path where the generated code will be saved.
	// If empty, the generated code will be saved in the same directory as
	// the input file with the name "<basename>.gen.go".
	Output string
	// Log is the path of the log file where the tool writes logs.
	Log string
	// DryRun instructs convergen not to write the generated code to the output path.
	DryRun bool
	// Prints instructs convergen to print the generated code to stdout.
	Prints bool

	// Cross-package type support
	// TypeSpec specifies the generic interface to instantiate (e.g., "TypeMapper[models.User,dto.UserDTO]")
	TypeSpec string
	// ImportMap maps package aliases to import paths (e.g., "models=./internal/models")
	ImportMap map[string]string
}

// String returns the string representation of the config.
func (c *Config) String() string {
	var sb strings.Builder

	sb.WriteString("config.Config{\n\tInput: \"")
	sb.WriteString(c.Input)
	sb.WriteString("\"\n\tOutput: \"")
	sb.WriteString(c.Output)
	sb.WriteString("\"\n\tLog: \"")
	sb.WriteString(c.Log)
	sb.WriteString("\"\n\tTypeSpec: \"")
	sb.WriteString(c.TypeSpec)
	sb.WriteString("\"\n\tImportMap: ")
	sb.WriteString(formatImportMap(c.ImportMap))
	sb.WriteString("\n}")

	return sb.String()
}

// formatImportMap formats the import map for string representation.
func formatImportMap(importMap map[string]string) string {
	if len(importMap) == 0 {
		return "{}"
	}

	var sb strings.Builder
	sb.WriteString("{")
	first := true
	for alias, path := range importMap {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(alias)
		sb.WriteString(": \"")
		sb.WriteString(path)
		sb.WriteString("\"")
		first = false
	}
	sb.WriteString("}")
	return sb.String()
}

// ParseArgs parses the command line arguments.
func (c *Config) ParseArgs() error {
	output := flag.String("out", "", "Set the output file path")
	logs := flag.Bool("log", false, "Write log messages to <output path>.log.")
	dryRun := flag.Bool("dry", false, "Perform a dry run without writing files.")
	prints := flag.Bool("print", false, "Print the resulting code to STDOUT as well.")

	// Cross-package type support flags
	typeSpec := flag.String("type", "", "Specify generic interface to instantiate (e.g., TypeMapper[models.User,dto.UserDTO])")
	imports := flag.String("imports", "", "Package alias mappings (e.g., models=./internal/models,dto=./pkg/dto)")

	flag.Usage = Usage
	flag.Parse()

	inputPath := flag.Arg(0)
	if inputPath == "" {
		inputPath = os.Getenv("GOFILE")
	}

	if inputPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	c.Input = inputPath
	if *output != "" {
		c.Output = *output
	} else {
		ext := path.Ext(inputPath)
		c.Output = inputPath[0:len(inputPath)-len(ext)] + ".gen" + ext
	}

	if *logs {
		ext := path.Ext(c.Output)
		c.Log = c.Output[0:len(c.Output)-len(ext)] + ".log"
	}

	c.DryRun = *dryRun
	c.Prints = *prints

	// Parse cross-package type configuration
	c.TypeSpec = strings.TrimSpace(*typeSpec)

	importMap, err := c.parseImportMap(*imports)
	if err != nil {
		return fmt.Errorf("failed to parse import mappings: %w", err)
	}
	c.ImportMap = importMap

	// Validate cross-package configuration consistency
	if err := c.validateCrossPackageConfig(); err != nil {
		return fmt.Errorf("invalid cross-package configuration: %w", err)
	}

	return nil
}

// parseImportMap parses import mappings from a string like "models=./internal/models,dto=./pkg/dto".
func (c *Config) parseImportMap(importsStr string) (map[string]string, error) {
	importMap := make(map[string]string)

	if importsStr == "" {
		return importMap, nil
	}

	// Split by comma to get individual mappings
	mappings := strings.Split(importsStr, ",")

	for _, mapping := range mappings {
		mapping = strings.TrimSpace(mapping)
		if mapping == "" {
			continue
		}

		// Split by equals sign to get alias and path
		parts := strings.Split(mapping, "=")
		if len(parts) != 2 {
			return nil, ErrInvalidImportMappingFormat
		}

		alias := strings.TrimSpace(parts[0])
		importPath := strings.TrimSpace(parts[1])

		if alias == "" {
			return nil, ErrEmptyPackageAlias
		}

		if importPath == "" {
			return nil, ErrEmptyImportPath
		}

		// Validate alias is a valid identifier
		if err := c.validatePackageAlias(alias); err != nil {
			return nil, fmt.Errorf("invalid package alias '%s': %w", alias, err)
		}

		// Validate import path
		if err := c.validateImportPath(importPath); err != nil {
			return nil, fmt.Errorf("invalid import path '%s': %w", importPath, err)
		}

		importMap[alias] = importPath
	}

	return importMap, nil
}

// validatePackageAlias validates that a package alias is a valid Go identifier.
func (c *Config) validatePackageAlias(alias string) error {
	if alias == "" {
		return ErrPackageAliasCannotBeEmpty
	}

	// Go identifier regex: starts with letter or underscore, followed by letters, digits, or underscores
	identifierRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !identifierRegex.MatchString(alias) {
		return ErrPackageAliasInvalidIdentifier
	}

	return nil
}

// validateImportPath validates that an import path is valid.
func (c *Config) validateImportPath(importPath string) error {
	if importPath == "" {
		return ErrImportPathCannotBeEmpty
	}

	// Basic validation - no spaces allowed
	if strings.Contains(importPath, " ") {
		return ErrImportPathCannotContainSpaces
	}

	return nil
}

// validateCrossPackageConfig validates the consistency of cross-package configuration.
func (c *Config) validateCrossPackageConfig() error {
	// If TypeSpec is provided, validate it
	if c.TypeSpec != "" {
		if err := c.validateTypeSpec(); err != nil {
			return fmt.Errorf("invalid type specification: %w", err)
		}

		// Check if TypeSpec contains qualified types that require imports
		if c.hasQualifiedTypes() && len(c.ImportMap) == 0 {
			return ErrTypeSpecWithQualifiedTypes
		}
	}

	// If ImportMap is provided without TypeSpec, that's valid - imports might be for other purposes

	return nil
}

// validateTypeSpec validates the type specification format.
func (c *Config) validateTypeSpec() error {
	// Basic format validation - should contain interface name
	if !c.isValidIdentifier(c.getInterfaceName()) {
		return ErrInvalidInterfaceName
	}

	return nil
}

// hasQualifiedTypes checks if the TypeSpec contains qualified type names (pkg.Type).
func (c *Config) hasQualifiedTypes() bool {
	// Simple check for dot notation in type arguments
	// This is a basic implementation - could be enhanced with full parsing
	if strings.Contains(c.TypeSpec, "[") && strings.Contains(c.TypeSpec, ".") {
		// Extract type arguments part
		start := strings.Index(c.TypeSpec, "[")
		end := strings.LastIndex(c.TypeSpec, "]")
		if start != -1 && end != -1 && end > start {
			typeArgs := c.TypeSpec[start+1 : end]
			return strings.Contains(typeArgs, ".")
		}
	}

	return false
}

// getInterfaceName extracts the interface name from TypeSpec.
func (c *Config) getInterfaceName() string {
	if c.TypeSpec == "" {
		return ""
	}

	// Extract interface name (part before '[' if generic)
	if idx := strings.Index(c.TypeSpec, "["); idx != -1 {
		return strings.TrimSpace(c.TypeSpec[:idx])
	}

	return strings.TrimSpace(c.TypeSpec)
}

// isValidIdentifier checks if a string is a valid Go identifier.
func (c *Config) isValidIdentifier(id string) bool {
	if id == "" {
		return false
	}

	identifierRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return identifierRegex.MatchString(id)
}

// HasCrossPackageTypes returns true if the configuration specifies cross-package types.
func (c *Config) HasCrossPackageTypes() bool {
	return c.TypeSpec != "" && c.hasQualifiedTypes()
}

// GetPackageAlias returns the import path for a given package alias.
func (c *Config) GetPackageAlias(alias string) (string, bool) {
	if c.ImportMap == nil {
		return "", false
	}

	path, exists := c.ImportMap[alias]
	return path, exists
}
