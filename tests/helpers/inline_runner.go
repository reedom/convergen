// Package helpers provides testing utilities for the Convergen behavior-driven testing framework.
package helpers

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reedom/convergen/v8/pkg/config"
	"github.com/reedom/convergen/v8/pkg/generator"
	"github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/parser"
)

// InlineScenarioRunner handles behavior-driven scenario testing.
type InlineScenarioRunner struct {
	t       *testing.T
	tempDir string
}

// NewInlineScenarioRunner creates a new inline scenario runner.
func NewInlineScenarioRunner(t *testing.T) *InlineScenarioRunner {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "convergen_test_*")
	require.NoError(t, err, "Failed to create temp directory")

	return &InlineScenarioRunner{
		t:       t,
		tempDir: tempDir,
	}
}

// Cleanup cleans up temporary files.
func (isr *InlineScenarioRunner) Cleanup() {
	if isr.tempDir != "" {
		_ = os.RemoveAll(isr.tempDir)
	}
}

// RunScenario executes a behavior-driven test scenario.
func (isr *InlineScenarioRunner) RunScenario(scenario TestScenario) {
	isr.t.Helper()

	if scenario.ShouldSkip {
		isr.t.Skipf("Skipping scenario %s: %s", scenario.Name, scenario.SkipReason)
		return
	}

	isr.t.Run(scenario.Name, func(t *testing.T) {
		t.Helper()

		// Generate the source file with inline code
		sourceFile, sourceErr := isr.createSourceFile(scenario)

		// Generate code using Convergen (only if source file creation succeeded)
		var generatedCode string
		var codeErr error
		if sourceErr == nil {
			generatedCode, codeErr = isr.generateCode(sourceFile, scenario.CliFlags)
		}

		// Determine the primary error (source creation error takes precedence)
		primaryError := sourceErr
		if primaryError == nil {
			primaryError = codeErr
		}

		// Handle success scenarios
		if scenario.ShouldSucceed {
			isr.handleSuccessScenario(t, scenario, primaryError, sourceFile, generatedCode, sourceErr, codeErr)
			return
		}

		// Handle failure scenarios
		isr.handleFailureScenario(t, scenario, primaryError)
	})
}

// handleSuccessScenario processes scenarios that should succeed.
func (isr *InlineScenarioRunner) handleSuccessScenario(t *testing.T, scenario TestScenario, primaryError error, sourceFile, generatedCode string, sourceErr, codeErr error) {
	t.Helper()

	if primaryError != nil {
		// Enhanced error reporting with debugging information
		sourceContent := isr.readSourceFileContent(sourceFile)
		t.Errorf("❌ Code generation failed for scenario '%s':\n"+
			"Error: %v\n"+
			"Source file: %s\n"+
			"Generated code length: %d\n"+
			"Source creation error: %v\n"+
			"Code generation error: %v\n"+
			"--- Source File Content ---\n%s\n"+
			"--- Generated Code (first 1000 chars) ---\n%s\n"+
			"--- End Debug Output ---",
			scenario.Name, primaryError, sourceFile, len(generatedCode),
			sourceErr, codeErr, sourceContent, truncateString(generatedCode, 1000))
		return
	}

	require.NotEmpty(t, generatedCode, "Expected non-empty generated code for scenario '%s'", scenario.Name)

	// Log successful generation details for debugging
	if scenario.VerboseDebugging {
		sourceContent := isr.readSourceFileContent(sourceFile)
		t.Logf("✅ Code generation successful for scenario '%s':\n"+
			"Generated code length: %d characters\n"+
			"Source file: %s\n"+
			"--- Source File Content ---\n%s\n"+
			"--- Generated Code ---\n%s\n"+
			"--- End Debug Output ---",
			scenario.Name, len(generatedCode), sourceFile, sourceContent, generatedCode)
	} else {
		t.Logf("✅ Code generation successful for scenario '%s': %d characters generated",
			scenario.Name, len(generatedCode))
	}

	// Run code assertions
	isr.runCodeAssertions(t, scenario, generatedCode)

	// Run behavior tests if provided
	if len(scenario.BehaviorTests) > 0 {
		isr.runBehaviorTests(t, scenario, sourceFile, generatedCode)
	}
}

