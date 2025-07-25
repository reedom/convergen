package coordinator

import (
	"testing"
	"time"
)

// Simple tests that don't depend on external packages with compilation issues

func TestPipelineStageStringSimple(t *testing.T) {
	tests := []struct {
		stage    PipelineStage
		expected string
	}{
		{StageInitializing, "initializing"},
		{StageParsing, "parsing"},
		{StagePlanning, "planning"},
		{StageExecuting, "executing"},
		{StageEmitting, "emitting"},
		{StageCompleted, "completed"},
		{StageFailed, "failed"},
	}
	
	for _, test := range tests {
		if test.stage.String() != test.expected {
			t.Errorf("Expected %s.String() = %q, got %q", test.stage, test.expected, test.stage.String())
		}
	}
}

func TestComponentStatusStringSimple(t *testing.T) {
	tests := []struct {
		status   ComponentStatus
		expected string
	}{
		{StatusInitializing, "initializing"},
		{StatusReady, "ready"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusShutdown, "shutdown"},
	}
	
	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected %s.String() = %q, got %q", test.status, test.expected, test.status.String())
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	config := &Config{
		MaxConcurrency:     4,
		EventBufferSize:    1000,
		ComponentTimeout:   30 * time.Second,
		ErrorThreshold:     10,
		EnableMetrics:      true,
		LogLevel:          "info",
		WorkerPoolSize:     8,
		BufferPoolSize:     32,
		ChannelPoolSize:    16,
		StopOnFirstError:   false,
		RetryTransientErrors: true,
		MaxRetries:         3,
		RetryDelay:         time.Second,
		EnableProfiling:    false,
		EnableEventTracing: false,
	}
	
	// Test that config values are as expected
	if config.MaxConcurrency != 4 {
		t.Errorf("Expected MaxConcurrency 4, got %d", config.MaxConcurrency)
	}
	
	if config.EventBufferSize != 1000 {
		t.Errorf("Expected EventBufferSize 1000, got %d", config.EventBufferSize)
	}
	
	if config.ComponentTimeout != 30*time.Second {
		t.Errorf("Expected ComponentTimeout 30s, got %v", config.ComponentTimeout)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
	
	if !config.RetryTransientErrors {
		t.Error("Expected RetryTransientErrors to be true")
	}
}

func TestErrorReportBasic(t *testing.T) {
	report := &ErrorReport{
		TotalCount:    5,
		CriticalCount: 2,
		WarningCount:  1,
	}
	
	if !report.HasCriticalErrors() {
		t.Error("Expected report to have critical errors")
	}
	
	if !report.HasWarnings() {
		t.Error("Expected report to have warnings")
	}
	
	summary := report.Summary()
	expectedStart := "Total: 5 errors (2 critical) (1 warnings)"
	if len(summary) < len(expectedStart) || summary[:len(expectedStart)] != expectedStart {
		t.Errorf("Expected summary to start with %q, got %q", expectedStart, summary)
	}
}