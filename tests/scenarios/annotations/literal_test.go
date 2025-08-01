package annotations

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestLiteralAnnotation(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"LiteralAnnotation",
			"literal",
		).WithCategory("annotations").
			WithDescription("Test literal annotation for direct value assignment").
			WithCodeChecks(
				helpers.Contains("func DomainToModel"),
				helpers.Contains("dst = &model.Pet{}"),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}