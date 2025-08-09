package helpers

import (
	"fmt"
	"regexp"
	"strings"
)

// CLIFlags represents CLI flags that can be set for testing scenarios.
type CLIFlags struct {
	StructLiteral   bool // Enable struct literal output
	NoStructLiteral bool // Disable struct literal output
	Verbose         bool // Enable verbose output
}

// TestScenario defines a behavior-driven test scenario for Convergen.
type TestScenario struct {
	Name        string
	Description string
	Category    string

	// Inline Code Definition
	SourceTypes string   // Go struct definitions
	Interface   string   // Converter interface definition
	Imports     []string // Additional imports needed

	// CLI Configuration
	CliFlags CLIFlags // CLI flags to simulate

	// Behavior Tests
	BehaviorTests []BehaviorTest

	// Code Expectations
	ShouldSucceed bool
	ExpectedError string
	CodeChecks    []CodeAssertion

	// Test metadata
	ShouldSkip       bool
	SkipReason       string
	VerboseDebugging bool // Enable detailed debugging output
}

// BehaviorTest defines a runtime behavior test.
type BehaviorTest struct {
	Name        string
	Description string
	TestFunc    string      // Name of generated function to test
	Input       interface{} // Input value for the function
	Expected    interface{} // Expected output value
	ShouldError bool        // Whether this test should produce an error
}

// InlineScenario creates a scenario with inline code definitions.
type InlineScenario struct {
	Name        string
	Description string
	SourceTypes string
	Interface   string
	Imports     []string
}

// WithBehaviorTests adds behavior tests to the scenario.
func (is InlineScenario) WithBehaviorTests(tests ...BehaviorTest) TestScenario {
	return TestScenario{
		Name:          is.Name,
		Description:   is.Description,
		SourceTypes:   is.SourceTypes,
		Interface:     is.Interface,
		Imports:       is.Imports,
		BehaviorTests: tests,
		ShouldSucceed: true,
	}
}

// AsTestScenario converts InlineScenario to TestScenario without behavior tests.
func (is InlineScenario) AsTestScenario() TestScenario {
	return TestScenario{
		Name:          is.Name,
		Description:   is.Description,
		SourceTypes:   is.SourceTypes,
		Interface:     is.Interface,
		Imports:       is.Imports,
		ShouldSucceed: true,
	}
}

// WithCodeChecks adds code assertions to the scenario.
func (ts TestScenario) WithCodeChecks(checks ...CodeAssertion) TestScenario {
	ts.CodeChecks = append(ts.CodeChecks, checks...)
	return ts
}

// WithCategory sets the test category.
func (ts TestScenario) WithCategory(category string) TestScenario {
	ts.Category = category
	return ts
}

// ShouldFail marks the scenario as expected to fail.
func (ts TestScenario) ShouldFail(expectedError string) TestScenario {
	ts.ShouldSucceed = false
	ts.ExpectedError = expectedError
	return ts
}

// WithVerboseDebugging enables verbose debugging output for this scenario.
func (ts TestScenario) WithVerboseDebugging() TestScenario {
	ts.VerboseDebugging = true
	return ts
}

// WithCliFlags sets CLI flags for the scenario.
func (ts TestScenario) WithCliFlags(flags CLIFlags) TestScenario {
	ts.CliFlags = flags
	return ts
}

// WithStructLiteral enables struct literal output for this scenario.
func (ts TestScenario) WithStructLiteral() TestScenario {
	ts.CliFlags.StructLiteral = true
	return ts
}

// WithNoStructLiteral disables struct literal output for this scenario.
func (ts TestScenario) WithNoStructLiteral() TestScenario {
	ts.CliFlags.NoStructLiteral = true
	return ts
}

// AssertionType defines the type of code assertion.
type AssertionType string

const (
	// AssertionContains checks if code contains a specific pattern.
	AssertionContains AssertionType = "contains"
	// AssertionNotContains checks if code does not contain a specific pattern.
	AssertionNotContains AssertionType = "not_contains"
	// AssertionRegex checks if code matches a regular expression.
	AssertionRegex AssertionType = "regex"
	// AssertionCompiles checks if code compiles successfully.
	AssertionCompiles AssertionType = "compiles"
	// AssertionExact checks if code matches exactly.
	AssertionExact AssertionType = "exact"
)

// CodeAssertion defines an assertion to be made against generated code.
type CodeAssertion struct {
	Type    AssertionType
	Pattern string
	Message string
}

// AssertionResult represents the result of running a code assertion.
type AssertionResult struct {
	Success bool
	Message string
	Details string
}

// Assert runs the assertion against the provided code and returns the result.
func (ca CodeAssertion) Assert(code string) AssertionResult {
	switch ca.Type {
	case AssertionContains:
		return ca.assertContains(code)
	case AssertionNotContains:
		return ca.assertNotContains(code)
	case AssertionRegex:
		return ca.assertRegex(code)
	case AssertionExact:
		return ca.assertExact(code)
	case AssertionCompiles:
		return ca.assertCompiles(code)
	default:
		return AssertionResult{
			Success: false,
			Message: fmt.Sprintf("Unknown assertion type: %s", ca.Type),
		}
	}
}

