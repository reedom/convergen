package coordinator

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewErrorHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	
	handler := NewErrorHandler(logger, config)
	
	if handler == nil {
		t.Fatal("NewErrorHandler returned nil")
	}
	
	// Verify it implements the interface
	var _ ErrorHandler = handler
}

func TestErrorHandlerCollectError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	testErr := errors.New("test error")
	handler.CollectError("test-component", testErr)
	
	report := handler.GetErrors()
	
	if report.TotalCount != 1 {
		t.Errorf("Expected 1 total error, got %d", report.TotalCount)
	}
	
	if len(report.Errors) != 1 {
		t.Errorf("Expected 1 error in list, got %d", len(report.Errors))
	}
	
	if report.Errors[0].Component != "test-component" {
		t.Errorf("Expected component 'test-component', got %q", report.Errors[0].Component)
	}
	
	if report.Errors[0].Error != testErr {
		t.Errorf("Expected error %v, got %v", testErr, report.Errors[0].Error)
	}
	
	if report.Errors[0].Stage != StageInitializing {
		t.Errorf("Expected stage %s, got %s", StageInitializing, report.Errors[0].Stage)
	}
}

func TestErrorHandlerCollectCriticalError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	testErr := errors.New("critical error")
	handler.CollectCriticalError("parser", testErr)
	
	report := handler.GetErrors()
	
	if report.TotalCount != 1 {
		t.Errorf("Expected 1 total error, got %d", report.TotalCount)
	}
	
	if report.CriticalCount != 1 {
		t.Errorf("Expected 1 critical error, got %d", report.CriticalCount)
	}
	
	if len(report.Critical) != 1 {
		t.Errorf("Expected 1 critical error in list, got %d", len(report.Critical))
	}
	
	if report.Critical[0] != testErr {
		t.Errorf("Expected critical error %v, got %v", testErr, report.Critical[0])
	}
	
	// Critical errors should stop pipeline
	if !handler.ShouldStop() {
		t.Error("Expected pipeline to stop due to critical error")
	}
}

func TestErrorHandlerCollectWarning(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	testErr := errors.New("warning message")
	handler.CollectWarning("planner", testErr)
	
	report := handler.GetErrors()
	
	if report.TotalCount != 1 {
		t.Errorf("Expected 1 total error, got %d", report.TotalCount)
	}
	
	if report.WarningCount != 1 {
		t.Errorf("Expected 1 warning, got %d", report.WarningCount)
	}
	
	if len(report.Warnings) != 1 {
		t.Errorf("Expected 1 warning in list, got %d", len(report.Warnings))
	}
	
	if report.Warnings[0] != testErr {
		t.Errorf("Expected warning %v, got %v", testErr, report.Warnings[0])
	}
	
	// Warnings alone should not stop pipeline
	if handler.ShouldStop() {
		t.Error("Expected pipeline not to stop due to warning")
	}
}

func TestErrorHandlerShouldStopOnFirstError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	config.StopOnFirstError = true
	handler := NewErrorHandler(logger, config)
	
	testErr := errors.New("first error")
	handler.CollectError("component", testErr)
	
	if !handler.ShouldStop() {
		t.Error("Expected pipeline to stop on first error")
	}
}

func TestErrorHandlerShouldStopOnThreshold(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	config.ErrorThreshold = 3
	config.StopOnFirstError = false
	handler := NewErrorHandler(logger, config)
	
	// Add errors up to threshold
	for i := 0; i < 3; i++ {
		handler.CollectError("component", errors.New("error"))
	}
	
	if !handler.ShouldStop() {
		t.Error("Expected pipeline to stop when error threshold reached")
	}
}

func TestErrorHandlerShouldStopOnMaxRetries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	config.MaxRetries = 2
	config.StopOnFirstError = false
	handler := NewErrorHandler(logger, config)
	
	// Add retryable errors exceeding max retries
	for i := 0; i < 3; i++ {
		handler.CollectError("component", errors.New("timeout"))
	}
	
	if !handler.ShouldStop() {
		t.Error("Expected pipeline to stop when max retries exceeded")
	}
}

