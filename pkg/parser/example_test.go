package parser_test

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/reedom/convergen/v8/pkg/parser"
)

// ExampleNewParser demonstrates basic parser usage with default configuration.
func ExampleNewParser() {
	// Create a parser for a source file containing convergen interfaces
	parser, err := parser.NewParser("models.go", "models_gen.go")
	if err != nil {
		log.Fatal(err)
	}

	// Parse the source file to extract method information
	methodsInfo, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	// Display discovered interfaces and methods
	for _, info := range methodsInfo {
		fmt.Printf("Interface marker: %s\n", info.Marker)
		fmt.Printf("Methods found: %d\n", len(info.Methods))
	}

	// Create a builder for code generation
	builder := parser.CreateBuilder()
	_ = builder // Use in generation pipeline

	// Generate base code without convergen annotations
	baseCode, err := parser.GenerateBaseCode()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Base code length: %d characters\n", len(baseCode))
}

// ExampleNewParserWithConfig demonstrates advanced parser configuration.
func ExampleNewParserWithConfig() {
	// Create custom parser configuration for high-performance scenarios
	config := &parser.ParserConfig{
		EnableConcurrentLoading: true,
		EnableMethodConcurrency: true,
		MaxConcurrentWorkers:    8,
		TypeResolutionTimeout:   30 * time.Second,
		CacheSize:               1000,
		EnableProgress:          true,
	}

	// Create parser with custom configuration
	parser, err := parser.NewParserWithConfig("large_models.go", "large_models_gen.go", config)
	if err != nil {
		log.Fatal(err)
	}

	// Parse with concurrent processing enabled
	methodsInfo, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %d interface(s) with concurrent processing\n", len(methodsInfo))
}

// ExampleParser_basicWorkflow demonstrates a complete basic parsing workflow.
func ExampleParser_basicWorkflow() {
	// Step 1: Create parser
	parser, err := parser.NewParser("example.go", "example_gen.go")
	if err != nil {
		log.Fatal(err)
	}

	// Step 2: Parse source file
	methodsInfo, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	// Step 3: Process discovered methods
	for i, info := range methodsInfo {
		fmt.Printf("Interface %d:\n", i+1)
		for j, method := range info.Methods {
			fmt.Printf("  Method %d: %s\n", j+1, method.Name())
		}
	}

	// Step 4: Generate base code for further processing
	baseCode, err := parser.GenerateBaseCode()
	if err != nil {
		log.Fatal(err)
	}

	// Step 5: Create builder for generation pipeline
	builder := parser.CreateBuilder()
	_ = builder

	fmt.Printf("Workflow completed successfully, base code ready\n")
	_ = baseCode
}

// ExampleParser_advancedConfiguration shows advanced configuration options.
func ExampleParser_advancedConfiguration() {
	// Configure for a large project with complex type hierarchies
	config := &parser.ParserConfig{
		EnableConcurrentLoading: true,             // Enable parallel package loading
		EnableMethodConcurrency: true,             // Enable parallel method processing
		MaxConcurrentWorkers:    16,               // High worker count for large projects
		TypeResolutionTimeout:   60 * time.Second, // Extended timeout for complex types
		CacheSize:               2000,             // Large cache for type resolution
		EnableProgress:          true,             // Track progress
	}

	sourcePath := filepath.Join("internal", "models", "converters.go")
	destPath := filepath.Join("internal", "models", "converters_gen.go")

	parser, err := parser.NewParserWithConfig(sourcePath, destPath, config)
	if err != nil {
		log.Fatal(err)
	}

	// Parse with performance monitoring
	methodsInfo, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	// Display metrics if available (this would be implementation-specific)
	fmt.Printf("Parsed %d interfaces with advanced configuration\n", len(methodsInfo))
}

// ExampleParser_errorHandling demonstrates proper error handling patterns.
func ExampleParser_errorHandling() {
	parser, err := parser.NewParser("nonexistent.go", "output.go")
	if err != nil {
		// Handle parser creation errors
		if strings.Contains(err.Error(), "no such file") {
			fmt.Println("Source file not found - please check the path")
			return
		}
		log.Fatal(err)
	}

	methodsInfo, err := parser.Parse()
	if err != nil {
		// Handle parsing errors
		if strings.Contains(err.Error(), "syntax error") {
			fmt.Println("Syntax error in source file - please fix and retry")
			return
		}
		if strings.Contains(err.Error(), "type resolution") {
			fmt.Println("Type resolution failed - check imports and dependencies")
			return
		}
		log.Fatal(err)
	}

	if len(methodsInfo) == 0 {
		fmt.Println("No convergen interfaces found in source file")
		return
	}

	fmt.Printf("Successfully parsed %d interface(s)\n", len(methodsInfo))
}

// ExampleParser_performanceOptimization shows performance optimization techniques.
func ExampleParser_performanceOptimization() {
	// Configuration optimized for performance
	config := &parser.ParserConfig{
		EnableConcurrentLoading: true,             // Enable concurrency for I/O bound operations
		EnableMethodConcurrency: true,             // Enable parallel method processing
		MaxConcurrentWorkers:    8,                // Optimal for most systems
		TypeResolutionTimeout:   10 * time.Second, // Shorter timeout for faster failure
		CacheSize:               500,              // Moderate cache size
		EnableProgress:          true,             // Monitor progress
	}

	parser, err := parser.NewParserWithConfig("fast_models.go", "fast_models_gen.go", config)
	if err != nil {
		log.Fatal(err)
	}

	// Time the parsing operation
	start := time.Now()
	methodsInfo, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(start)

	fmt.Printf("Parsed %d interfaces in %v\n", len(methodsInfo), duration)
}

// ExampleParser_memoryOptimization demonstrates memory-conscious configuration.
func ExampleParser_memoryOptimization() {
	// Configuration optimized for low memory usage
	config := &parser.ParserConfig{
		EnableConcurrentLoading: false,            // Disable concurrency to save memory
		EnableMethodConcurrency: false,            // Process sequentially
		MaxConcurrentWorkers:    1,                // Single worker
		TypeResolutionTimeout:   30 * time.Second, // Allow longer processing time
		CacheSize:               100,              // Smaller cache
		EnableProgress:          false,            // Disable progress to save memory
	}

	parser, err := parser.NewParserWithConfig("memory_efficient.go", "memory_efficient_gen.go", config)
	if err != nil {
		log.Fatal(err)
	}

	methodsInfo, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Memory-efficient parsing completed: %d interfaces\n", len(methodsInfo))
}
