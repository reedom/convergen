package annotations

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestTypecastAnnotation(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"TypecastAnnotation",
			"typecast",
		).WithCategory("annotations").
			WithDescription("Test typecast annotation for type conversion").
			WithCodeChecks(
				helpers.Contains("func DomainToModel"),
				helpers.Contains("dst = &model.User{}"),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}