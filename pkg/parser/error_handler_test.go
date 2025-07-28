package parser

import (
	"context"
	"errors"
	"go/token"
	"strings"
	"testing"
)

func TestErrorHandler_CreateError(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	tests := []struct {
		name         string
		opts         ErrorOptions
		wantCode     string
		wantSeverity ErrorSeverity
		wantCategory ErrorCategory
	}{
		{
			name: "basic_error",
			opts: ErrorOptions{
				Message:  "test error message",
				Code:     "TEST_ERROR",
				Category: CategorySyntax,
				Severity: SeverityError,
			},
			wantCode:     "TEST_ERROR",
			wantSeverity: SeverityError,
			wantCategory: CategorySyntax,
		},
		{
			name: "error_with_context",
			opts: ErrorOptions{
				Message:   "method processing failed",
				Code:      "METHOD_ERROR",
				Category:  CategoryType,
				Severity:  SeverityError,
				Method:    "TestMethod",
				Interface: "TestInterface",
			},
			wantCode:     "METHOD_ERROR",
			wantSeverity: SeverityError,
			wantCategory: CategoryType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.CreateError(tt.opts)

			if err.Code != tt.wantCode {
				t.Errorf("CreateError().Code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.Severity != tt.wantSeverity {
				t.Errorf("CreateError().Severity = %v, want %v", err.Severity, tt.wantSeverity)
			}

			if err.Category != tt.wantCategory {
				t.Errorf("CreateError().Category = %v, want %v", err.Category, tt.wantCategory)
			}

			if err.Message != tt.opts.Message {
				t.Errorf("CreateError().Message = %v, want %v", err.Message, tt.opts.Message)
			}

			// Test Error() method
			errorStr := err.Error()
			if !strings.Contains(errorStr, tt.opts.Message) {
				t.Errorf("Error() = %v, should contain %v", errorStr, tt.opts.Message)
			}

			// Test that suggestions are generated
			if len(err.Suggestions) == 0 {
				t.Error("CreateError() should generate suggestions")
			}
		})
	}
}

