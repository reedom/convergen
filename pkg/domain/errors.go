package domain

import (
	"fmt"
	"strings"
	"time"
)

// ErrorCode categorizes different types of generation errors
type ErrorCode int

const (
	ErrTypeResolution ErrorCode = iota
	ErrIncompatibleTypes
	ErrCircularDependency
	ErrInvalidAnnotation
	ErrConverterNotFound
	ErrCodeGeneration
	ErrResourceExhausted
	ErrTimeout
	ErrValidation
	ErrInternal
)

func (e ErrorCode) String() string {
	switch e {
	case ErrTypeResolution:
		return "TYPE_RESOLUTION"
	case ErrIncompatibleTypes:
		return "INCOMPATIBLE_TYPES"
	case ErrCircularDependency:
		return "CIRCULAR_DEPENDENCY"
	case ErrInvalidAnnotation:
		return "INVALID_ANNOTATION"
	case ErrConverterNotFound:
		return "CONVERTER_NOT_FOUND"
	case ErrCodeGeneration:
		return "CODE_GENERATION"
	case ErrResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case ErrTimeout:
		return "TIMEOUT"
	case ErrValidation:
		return "VALIDATION"
	case ErrInternal:
		return "INTERNAL"
	default:
		return "UNKNOWN"
	}
}

// ProcessingPhase identifies where an error occurred
type ProcessingPhase int

const (
	PhaseParsing ProcessingPhase = iota
	PhasePlanning
	PhaseExecution
	PhaseEmission
	PhaseValidation
)

func (p ProcessingPhase) String() string {
	switch p {
	case PhaseParsing:
		return "parsing"
	case PhasePlanning:
		return "planning"
	case PhaseExecution:
		return "execution"
	case PhaseEmission:
		return "emission"
	case PhaseValidation:
		return "validation"
	default:
		return "unknown"
	}
}

// SourceLocation represents a location in source code
type SourceLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Offset int    `json:"offset"`
}

func (sl *SourceLocation) String() string {
	if sl.File == "" {
		return "unknown location"
	}
	return fmt.Sprintf("%s:%d:%d", sl.File, sl.Line, sl.Column)
}

// GenerationError provides rich error context for generation failures
type GenerationError struct {
	Code      ErrorCode               `json:"code"`
	Message   string                  `json:"message"`
	Phase     ProcessingPhase         `json:"phase"`
	Method    string                  `json:"method"`
	Field     string                  `json:"field"`
	Source    *SourceLocation         `json:"source"`
	Cause     error                   `json:"-"` // Original error (not serialized)
	CauseText string                  `json:"cause_text"`
	Context   map[string]interface{}  `json:"context"`
	Timestamp time.Time               `json:"timestamp"`
	Hints     []string                `json:"hints"`
}

// NewGenerationError creates a new generation error
func NewGenerationError(code ErrorCode, message string, phase ProcessingPhase) *GenerationError {
	return &GenerationError{
		Code:      code,
		Message:   message,
		Phase:     phase,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Hints:     make([]string, 0),
	}
}

// WithMethod adds method context to the error
func (e *GenerationError) WithMethod(method string) *GenerationError {
	e.Method = method
	return e
}

// WithField adds field context to the error
func (e *GenerationError) WithField(field string) *GenerationError {
	e.Field = field
	return e
}

// WithSource adds source location to the error
func (e *GenerationError) WithSource(source *SourceLocation) *GenerationError {
	e.Source = source
	return e
}

// WithCause adds the underlying cause
func (e *GenerationError) WithCause(cause error) *GenerationError {
	e.Cause = cause
	if cause != nil {
		e.CauseText = cause.Error()
	}
	return e
}

// WithContext adds context information
func (e *GenerationError) WithContext(key string, value interface{}) *GenerationError {
	e.Context[key] = value
	return e
}

// WithHint adds a helpful hint for resolving the error
func (e *GenerationError) WithHint(hint string) *GenerationError {
	e.Hints = append(e.Hints, hint)
	return e
}

// Error implements the error interface
func (e *GenerationError) Error() string {
	var parts []string
	
	// Add phase and code
	parts = append(parts, fmt.Sprintf("[%s:%s]", e.Phase, e.Code))
	
	// Add location if available
	if e.Source != nil {
		parts = append(parts, e.Source.String())
	}
	
	// Add method and field context
	if e.Method != "" {
		if e.Field != "" {
			parts = append(parts, fmt.Sprintf("%s.%s", e.Method, e.Field))
		} else {
			parts = append(parts, e.Method)
		}
	}
	
	// Add the main message
	parts = append(parts, e.Message)
	
	// Add cause if present
	if e.CauseText != "" {
		parts = append(parts, fmt.Sprintf("caused by: %s", e.CauseText))
	}
	
	return strings.Join(parts, " ")
}