func TestErrorHandlerReset(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	// Add various errors
	handler.CollectError("component1", errors.New("error1"))
	handler.CollectCriticalError("component2", errors.New("critical"))
	handler.CollectWarning("component3", errors.New("warning"))
	
	// Verify errors exist
	report := handler.GetErrors()
	if report.TotalCount == 0 {
		t.Error("Expected errors before reset")
	}
	
	// Reset
	handler.Reset()
	
	// Verify errors are cleared
	report = handler.GetErrors()
	if report.TotalCount != 0 {
		t.Errorf("Expected 0 errors after reset, got %d", report.TotalCount)
	}
	
	if report.CriticalCount != 0 {
		t.Errorf("Expected 0 critical errors after reset, got %d", report.CriticalCount)
	}
	
	if report.WarningCount != 0 {
		t.Errorf("Expected 0 warnings after reset, got %d", report.WarningCount)
	}
	
	// Should not stop after reset
	if handler.ShouldStop() {
		t.Error("Expected pipeline not to stop after reset")
	}
}

func TestErrorHandlerSetErrorThreshold(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	config.ErrorThreshold = 5
	handler := NewErrorHandler(logger, config)
	
	// Set new threshold
	handler.SetErrorThreshold(2)
	
	// Add errors up to new threshold
	handler.CollectError("component", errors.New("error1"))
	handler.CollectError("component", errors.New("error2"))
	
	if !handler.ShouldStop() {
		t.Error("Expected pipeline to stop with new error threshold")
	}
}

func TestErrorHandlerGetErrorStats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	// Add various errors
	handler.CollectError("parser", errors.New("parse error"))
	handler.CollectError("parser", errors.New("another parse error"))
	handler.CollectCriticalError("executor", errors.New("critical error"))
	handler.CollectWarning("planner", errors.New("warning"))
	
	stats := handler.GetErrorStats()
	
	expectedStats := map[string]int64{
		"total_errors":    4,
		"critical_errors": 1,
		"warnings":        1,
		"parser_errors":   2,
		"executor_errors": 1,
		"planner_errors":  1,
	}
	
	for key, expected := range expectedStats {
		if actual, exists := stats[key]; !exists {
			t.Errorf("Expected stat %s not found", key)
		} else if actual != expected {
			t.Errorf("Expected %s=%d, got %d", key, expected, actual)
		}
	}
}

func TestErrorHandlerRetryableErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	tests := []struct {
		error     error
		retryable bool
	}{
		{errors.New("timeout occurred"), true},
		{errors.New("connection refused"), true},
		{errors.New("temporary failure"), true},
		{errors.New("syntax error"), false},
		{errors.New("invalid configuration"), false},
	}
	
	for _, test := range tests {
		handler.Reset()
		handler.CollectError("component", test.error)
		
		report := handler.GetErrors()
		if len(report.Errors) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(report.Errors))
		}
		
		actual := report.Errors[0].Retryable
		if actual != test.retryable {
			t.Errorf("Error %q: expected retryable=%v, got %v", 
				test.error.Error(), test.retryable, actual)
		}
	}
}

func TestErrorHandlerComponentStageMapping(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	tests := []struct {
		component string
		stage     PipelineStage
	}{
		{"parser", StageParsing},
		{"planner", StagePlanning},
		{"executor", StageExecuting},
		{"emitter", StageEmitting},
		{"unknown", StageInitializing},
	}
	
	for _, test := range tests {
		handler.Reset()
		handler.CollectError(test.component, errors.New("test error"))
		
		report := handler.GetErrors()
		if len(report.Errors) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(report.Errors))
		}
		
		actual := report.Errors[0].Stage
		if actual != test.stage {
			t.Errorf("Component %q: expected stage %s, got %s", 
				test.component, test.stage, actual)
		}
	}
}

// Test ErrorReport methods

func TestErrorReportHasCriticalErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	report := handler.GetErrors()
	if report.HasCriticalErrors() {
		t.Error("Expected no critical errors initially")
	}
	
	handler.CollectCriticalError("component", errors.New("critical"))
	report = handler.GetErrors()
	if !report.HasCriticalErrors() {
		t.Error("Expected critical errors after adding one")
	}
}

