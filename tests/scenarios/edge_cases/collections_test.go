package edge_cases

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestCollectionTypes(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"SliceHandling",
			"slice",
		).WithCategory("edge_cases").
			WithDescription("Test slice type handling").
			WithCodeChecks(
				helpers.Contains("func Copy"),
				helpers.CompilesSuccessfully(),
			),

		helpers.QuickScenario(
			"MapHandling",
			"maps",
		).WithCategory("edge_cases").
			WithDescription("Test map type handling").
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}