// Unwrap returns the underlying cause
func (e *GenerationError) Unwrap() error {
	return e.Cause
}

// IsRetryable indicates if the error might be resolved by retrying
func (e *GenerationError) IsRetryable() bool {
	switch e.Code {
	case ErrResourceExhausted, ErrTimeout:
		return true
	default:
		return false
	}
}

// Severity returns the error severity level
func (e *GenerationError) Severity() DiagnosticLevel {
	switch e.Code {
	case ErrValidation, ErrInvalidAnnotation:
		return DiagnosticWarning
	default:
		return DiagnosticError
	}
}

// ErrorCollector aggregates errors from concurrent operations
type ErrorCollector struct {
	errors   []*GenerationError
	maxErrors int
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(maxErrors int) *ErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100 // Default limit
	}
	
	return &ErrorCollector{
		errors:   make([]*GenerationError, 0),
		maxErrors: maxErrors,
	}
}

// Collect adds an error to the collection
func (ec *ErrorCollector) Collect(err *GenerationError) {
	if len(ec.errors) < ec.maxErrors {
		ec.errors = append(ec.errors, err)
	}
}

// CollectError creates and collects a generation error
func (ec *ErrorCollector) CollectError(code ErrorCode, message string, phase ProcessingPhase) {
	err := NewGenerationError(code, message, phase)
	ec.Collect(err)
}

// HasErrors returns true if any errors have been collected
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []*GenerationError {
	return append([]*GenerationError(nil), ec.errors...) // defensive copy
}

// ErrorsByPhase returns errors grouped by processing phase
func (ec *ErrorCollector) ErrorsByPhase() map[ProcessingPhase][]*GenerationError {
	result := make(map[ProcessingPhase][]*GenerationError)
	
	for _, err := range ec.errors {
		result[err.Phase] = append(result[err.Phase], err)
	}
	
	return result
}

// ErrorsByCode returns errors grouped by error code
func (ec *ErrorCollector) ErrorsByCode() map[ErrorCode][]*GenerationError {
	result := make(map[ErrorCode][]*GenerationError)
	
	for _, err := range ec.errors {
		result[err.Code] = append(result[err.Code], err)
	}
	
	return result
}

// Summary returns a summary of collected errors
func (ec *ErrorCollector) Summary() string {
	if len(ec.errors) == 0 {
		return "No errors"
	}
	
	codeCount := make(map[ErrorCode]int)
	phaseCount := make(map[ProcessingPhase]int)
	
	for _, err := range ec.errors {
		codeCount[err.Code]++
		phaseCount[err.Phase]++
	}
	
	var parts []string
	parts = append(parts, fmt.Sprintf("Total errors: %d", len(ec.errors)))
	
	// Add breakdown by phase
	if len(phaseCount) > 0 {
		var phaseParts []string
		for phase, count := range phaseCount {
			phaseParts = append(phaseParts, fmt.Sprintf("%s: %d", phase, count))
		}
		parts = append(parts, "By phase: "+strings.Join(phaseParts, ", "))
	}
	
	// Add breakdown by code
	if len(codeCount) > 0 {
		var codeParts []string
		for code, count := range codeCount {
			codeParts = append(codeParts, fmt.Sprintf("%s: %d", code, count))
		}
		parts = append(parts, "By type: "+strings.Join(codeParts, ", "))
	}
	
	return strings.Join(parts, "; ")
}

// ToError converts the collected errors to a single error
func (ec *ErrorCollector) ToError() error {
	if len(ec.errors) == 0 {
		return nil
	}
	
	if len(ec.errors) == 1 {
		return ec.errors[0]
	}
	
	return &MultiError{
		Errors:  ec.errors,
		Summary: ec.Summary(),
	}
}

// MultiError represents multiple generation errors
type MultiError struct {
	Errors  []*GenerationError `json:"errors"`
	Summary string             `json:"summary"`
}

// Error implements the error interface
func (me *MultiError) Error() string {
	return fmt.Sprintf("multiple generation errors: %s", me.Summary)
}

// Unwrap returns the first error (for compatibility with error unwrapping)
func (me *MultiError) Unwrap() error {
	if len(me.Errors) > 0 {
		return me.Errors[0]
	}
	return nil
}

// Count returns the number of errors
func (me *MultiError) Count() int {
	return len(me.Errors)
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	Rule      string      `json:"rule"`
	Message   string      `json:"message"`
	Path      []string    `json:"path"`
}

