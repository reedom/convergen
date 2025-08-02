package examples

import (
	"fmt"
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// Example: Custom scenario builder
// This shows how to create reusable scenario builders for specific patterns
func TestExampleCustomScenarioBuilder(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Use a custom scenario builder (defined below)
	scenario := CustomMappingScenario("FirstName", "LastName", "FullName").
		WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("dst.FullName = TransformFirstName(src.FirstName)"),
			helpers.Contains("dst.LastName = src.LastName"),
			helpers.Contains("dst.Other = src.Other"),
		)

	runner.RunScenario(scenario)
}

// CustomMappingScenario creates a scenario for testing custom field mapping
func CustomMappingScenario(field1, field2, destField string) helpers.InlineScenario {
	return helpers.NewInlineScenario(
		fmt.Sprintf("CustomMapping_%s_%s_to_%s", field1, field2, destField),
		fmt.Sprintf("Test mapping %s with converter and %s directly to %s", field1, field2, destField),
	).WithTypes(fmt.Sprintf(`
type Source struct {
	%s string
	%s string
	Other string
}

type Dest struct {
	%s string
	%s string
	Other string
}

// Converter function to transform first field
func Transform%s(value string) string {
	return "transformed_" + value
}`, field1, field2, destField, field2, field1)).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :conv Transform%s %s %s
	Convert(*Source) *Dest
}`, field1, field1, destField))
}

// Example: Testing complex annotations
// This demonstrates testing multiple annotations working together
func TestExampleComplexAnnotations(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"MultipleAnnotations",
		"Test multiple annotations working together",
	).WithTypes(`
type User struct {
	FirstName string
	LastName  string
	Password  string
	Age       int
}

type UserModel struct {
	FirstName      string
	LastName       string
	HashedPassword string
	Age            int
	Status         string
	IsActive       bool
}

// Custom converter functions
func HashPassword(password string) string {
	return "hashed_" + password
}

func IsAdult(age int) bool {
	return 18 <= age
}`).WithInterface(`
type Convergen interface {
	// :conv HashPassword Password HashedPassword
	// :conv IsAdult Age IsActive
	// :literal Status "active"
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("dst.FirstName = src.FirstName"),
			helpers.Contains("dst.LastName = src.LastName"),
			helpers.Contains("HashPassword(src.Password)"),
			helpers.Contains("IsAdult(src.Age)"),
			helpers.Contains(`dst.Status = "active"`),
		)

	runner.RunScenario(scenario)
}

// Example: Testing edge cases
// This shows how to test boundary conditions and edge cases
func TestExampleEdgeCases(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	testCases := []struct {
		name        string
		description string
		sourceTypes string
		interface_  string
		shouldFail  bool
		errorMsg    string
		checks      []helpers.CodeAssertion
	}{
		{
			name:        "EmptyStructs",
			description: "Test conversion between empty structs",
			sourceTypes: `
type Empty1 struct {}
type Empty2 struct {}`,
			interface_: `
type Convergen interface {
	Convert(*Empty1) *Empty2
}`,
			shouldFail: false,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("dst = &Empty2{}"),
			},
		},
		{
			name:        "SingleField",
			description: "Test conversion with single field",
			sourceTypes: `
type Single1 struct { Value string }
type Single2 struct { Value string }`,
			interface_: `
type Convergen interface {
	Convert(*Single1) *Single2
}`,
			shouldFail: false,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("dst.Value = src.Value"),
			},
		},
		{
			name:        "MismatchedFields",
			description: "Test handling of mismatched field types",
			sourceTypes: `
type Source struct { Value string }
type Dest struct { Value int }`,
			interface_: `
type Convergen interface {
	Convert(*Source) *Dest
}`,
			shouldFail: false,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				// The function should be generated even with mismatched types
				// Convergen handles this by omitting the problematic field assignment
			},
		},
	}

	var scenarios []helpers.TestScenario
	for _, tc := range testCases {
		scenario := helpers.NewInlineScenario(tc.name, tc.description).
			WithTypes(tc.sourceTypes).
			WithInterface(tc.interface_).
			WithBehaviorTests()

		if tc.shouldFail {
			scenario = scenario.ShouldFail(tc.errorMsg)
		} else {
			scenario = scenario.WithCodeChecks(tc.checks...)
		}

		scenarios = append(scenarios, scenario)
	}

	runner.RunScenarios(scenarios)
}

// Example: Performance testing pattern
// This shows how to structure performance-focused tests
func TestExamplePerformanceTesting(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test with varying struct sizes to check performance characteristics
	structSizes := []int{5, 10, 20, 50}

	var scenarios []helpers.TestScenario
	for _, size := range structSizes {
		scenario := createLargeStructScenario(size)
		scenarios = append(scenarios, scenario)
	}

	runner.RunScenarios(scenarios)
}

