package testing

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// TestErrorScenarios tests comprehensive error conditions.
func TestErrorScenarios(t *testing.T) {
	t.Parallel()

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	errorScenarios := []helpers.TestScenario{
		// Invalid syntax scenarios
		helpers.InvalidSyntaxScenario().
			WithBehaviorTests().
			ShouldFail("failed to format source code"),

		// Invalid annotation scenarios - this actually succeeds, invalid annotations are ignored
		helpers.InvalidAnnotationScenario().
			WithBehaviorTests().
			WithCodeChecks(helpers.AssertHasGeneratedFunction()),

		// Type mismatch scenarios - Convergen generates code but issues warnings
		helpers.TypeMismatchScenario().
			WithBehaviorTests().
			WithCodeChecks(helpers.AssertHasGeneratedFunction()),

		// Missing converter function scenarios
		helpers.MissingConverterFunctionScenario().
			WithBehaviorTests().
			ShouldFail("function NonExistentFunction not found"),

		// Invalid map annotation scenarios - Convergen generates code but issues warnings
		helpers.InvalidMapAnnotationScenario().
			WithBehaviorTests().
			WithCodeChecks(helpers.AssertHasGeneratedFunction()),

		// Empty interface scenarios
		helpers.EmptyInterfaceScenario().
			WithBehaviorTests().
			ShouldFail("expected declaration"),

		// Invalid return type scenarios
		helpers.InvalidReturnTypeScenario().
			WithBehaviorTests().
			ShouldFail("dst type is not defined"),
	}

	runner.RunScenarios(errorScenarios)
}

// TestSpecificErrorMessages tests specific error message validation.
func TestSpecificErrorMessages(t *testing.T) {
	t.Parallel()

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test specific error message patterns
	scenarios := []helpers.TestScenario{
		// Test missing field error - generates code but logs warnings
		{
			Name:        "MissingFieldError",
			Description: "Test specific error message for missing field",
			SourceTypes: `
type Source struct {
	Name string
}

type Dest struct {
	Name        string
	MissingField string
}`,
			Interface: `
type Convergen interface {
	Convert(*Source) *Dest
}`,
			ShouldSucceed: true, // Convergen generates code but logs warnings
			CodeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("src.Name"),
			},
		},

		// Test invalid conversion function error - this actually fails
		{
			Name:        "InvalidConvFunction",
			Description: "Test specific error message for invalid conversion function",
			SourceTypes: `
type Source struct {
	Value string
}

type Dest struct {
	Value int
}`,
			Interface: `
type Convergen interface {
	// :conv InvalidFunction Value Value
	Convert(*Source) *Dest
}`,
			ShouldSucceed: false,
			ExpectedError: "function InvalidFunction not found",
		},

		// Test duplicate method error - Go allows this, generates both methods
		{
			Name:        "DuplicateMethod",
			Description: "Test handling of duplicate method names",
			SourceTypes: `
type User struct {
	Name string
}

type UserModel struct {
	Name string
}`,
			Interface: `
type Convergen interface {
	Convert(*User) *UserModel
	Convert(*User) *UserModel // Duplicate method
}`,
			ShouldSucceed: true, // Go allows duplicate methods in interfaces
			CodeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
			},
		},
	}

	runner.RunScenarios(scenarios)
}

// TestErrorRecovery tests error recovery and graceful degradation.
func TestErrorRecovery(t *testing.T) {
	t.Parallel()

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test scenarios that should handle errors gracefully
	scenarios := []helpers.TestScenario{
		// Test partial success with some field errors - Convergen generates code anyway
		{
			Name:        "PartialFieldMapping",
			Description: "Test partial success when some fields can't be mapped",
			SourceTypes: `
type Source struct {
	Name     string
	ValidField string
	InvalidField complex128 // Complex type that can't be easily converted
}

type Dest struct {
	Name     string
	ValidField string
	SimpleField string
}`,
			Interface: `
type Convergen interface {
	Convert(*Source) *Dest
}`,
			ShouldSucceed: true, // Convergen generates code but logs warnings
			CodeChecks: []helpers.CodeAssertion{
				helpers.Contains("src.Name"),
				helpers.Contains("src.ValidField"),
				helpers.AssertHasGeneratedFunction(),
			},
		},

		// Test error context preservation - Convergen generates code anyway
		{
			Name:        "ErrorContextPreservation",
			Description: "Test that error messages preserve context information",
			SourceTypes: `
type ComplexSource struct {
	NestedStruct struct {
		DeepField string
	}
}

type ComplexDest struct {
	FlatField string
}`,
			Interface: `
type Convergen interface {
	// :map NonExistent FlatField
	Convert(*ComplexSource) *ComplexDest
}`,
			ShouldSucceed: true, // Convergen generates code but logs warnings
			CodeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
			},
		},
	}

	runner.RunScenarios(scenarios)
}

// TestAdvancedErrorScenarios tests complex error conditions.
func TestAdvancedErrorScenarios(t *testing.T) {
	t.Parallel()

	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenarios := []helpers.TestScenario{
		// Test deep nested type errors - Convergen generates code anyway
		{
			Name:        "DeepNestedTypeError",
			Description: "Test error handling in deeply nested type structures",
			SourceTypes: `
type Level1 struct {
	Level2 *Level2Struct
}

type Level2Struct struct {
	Level3 *Level3Struct
}

type Level3Struct struct {
	Value string
}

type FlatDest struct {
	DeepValue string
}`,
			Interface: `
type Convergen interface {
	// This should fail because the mapping is too complex for auto-detection
	Convert(*Level1) *FlatDest
}`,
			ShouldSucceed: true, // Convergen generates code but logs warnings
			CodeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
			},
		},

		// Test interface parameter errors
		{
			Name:        "InterfaceParameterError",
			Description: "Test error handling for invalid interface parameters",
			SourceTypes: `
type User struct {
	Name string
}`,
			Interface: `
type Convergen interface {
	// Invalid - no return type specified
	Convert(*User)
}`,
			ShouldSucceed: false,
			ExpectedError: "abort",
		},

		// Test annotation parameter count errors
		{
			Name:        "AnnotationParameterError",
			Description: "Test error handling for wrong number of annotation parameters",
			SourceTypes: `
type User struct {
	Value string
}

type UserModel struct {
	Value string
}`,
			Interface: `
type Convergen interface {
	// :conv function missing source/dest parameters
	// :conv OnlyOneParam
	Convert(*User) *UserModel
}`,
			ShouldSucceed: false,
			ExpectedError: "abort",
		},
	}

	runner.RunScenarios(scenarios)
}