func TestErrorHandler_Wrap(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	originalErr := errors.New("original error")

	wrappedErr := handler.Wrap(originalErr, ErrorOptions{
		Message:  "wrapped error message",
		Code:     "WRAPPED_ERROR",
		Category: CategoryGeneral,
		Severity: SeverityError,
	})

	if wrappedErr.Cause != originalErr {
		t.Errorf("Wrap().Cause = %v, want %v", wrappedErr.Cause, originalErr)
	}

	if wrappedErr.Message != "wrapped error message" {
		t.Errorf("Wrap().Message = %v, want %v", wrappedErr.Message, "wrapped error message")
	}

	// Test unwrapping
	if unwrapped := wrappedErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestErrorHandler_WrapWithContext(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	ctx := context.Background()
	ctx = context.WithValue(ctx, "method", "TestMethod")
	ctx = context.WithValue(ctx, "interface", "TestInterface")

	originalErr := errors.New("context error")

	wrappedErr := handler.WrapWithContext(ctx, originalErr, "operation failed")

	if wrappedErr.Method != "TestMethod" {
		t.Errorf("WrapWithContext().Method = %v, want %v", wrappedErr.Method, "TestMethod")
	}

	if wrappedErr.Interface != "TestInterface" {
		t.Errorf("WrapWithContext().Interface = %v, want %v", wrappedErr.Interface, "TestInterface")
	}
}

func TestErrorHandler_RecoverFromPanic(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	panicValue := "test panic"
	err := handler.RecoverFromPanic(panicValue)

	if err.Code != "PANIC_RECOVERED" {
		t.Errorf("RecoverFromPanic().Code = %v, want %v", err.Code, "PANIC_RECOVERED")
	}

	if err.Severity != SeverityCritical {
		t.Errorf("RecoverFromPanic().Severity = %v, want %v", err.Severity, SeverityCritical)
	}

	if !strings.Contains(err.Message, panicValue) {
		t.Errorf("RecoverFromPanic().Message = %v, should contain %v", err.Message, panicValue)
	}

	// Check metadata
	if panicVal, exists := err.Metadata["panic_value"]; !exists || panicVal != panicValue {
		t.Errorf("RecoverFromPanic().Metadata[panic_value] = %v, want %v", panicVal, panicValue)
	}
}

func TestErrorHandler_Chain(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	primary := handler.CreateError(ErrorOptions{
		Message:  "primary error",
		Code:     "PRIMARY_ERROR",
		Category: CategoryGeneral,
		Severity: SeverityError,
	})

	related1 := handler.CreateError(ErrorOptions{
		Message:  "related error 1",
		Code:     "RELATED_ERROR_1",
		Category: CategoryValidation,
		Severity: SeverityWarning,
	})

	related2 := handler.CreateError(ErrorOptions{
		Message:  "related error 2",
		Code:     "RELATED_ERROR_2",
		Category: CategoryType,
		Severity: SeverityError,
	})

	chained := handler.Chain(primary, related1, related2)

	if len(chained.Related) != 2 {
		t.Errorf("Chain().Related length = %v, want %v", len(chained.Related), 2)
	}

	if chained.Related[0] != related1 {
		t.Errorf("Chain().Related[0] = %v, want %v", chained.Related[0], related1)
	}

	if chained.Related[1] != related2 {
		t.Errorf("Chain().Related[1] = %v, want %v", chained.Related[1], related2)
	}
}

func TestErrorHandler_CategorizeError(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "syntax_error",
			err:      errors.New("syntax error in file"),
			expected: CategorySyntax,
		},
		{
			name:     "type_error",
			err:      errors.New("type signature mismatch"),
			expected: CategoryType,
		},
		{
			name:     "annotation_error",
			err:      errors.New("invalid annotation format"),
			expected: CategoryAnnotation,
		},
		{
			name:     "generation_error",
			err:      errors.New("code generation failed"),
			expected: CategoryGeneration,
		},
		{
			name:     "validation_error",
			err:      errors.New("validation failed for interface"),
			expected: CategoryValidation,
		},
		{
			name:     "concurrency_error",
			err:      errors.New("concurrent operation timeout"),
			expected: CategoryConcurrency,
		},
		{
			name:     "performance_error",
			err:      errors.New("memory usage exceeded"),
			expected: CategoryPerformance,
		},
		{
			name:     "general_error",
			err:      errors.New("unknown error occurred"),
			expected: CategoryGeneral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := handler.categorizeError(tt.err)
			if category != tt.expected {
				t.Errorf("categorizeError() = %v, want %v", category, tt.expected)
			}
		})
	}
}

func TestErrorHandler_DetermineSeverity(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	tests := []struct {
		name     string
		err      error
		expected ErrorSeverity
	}{
		{
			name:     "critical_error",
			err:      errors.New("panic: fatal error occurred"),
			expected: SeverityCritical,
		},
		{
			name:     "error",
			err:      errors.New("operation failed with error"),
			expected: SeverityError,
		},
		{
			name:     "warning",
			err:      errors.New("warning: deprecated usage"),
			expected: SeverityWarning,
		},
		{
			name:     "default_error",
			err:      errors.New("some unknown issue"),
			expected: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := handler.determineSeverity(tt.err)
			if severity != tt.expected {
				t.Errorf("determineSeverity() = %v, want %v", severity, tt.expected)
			}
		})
	}
}

