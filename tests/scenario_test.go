package testing

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestScenarios(t *testing.T) {
	t.Parallel()

	// Create scenario runner
	runner := helpers.NewScenarioRunner(t)

	// Define test scenarios (migrated from usecases_test.go)
	scenarios := []helpers.TestScenario{
		helpers.NewScenario(
			"AdditionalArgs",
			"fixtures/usecase/additionalargs/setup.go",
			"fixtures/usecase/additionalargs/setup.gen.go",
		).WithCategory("basic").WithDescription("Test additional arguments in conversion functions"),

		helpers.NewScenario(
			"Converter",
			"fixtures/usecase/converter/setup.go",
			"fixtures/usecase/converter/setup.gen.go",
		).WithCategory("basic").WithDescription("Test basic converter functionality"),

		helpers.NewScenario(
			"Embedded",
			"fixtures/usecase/embedded/setup.go",
			"fixtures/usecase/embedded/setup.gen.go",
		).WithCategory("embedded").WithDescription("Test embedded struct handling"),

		helpers.NewScenario(
			"Getter",
			"fixtures/usecase/getter/setup.go",
			"fixtures/usecase/getter/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test getter annotation functionality"),

		helpers.NewScenario(
			"Literal",
			"fixtures/usecase/literal/setup.go",
			"fixtures/usecase/literal/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test literal annotation"),

		helpers.NewScenario(
			"NoCase",
			"fixtures/usecase/nocase/setup.go",
			"fixtures/usecase/nocase/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test nocase annotation"),

		helpers.NewScenario(
			"MapName",
			"fixtures/usecase/mapname/setup.go",
			"fixtures/usecase/mapname/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test field name mapping"),

		helpers.NewScenario(
			"Maps",
			"fixtures/usecase/maps/setup.go",
			"fixtures/usecase/maps/setup.gen.go",
		).WithCategory("collections").WithDescription("Test map type handling"),

		helpers.NewScenario(
			"MultiInterface",
			"fixtures/usecase/multi_intf/setup.go",
			"fixtures/usecase/multi_intf/setup.gen.go",
		).WithCategory("advanced").WithDescription("Test multiple interface handling"),

		helpers.NewScenario(
			"Postprocess",
			"fixtures/usecase/postprocess/setup.go",
			"fixtures/usecase/postprocess/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test postprocess annotation"),

		helpers.NewScenario(
			"Ref",
			"fixtures/usecase/ref/setup.go",
			"fixtures/usecase/ref/generated/setup.gen.go",
		).WithCategory("advanced").WithDescription("Test reference handling"),

		helpers.NewScenario(
			"Simple",
			"fixtures/usecase/simple/setup.go",
			"fixtures/usecase/simple/setup.gen.go",
		).WithCategory("basic").WithDescription("Test simple struct conversion"),

		helpers.NewScenario(
			"Slice",
			"fixtures/usecase/slice/setup.go",
			"fixtures/usecase/slice/setup.gen.go",
		).WithCategory("collections").WithDescription("Test slice type handling"),

		helpers.NewScenario(
			"Stringer",
			"fixtures/usecase/stringer/setup.go",
			"fixtures/usecase/stringer/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test stringer annotation"),

		helpers.NewScenario(
			"Style",
			"fixtures/usecase/style/setup.go",
			"fixtures/usecase/style/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test style annotation"),

		helpers.NewScenario(
			"Typecast",
			"fixtures/usecase/typecast/setup.go",
			"fixtures/usecase/typecast/setup.gen.go",
		).WithCategory("annotations").WithDescription("Test typecast annotation"),

		// Enhanced scenarios with code assertions
		helpers.NewScenario(
			"SimpleWithAssertions",
			"fixtures/usecase/simple/setup.go",
			"fixtures/usecase/simple/setup.gen.go",
		).WithCategory("basic").
			WithDescription("Test simple conversion with code quality checks").
			WithCodeChecks(
				helpers.Contains("func DomainToModel"),
				helpers.Contains("dst = &model.Pet{}"),
				helpers.NotContains("panic"),
				helpers.CompilesSuccessfully(),
			),
	}

	// Run all scenarios
	runner.RunScenarios(scenarios)
}

// TestScenariosByCategory runs scenarios grouped by category
func TestScenariosByCategory(t *testing.T) {
	t.Parallel()

	testCategories := map[string][]helpers.TestScenario{
		"basic": {
			helpers.NewScenario(
				"Simple",
				"fixtures/usecase/simple/setup.go",
				"fixtures/usecase/simple/setup.gen.go",
			),
		},
		"annotations": {
			helpers.NewScenario(
				"Style",
				"fixtures/usecase/style/setup.go",
				"fixtures/usecase/style/setup.gen.go",
			).WithCodeChecks(
				helpers.Contains("func ArgToArg"),
				helpers.CompilesSuccessfully(),
			),
		},
	}

	for category, scenarios := range testCategories {
		category := category // capture for parallel execution
		scenarios := scenarios

		t.Run(category, func(t *testing.T) {
			t.Parallel()
			runner := helpers.NewScenarioRunner(t)
			runner.RunScenarios(scenarios)
		})
	}
}