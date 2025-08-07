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

// TestGenericsPatterns demonstrates advanced generics testing patterns.
// This shows how to test generics features across different scenarios.
func TestGenericsPatterns(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test different generic patterns
	genericsPatterns := []struct {
		name     string
		scenario helpers.TestScenario
	}{
		{
			"BasicGeneric",
			helpers.BasicGenericInterfaceScenario().WithBehaviorTests(),
		},
		{
			"GenericWithConstraints",
			helpers.GenericWithConstraintsScenario().WithBehaviorTests(),
		},
		{
			"MultipleTypeParams",
			helpers.MultipleTypeParametersScenario().WithBehaviorTests(),
		},
		{
			"GenericWithAnnotations",
			helpers.GenericWithAnnotationsScenario().WithBehaviorTests().
				WithCodeChecks(helpers.Contains("return")),
		},
	}

	scenarios := make([]helpers.TestScenario, 0, len(genericsPatterns))
	for _, pattern := range genericsPatterns {
		scenario := pattern.scenario.WithCategory("generics_patterns")
		scenarios = append(scenarios, scenario)
	}

	runner.RunScenarios(scenarios)
}

// TestGenericPerformancePatterns shows performance testing with generics.
// This demonstrates testing generics with varying type complexities.
func TestGenericPerformancePatterns(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test with different generic type complexities
	typeComplexities := []string{"string", "int"}
	scenarios := make([]helpers.TestScenario, 0, len(typeComplexities))

	for _, typeParam := range typeComplexities {
		scenario := createGenericTypeScenario(typeParam)
		scenarios = append(scenarios, scenario)
	}

	// Add a struct type scenario separately with proper type definition
	structScenario := createStructTypeScenario()
	scenarios = append(scenarios, structScenario)

	runner.RunScenarios(scenarios)
}

// TestGenericErrorHandling demonstrates error handling patterns for generics.
// This shows how to test various generics error conditions.
func TestGenericErrorHandling(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	errorScenarios := []helpers.TestScenario{
		helpers.InvalidGenericSyntaxScenario().
			WithBehaviorTests().
			WithVerboseDebugging().
			WithCodeChecks(helpers.CompilesSuccessfully()),

		helpers.UnsupportedConstraintScenario().
			WithBehaviorTests().
			WithCodeChecks(helpers.CompilesSuccessfully()),

		helpers.CircularConstraintScenario().
			WithBehaviorTests().
			WithCodeChecks(helpers.CompilesSuccessfully()),
	}

	for _, scenario := range errorScenarios {
		runner.RunScenario(scenario.WithCategory("generics_errors"))
	}
}

// TestGenericComplexTypes demonstrates testing with complex generic type scenarios.
// This shows advanced generic patterns with nested types and constraints.
func TestGenericComplexTypes(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	complexScenarios := []helpers.TestScenario{
		helpers.NestedGenericTypesScenario().
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("MapNested"),
				helpers.CompilesSuccessfully(),
			),

		helpers.GenericWithInterfaceConstraintsScenario().
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("ConvertStringer"),
				helpers.CompilesSuccessfully(),
			),

		helpers.GenericFieldMappingScenario().
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("MapFields"),
				helpers.Contains("Data"),
				helpers.CompilesSuccessfully(),
			),
	}

	for _, scenario := range complexScenarios {
		runner.RunScenario(scenario.WithCategory("generics_complex"))
	}
}

// Custom generics scenario builder for performance testing.
func createGenericTypeScenario(typeParam string) helpers.TestScenario {
	name := fmt.Sprintf("GenericType_%s", typeParam)

	return helpers.NewInlineScenario(
		name,
		fmt.Sprintf("Test generic conversion with %s type parameter", typeParam),
	).WithTypes(fmt.Sprintf(`
type Container[T any] struct {
	Data T
	Meta string
}

type Result struct {
	Data %s
	Meta string
}`, typeParam)).
		WithInterface(`
type Convergen[T any] interface {
	Convert(Container[T]) Result
}`).WithBehaviorTests().
		WithCodeChecks(helpers.AssertHasGeneratedFunction())
}

// createStructTypeScenario creates a scenario with a struct type parameter.
func createStructTypeScenario() helpers.TestScenario {
	return helpers.NewInlineScenario(
		"GenericType_StructValue",
		"Test generic conversion with struct type parameter",
	).WithTypes(`
type StructValue struct {
	Value string
}

type Container[T any] struct {
	Data T
	Meta string
}

type Result struct {
	Data StructValue
	Meta string
}`).
		WithInterface(`
type Convergen[T any] interface {
	Convert(Container[T]) Result
}`).WithBehaviorTests().
		WithCodeChecks(helpers.AssertHasGeneratedFunction())
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