func TestErrorHandler_IsTemporary(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "temporary_concurrency_error",
			err: handler.CreateError(ErrorOptions{
				Category: CategoryConcurrency,
				Severity: SeverityError,
				Message:  "timeout error",
			}),
			expected: true,
		},
		{
			name: "temporary_performance_error",
			err: handler.CreateError(ErrorOptions{
				Category: CategoryPerformance,
				Severity: SeverityError,
				Message:  "performance issue",
			}),
			expected: true,
		},
		{
			name: "temporary_warning",
			err: handler.CreateError(ErrorOptions{
				Category: CategoryGeneral,
				Severity: SeverityWarning,
				Message:  "warning message",
			}),
			expected: true,
		},
		{
			name: "non_temporary_syntax_error",
			err: handler.CreateError(ErrorOptions{
				Category: CategorySyntax,
				Severity: SeverityError,
				Message:  "syntax error",
			}),
			expected: false,
		},
		{
			name:     "non_contextual_error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsTemporary(tt.err)
			if result != tt.expected {
				t.Errorf("IsTemporary() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorHandler_ShouldRetry(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	temporaryErr := handler.CreateError(ErrorOptions{
		Category: CategoryConcurrency,
		Severity: SeverityError,
		Message:  "timeout error",
	})

	permanentErr := handler.CreateError(ErrorOptions{
		Category: CategorySyntax,
		Severity: SeverityError,
		Message:  "syntax error",
	})

	tests := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		expected    bool
	}{
		{
			name:        "should_retry_temporary",
			err:         temporaryErr,
			attempt:     1,
			maxAttempts: 3,
			expected:    true,
		},
		{
			name:        "should_not_retry_permanent",
			err:         permanentErr,
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "should_not_retry_max_attempts",
			err:         temporaryErr,
			attempt:     3,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "should_not_retry_nil_error",
			err:         nil,
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ShouldRetry(tt.err, tt.attempt, tt.maxAttempts)
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorHandler_GetRetryDelay(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	// Test that delay increases with attempt number
	delay1 := handler.GetRetryDelay(1)
	delay2 := handler.GetRetryDelay(2)
	delay3 := handler.GetRetryDelay(3)

	if delay1 >= delay2 {
		t.Errorf("GetRetryDelay(1) = %v should be less than GetRetryDelay(2) = %v", delay1, delay2)
	}

	if delay2 >= delay3 {
		t.Errorf("GetRetryDelay(2) = %v should be less than GetRetryDelay(3) = %v", delay2, delay3)
	}

	// Test that all delays are positive
	if delay1 <= 0 {
		t.Errorf("GetRetryDelay(1) = %v should be positive", delay1)
	}
}

func TestContextualError_Is(t *testing.T) {
	fset := token.NewFileSet()
	handler := NewErrorHandler(fset, "test.go")

	err1 := handler.CreateError(ErrorOptions{
		Code:     "TEST_ERROR",
		Category: CategorySyntax,
		Message:  "test error",
	})

	err2 := handler.CreateError(ErrorOptions{
		Code:     "TEST_ERROR",
		Category: CategorySyntax,
		Message:  "different message",
	})

	err3 := handler.CreateError(ErrorOptions{
		Code:     "DIFFERENT_ERROR",
		Category: CategorySyntax,
		Message:  "test error",
	})

	regularErr := errors.New("regular error")

	// Test identity
	if !errors.Is(err1, err2) {
		t.Error("errors with same code and category should be considered equal")
	}

	if errors.Is(err1, err3) {
		t.Error("errors with different codes should not be considered equal")
	}

	if errors.Is(err1, regularErr) {
		t.Error("contextual error should not equal regular error")
	}
}

// Test string representations
func TestStringRepresentations(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{ String() string }
		expected string
	}{
		{"severity_info", SeverityInfo, "INFO"},
		{"severity_warning", SeverityWarning, "WARNING"},
		{"severity_error", SeverityError, "ERROR"},
		{"severity_critical", SeverityCritical, "CRITICAL"},
		{"category_syntax", CategorySyntax, "SYNTAX"},
		{"category_type", CategoryType, "TYPE"},
		{"category_annotation", CategoryAnnotation, "ANNOTATION"},
		{"category_general", CategoryGeneral, "GENERAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
