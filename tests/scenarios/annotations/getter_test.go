package annotations

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestGetterAnnotation(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"GetterAnnotation",
			"getter",
		).WithCategory("annotations").
			WithDescription("Test getter annotation for property access").
			WithCodeChecks(
				helpers.Contains("func DomainToModel"),
				helpers.Contains("dst = &model.Pet{}"),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}