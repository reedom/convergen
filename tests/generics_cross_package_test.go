package testing

import (
	"testing"

	"github.com/reedom/convergen/v9/tests/helpers"
)

// TestCrossPackageTypeResolution tests the cross-package type resolution functionality.
func TestGenericsCrossPackageBasic(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	tests := []struct {
		name          string
		description   string
		types         string
		interfaceCode string
		imports       []string
		expectSuccess bool
		codeChecks    []helpers.CodeAssertion
	}{
		{
			name:        "BasicCrossPackageUserConversion",
			description: "Should convert between cross-package user types",
			imports:     []string{},
			types: `
// Simple local types for cross-package testing
type UserSource struct {
	ID   int
	Name string
	Email string
}

type UserTarget struct {
	ID   int
	Name string
	Email string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertUser(UserSource) UserTarget
}`,
			expectSuccess: true,
			codeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertUser"),
				helpers.Contains("src.ID"),
				helpers.Contains("src.Name"),
				helpers.Contains("src.Email"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "GenericContainerConversion",
			description: "Should convert generic container types (but convergen cannot handle different type parameters)",
			imports:     []string{},
			types: `
// Local generic types (convergen cannot map between different type parameters T->U)
type Container[T any] struct {
	Value T
	Metadata string
}

type ContainerDTO[T any] struct {
	Value T
	Metadata string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertContainer(Container[string]) ContainerDTO[string]
}`,
			expectSuccess: true,
			codeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertContainer"),
				helpers.Contains("Value"),
				helpers.Contains("Metadata"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "MultipleTypeParametersConversion",
			description: "Should handle concrete comparable types (generics with different type parameters fail)",
			imports:     []string{},
			types: `
type ComparableData struct {
	ID   int
	Name string
}

type ComparableDTO struct {
	ID   int
	Name string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertComparable(ComparableData) ComparableDTO
}`,
			expectSuccess: true,
			codeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertComparable"),
				helpers.Contains("ID"),
				helpers.Contains("Name"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interfaceCode).
				WithImports(tt.imports...).
				WithBehaviorTests().
				WithCodeChecks(tt.codeChecks...)

			if !tt.expectSuccess {
				scenario = scenario.ShouldFail("expected error")
			}

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsCrossPackageConstraints tests constraint-based conversions across packages.
func TestGenericsCrossPackageConstraints(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	constraintTests := []struct {
		name          string
		description   string
		types         string
		interfaceCode string
		imports       []string
		checks        []helpers.CodeAssertion
	}{
		{
			name:        "ComparableConstraintConversion",
			description: "Should handle concrete comparable types (generic constraints not supported)",
			imports:     []string{},
			types: `
type ComparableSource struct {
	ID   int
	Name string
}

type ComparableTarget struct {
	ID   int
	Name string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertComparable(ComparableSource) ComparableTarget
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertComparable"),
				helpers.Contains("ID"),
				helpers.Contains("Name"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "StringerConstraintConversion",
			description: "Should handle concrete types with string values (generic constraints not supported)",
			imports:     []string{},
			types: `
type StringerSource struct {
	Value string
}

type StringerTarget struct {
	Value string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertStringer(StringerSource) StringerTarget
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertStringer"),
				helpers.Contains("Value"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "MixedConstraintsConversion",
			description: "Should handle concrete mixed types (generic constraints not supported)",
			imports:     []string{},
			types: `
type MixedComparableSource struct {
	ID   int
	Name string
}

type MixedStringerSource struct {
	Value string
}

type MixedTarget struct {
	ID    int
	Name  string
	Value string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertComparable(MixedComparableSource) MixedTarget
	ConvertStringer(MixedStringerSource) MixedTarget
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertComparable"),
				helpers.Contains("func ConvertStringer"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range constraintTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interfaceCode).
				WithImports(tt.imports...).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsCrossPackageNestedTypes tests nested generic types across packages.
func TestGenericsCrossPackageNestedTypes(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	nestedTests := []struct {
		name          string
		description   string
		types         string
		interfaceCode string
		imports       []string
		checks        []helpers.CodeAssertion
	}{
		{
			name:        "NestedGenericContainerConversion",
			description: "Should convert concrete nested container types",
			imports:     []string{},
			types: `
type NestedSource struct {
	Items    []string
	Metadata string
	Count    int
}

type NestedTarget struct {
	Items    []string
	Metadata string
	Count    int
}`,
			interfaceCode: `
type Convergen interface {
	ConvertNested(NestedSource) NestedTarget
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertNested"),
				helpers.Contains("Items"),
				helpers.Contains("Metadata"),
				helpers.Contains("Count"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "ComplexNestedGenericConversion",
			description: "Should handle concrete complex nested structures",
			imports:     []string{},
			types: `
type ComplexNestedSource struct {
	Value string
}

type ComplexNestedTarget struct {
	Value string
}`,
			interfaceCode: `
type Convergen interface {
	ConvertComplex(ComplexNestedSource) ComplexNestedTarget
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertComplex"),
				helpers.Contains("Value"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range nestedTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interfaceCode).
				WithImports(tt.imports...).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsCrossPackageErrorScenarios tests error conditions for cross-package scenarios.
func TestGenericsCrossPackageErrorScenarios(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	errorTests := []struct {
		name          string
		description   string
		types         string
		interfaceCode string
		imports       []string
		expectedError string
	}{
		{
			name:        "MissingPackageImport",
			description: "Should fail gracefully when package import is missing",
			imports:     []string{}, // No imports
			types: `
type MissingSource struct {
	ID   int
	Name string
}

type MissingTarget struct {
	ID   int
	Name string
}`,
			interfaceCode: `
// :convergen
type MissingConverter interface {
	ConvertUser(models.User) dto.UserDTO  // Use undefined types directly in method
}`,
			expectedError: "not defined",
		},
		{
			name:        "InvalidConstraintType",
			description: "Should fail when destination type is invalid",
			imports:     []string{},
			types: `
type ValidSource struct {
	Data string
}`,
			interfaceCode: `
// :convergen
type InvalidConverter interface {
	Convert(ValidSource) string  // string is not a valid destination type
}`,
			expectedError: "dst type should be a struct",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interfaceCode).
				WithImports(tt.imports...).
				WithBehaviorTests().
				ShouldFail(tt.expectedError)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsCrossPackagePerformance tests performance characteristics.
func TestGenericsCrossPackagePerformance(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test with multiple concrete types for performance
	scenario := helpers.NewInlineScenario(
		"MultipleTypeParametersPerformance",
		"Should handle multiple concrete types efficiently",
	).WithImports().
		WithTypes(`
type Source1 struct {
	ID   int
	Name string
}

type Source2 struct {
	ID   int
	Name string
}

type Source3 struct {
	Value string
}

type Target1 struct {
	ID   int
	Name string
}

type Target2 struct {
	ID   int
	Name string
}

type Target3 struct {
	Value string
}`).
		WithInterface(`
type Convergen interface {
	ConvertFirst(Source1) Target1
	ConvertSecond(Source2) Target2
	ConvertThird(Source3) Target3
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("func ConvertFirst"),
			helpers.Contains("func ConvertSecond"),
			helpers.Contains("func ConvertThird"),
			helpers.CompilesSuccessfully(),
		)

	runner.RunScenario(scenario)
}
