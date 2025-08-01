package edge_cases

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestMultiInterface(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"MultipleInterfaces",
			"multi_intf",
		).WithCategory("edge_cases").
			WithDescription("Test multiple interface handling").
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}