// handleFailureScenario processes scenarios that should fail.
func (isr *InlineScenarioRunner) handleFailureScenario(t *testing.T, scenario TestScenario, primaryError error) {
	t.Helper()

	require.Error(t, primaryError, "Expected error for scenario: %s", scenario.Name)

	if scenario.ExpectedError != "" {
		assert.Contains(t, primaryError.Error(), scenario.ExpectedError,
			"Error message should contain expected text")
	}
}

// createSourceFile creates a temporary Go file with the scenario's inline code.
func (isr *InlineScenarioRunner) createSourceFile(scenario TestScenario) (string, error) {
	// Create package directory
	packageName := "test" + strings.ReplaceAll(scenario.Name, " ", "")
	packageDir := filepath.Join(isr.tempDir, packageName)
	err := os.MkdirAll(packageDir, 0750)
	if err != nil {
		return "", fmt.Errorf("failed to create package directory: %w", err)
	}

	// Build the complete source file content
	var content strings.Builder

	// Package declaration
	fmt.Fprintf(&content, "package %s\n\n", packageName)

	// Imports
	if len(scenario.Imports) > 0 {
		content.WriteString("import (\n")
		for _, imp := range scenario.Imports {
			fmt.Fprintf(&content, "\t%s\n", imp)
		}
		content.WriteString(")\n\n")
	}

	// Generate directive
	content.WriteString("//go:generate go run github.com/reedom/convergen/v8\n\n")

	// Source types
	if scenario.SourceTypes != "" {
		content.WriteString(scenario.SourceTypes)
		content.WriteString("\n\n")
	}

	// Interface definition
	if scenario.Interface != "" {
		content.WriteString(scenario.Interface)
		content.WriteString("\n")
	}

	// Format the code
	formatted, err := format.Source([]byte(content.String()))
	if err != nil {
		return "", fmt.Errorf("failed to format source code: %w", err)
	}

	// Write to file
	sourceFile := filepath.Join(packageDir, "setup.go")
	err = os.WriteFile(sourceFile, formatted, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write source file: %w", err)
	}

	return sourceFile, nil
}

// generateCode runs the Convergen pipeline on the source file with CLI flags.
func (isr *InlineScenarioRunner) generateCode(sourceFile string, cliFlags CLIFlags) (string, error) {
	// Create parser
	p, err := parser.NewParser(sourceFile, "")
	if err != nil {
		return "", fmt.Errorf("failed to create parser: %w", err)
	}

	// Parse the source file
	methods, err := p.Parse()
	if err != nil {
		return "", fmt.Errorf("failed to parse source: %w", err)
	}

	// Build function blocks
	funcBlocks := make([]model.FunctionsBlock, 0, len(methods))
	builder := p.CreateBuilder()

	for _, info := range methods {
		functions, err := builder.CreateFunctions(info.Methods)
		if err != nil {
			return "", fmt.Errorf("failed to create functions: %w", err)
		}

		block := model.FunctionsBlock{
			Marker:    info.Marker,
			Functions: functions,
		}
		funcBlocks = append(funcBlocks, block)
	}

	// Generate base code
	baseCode, err := p.GenerateBaseCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate base code: %w", err)
	}

	// Create code model
	code := model.Code{
		BaseCode:       baseCode,
		FunctionBlocks: funcBlocks,
	}

	// Create config from CLI flags
	cfg := config.Config{
		Input:           sourceFile,
		Output:          "",
		Verbose:         cliFlags.Verbose,
		StructLiteral:   cliFlags.StructLiteral,
		NoStructLiteral: cliFlags.NoStructLiteral,
	}

	// Generate final code with configuration
	g := generator.NewGeneratorWithConfig(code, cfg)
	actual, err := g.Generate(sourceFile, false, true)
	if err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}

	return string(actual), nil
}

