package annotations

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestStyleAnnotation(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"StyleAnnotation",
			"style",
		).WithCategory("annotations").
			WithDescription("Test style annotation for function signature styles").
			WithCodeChecks(
				helpers.AssertFunction("ArgToArg"),
				helpers.Contains("dst.ID = pet.ID"),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}