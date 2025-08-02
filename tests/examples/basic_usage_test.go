package examples

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// TestFrameworkBasics demonstrates the fundamental testing framework usage.
// This shows the essential components every test needs.
func TestFrameworkBasics(t *testing.T) {
	// Step 1: Create test runner with automatic cleanup
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup() // Always defer cleanup

	// Step 2: Define a simple test scenario
	scenario := helpers.NewInlineScenario(
		"MyFirstTest",                    // Test name
		"Simple framework usage example", // Description
	).WithTypes(`
type User struct {
	Name string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *User
}`).WithBehaviorTests() // Convert to TestScenario

	// Step 3: Execute the test
	runner.RunScenario(scenario)
}

// TestCodeAssertions demonstrates how to validate generated code.
// This shows the most common assertion patterns.
func TestCodeAssertions(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"AssertionExample",
		"How to use code assertions",
	).WithTypes(`
type User struct {
	Name  string
	Email string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *User
}`).WithBehaviorTests().
		WithCodeChecks(
			// Essential assertion - verify function was generated
			helpers.AssertHasGeneratedFunction(),

			// Check for specific patterns in generated code
			helpers.Contains("func Convert"),
			helpers.Contains("src.Name"),

			// Verify code compiles
			helpers.CompilesSuccessfully(),
		)

	runner.RunScenario(scenario)
}

// TestBuiltInScenarios shows how to use pre-built scenario helpers.
// This demonstrates the framework's convenience builders.
func TestBuiltInScenarios(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Use built-in scenario builders for common patterns
	scenarios := []helpers.TestScenario{
		// Test different styles
		helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
		helpers.StyleAnnotationScenario("arg").WithBehaviorTests(),

		// Test match strategies
		helpers.MatchAnnotationScenario("name").WithBehaviorTests(),

		// Test custom converters
		helpers.ConvertAnnotationScenario("HashPassword", "Password", "HashedPassword").WithBehaviorTests(),
	}

	// Run multiple scenarios
	runner.RunScenarios(scenarios)
}

// TestErrorScenarios demonstrates how to test failure cases.
// This shows how to verify error conditions.
func TestErrorScenarios(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test a scenario that should fail
	scenario := helpers.NewInlineScenario(
		"ExpectedFailure",
		"Testing error conditions",
	).WithTypes(`
type User struct {
	Password string
}`).WithInterface(`
type Convergen interface {
	// :conv NonExistentFunction Password Hashed  
	Convert(*User) *User
}`).WithBehaviorTests().
		ShouldFail("function NonExistentFunction not found") // Expect specific error

	runner.RunScenario(scenario)
}

// TestBatchExecution shows how to run multiple related tests efficiently.
// This demonstrates organizing and running test groups.
func TestBatchExecution(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Define common types for reuse
	commonTypes := `
type User struct {
	Name string
	Age  int
}`

	// Create multiple scenarios testing different aspects
	scenarios := []helpers.TestScenario{
		helpers.NewInlineScenario("Test1", "First test").
			WithTypes(commonTypes).
			WithInterface(`type Convergen interface { Convert(*User) *User }`).
			WithBehaviorTests().
			WithCategory("group1"),

		helpers.NewInlineScenario("Test2", "Second test").
			WithTypes(commonTypes).
			WithInterface(`type Convergen interface { Convert(*User) *User }`).
			WithBehaviorTests().
			WithCategory("group1"),
	}

	// Run all scenarios in batch
	runner.RunScenarios(scenarios)
}

// TestWithImports demonstrates how to test scenarios requiring imports.
// This shows handling external dependencies in tests.
func TestWithImports(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"ImportsExample",
		"Testing with external imports",
	).WithTypes(`
type Event struct {
	Name      string
	Timestamp time.Time
}`).WithInterface(`
type Convergen interface {
	Convert(*Event) *Event
}`).WithImports("time"). // Add required imports
		WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("time.Time"),
		)

	runner.RunScenario(scenario)
}
