package helpers

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reedom/convergen/v8/pkg/generator"
	"github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/parser"
)

// ScenarioRunner handles the execution of test scenarios
type ScenarioRunner struct {
	t *testing.T
}

// NewScenarioRunner creates a new scenario runner
func NewScenarioRunner(t *testing.T) *ScenarioRunner {
	return &ScenarioRunner{t: t}
}

// RunScenario executes a single test scenario
func (sr *ScenarioRunner) RunScenario(scenario TestScenario) {
	sr.t.Helper()

	if scenario.ShouldSkip {
		sr.t.Skipf("Skipping scenario %s: %s", scenario.Name, scenario.SkipReason)
		return
	}

	sr.t.Run(scenario.Name, func(t *testing.T) {
		t.Helper()

		// Generate code using Convergen pipeline
		generatedCode, err := sr.generateCode(scenario.SourceFile)

		if scenario.ShouldSucceed {
			// Expect success
			require.NoError(t, err, "Expected successful code generation for scenario: %s", scenario.Name)
			require.NotEmpty(t, generatedCode, "Expected non-empty generated code")

			// Compare with expected file if provided
			if scenario.ExpectedFile != "" {
				expectedCode := sr.loadExpectedCode(t, scenario.ExpectedFile)
				assert.Equal(t, expectedCode, generatedCode, "Generated code should match expected output")
			}

			// Run code assertions
			sr.runCodeAssertions(t, scenario, generatedCode)

		} else {
			// Expect error
			require.Error(t, err, "Expected error for scenario: %s", scenario.Name)

			// Check error message if provided
			if scenario.ExpectedError != "" {
				assert.Contains(t, err.Error(), scenario.ExpectedError, 
					"Error message should contain expected text")
			}
		}
	})
}

// RunScenarios executes multiple test scenarios
func (sr *ScenarioRunner) RunScenarios(scenarios []TestScenario) {
	sr.t.Helper()

	for _, scenario := range scenarios {
		sr.RunScenario(scenario)
	}
}

// generateCode runs the Convergen pipeline on the source file
func (sr *ScenarioRunner) generateCode(sourceFile string) (string, error) {
	// Create parser (similar to existing usecases_test.go)
	p, err := parser.NewParser(sourceFile, "")
	if err != nil {
		return "", fmt.Errorf("failed to create parser: %w", err)
	}

	// Parse the source file
	methods, err := p.Parse()
	if err != nil {
		return "", fmt.Errorf("failed to parse source: %w", err)
	}

	// Build function blocks (following existing pattern)
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

// loadExpectedCode loads the expected code from file
func (sr *ScenarioRunner) loadExpectedCode(t *testing.T, expectedFile string) string {
	t.Helper()

	expectedBytes, err := os.ReadFile(expectedFile)
	require.NoError(t, err, "Failed to read expected file: %s", expectedFile)

	return string(expectedBytes)
}

// runCodeAssertions executes all code assertions for a scenario
func (sr *ScenarioRunner) runCodeAssertions(t *testing.T, scenario TestScenario, generatedCode string) {
	t.Helper()

	for i, assertion := range scenario.CodeChecks {
		result := assertion.Assert(generatedCode)
		
		if !result.Success {
			t.Errorf("Assertion %d failed for scenario %s: %s\nDetails: %s", 
				i+1, scenario.Name, result.Message, result.Details)
		}
	}
}

// Helper functions for creating test scenarios

// NewScenario creates a new test scenario with default values
func NewScenario(name, sourceFile, expectedFile string) TestScenario {
	return TestScenario{
		Name:          name,
		SourceFile:    sourceFile,
		ExpectedFile:  expectedFile,
		ShouldSucceed: true,
		CodeChecks:    []CodeAssertion{},
	}
}

// NewErrorScenario creates a new test scenario that expects an error
func NewErrorScenario(name, sourceFile, expectedError string) TestScenario {
	return TestScenario{
		Name:          name,
		SourceFile:    sourceFile,
		ShouldSucceed: false,
		ExpectedError: expectedError,
		CodeChecks:    []CodeAssertion{},
	}
}

// WithCategory adds a category to a scenario
func (ts TestScenario) WithCategory(category string) TestScenario {
	ts.Category = category
	return ts
}

// WithDescription adds a description to a scenario
func (ts TestScenario) WithDescription(description string) TestScenario {
	ts.Description = description
	return ts
}

// WithCodeChecks adds code assertions to a scenario
func (ts TestScenario) WithCodeChecks(checks ...CodeAssertion) TestScenario {
	ts.CodeChecks = append(ts.CodeChecks, checks...)
	return ts
}

// SkipScenario marks a scenario to be skipped
func (ts TestScenario) SkipScenario(reason string) TestScenario {
	ts.ShouldSkip = true
	ts.SkipReason = reason
	return ts
}