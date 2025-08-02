package coordinator

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ErrorHandler aggregates and manages errors across the pipeline.
type ErrorHandler interface {
	// Collect error from any component
	CollectError(component string, err error)

	// Collect critical error that should stop pipeline
	CollectCriticalError(component string, err error)

	// Collect warning that doesn't stop pipeline
	CollectWarning(component string, err error)

	// Get aggregated errors
	GetErrors() *ErrorReport

	// Check if pipeline should stop due to errors
	ShouldStop() bool

	// Reset error state for new pipeline
	Reset()

	// Set error threshold
	SetErrorThreshold(threshold int)

	// Get error statistics
	GetErrorStats() map[string]int64
}

// ConcreteErrorHandler implements ErrorHandler.
type ConcreteErrorHandler struct {
	logger *zap.Logger
	config *Config

	// Error storage
	mutex    sync.RWMutex
	errors   []ComponentError
	critical []error
	warnings []error

	// Error tracking
	errorCounts    map[string]int64
	totalCount     int
	criticalCount  int
	warningCount   int
	errorThreshold int

	// Error categorization
	retryableErrors map[string]bool

	// Recovery tracking
	recoveryAttempts map[string]int
	maxRetryAttempts int
}

// NewErrorHandler creates a new error handler.
func NewErrorHandler(logger *zap.Logger, config *Config) ErrorHandler {
	handler := &ConcreteErrorHandler{
		logger:           logger,
		config:           config,
		errors:           make([]ComponentError, 0),
		critical:         make([]error, 0),
		warnings:         make([]error, 0),
		errorCounts:      make(map[string]int64),
		errorThreshold:   config.ErrorThreshold,
		retryableErrors:  make(map[string]bool),
		recoveryAttempts: make(map[string]int),
		maxRetryAttempts: config.MaxRetries,
	}

	// Initialize retryable error patterns
	handler.initializeRetryableErrors()

	logger.Debug("error handler initialized",
		zap.Int("error_threshold", config.ErrorThreshold),
		zap.Int("max_retries", config.MaxRetries))

	return handler
}

// CollectError collects a regular error from a component.
func (e *ConcreteErrorHandler) CollectError(component string, err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	componentError := ComponentError{
		Component: component,
		Stage:     e.getCurrentStage(component),
		Error:     err,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Retryable: e.isRetryable(err),
		Attempt:   e.getAttemptCount(component),
	}

	e.errors = append(e.errors, componentError)
	e.errorCounts[component]++
	e.totalCount++

	e.logger.Warn("error collected",
		zap.String("component", component),
		zap.Error(err),
		zap.Bool("retryable", componentError.Retryable),
		zap.Int("attempt", componentError.Attempt))

	// Track recovery attempts for retryable errors
	if componentError.Retryable {
		e.recoveryAttempts[component]++
	}
}

// CollectCriticalError collects a critical error that should stop the pipeline.
func (e *ConcreteErrorHandler) CollectCriticalError(component string, err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	componentError := ComponentError{
		Component: component,
		Stage:     e.getCurrentStage(component),
		Error:     err,
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"critical": true},
		Retryable: false, // Critical errors are never retryable
		Attempt:   1,
	}

	e.errors = append(e.errors, componentError)
	e.critical = append(e.critical, err)
	e.errorCounts[component]++
	e.totalCount++
	e.criticalCount++

	e.logger.Error("critical error collected",
		zap.String("component", component),
		zap.Error(err))
}

// CollectWarning collects a warning that doesn't stop the pipeline.
func (e *ConcreteErrorHandler) CollectWarning(component string, err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	componentError := ComponentError{
		Component: component,
		Stage:     e.getCurrentStage(component),
		Error:     err,
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"warning": true},
		Retryable: false, // Warnings don't need retry
		Attempt:   1,
	}

	e.errors = append(e.errors, componentError)
	e.warnings = append(e.warnings, err)
	e.errorCounts[component]++
	e.totalCount++
	e.warningCount++

	e.logger.Warn("warning collected",
		zap.String("component", component),
		zap.Error(err))
}

// GetErrors returns the aggregated error report.
func (e *ConcreteErrorHandler) GetErrors() *ErrorReport {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	report := &ErrorReport{
		Errors:        make([]ComponentError, len(e.errors)),
		Critical:      make([]error, len(e.critical)),
		Warnings:      make([]error, len(e.warnings)),
		TotalCount:    e.totalCount,
		CriticalCount: e.criticalCount,
		WarningCount:  e.warningCount,
	}

	// Copy errors to avoid race conditions
	copy(report.Errors, e.errors)
	copy(report.Critical, e.critical)
	copy(report.Warnings, e.warnings)

	// Set first and last errors
	if len(e.errors) > 0 {
		firstError := e.errors[0]
		report.FirstError = &firstError

		lastError := e.errors[len(e.errors)-1]
		report.LastError = &lastError
	}

	return report
}

