package annotations

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestPostprocessAnnotation(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"PostprocessAnnotation",
			"postprocess",
		).WithCategory("annotations").
			WithDescription("Test postprocess annotation for post-conversion processing").
			WithCodeChecks(
				helpers.Contains("func DomainToModel"),
				helpers.Contains("dst = &model.Pet{}"),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}