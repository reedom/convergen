package testing

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestBehaviorDrivenScenarios(t *testing.T) {
	t.Parallel()

	// Create inline scenario runner
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenarios := []helpers.TestScenario{
		// Basic conversion test
		helpers.NewInlineScenario(
			"BasicConversion",
			"Test basic struct-to-struct conversion",
		).WithTypes(helpers.SimpleUserTypes()).
			WithInterface(helpers.SimpleConverter("Convert(*User) *UserModel")).
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func Convert"),
				helpers.CompilesSuccessfully(),
			),

		// Style annotation tests
		helpers.StyleAnnotationScenario("return").
			WithBehaviorTests().
			WithCodeChecks(
				helpers.Contains("func Convert"),
				helpers.Contains("return"),
				helpers.CompilesSuccessfully(),
			),

		helpers.StyleAnnotationScenario("arg").
			WithBehaviorTests().
			WithCodeChecks(
				helpers.Contains("func Convert"),
				helpers.CompilesSuccessfully(),
			),

		// Match annotation tests  
		helpers.MatchAnnotationScenario("name").
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),

		helpers.MatchAnnotationScenario("none").
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),

		// Conv annotation test
		helpers.ConvertAnnotationScenario("HashPassword", "Password", "HashedPassword").
			WithBehaviorTests().
			WithCodeChecks(
				helpers.Contains("HashPassword("),
				helpers.Contains("HashedPassword"),
				helpers.CompilesSuccessfully(),
			),

		// Literal annotation test
		helpers.LiteralAnnotationScenario("Status", `"active"`).
			WithBehaviorTests().
			WithCodeChecks(
				helpers.Contains(`Status: "active"`),
				helpers.CompilesSuccessfully(),
			),

		// Skip annotation test
		helpers.SkipAnnotationScenario("Password").
			WithBehaviorTests().
			WithCodeChecks(
				helpers.NotContains("Password"),
				helpers.CompilesSuccessfully(),
			),

		// Error scenario - invalid annotation
		helpers.NewInlineScenario(
			"InvalidAnnotation",
			"Test invalid annotation handling",
		).WithTypes(helpers.SimpleUserTypes()).
			WithInterface(`
type Convergen interface {
	// :invalid_annotation
	Convert(*User) *UserModel
}`).WithBehaviorTests().
			ShouldFail("invalid annotation"),
	}

	// Run all scenarios
	runner.RunScenarios(scenarios)
}

func TestAnnotationCoverage(t *testing.T) {
	t.Parallel()

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test each annotation individually for comprehensive coverage
	annotationTests := map[string]helpers.TestScenario{
		"match_name": helpers.MatchAnnotationScenario("name").WithBehaviorTests(),
		"match_none": helpers.MatchAnnotationScenario("none").WithBehaviorTests(),
		"style_return": helpers.StyleAnnotationScenario("return").WithBehaviorTests(),
		"style_arg": helpers.StyleAnnotationScenario("arg").WithBehaviorTests(),
		"conv_custom": helpers.ConvertAnnotationScenario("HashPassword", "Password", "HashedPassword").WithBehaviorTests(),
		"literal_string": helpers.LiteralAnnotationScenario("Status", `"active"`).WithBehaviorTests(),
		"skip_field": helpers.SkipAnnotationScenario("Password").WithBehaviorTests(),
	}

	for name, scenario := range annotationTests {
		t.Run(name, func(t *testing.T) {
			runner.RunScenario(scenario.WithCategory("annotations"))
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	edgeCases := []helpers.TestScenario{
		// Empty struct test
		helpers.NewInlineScenario(
			"EmptyStruct",
			"Test conversion with empty structs",
		).WithTypes(`
type Empty struct {}
type EmptyModel struct {}`).
			WithInterface(helpers.SimpleConverter("Convert(*Empty) *EmptyModel")).
			WithBehaviorTests().
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),

		// Nested struct test
		helpers.NewInlineScenario(
			"NestedStruct",
			"Test conversion with nested structs",
		).WithTypes(`
type Address struct {
	Street string
	City   string
}

type User struct {
	Name    string
	Address Address
}

type UserModel struct {
	Name    string
	Address Address
}`).WithInterface(helpers.SimpleConverter("Convert(*User) *UserModel")).
			WithBehaviorTests().
			WithCodeChecks(
				helpers.Contains("Address: src.Address"),
				helpers.CompilesSuccessfully(),
			),
	}

	for _, scenario := range edgeCases {
		runner.RunScenario(scenario.WithCategory("edge_cases"))
	}
}