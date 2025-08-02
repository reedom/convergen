package examples

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// TestDebugExample demonstrates how to use enhanced debugging features.
// This test shows how to get detailed information when tests fail.
func TestDebugExample(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Example 1: Basic debugging - shows failure information without full code
	basicScenario := helpers.NewInlineScenario(
		"BasicDebugExample",
		"Example of basic test debugging",
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
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("src.Name"), // This should pass
		)

	runner.RunScenario(basicScenario)

	// Example 2: Verbose debugging - shows full source and generated code
	verboseScenario := helpers.NewInlineScenario(
		"VerboseDebugExample",
		"Example of verbose debugging with full code output",
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
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("src.Name"),
		).
		WithVerboseDebugging() // Enable verbose debugging

	runner.RunScenario(verboseScenario)

	// Example 3: Using the debug helper function
	debugScenario := helpers.WithDebug(
		helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
	)

	runner.RunScenario(debugScenario)
}

// TestFailureExample demonstrates what enhanced debugging looks like when tests fail.
// Uncomment the failing assertion to see the debugging output.
func TestFailureExample(t *testing.T) {
	t.Skip("Skipping failure example - uncomment failing assertion to test debugging")

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// This scenario would fail and show debugging information
	failingScenario := helpers.NewInlineScenario(
		"FailureDebugExample",
		"Example of test failure with debugging",
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
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			// Uncomment this line to see failure debugging:
			// helpers.Contains("NonExistentPattern"), // This would fail
		).
		WithVerboseDebugging() // Shows full code on failure

	runner.RunScenario(failingScenario)
}

// TestMinimalFailureInfo demonstrates concise failure reporting.
func TestMinimalFailureInfo(t *testing.T) {
	t.Skip("Skipping failure example - uncomment failing assertion to test minimal debugging")

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// This scenario would fail with minimal info (no verbose debugging)
	failingScenario := helpers.NewInlineScenario(
		"MinimalFailureExample",
		"Example of test failure with minimal info",
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
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			// Uncomment this line to see minimal failure info:
			// helpers.Contains("NonExistentPattern"), // This would fail
		)
	// No .WithVerboseDebugging() - shows concise failure info

	runner.RunScenario(failingScenario)
}
