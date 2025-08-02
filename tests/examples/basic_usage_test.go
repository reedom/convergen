package examples

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// Example: Basic scenario testing
// This demonstrates the simplest way to test a conversion scenario
func TestExampleBasicScenario(t *testing.T) {
	// Create test runner with automatic cleanup
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Define a simple test scenario
	scenario := helpers.NewInlineScenario(
		"BasicUserConversion",
		"Convert User struct to UserModel struct",
	).WithTypes(`
// Source type - what we're converting from
type User struct {
	ID   uint64
	Name string
	Email string
}

// Destination type - what we're converting to
type UserModel struct {
	ID   uint64
	Name string
	Email string
}`).WithInterface(`
// Converter interface with Convergen annotations
type Convergen interface {
	Convert(*User) *UserModel
}`).WithBehaviorTests(). // Enable behavior testing
		WithCodeChecks( // Add code assertions
			helpers.AssertHasGeneratedFunction(), // Verify a function was generated
			helpers.Contains("src.Name"),         // Verify Name field is mapped
			helpers.Contains("src.Email"),        // Verify Email field is mapped
		)

	// Execute the test scenario
	runner.RunScenario(scenario)
}

// Example: Testing with annotations
// This shows how to test Convergen annotations
func TestExampleAnnotationTesting(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"StyleAnnotation",
		"Test :style annotation with return style",
	).WithTypes(`
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64
	Name string
}`).WithInterface(`
type Convergen interface {
	// :style return
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("&UserModel{"),
		)

	runner.RunScenario(scenario)
}

// Example: Using built-in scenario builders
// This demonstrates using pre-built scenario helpers
func TestExampleBuiltInScenarios(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Use built-in scenario builders for common patterns
	scenarios := []helpers.TestScenario{
		// Test different style annotations
		helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
		helpers.StyleAnnotationScenario("arg").WithBehaviorTests(),

		// Test match algorithms
		helpers.MatchAnnotationScenario("name").WithBehaviorTests(),
		helpers.MatchAnnotationScenario("none").WithBehaviorTests(),

		// Test custom converter functions
		helpers.ConvertAnnotationScenario("HashPassword", "Password", "HashedPassword").
			WithBehaviorTests(),
	}

	// Run all scenarios
	runner.RunScenarios(scenarios)
}

// Example: Error testing
// This shows how to test error conditions
func TestExampleErrorTesting(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test a scenario that should fail
	scenario := helpers.NewInlineScenario(
		"MissingConverter",
		"Test error when converter function is missing",
	).WithTypes(`
type User struct {
	Password string
}

type UserModel struct {
	HashedPassword string
}`).WithInterface(`
type Convergen interface {
	// :conv NonExistentFunction Password HashedPassword
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		ShouldFail("function NonExistentFunction not found") // Expect specific error

	runner.RunScenario(scenario)
}

// Example: Complex type structures
// This demonstrates testing with complex nested types
func TestExampleComplexTypes(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"NestedStructConversion",
		"Test conversion of nested struct hierarchies",
	).WithTypes(`
type Address struct {
	Street string
	City   string
	Country string
}

type User struct {
	ID      uint64
	Name    string
	Address Address
	Tags    []string
}

type UserModel struct {
	ID      uint64
	Name    string
	Address Address
	Tags    []string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("Address: src.Address"),
			helpers.Contains("Tags: src.Tags"),
		)

	runner.RunScenario(scenario)
}

// Example: Multiple assertions
// This shows different types of code assertions
func TestExampleMultipleAssertions(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"AssertionExample",
		"Demonstrate different assertion types",
	).WithTypes(`
type User struct {
	FirstName string
	LastName  string
	Age       int
}

type UserModel struct {
	FullName string
	Age      int
	Status   string
}`).WithInterface(`
type Convergen interface {
	// :map FirstName,LastName FullName
	// :literal Status "active"
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			// Function existence
			helpers.AssertHasGeneratedFunction(),

			// String containment
			helpers.Contains("src.FirstName"),
			helpers.Contains("src.LastName"),

			// String absence
			helpers.NotContains("Password"),

			// Regex matching
			helpers.MatchesRegex(`Status:\s*"active"`),

			// Compilation success
			helpers.CompilesSuccessfully(),
		)

	runner.RunScenario(scenario)
}

// Example: Batch testing with categories
// This shows how to organize and run multiple related tests
func TestExampleBatchTesting(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Define common types for reuse
	commonTypes := `
type User struct {
	ID   uint64
	Name string
	Age  int
}

type UserModel struct {
	ID   uint64
	Name string
	Age  int
}`

	// Create multiple scenarios testing different aspects
	scenarios := []helpers.TestScenario{
		helpers.NewInlineScenario("ReturnStyle", "Test return style").
			WithTypes(commonTypes).
			WithInterface(`
type Convergen interface {
	// :style return
	Convert(*User) *UserModel
}`).WithBehaviorTests().
			WithCategory("style"),

		helpers.NewInlineScenario("ArgStyle", "Test arg style").
			WithTypes(commonTypes).
			WithInterface(`
type Convergen interface {
	// :style arg
	Convert(*User) *UserModel
}`).WithBehaviorTests().
			WithCategory("style"),

		helpers.NewInlineScenario("NameMatch", "Test name matching").
			WithTypes(commonTypes).
			WithInterface(`
type Convergen interface {
	// :match name
	Convert(*User) *UserModel
}`).WithBehaviorTests().
			WithCategory("match"),
	}

	// Run all scenarios in batch
	runner.RunScenarios(scenarios)
}

// Example: Testing with imports
// This shows how to test scenarios that require additional imports
func TestExampleWithImports(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"TimeConversion",
		"Test conversion involving time types",
	).WithTypes(`
type Event struct {
	Name      string
	Timestamp time.Time
}

type EventModel struct {
	Name      string
	Timestamp time.Time
}`).WithInterface(`
type Convergen interface {
	Convert(*Event) *EventModel
}`).WithImports("time"). // Add required imports
		WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("Timestamp: src.Timestamp"),
		)

	runner.RunScenario(scenario)
}
