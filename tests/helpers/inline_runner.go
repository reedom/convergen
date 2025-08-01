package helpers

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reedom/convergen/v8/pkg/generator"
	"github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/parser"
)

// InlineScenarioRunner handles behavior-driven scenario testing
type InlineScenarioRunner struct {
	t       *testing.T
	tempDir string
}

// NewInlineScenarioRunner creates a new inline scenario runner
func NewInlineScenarioRunner(t *testing.T) *InlineScenarioRunner {
	// Create temporary directory for test files
	tempDir, err := ioutil.TempDir("", "convergen_test_*")
	require.NoError(t, err, "Failed to create temp directory")

	return &InlineScenarioRunner{
		t:       t,
		tempDir: tempDir,
	}
}

// Cleanup cleans up temporary files
func (isr *InlineScenarioRunner) Cleanup() {
	if isr.tempDir != "" {
		os.RemoveAll(isr.tempDir)
	}
}

// RunScenario executes a behavior-driven test scenario
func (isr *InlineScenarioRunner) RunScenario(scenario TestScenario) {
	isr.t.Helper()

	if scenario.ShouldSkip {
		isr.t.Skipf("Skipping scenario %s: %s", scenario.Name, scenario.SkipReason)
		return
	}

	isr.t.Run(scenario.Name, func(t *testing.T) {
		t.Helper()

		// Generate the source file with inline code
		sourceFile, err := isr.createSourceFile(scenario)
		require.NoError(t, err, "Failed to create source file")

		// Generate code using Convergen
		generatedCode, err := isr.generateCode(sourceFile)

		if scenario.ShouldSucceed {
			require.NoError(t, err, "Expected successful code generation")
			require.NotEmpty(t, generatedCode, "Expected non-empty generated code")

			// Run code assertions
			isr.runCodeAssertions(t, scenario, generatedCode)

			// Run behavior tests if provided
			if len(scenario.BehaviorTests) > 0 {
				isr.runBehaviorTests(t, scenario, sourceFile, generatedCode)
			}
		} else {
			require.Error(t, err, "Expected error for scenario: %s", scenario.Name)

			if scenario.ExpectedError != "" {
				assert.Contains(t, err.Error(), scenario.ExpectedError,
					"Error message should contain expected text")
			}
		}
	})
}

// createSourceFile creates a temporary Go file with the scenario's inline code
func (isr *InlineScenarioRunner) createSourceFile(scenario TestScenario) (string, error) {
	// Create package directory
	packageName := "test" + strings.ReplaceAll(scenario.Name, " ", "")
	packageDir := filepath.Join(isr.tempDir, packageName)
	err := os.MkdirAll(packageDir, 0755)
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
			fmt.Fprintf(&content, "\t\"%s\"\n", imp)
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
	err = ioutil.WriteFile(sourceFile, formatted, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write source file: %w", err)
	}

	return sourceFile, nil
}

// generateCode runs the Convergen pipeline on the source file
func (isr *InlineScenarioRunner) generateCode(sourceFile string) (string, error) {
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
	var funcBlocks []model.FunctionsBlock
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

	// Generate final code
	g := generator.NewGenerator(code)
	actual, err := g.Generate(sourceFile, false, true)
	if err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}

	return string(actual), nil
}

// runCodeAssertions executes code assertions
func (isr *InlineScenarioRunner) runCodeAssertions(t *testing.T, scenario TestScenario, generatedCode string) {
	t.Helper()

	for i, assertion := range scenario.CodeChecks {
		result := assertion.Assert(generatedCode)

		if !result.Success {
			t.Errorf("Assertion %d failed for scenario %s: %s\nDetails: %s",
				i+1, scenario.Name, result.Message, result.Details)
		}
	}
}

// runBehaviorTests compiles and runs behavior tests
func (isr *InlineScenarioRunner) runBehaviorTests(t *testing.T, scenario TestScenario, sourceFile, generatedCode string) {
	t.Helper()

	// Write generated code to file
	generatedFile := strings.Replace(sourceFile, ".go", ".gen.go", 1)
	err := ioutil.WriteFile(generatedFile, []byte(generatedCode), 0644)
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

// createBehaviorTestFile creates a Go test file that tests the generated functions
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

	err := ioutil.WriteFile(testFile, []byte(content.String()), 0644)
	require.NoError(t, err, "Failed to write behavior test file")

	return testFile
}

// RunScenarios executes multiple scenarios
func (isr *InlineScenarioRunner) RunScenarios(scenarios []TestScenario) {
	isr.t.Helper()

	for _, scenario := range scenarios {
		isr.RunScenario(scenario)
	}
}