func (ca CodeAssertion) assertContains(code string) AssertionResult {
	contains := strings.Contains(code, ca.Pattern)
	message := ca.Message
	if message == "" {
		message = fmt.Sprintf("Expected code to contain: %s", ca.Pattern)
	}

	return AssertionResult{
		Success: contains,
		Message: message,
		Details: fmt.Sprintf("Looking for pattern: %s", ca.Pattern),
	}
}

func (ca CodeAssertion) assertNotContains(code string) AssertionResult {
	contains := strings.Contains(code, ca.Pattern)
	message := ca.Message
	if message == "" {
		message = fmt.Sprintf("Expected code to NOT contain: %s", ca.Pattern)
	}

	return AssertionResult{
		Success: !contains,
		Message: message,
		Details: fmt.Sprintf("Checking absence of pattern: %s", ca.Pattern),
	}
}

func (ca CodeAssertion) assertRegex(code string) AssertionResult {
	regex, err := regexp.Compile(ca.Pattern)
	if err != nil {
		return AssertionResult{
			Success: false,
			Message: fmt.Sprintf("Invalid regex pattern: %s", ca.Pattern),
			Details: err.Error(),
		}
	}

	matches := regex.MatchString(code)
	message := ca.Message
	if message == "" {
		message = fmt.Sprintf("Expected code to match regex: %s", ca.Pattern)
	}

	return AssertionResult{
		Success: matches,
		Message: message,
		Details: fmt.Sprintf("Regex pattern: %s", ca.Pattern),
	}
}

func (ca CodeAssertion) assertExact(code string) AssertionResult {
	matches := strings.TrimSpace(code) == strings.TrimSpace(ca.Pattern)
	message := ca.Message
	if message == "" {
		message = "Expected exact code match"
	}

	return AssertionResult{
		Success: matches,
		Message: message,
		Details: fmt.Sprintf("Expected exact match with pattern length: %d", len(ca.Pattern)),
	}
}

func (ca CodeAssertion) assertCompiles(code string) AssertionResult {
	// For now, this is a placeholder - actual compilation checking would require
	// more complex implementation with go/parser and go/types
	message := ca.Message
	if message == "" {
		message = "Expected code to compile successfully"
	}

	// Basic syntax check - look for obvious issues
	hasPackage := strings.Contains(code, "package ")
	hasFunc := strings.Contains(code, "func ")

	success := hasPackage && hasFunc
	details := "Basic syntax validation (package + func declarations)"

	return AssertionResult{
		Success: success,
		Message: message,
		Details: details,
	}
}

// Helper functions for creating common assertions.

// Contains creates a "contains" assertion.
func Contains(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionContains,
		Pattern: pattern,
	}
}

// NotContains creates a "not contains" assertion.
func NotContains(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionNotContains,
		Pattern: pattern,
	}
}

// MatchesRegex creates a regex assertion.
func MatchesRegex(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionRegex,
		Pattern: pattern,
	}
}

// CompilesSuccessfully creates a compilation assertion.
func CompilesSuccessfully() CodeAssertion {
	return CodeAssertion{
		Type: AssertionCompiles,
	}
}

// ExactMatch creates an exact match assertion.
func ExactMatch(expected string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionExact,
		Pattern: expected,
	}
}

// AssertHasGeneratedFunction checks for any generated function (including receiver methods).
func AssertHasGeneratedFunction() CodeAssertion {
	return MatchesRegex(`func (\(\w+ \*\w+\) )?\w+\(.*\).*\{`)
}

// AssertFunction checks for the presence of a function with the given name.
func AssertFunction(funcName string) CodeAssertion {
	return Contains(fmt.Sprintf("func %s", funcName))
}

// Error-specific assertion helpers.

// AssertErrorContains creates an assertion for error message validation.
func AssertErrorContains(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionContains,
		Pattern: pattern,
		Message: fmt.Sprintf("Expected error message to contain: %s", pattern),
	}
}

// AssertErrorType creates an assertion for specific error types.
func AssertErrorType(errorType string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionContains,
		Pattern: errorType,
		Message: fmt.Sprintf("Expected error of type: %s", errorType),
	}
}

// AssertParseError checks for parsing-related errors.
func AssertParseError() CodeAssertion {
	return CodeAssertion{
		Type:    AssertionRegex,
		Pattern: `(parse|syntax|expected|unexpected)`,
		Message: "Expected parse or syntax error",
	}
}

// AssertTypeError checks for type-related errors.
func AssertTypeError() CodeAssertion {
	return CodeAssertion{
		Type:    AssertionRegex,
		Pattern: `(type|assignment|mismatch|incompatible)`,
		Message: "Expected type-related error",
	}
}

// AssertAnnotationError checks for annotation-related errors.
func AssertAnnotationError() CodeAssertion {
	return CodeAssertion{
		Type:    AssertionRegex,
		Pattern: `(annotation|invalid|unknown)`,
		Message: "Expected annotation-related error",
	}
}
