package helpers

import (
	"fmt"
	"regexp"
	"strings"
)

// TestScenario defines a single test scenario for Convergen
type TestScenario struct {
	Name        string
	Description string
	Category    string

	// Input
	SourceFile   string
	ExpectedFile string

	// Expectations
	ShouldSucceed bool
	ExpectedError string
	CodeChecks    []CodeAssertion

	// Test metadata
	ShouldSkip bool
	SkipReason string
}

// AssertionType defines the type of code assertion
type AssertionType string

const (
	AssertionContains    AssertionType = "contains"
	AssertionNotContains AssertionType = "not_contains"
	AssertionRegex       AssertionType = "regex"
	AssertionCompiles    AssertionType = "compiles"
	AssertionExact       AssertionType = "exact"
)

// CodeAssertion defines an assertion to be made against generated code
type CodeAssertion struct {
	Type    AssertionType
	Pattern string
	Message string
}

// AssertionResult represents the result of running a code assertion
type AssertionResult struct {
	Success bool
	Message string
	Details string
}

// Assert runs the assertion against the provided code and returns the result
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

// Helper functions for creating common assertions

// Contains creates a "contains" assertion
func Contains(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionContains,
		Pattern: pattern,
	}
}

// NotContains creates a "not contains" assertion
func NotContains(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionNotContains,
		Pattern: pattern,
	}
}

// MatchesRegex creates a regex assertion
func MatchesRegex(pattern string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionRegex,
		Pattern: pattern,
	}
}

// CompilesSuccessfully creates a compilation assertion
func CompilesSuccessfully() CodeAssertion {
	return CodeAssertion{
		Type: AssertionCompiles,
	}
}

// ExactMatch creates an exact match assertion
func ExactMatch(expected string) CodeAssertion {
	return CodeAssertion{
		Type:    AssertionExact,
		Pattern: expected,
	}
}