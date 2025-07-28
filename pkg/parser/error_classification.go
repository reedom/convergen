package parser

import (
	"strings"
)

// ErrorPattern represents an error classification pattern
type ErrorPattern struct {
	Keywords   []string
	Category   ErrorCategory
	Severity   ErrorSeverity
	Suggestion string
}

// errorPatterns defines all error classification patterns in order of priority
var errorPatterns = []ErrorPattern{
	// Critical errors
	{
		Keywords:   []string{"panic", "fatal", "crashed"},
		Category:   CategoryGeneral,
		Severity:   SeverityCritical,
		Suggestion: "This is a critical system error. Please report this issue with full stack trace.",
	},

	// Syntax errors
	{
		Keywords:   []string{"syntax", "parse", "parsing", "scanner", "token", "unexpected"},
		Category:   CategorySyntax,
		Severity:   SeverityError,
		Suggestion: "Check Go syntax in the source file. Ensure all brackets, parentheses, and quotes are properly closed.",
	},

	// Type errors
	{
		Keywords:   []string{"type", "signature", "type mismatch", "undefined", "not declared"},
		Category:   CategoryType,
		Severity:   SeverityError,
		Suggestion: "Verify that all types are properly imported and defined. Check method signatures match interface requirements.",
	},

	// Annotation errors
	{
		Keywords:   []string{"annotation", "comment", "notation", ":map", ":conv", ":skip"},
		Category:   CategoryAnnotation,
		Severity:   SeverityError,
		Suggestion: "Check annotation syntax. Common formats: :map field1 field2, :conv converter srcField, :skip pattern.",
	},

	// Generation errors
	{
		Keywords:   []string{"generate", "generation", "code", "emit", "template"},
		Category:   CategoryGeneration,
		Severity:   SeverityError,
		Suggestion: "Code generation failed. Check that source and destination types are compatible.",
	},

	// Validation errors
	{
		Keywords:   []string{"validate", "validation", "invalid", "required", "missing", "must have"},
		Category:   CategoryValidation,
		Severity:   SeverityError,
		Suggestion: "Validate input parameters and interface definitions. Ensure methods have required parameters and return values.",
	},

	// Concurrency errors
	{
		Keywords:   []string{"concurrent", "timeout", "context", "deadline", "cancelled"},
		Category:   CategoryConcurrency,
		Severity:   SeverityWarning,
		Suggestion: "Operation timed out or was cancelled. Try increasing timeout or reducing concurrent workers.",
	},

	// Performance warnings
	{
		Keywords:   []string{"performance", "memory", "slow", "resource", "limit"},
		Category:   CategoryPerformance,
		Severity:   SeverityWarning,
		Suggestion: "Performance issue detected. Consider optimizing or reducing resource usage.",
	},

	// Deprecation warnings
	{
		Keywords:   []string{"warning", "deprecated", "obsolete"},
		Category:   CategoryGeneral,
		Severity:   SeverityWarning,
		Suggestion: "Consider updating to use recommended alternatives.",
	},
}

// ClassifyError classifies an error using predefined patterns
func ClassifyError(err error) (ErrorCategory, ErrorSeverity, string) {
	if err == nil {
		return CategoryGeneral, SeverityInfo, ""
	}

	message := strings.ToLower(err.Error())

	// Try to match against patterns in priority order
	for _, pattern := range errorPatterns {
		for _, keyword := range pattern.Keywords {
			if strings.Contains(message, keyword) {
				return pattern.Category, pattern.Severity, pattern.Suggestion
			}
		}
	}

	// Default fallback
	return CategoryGeneral, SeverityError, "Please check the error message for more details."
}

// ClassifyErrorWithContext classifies an error with additional context
func ClassifyErrorWithContext(err error, context string) (ErrorCategory, ErrorSeverity, string) {
	category, severity, suggestion := ClassifyError(err)

	// Enhance suggestions based on context
	if context != "" {
		contextLower := strings.ToLower(context)
		switch {
		case strings.Contains(contextLower, "method"):
			suggestion = "Method-related issue: " + suggestion
		case strings.Contains(contextLower, "interface"):
			suggestion = "Interface-related issue: " + suggestion
		case strings.Contains(contextLower, "package"):
			suggestion = "Package-related issue: " + suggestion
		}
	}

	return category, severity, suggestion
}

// GetCategoryDescription returns a human-readable description of an error category
func GetCategoryDescription(category ErrorCategory) string {
	switch category {
	case CategorySyntax:
		return "Syntax Error"
	case CategoryType:
		return "Type Error"
	case CategoryAnnotation:
		return "Annotation Error"
	case CategoryGeneration:
		return "Code Generation Error"
	case CategoryValidation:
		return "Validation Error"
	case CategoryConcurrency:
		return "Concurrency Issue"
	case CategoryPerformance:
		return "Performance Issue"
	case CategoryGeneral:
		return "General Error"
	default:
		return "Unknown Error"
	}
}

// GetSeverityDescription returns a human-readable description of error severity
func GetSeverityDescription(severity ErrorSeverity) string {
	switch severity {
	case SeverityCritical:
		return "Critical"
	case SeverityError:
		return "Error"
	case SeverityWarning:
		return "Warning"
	case SeverityInfo:
		return "Information"
	default:
		return "Unknown"
	}
}

// IsRetryableError determines if an error might be resolved by retrying
func IsRetryableError(category ErrorCategory) bool {
	switch category {
	case CategoryConcurrency, CategoryPerformance:
		return true
	default:
		return false
	}
}

// IsSkippableError determines if an error can be safely skipped
func IsSkippableError(severity ErrorSeverity) bool {
	return severity == SeverityWarning || severity == SeverityInfo
}
