package edge_cases

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestEmbeddedStructs(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"EmbeddedStructs",
			"embedded",
		).WithCategory("edge_cases").
			WithDescription("Test embedded struct handling").
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}