// Helper function to create scenarios with large structs
func createLargeStructScenario(fieldCount int) helpers.TestScenario {
	// Generate struct with specified number of fields
	sourceFields := ""
	destFields := ""
	for i := 0; i < fieldCount; i++ {
		field := fmt.Sprintf("Field%d string\n", i)
		sourceFields += "	" + field
		destFields += "	" + field
	}

	types := fmt.Sprintf(`
type LargeSource struct {
%s}

type LargeDest struct {
%s}`, sourceFields, destFields)

	return helpers.NewInlineScenario(
		fmt.Sprintf("LargeStruct_%dFields", fieldCount),
		fmt.Sprintf("Test conversion with %d fields", fieldCount),
	).WithTypes(types).
		WithInterface(`
type Convergen interface {
	Convert(*LargeSource) *LargeDest
}`).WithBehaviorTests().
		WithCodeChecks(helpers.AssertHasGeneratedFunction())
}

// Example: Testing with generic types (Current Limitation)
// This demonstrates that Convergen currently doesn't support Go generics
// The test documents this limitation by expecting failure when generics are used
func TestExampleGenericTypes(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"GenericConversion",
		"Test that demonstrates Convergen's current limitation with Go generics (expected to fail)",
	).WithTypes(`
// Generic container type
type Container[T any] struct {
	Value T
	Count int
}

// Specific type aliases using generics
type StringContainer = Container[string]
type IntContainer = Container[int]

// Generic wrapper type
type Wrapper[T any] struct {
	Data     T
	Metadata map[string]string
}

type StringWrapper = Wrapper[string]
type IntWrapper = Wrapper[int]`).WithInterface(`
type Convergen interface {
	// Test: Can Convergen handle generic type aliases?
	ConvertContainer(*StringContainer) *StringContainer
	ConvertWrapper(*StringWrapper) *StringWrapper
}`).WithBehaviorTests().
		ShouldFail("missing ',' in parameter list")

	runner.RunScenario(scenario)
}

// Example: Testing with interfaces and embedded types
// This shows testing complex type relationships
func TestExampleComplexTypeRelationships(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"ComplexTypeRelationships",
		"Test conversion with embedded types and interfaces",
	).WithTypes(`
// Base types
type BaseEntity struct {
	ID        uint64
	CreatedAt time.Time
}

type Identifiable interface {
	GetID() uint64
}

// Source types with embedding
type User struct {
	BaseEntity
	Name  string
	Email string
}

func (u User) GetID() uint64 {
	return u.ID
}

// Destination types
type UserModel struct {
	BaseEntity
	Name  string
	Email string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *UserModel
}`).WithImports("time").
		WithBehaviorTests().
		WithCodeChecks(
			helpers.AssertHasGeneratedFunction(),
			helpers.Contains("dst.BaseEntity = src.BaseEntity"),
			helpers.Contains("dst.Name = src.Name"),
			helpers.Contains("dst.Email = src.Email"),
		)

	runner.RunScenario(scenario)
}

// Example: Testing graceful degradation patterns
// This shows how Convergen handles problematic scenarios by continuing with partial success
// rather than failing completely (graceful degradation, not true error recovery)
func TestExampleGracefulDegradationPatterns(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	// Test scenarios demonstrating graceful degradation
	scenarios := []helpers.TestScenario{
		{
			Name:        "PartialMappingSuccess",
			Description: "Convergen generates code for mappable fields while gracefully skipping problematic ones",
			SourceTypes: `
type Source struct {
	GoodField    string
	UnmatchType  complex128 // Type that's hard to convert
	AnotherGood  int
}

type Dest struct {
	GoodField    string
	UnmatchType  int64
	AnotherGood  int
	UnmappedField string // This won't have a source
}`,
			Interface: `
type Convergen interface {
	Convert(*Source) *Dest
}`,
			ShouldSucceed: true, // Demonstrates graceful degradation - generates partial code
			CodeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("dst.GoodField = src.GoodField"),
				helpers.Contains("dst.AnotherGood = src.AnotherGood"),
				helpers.Contains("// no match: dst.UnmatchType"),
				helpers.Contains("// no match: dst.UnmappedField"),
			},
			VerboseDebugging: true,
		},
		{
			Name:        "InvalidAnnotationHandling",
			Description: "Convergen ignores invalid annotations and continues with basic conversion",
			SourceTypes: `
type Source struct {
	Field1 string
	Field2 int
}

type Dest struct {
	Field1 string
	Field2 int
}`,
			Interface: `
type Convergen interface {
	// :invalid_annotation_that_should_be_ignored
	// :another_bad_annotation with params
	Convert(*Source) *Dest
}`,
			ShouldSucceed: true, // Should still generate basic conversion
			CodeChecks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
			},
		},
	}

	runner.RunScenarios(scenarios)
}