func TestErrorReportHasWarnings(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	report := handler.GetErrors()
	if report.HasWarnings() {
		t.Error("Expected no warnings initially")
	}
	
	handler.CollectWarning("component", errors.New("warning"))
	report = handler.GetErrors()
	if !report.HasWarnings() {
		t.Error("Expected warnings after adding one")
	}
}

func TestErrorReportGetErrorsByComponent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	handler.CollectError("parser", errors.New("parse error 1"))
	handler.CollectError("parser", errors.New("parse error 2"))
	handler.CollectError("executor", errors.New("execute error"))
	
	report := handler.GetErrors()
	errorsByComponent := report.GetErrorsByComponent()
	
	if len(errorsByComponent["parser"]) != 2 {
		t.Errorf("Expected 2 parser errors, got %d", len(errorsByComponent["parser"]))
	}
	
	if len(errorsByComponent["executor"]) != 1 {
		t.Errorf("Expected 1 executor error, got %d", len(errorsByComponent["executor"]))
	}
}

func TestErrorReportGetErrorsByStage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	handler.CollectError("parser", errors.New("parse error"))
	handler.CollectError("executor", errors.New("execute error"))
	
	report := handler.GetErrors()
	errorsByStage := report.GetErrorsByStage()
	
	if len(errorsByStage[StageParsing]) != 1 {
		t.Errorf("Expected 1 parsing error, got %d", len(errorsByStage[StageParsing]))
	}
	
	if len(errorsByStage[StageExecuting]) != 1 {
		t.Errorf("Expected 1 executing error, got %d", len(errorsByStage[StageExecuting]))
	}
}

func TestErrorReportGetRetryableErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	handler.CollectError("component", errors.New("timeout"))
	handler.CollectError("component", errors.New("syntax error"))
	
	report := handler.GetErrors()
	retryableErrors := report.GetRetryableErrors()
	
	if len(retryableErrors) != 1 {
		t.Errorf("Expected 1 retryable error, got %d", len(retryableErrors))
	}
	
	if retryableErrors[0].Error.Error() != "timeout" {
		t.Errorf("Expected timeout error, got %q", retryableErrors[0].Error.Error())
	}
}

func TestErrorReportSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	// Test empty report
	report := handler.GetErrors()
	summary := report.Summary()
	if summary != "No errors" {
		t.Errorf("Expected 'No errors', got %q", summary)
	}
	
	// Test with errors
	handler.CollectError("parser", errors.New("first error"))
	handler.CollectCriticalError("executor", errors.New("critical error"))
	handler.CollectWarning("planner", errors.New("warning"))
	
	report = handler.GetErrors()
	summary = report.Summary()
	
	expected := "Total: 3 errors (1 critical) (1 warnings) | First: first error in parser"
	if summary != expected {
		t.Errorf("Expected summary %q, got %q", expected, summary)
	}
}

// Concurrent access tests

func TestErrorHandlerConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	done := make(chan bool, 10)
	
	// Concurrent error collection
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				switch j % 3 {
				case 0:
					handler.CollectError("component", errors.New("error"))
				case 1:
					handler.CollectWarning("component", errors.New("warning"))
				case 2:
					_ = handler.GetErrors()
				}
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify final state is consistent
	report := handler.GetErrors()
	if report.TotalCount != len(report.Errors) {
		t.Errorf("Inconsistent error count: total=%d, actual=%d", 
			report.TotalCount, len(report.Errors))
	}
}

// Benchmark tests

func BenchmarkErrorHandlerCollectError(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	testErr := errors.New("benchmark error")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		handler.CollectError("component", testErr)
	}
}

func BenchmarkErrorHandlerGetErrors(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	// Pre-populate with errors
	for i := 0; i < 100; i++ {
		handler.CollectError("component", errors.New("error"))
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = handler.GetErrors()
	}
}

func BenchmarkErrorHandlerShouldStop(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	handler := NewErrorHandler(logger, config)
	
	// Add some errors
	for i := 0; i < 5; i++ {
		handler.CollectError("component", errors.New("error"))
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = handler.ShouldStop()
	}
}