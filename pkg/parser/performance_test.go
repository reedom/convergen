package parser

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestPackageLoader_LoadPackageConcurrent(t *testing.T) {
	tests := []struct {
		name          string
		maxWorkers    int
		timeout       time.Duration
		expectError   bool
		expectedCache bool
	}{
		{
			name:          "valid_package_loading",
			maxWorkers:    2,
			timeout:       10 * time.Second,
			expectError:   false,
			expectedCache: true,
		},
		{
			name:        "timeout_handling",
			maxWorkers:  1,
			timeout:     1 * time.Nanosecond, // Very short timeout
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewPackageLoader(tt.maxWorkers, tt.timeout)

			// Test with invalid path first
			ctx := context.Background()
			result, err := loader.LoadPackageConcurrent(ctx, "nonexistent.go", "")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				// For valid cases, we expect an error since we're using invalid paths
				// This tests the error handling path
				if err == nil {
					t.Error("expected error for invalid path")
				}
			}

			if result == nil {
				t.Error("expected result struct even on error")
			}
		})
	}
}

func TestPackageLoader_CacheManagement(t *testing.T) {
	loader := NewPackageLoader(2, 5*time.Second)

	// Test cache clearing
	loader.ClearCache()

	// Test cache stats
	hits, misses := loader.GetCacheStats()
	if hits != 0 {
		t.Errorf("expected 0 cache hits after clear, got %d", hits)
	}

	if misses != 0 {
		t.Errorf("expected 0 cache misses, got %d", misses)
	}
}

func TestConcurrentMethodProcessor_ProcessingMetrics(t *testing.T) {
	// Create a mock parser (we'll need to adapt this based on actual Parser structure)
	parser := &Parser{
		config: &ParserConfig{
			MaxConcurrentWorkers:  2,
			TypeResolutionTimeout: 5 * time.Second,
		},
	}

	logger := zap.NewNop()
	processor := NewConcurrentMethodProcessor(parser, 2, 5*time.Second, logger)

	// Test metrics initialization
	metrics := processor.GetMetrics()
	if metrics.TotalMethods != 0 {
		t.Errorf("expected 0 total methods initially, got %d", metrics.TotalMethods)
	}

	if metrics.SuccessfulMethods != 0 {
		t.Errorf("expected 0 successful methods initially, got %d", metrics.SuccessfulMethods)
	}

	if metrics.FailedMethods != 0 {
		t.Errorf("expected 0 failed methods initially, got %d", metrics.FailedMethods)
	}
}

func TestParserConfig_DefaultValues(t *testing.T) {
	// Test NewParser with default config
	_, err := NewParser("nonexistent.go", "output.go")

	// We expect an error since the file doesn't exist, but we can test that the function doesn't panic
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParserConfig_CustomConfig(t *testing.T) {
	config := &ParserConfig{
		MaxConcurrentWorkers:    8,
		TypeResolutionTimeout:   60 * time.Second,
		EnableConcurrentLoading: true,
		EnableMethodConcurrency: true,
	}

	// Test NewParserWithConfig with custom config
	_, err := NewParserWithConfig("nonexistent.go", "output.go", config)

	// We expect an error since the file doesn't exist
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	// The error should mention concurrent loading since that's enabled
	if !strings.Contains(err.Error(), "concurrent") {
		t.Error("expected error message to mention concurrent processing")
	}
}

func TestParserConfig_DisabledConcurrency(t *testing.T) {
	config := &ParserConfig{
		MaxConcurrentWorkers:    4,
		TypeResolutionTimeout:   30 * time.Second,
		EnableConcurrentLoading: false, // Disabled
		EnableMethodConcurrency: false, // Disabled
	}

	// Test NewParserWithConfig with disabled concurrency
	_, err := NewParserWithConfig("nonexistent.go", "output.go", config)

	// We expect an error since the file doesn't exist
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	// The error should NOT mention concurrent loading since that's disabled
	if strings.Contains(err.Error(), "concurrent") {
		t.Error("expected error message to NOT mention concurrent processing when disabled")
	}
}

// Benchmark tests for performance comparison.
func BenchmarkPackageLoader_Sequential(b *testing.B) {
	loader := NewPackageLoader(1, 10*time.Second) // Single worker
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Benchmark with invalid path (tests the loading path without actual file I/O)
		_, _ = loader.LoadPackageConcurrent(ctx, "benchmark.go", "")
	}
}

func BenchmarkPackageLoader_Concurrent(b *testing.B) {
	loader := NewPackageLoader(4, 10*time.Second) // Multiple workers
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Benchmark with invalid path (tests the loading path without actual file I/O)
		_, _ = loader.LoadPackageConcurrent(ctx, "benchmark.go", "")
	}
}

func BenchmarkConcurrentMethodProcessor_Creation(b *testing.B) {
	parser := &Parser{
		config: &ParserConfig{
			MaxConcurrentWorkers:  4,
			TypeResolutionTimeout: 5 * time.Second,
		},
	}
	logger := zap.NewNop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewConcurrentMethodProcessor(parser, 4, 5*time.Second, logger)
	}
}

// Test error classification functions.
func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isTypeRes bool
		isAnnot   bool
	}{
		{
			name:      "nil_error",
			err:       nil,
			isTypeRes: false,
			isAnnot:   false,
		},
		{
			name:      "generic_error",
			err:       errors.New("generic error"), // Generic error
			isTypeRes: false,
			isAnnot:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				return // Skip nil error test
			}

			if got := isTypeResolutionError(tt.err); got != tt.isTypeRes {
				t.Errorf("isTypeResolutionError() = %v, want %v", got, tt.isTypeRes)
			}

			if got := isAnnotationError(tt.err); got != tt.isAnnot {
				t.Errorf("isAnnotationError() = %v, want %v", got, tt.isAnnot)
			}
		})
	}
}
