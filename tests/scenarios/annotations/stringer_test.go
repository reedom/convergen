package annotations

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestStringerAnnotation(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"StringerAnnotation",
			"stringer",
		).WithCategory("annotations").
			WithDescription("Test stringer annotation for String() method usage").
			WithCodeChecks(
				helpers.Contains("func LocalToModel"),
				helpers.Contains("dst = &model.Pet{}"),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}