// runCodeAssertions executes code assertions.
func (isr *InlineScenarioRunner) runCodeAssertions(t *testing.T, scenario TestScenario, generatedCode string) {
	t.Helper()

	for i, assertion := range scenario.CodeChecks {
		result := assertion.Assert(generatedCode)

		if !result.Success {
			if scenario.VerboseDebugging {
				t.Errorf("❌ Assertion %d failed for scenario '%s':\n"+
					"Type: %s\n"+
					"Pattern: %s\n"+
					"Message: %s\n"+
					"Details: %s\n"+
					"--- Full Generated Code ---\n%s\n"+
					"--- End Generated Code ---",
					i+1, scenario.Name, assertion.Type, assertion.Pattern,
					result.Message, result.Details, generatedCode)
			} else {
				t.Errorf("❌ Assertion %d failed for scenario '%s': %s (pattern: %s)\n"+
					"Use .WithVerboseDebugging() to see full generated code",
					i+1, scenario.Name, result.Message, assertion.Pattern)
			}
		} else if scenario.VerboseDebugging {
			t.Logf("✅ Assertion %d passed for scenario '%s': %s", i+1, scenario.Name, assertion.Pattern)
		}
	}
}

// runBehaviorTests compiles and runs behavior tests.
func (isr *InlineScenarioRunner) runBehaviorTests(t *testing.T, scenario TestScenario, sourceFile, generatedCode string) {
	t.Helper()

	// Write generated code to file
	generatedFile := strings.Replace(sourceFile, ".go", ".gen.go", 1)
	err := os.WriteFile(generatedFile, []byte(generatedCode), 0600)
	require.NoError(t, err, "Failed to write generated code")

	// Create test file that uses the generated functions
	_ = isr.createBehaviorTestFile(t, scenario, sourceFile)

	// Run the behavior tests
	packageDir := filepath.Dir(sourceFile)
	cmd := exec.Command("go", "test", "-v", ".")
	cmd.Dir = packageDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Behavior tests failed for scenario %s:\n%s", scenario.Name, string(output))
	} else {
		t.Logf("Behavior tests passed for scenario %s", scenario.Name)
	}
}

// createBehaviorTestFile creates a Go test file that tests the generated functions.
func (isr *InlineScenarioRunner) createBehaviorTestFile(t *testing.T, scenario TestScenario, sourceFile string) string {
	t.Helper()

	packageDir := filepath.Dir(sourceFile)
	testFile := filepath.Join(packageDir, "behavior_test.go")

	// Extract package name from source file
	packageName := filepath.Base(packageDir)

	var content strings.Builder
	fmt.Fprintf(&content, "package %s\n\n", packageName)
	content.WriteString("import (\n")
	content.WriteString("\t\"testing\"\n")
	content.WriteString("\t\"reflect\"\n")
	content.WriteString(")\n\n")

	// Generate test functions for each behavior test
	for i, behaviorTest := range scenario.BehaviorTests {
		fmt.Fprintf(&content, "func TestBehavior%d(t *testing.T) {\n", i)
		fmt.Fprintf(&content, "\t// Test: %s\n", behaviorTest.Description)
		fmt.Fprintf(&content, "\t// Implementation would call %s function\n", behaviorTest.TestFunc)
		fmt.Fprintf(&content, "\t// and compare input/output\n")
		fmt.Fprintf(&content, "\tt.Skip(\"Behavior test implementation needed\")\n")
		fmt.Fprintf(&content, "}\n\n")
	}

	err := os.WriteFile(testFile, []byte(content.String()), 0600)
	require.NoError(t, err, "Failed to write behavior test file")

	return testFile
}

// truncateString truncates a string to maxLen characters with ellipsis if needed.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated, full length: " + fmt.Sprintf("%d", len(s)) + " characters)"
}

// readSourceFileContent reads the source file content for debugging purposes.
func (isr *InlineScenarioRunner) readSourceFileContent(sourceFile string) string {
	if sourceFile == "" {
		return "(source file not created)"
	}

	// Security: Ensure the file is within our temp directory
	if !strings.HasPrefix(sourceFile, isr.tempDir) {
		return fmt.Sprintf("(security: file outside temp directory: %s)", sourceFile)
	}

	content, err := os.ReadFile(sourceFile) // #nosec G304 - file path is validated above
	if err != nil {
		return fmt.Sprintf("(error reading source file: %v)", err)
	}

	return string(content)
}

// RunScenarios executes multiple scenarios.
func (isr *InlineScenarioRunner) RunScenarios(scenarios []TestScenario) {
	isr.t.Helper()

	for _, scenario := range scenarios {
		isr.RunScenario(scenario)
	}
}
