package parser

import (
	"context"
	"fmt"
	"go/token"
	"runtime"
	"strings"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns the string representation of ErrorSeverity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ErrorCategory represents different categories of parsing errors
type ErrorCategory int

const (
	CategoryGeneral ErrorCategory = iota
	CategorySyntax
	CategoryType
	CategoryAnnotation
	CategoryGeneration
	CategoryValidation
	CategoryConcurrency
	CategoryPerformance
)

// String returns the string representation of ErrorCategory
func (c ErrorCategory) String() string {
	switch c {
	case CategoryGeneral:
		return "GENERAL"
	case CategorySyntax:
		return "SYNTAX"
	case CategoryType:
		return "TYPE"
	case CategoryAnnotation:
		return "ANNOTATION"
	case CategoryGeneration:
		return "GENERATION"
	case CategoryValidation:
		return "VALIDATION"
	case CategoryConcurrency:
		return "CONCURRENCY"
	case CategoryPerformance:
		return "PERFORMANCE"
	default:
		return "UNKNOWN"
	}
}

// ContextualError represents a rich error with comprehensive context
type ContextualError struct {
	// Core error information
	Message   string        `json:"message"`
	Cause     error         `json:"cause,omitempty"`
	Code      string        `json:"code"`
	Category  ErrorCategory `json:"category"`
	Severity  ErrorSeverity `json:"severity"`
	Timestamp time.Time     `json:"timestamp"`

	// Context information
	SourcePath string         `json:"source_path,omitempty"`
	Position   token.Position `json:"position,omitempty"`
	Method     string         `json:"method,omitempty"`
	Interface  string         `json:"interface,omitempty"`
	Package    string         `json:"package,omitempty"`

	// Debugging information
	StackTrace []string               `json:"stack_trace,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`

	// Suggestions for resolution
	Suggestions []string `json:"suggestions,omitempty"`

	// Related errors (for error chains)
	Related []*ContextualError `json:"related,omitempty"`
}

// Error implements the error interface
func (ce *ContextualError) Error() string {
	var builder strings.Builder

	// Add position if available
	if ce.Position.IsValid() {
		builder.WriteString(fmt.Sprintf("%s: ", ce.Position))
	}

	// Add severity and category
	builder.WriteString(fmt.Sprintf("[%s:%s] ", ce.Severity, ce.Category))

	// Add error code if available
	if ce.Code != "" {
		builder.WriteString(fmt.Sprintf("(%s) ", ce.Code))
	}

	// Add main message
	builder.WriteString(ce.Message)

	// Add context information
	if ce.Interface != "" {
		builder.WriteString(fmt.Sprintf(" [interface: %s]", ce.Interface))
	}
	if ce.Method != "" {
		builder.WriteString(fmt.Sprintf(" [method: %s]", ce.Method))
	}

	return builder.String()
}

// Unwrap returns the underlying cause error for error unwrapping
func (ce *ContextualError) Unwrap() error {
	return ce.Cause
}

// Is implements error identity checking
func (ce *ContextualError) Is(target error) bool {
	if otherContextual, ok := target.(*ContextualError); ok {
		return ce.Code == otherContextual.Code && ce.Category == otherContextual.Category
	}
	return false
}

// ErrorHandler provides enhanced error handling capabilities
type ErrorHandler struct {
	fileSet          *token.FileSet
	sourcePath       string
	enableStackTrace bool
	maxSuggestions   int
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(fset *token.FileSet, sourcePath string) *ErrorHandler {
	return &ErrorHandler{
		fileSet:          fset,
		sourcePath:       sourcePath,
		enableStackTrace: true,
		maxSuggestions:   3,
	}
}

// CreateError creates a new contextual error with comprehensive information
func (eh *ErrorHandler) CreateError(opts ErrorOptions) *ContextualError {
	err := &ContextualError{
		Message:   opts.Message,
		Cause:     opts.Cause,
		Code:      opts.Code,
		Category:  opts.Category,
		Severity:  opts.Severity,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Set source path
	if opts.SourcePath != "" {
		err.SourcePath = opts.SourcePath
	} else if eh.sourcePath != "" {
		err.SourcePath = eh.sourcePath
	}

	// Set position information
	if opts.Position.IsValid() {
		err.Position = opts.Position
	} else if opts.TokenPos != token.NoPos && eh.fileSet != nil {
		err.Position = eh.fileSet.Position(opts.TokenPos)
	}

	// Set context information
	err.Method = opts.Method
	err.Interface = opts.Interface
	err.Package = opts.Package

	// Add metadata
	for k, v := range opts.Metadata {
		err.Metadata[k] = v
	}

	// Generate stack trace if enabled
	if eh.enableStackTrace {
		err.StackTrace = eh.captureStackTrace()
	}

	// Generate suggestions
	err.Suggestions = eh.generateSuggestions(err)

	return err
}

// ErrorOptions contains options for creating contextual errors
type ErrorOptions struct {
	Message    string
	Cause      error
	Code       string
	Category   ErrorCategory
	Severity   ErrorSeverity
	SourcePath string
	Position   token.Position
	TokenPos   token.Pos
	Method     string
	Interface  string
	Package    string
	Metadata   map[string]interface{}
}

// Wrap wraps an existing error with additional context
func (eh *ErrorHandler) Wrap(err error, opts ErrorOptions) *ContextualError {
	opts.Cause = err
	if opts.Message == "" {
		opts.Message = err.Error()
	}
	return eh.CreateError(opts)
}

// WrapWithContext wraps an error with context information
func (eh *ErrorHandler) WrapWithContext(ctx context.Context, err error, message string) *ContextualError {
	opts := ErrorOptions{
		Message:  message,
		Cause:    err,
		Category: eh.categorizeError(err),
		Severity: eh.determineSeverity(err),
	}

	// Extract context information if available
	if ctx != nil {
		if value := ctx.Value("method"); value != nil {
			if method, ok := value.(string); ok {
				opts.Method = method
			}
		}
		if value := ctx.Value("interface"); value != nil {
			if intf, ok := value.(string); ok {
				opts.Interface = intf
			}
		}
	}

	return eh.CreateError(opts)
}

// Chain creates a chain of related errors
func (eh *ErrorHandler) Chain(primary *ContextualError, related ...*ContextualError) *ContextualError {
	if primary == nil {
		return nil
	}

	primary.Related = append(primary.Related, related...)
	return primary
}

// RecoverFromPanic recovers from a panic and converts it to a contextual error
func (eh *ErrorHandler) RecoverFromPanic(recovered interface{}) *ContextualError {
	return eh.CreateError(ErrorOptions{
		Message:  fmt.Sprintf("panic recovered: %v", recovered),
		Code:     "PANIC_RECOVERED",
		Category: CategoryGeneral,
		Severity: SeverityCritical,
		Metadata: map[string]interface{}{
			"panic_value": recovered,
		},
	})
}

// captureStackTrace captures the current stack trace
func (eh *ErrorHandler) captureStackTrace() []string {
	const maxFrames = 10
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(3, pcs) // Skip captureStackTrace, CreateError, and the caller

	frames := runtime.CallersFrames(pcs[:n])
	var stackTrace []string

	for {
		frame, more := frames.Next()

		// Skip internal Go runtime frames
		if !strings.Contains(frame.File, "convergen") {
			if !more {
				break
			}
			continue
		}

		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))

		if !more {
			break
		}
	}

	return stackTrace
}

// categorizeError attempts to categorize an error based on its content using simplified classification
func (eh *ErrorHandler) categorizeError(err error) ErrorCategory {
	category, _, _ := ClassifyError(err)
	return category
}

// determineSeverity determines the severity of an error using simplified classification
func (eh *ErrorHandler) determineSeverity(err error) ErrorSeverity {
	_, severity, _ := ClassifyError(err)
	return severity
}

// generateSuggestions generates helpful suggestions for resolving errors using simplified classification
func (eh *ErrorHandler) generateSuggestions(err *ContextualError) []string {
	if err.Cause != nil {
		// Get the primary suggestion from error classification
		_, _, suggestion := ClassifyError(err.Cause)
		if suggestion != "" {
			return []string{suggestion}
		}
	}

	// Fallback to category-based suggestion
	_, _, suggestion := ClassifyErrorWithContext(err, err.Method+"|"+err.Interface)
	if suggestion != "" {
		return []string{suggestion}
	}

	return []string{"Check the error message for more details."}
}

// IsTemporary checks if an error is temporary and might succeed on retry using simplified classification
func (eh *ErrorHandler) IsTemporary(err error) bool {
	if contextual, ok := err.(*ContextualError); ok {
		return IsRetryableError(contextual.Category) || IsSkippableError(contextual.Severity)
	}

	// Fallback: classify the error directly
	category, severity, _ := ClassifyError(err)
	return IsRetryableError(category) || IsSkippableError(severity)
}

// ShouldRetry determines if an operation should be retried based on the error
func (eh *ErrorHandler) ShouldRetry(err error, attempt int, maxAttempts int) bool {
	if err == nil || attempt >= maxAttempts {
		return false
	}

	return eh.IsTemporary(err)
}

// GetRetryDelay calculates the delay before retrying an operation
func (eh *ErrorHandler) GetRetryDelay(attempt int) time.Duration {
	// Exponential backoff with jitter
	base := time.Duration(100<<uint(attempt)) * time.Millisecond
	jitter := time.Duration(attempt*50) * time.Millisecond
	return base + jitter
}