// NewValidationError creates a new validation error
func NewValidationError(field, rule, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Rule:    rule,
		Message: message,
		Path:    make([]string, 0),
	}
}

// WithValue adds the invalid value to the error
func (ve *ValidationError) WithValue(value interface{}) *ValidationError {
	ve.Value = value
	return ve
}

// WithPath adds path context to the error
func (ve *ValidationError) WithPath(path []string) *ValidationError {
	ve.Path = append([]string(nil), path...)
	return ve
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	var parts []string
	
	if len(ve.Path) > 0 {
		parts = append(parts, strings.Join(ve.Path, "."))
	}
	
	parts = append(parts, ve.Field)
	parts = append(parts, fmt.Sprintf("validation failed [%s]: %s", ve.Rule, ve.Message))
	
	if ve.Value != nil {
		parts = append(parts, fmt.Sprintf("(value: %v)", ve.Value))
	}
	
	return strings.Join(parts, " ")
}

// Common error creation helpers

// ErrTypeResolutionFailed creates a type resolution error
func ErrTypeResolutionFailed(typeName string, cause error) *GenerationError {
	return NewGenerationError(ErrTypeResolution, fmt.Sprintf("failed to resolve type: %s", typeName), PhaseParsing).
		WithCause(cause).
		WithContext("type_name", typeName)
}

// ErrIncompatibleTypeConversion creates an incompatible types error
func ErrIncompatibleTypeConversion(sourceType, destType Type) *GenerationError {
	return NewGenerationError(ErrIncompatibleTypes, 
		fmt.Sprintf("cannot convert from %s to %s", sourceType.String(), destType.String()), 
		PhasePlanning).
		WithContext("source_type", sourceType.String()).
		WithContext("dest_type", destType.String()).
		WithHint("Consider using a custom converter function with :conv annotation").
		WithHint("Check if type casting is enabled with :typecast annotation")
}

// ErrCircularFieldDependency creates a circular dependency error
func ErrCircularFieldDependency(cycle []string) *GenerationError {
	return NewGenerationError(ErrCircularDependency, 
		fmt.Sprintf("circular dependency detected: %s", strings.Join(cycle, " -> ")), 
		PhasePlanning).
		WithContext("cycle", cycle).
		WithHint("Review field mappings to break the circular dependency")
}

// ErrInvalidAnnotationSyntax creates an invalid annotation error
func ErrInvalidAnnotationSyntax(annotation, reason string) *GenerationError {
	return NewGenerationError(ErrInvalidAnnotation, 
		fmt.Sprintf("invalid annotation syntax: %s (%s)", annotation, reason), 
		PhaseParsing).
		WithContext("annotation", annotation).
		WithContext("reason", reason).
		WithHint("Check the annotation syntax in the documentation")
}

// ErrConverterFunctionNotFound creates a converter not found error
func ErrConverterFunctionNotFound(funcName, pkg string) *GenerationError {
	return NewGenerationError(ErrConverterNotFound, 
		fmt.Sprintf("converter function not found: %s in package %s", funcName, pkg), 
		PhasePlanning).
		WithContext("function_name", funcName).
		WithContext("package", pkg).
		WithHint("Ensure the converter function is exported and properly imported")
}

// ErrCodeGenerationFailed creates a code generation error
func ErrCodeGenerationFailed(reason string, cause error) *GenerationError {
	return NewGenerationError(ErrCodeGeneration, 
		fmt.Sprintf("code generation failed: %s", reason), 
		PhaseEmission).
		WithCause(cause).
		WithContext("reason", reason)
}

// ErrResourceLimitExceeded creates a resource exhaustion error
func ErrResourceLimitExceeded(resource string, limit, requested int) *GenerationError {
	return NewGenerationError(ErrResourceExhausted, 
		fmt.Sprintf("%s limit exceeded: requested %d, limit %d", resource, requested, limit), 
		PhaseExecution).
		WithContext("resource", resource).
		WithContext("limit", limit).
		WithContext("requested", requested).
		WithHint("Consider reducing concurrency or increasing resource limits")
}

// ErrProcessingTimeout creates a timeout error
func ErrProcessingTimeout(operation string, timeoutMS int) *GenerationError {
	return NewGenerationError(ErrTimeout, 
		fmt.Sprintf("%s timed out after %d ms", operation, timeoutMS), 
		PhaseExecution).
		WithContext("operation", operation).
		WithContext("timeout_ms", timeoutMS).
		WithHint("Consider increasing timeout or reducing processing complexity")
}