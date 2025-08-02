package examples

import (
	"fmt"
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// TestCustomScenarioBuilder shows how to create reusable scenario builders.
// This demonstrates building your own scenario factory functions.
func TestCustomScenarioBuilder(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Use a custom scenario builder (defined below)
	scenario := CustomUserScenario("UserToModel").
		WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("func Convert"),
		)

	runner.RunScenario(scenario)
}

// CustomUserScenario creates a reusable scenario for user conversion tests.
// This shows how to build domain-specific test builders.
func CustomUserScenario(name string) helpers.InlineScenario {
	return helpers.NewInlineScenario(
		name,
		"Custom user conversion scenario",
	).WithTypes(`
type User struct {
	Name  string
	Email string
}

type UserModel struct {
	Name  string
	Email string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *UserModel
}`)
}

// TestVerboseDebugging demonstrates debugging capabilities.
// This shows how to get detailed information when tests fail.
func TestVerboseDebugging(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Enable verbose debugging to see full generated code
	scenario := helpers.NewInlineScenario(
		"DebuggingExample",
		"How to debug test failures",
	).WithTypes(`
type User struct {
	Name string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *User
}`).WithBehaviorTests().
		WithVerboseDebugging(). // Enable detailed debugging output
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("src.Name"),
		)

	runner.RunScenario(scenario)
}

// TestDebugHelper shows using the debug helper function.
// This demonstrates the quickest way to add debugging to existing scenarios.
func TestDebugHelper(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Use debug helper to wrap any existing scenario
	debugScenario := helpers.WithDebug(
		helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
	)

	runner.RunScenario(debugScenario)
}

// TestCustomAssertions demonstrates creating domain-specific assertions.
// This shows how to build reusable assertion patterns.
func TestCustomAssertions(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"CustomAssertions",
		"Using custom assertion functions",
	).WithTypes(`
type User struct {
	Name  string
	Email string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *User
}`).WithBehaviorTests().
		WithCodeChecks(
			// Standard assertions
			helpers.AssertHasGeneratedFunction(),

			// Custom domain-specific assertions
			AssertUserConversion(),
			AssertFieldMapping("Name"),
			AssertFieldMapping("Email"),
		)

	runner.RunScenario(scenario)
}

// Custom assertion functions for domain-specific validation.
func AssertUserConversion() helpers.CodeAssertion {
	return helpers.MatchesRegex(`func\s+Convert\([^)]*\*User[^)]*\)\s*\([^)]*\*User\)`)
}

func AssertFieldMapping(fieldName string) helpers.CodeAssertion {
	return helpers.Contains(fmt.Sprintf("src.%s", fieldName))
}

// TestDataDrivenTests shows how to use table-driven test patterns.
// This demonstrates testing multiple variations efficiently.
func TestDataDrivenTests(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	testCases := []struct {
		name        string
		style       string
		expectation string
	}{
		{"ReturnStyle", "return", "return"},
		{"ArgStyle", "arg", "func Convert"},
	}

	scenarios := make([]helpers.TestScenario, 0, len(testCases))
	for _, tc := range testCases {
		scenario := helpers.StyleAnnotationScenario(tc.style).
			WithBehaviorTests().
			WithCodeChecks(helpers.Contains(tc.expectation))
		scenarios = append(scenarios, scenario)
	}

	runner.RunScenarios(scenarios)
}

// TestConditionalAssertions shows how to create conditional test logic.
// This demonstrates advanced assertion patterns.
func TestConditionalAssertions(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"ConditionalExample",
		"Testing with conditional assertions",
	).WithTypes(`
type User struct {
	Name     string
	Password string
}`).WithInterface(`
type Convergen interface {
	// :skip Password
	Convert(*User) *User
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("src.Name"),
			helpers.NotContains("src.Password"), // Should be skipped
		)

	runner.RunScenario(scenario)
}

// TestScenarioCategories demonstrates organizing tests with categories.
// This shows how to group related tests for better organization.
func TestScenarioCategories(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenarios := []helpers.TestScenario{
		helpers.StyleAnnotationScenario("return").WithBehaviorTests().WithCategory("style"),
		helpers.StyleAnnotationScenario("arg").WithBehaviorTests().WithCategory("style"),
		helpers.MatchAnnotationScenario("name").WithBehaviorTests().WithCategory("matching"),
		helpers.MatchAnnotationScenario("none").WithBehaviorTests().WithCategory("matching"),
	}

	// Run scenarios - categories help with organization and filtering
	runner.RunScenarios(scenarios)
}

// TestPerformancePattern shows how to structure performance-focused tests.
// This demonstrates patterns for testing with varying complexity.
func TestPerformancePattern(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test with varying struct sizes
	structSizes := []int{1, 5, 10}
	scenarios := make([]helpers.TestScenario, 0, len(structSizes))

	for _, size := range structSizes {
		scenario := createVariableSizeScenario(size)
		scenarios = append(scenarios, scenario)
	}

	runner.RunScenarios(scenarios)
}

// Helper function to create scenarios with different complexities.
func createVariableSizeScenario(fieldCount int) helpers.TestScenario {
	// Generate struct with specified number of fields
	fields := ""
	for i := 0; i < fieldCount; i++ {
		fields += fmt.Sprintf("\tField%d string\n", i)
	}

	types := fmt.Sprintf(`
type TestStruct struct {
%s}`, fields)

	return helpers.NewInlineScenario(
		fmt.Sprintf("Size_%dFields", fieldCount),
		fmt.Sprintf("Test with %d fields", fieldCount),
	).WithTypes(types).
		WithInterface(`
type Convergen interface {
	Convert(*TestStruct) *TestStruct
}`).WithBehaviorTests().
		WithCodeChecks(helpers.AssertHasGeneratedFunction())
}