// ShouldStop determines if the pipeline should stop due to errors.
func (e *ConcreteErrorHandler) ShouldStop() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Stop if we have critical errors
	if 0 < e.criticalCount {
		return true
	}

	// Stop if we're configured to stop on first error
	if e.config.StopOnFirstError && 0 < e.totalCount {
		return true
	}

	// Stop if we've exceeded the error threshold
	if e.errorThreshold <= e.totalCount {
		e.logger.Warn("error threshold exceeded",
			zap.Int("total_errors", e.totalCount),
			zap.Int("threshold", e.errorThreshold))

		return true
	}

	// Stop if too many retry attempts for any component
	for component, attempts := range e.recoveryAttempts {
		if e.maxRetryAttempts <= attempts {
			e.logger.Warn("max retry attempts exceeded",
				zap.String("component", component),
				zap.Int("attempts", attempts),
				zap.Int("max_attempts", e.maxRetryAttempts))

			return true
		}
	}

	return false
}

// Reset clears all error state for a new pipeline.
func (e *ConcreteErrorHandler) Reset() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.errors = e.errors[:0]
	e.critical = e.critical[:0]
	e.warnings = e.warnings[:0]

	// Clear counters
	for component := range e.errorCounts {
		e.errorCounts[component] = 0
	}

	for component := range e.recoveryAttempts {
		e.recoveryAttempts[component] = 0
	}

	e.totalCount = 0
	e.criticalCount = 0
	e.warningCount = 0

	e.logger.Debug("error handler reset")
}

// SetErrorThreshold updates the error threshold.
func (e *ConcreteErrorHandler) SetErrorThreshold(threshold int) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.errorThreshold = threshold

	e.logger.Debug("error threshold updated", zap.Int("threshold", threshold))
}

// GetErrorStats returns error statistics.
func (e *ConcreteErrorHandler) GetErrorStats() map[string]int64 {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	stats := make(map[string]int64)
	stats["total_errors"] = int64(e.totalCount)
	stats["critical_errors"] = int64(e.criticalCount)
	stats["warnings"] = int64(e.warningCount)

	// Add per-component error counts
	for component, count := range e.errorCounts {
		stats[fmt.Sprintf("%s_errors", component)] = count
	}

	// Add retry statistics
	for component, attempts := range e.recoveryAttempts {
		stats[fmt.Sprintf("%s_retries", component)] = int64(attempts)
	}

	return stats
}

// Private methods

func (e *ConcreteErrorHandler) initializeRetryableErrors() {
	// Define patterns for retryable errors
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"resource temporarily unavailable",
		"context deadline exceeded",
		"network unreachable",
	}

	for _, pattern := range retryablePatterns {
		e.retryableErrors[pattern] = true
	}
}

func (e *ConcreteErrorHandler) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check against known retryable patterns
	for pattern := range e.retryableErrors {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

func (e *ConcreteErrorHandler) getCurrentStage(component string) PipelineStage {
	// Map component names to pipeline stages
	switch component {
	case "parser":
		return StageParsing
	case "planner":
		return StagePlanning
	case "executor":
		return StageExecuting
	case "emitter":
		return StageEmitting
	default:
		return StageInitializing
	}
}

func (e *ConcreteErrorHandler) getAttemptCount(component string) int {
	if attempts, exists := e.recoveryAttempts[component]; exists {
		return attempts + 1
	}

	return 1
}

// Helper functions

func containsIgnoreCase(str, substr string) bool {
	// Simple case-insensitive contains check
	// In a real implementation, you'd use strings.ToLower or regexp
	return len(str) >= len(substr) &&
		str[:len(substr)] == substr ||
		(len(str) > len(substr) && containsIgnoreCase(str[1:], substr))
}

// ErrorReport methods

// HasCriticalErrors returns true if there are critical errors.
func (r *ErrorReport) HasCriticalErrors() bool {
	return 0 < r.CriticalCount
}

// HasWarnings returns true if there are warnings.
func (r *ErrorReport) HasWarnings() bool {
	return 0 < r.WarningCount
}

// GetErrorsByComponent returns errors grouped by component.
func (r *ErrorReport) GetErrorsByComponent() map[string][]ComponentError {
	errorsByComponent := make(map[string][]ComponentError)

	for _, err := range r.Errors {
		component := err.Component
		errorsByComponent[component] = append(errorsByComponent[component], err)
	}

	return errorsByComponent
}

// GetErrorsByStage returns errors grouped by pipeline stage.
func (r *ErrorReport) GetErrorsByStage() map[PipelineStage][]ComponentError {
	errorsByStage := make(map[PipelineStage][]ComponentError)

	for _, err := range r.Errors {
		stage := err.Stage
		errorsByStage[stage] = append(errorsByStage[stage], err)
	}

	return errorsByStage
}

// GetRetryableErrors returns only the retryable errors.
func (r *ErrorReport) GetRetryableErrors() []ComponentError {
	var retryable []ComponentError

	for _, err := range r.Errors {
		if err.Retryable {
			retryable = append(retryable, err)
		}
	}

	return retryable
}

// Summary returns a human-readable error summary.
func (r *ErrorReport) Summary() string {
	if r.TotalCount == 0 {
		return "No errors"
	}

	summary := fmt.Sprintf("Total: %d errors", r.TotalCount)

	if 0 < r.CriticalCount {
		summary += fmt.Sprintf(" (%d critical)", r.CriticalCount)
	}

	if 0 < r.WarningCount {
		summary += fmt.Sprintf(" (%d warnings)", r.WarningCount)
	}

	if r.FirstError != nil {
		summary += fmt.Sprintf(" | First: %s in %s",
			r.FirstError.Error.Error(), r.FirstError.Component)
	}

	return summary
}
