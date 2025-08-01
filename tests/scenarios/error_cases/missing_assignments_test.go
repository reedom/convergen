package error_cases

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

func TestMissingAssignments(t *testing.T) {
	t.Parallel()

	runner := helpers.NewScenarioRunner(t)

	scenarios := []helpers.TestScenario{
		helpers.QuickScenario(
			"NoCase",
			"nocase",
		).WithCategory("error_cases").
			WithDescription("Test handling of case conversion issues").
			WithCodeChecks(
				helpers.AssertHasGeneratedFunction(),
				helpers.CompilesSuccessfully(),
			),
	}

	runner.RunScenarios(scenarios)
}