// Example: Custom assertion patterns
// This demonstrates creating domain-specific assertions
func TestExampleCustomAssertions(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"CustomAssertions",
		"Demonstrate custom assertion patterns",
	).WithTypes(`
type User struct {
	Name  string
	Email string
	Age   int
}

type UserModel struct {
	Name  string
	Email string
	Age   int
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *UserModel
}`).WithBehaviorTests().
		WithCodeChecks(
			// Standard assertions
			helpers.AssertHasGeneratedFunction(),

			// Custom domain-specific assertions
			AssertUserConversion(),
			AssertFieldMapping("Name"),
			AssertFieldMapping("Email"),
			AssertFieldMapping("Age"),
		)

	runner.RunScenario(scenario)
}

// Custom assertion functions for domain-specific validation
func AssertUserConversion() helpers.CodeAssertion {
	return helpers.MatchesRegex(`func\s+Convert\([^)]*\*User[^)]*\)\s*\([^)]*\*UserModel\)`)
}

func AssertFieldMapping(fieldName string) helpers.CodeAssertion {
	return helpers.Contains(fmt.Sprintf("dst.%s = src.%s", fieldName, fieldName))
}

// Example: Comprehensive integration test
// This shows how to create a comprehensive test that covers multiple aspects
func TestExampleComprehensiveIntegration(t *testing.T) {
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	scenario := helpers.NewInlineScenario(
		"ComprehensiveIntegration",
		"Comprehensive test covering multiple Convergen features",
	).WithTypes(`
// Domain models demonstrating complex mappings
type Address struct {
	Street  string
	City    string
	Country string
}

// Source with various field types and naming patterns
type User struct {
	ID        uint64
	FirstName string
	LastName  string
	email     string        // lowercase - will be skipped by default matching
	Password  string
	Age       int
	Address   Address
	Tags      []string
	IsActive  bool
	SecretKey string        // Will be explicitly skipped
	TempData  string        // Will be skipped by pattern
}

// Destination with different field names and types
type UserProfile struct {
	UserID         uint64        // :map ID UserID
	FirstName      string        // Direct match
	LastName       string        // Direct match
	ContactEmail   string        // :map email ContactEmail (case difference)
	HashedPassword string        // :conv HashPassword Password HashedPassword
	Age            int           // Direct match
	Location       Address       // :map Address Location
	Keywords       []string      // :map Tags Keywords
	Status         string        // :literal Status "active"
	AdultUser      bool          // :conv DetermineAdultStatus Age AdultUser
	// Note: SecretKey and TempData will be skipped
}

// Custom converter functions
func HashPassword(password string) string {
	return "bcrypt_" + password
}

func DetermineAdultStatus(age int) bool {
	return 18 <= age
}`).WithInterface(`
type Convergen interface {
	// Comprehensive test of multiple annotation features:
	// :map - Explicit field mapping between different names
	// :conv - Custom converter functions
	// :literal - Static value assignment
	// :skip - Exclude specific fields from conversion
	// :match - Field matching strategy

	// :match name                           // Use name-based matching as primary strategy
	// :map ID UserID                        // Map ID to UserID (different field names)
	// :map email ContactEmail               // Map lowercase email to ContactEmail
	// :conv HashPassword Password HashedPassword  // Convert using custom function
	// :map Address Location                 // Map struct field to different name
	// :map Tags Keywords                    // Map slice to different name
	// :literal Status "active"              // Set literal value
	// :conv DetermineAdultStatus Age AdultUser    // Convert using boolean function
	// :skip SecretKey                       // Explicitly skip sensitive field
	// :skip Temp*                           // Skip fields matching pattern
	Convert(*User) *UserProfile
}`).WithBehaviorTests().
		WithCodeChecks(
			// Function structure
			helpers.AssertHasGeneratedFunction(),

			// :map annotations - Explicit field mappings
			helpers.Contains("dst.UserID = src.ID"),          // :map ID UserID
			helpers.Contains("dst.ContactEmail = src.email"), // :map email ContactEmail
			helpers.Contains("dst.Location = src.Address"),   // :map Address Location
			helpers.Contains("dst.Keywords = src.Tags"),      // :map Tags Keywords

			// Direct field matches (name-based matching)
			helpers.Contains("dst.FirstName = src.FirstName"),
			helpers.Contains("dst.LastName = src.LastName"),
			helpers.Contains("dst.Age = src.Age"),

			// :conv annotations - Custom converter functions
			helpers.Contains("dst.HashedPassword = HashPassword(src.Password)"),
			helpers.Contains("dst.AdultUser = DetermineAdultStatus(src.Age)"),

			// :literal annotation - Static value assignment
			helpers.Contains(`dst.Status = "active"`),

			// :skip behavior - Ensure skipped fields are not mapped
			helpers.NotContains("dst.SecretKey"), // Should be skipped
			helpers.NotContains("src.SecretKey"), // Should not appear in generated code
			helpers.NotContains("dst.TempData"),  // Should be skipped by pattern
			helpers.NotContains("src.TempData"),  // Should not appear in generated code

			// Ensure password is converted, not directly copied
			helpers.NotContains("dst.Password = src.Password"),
		)

	runner.RunScenario(scenario)
}
