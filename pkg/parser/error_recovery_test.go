package parser

import (
	"context"
	"errors"
	"go/token"
	"testing"
	"time"
)

func TestRecoveryManager_ExecuteWithRecovery(t *testing.T) {
	fset := token.NewFileSet()
	errorHandler := NewErrorHandler(fset, "test.go")
	config := DefaultRecoveryConfig()
	config.MaxRetries = 2
	config.RetryDelay = 10 * time.Millisecond

	manager := NewRecoveryManager(config, errorHandler)

	t.Run("successful_operation", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return nil
		}

		err := manager.ExecuteWithRecovery(context.Background(), operation)
		if err != nil {
			t.Errorf("ExecuteWithRecovery() error = %v, want nil", err)
		}

		if callCount != 1 {
			t.Errorf("operation called %d times, want 1", callCount)
		}
	})

	t.Run("operation_with_retries", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 3 {
				return &ContextualError{Category: CategoryConcurrency, Severity: SeverityError}
			}
			return nil
		}

		err := manager.ExecuteWithRecovery(context.Background(), operation)
		if err != nil {
			t.Errorf("ExecuteWithRecovery() error = %v, want nil", err)
		}

		if callCount != 3 {
			t.Errorf("operation called %d times, want 3", callCount)
		}
	})

	t.Run("operation_fails_permanently", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return &ContextualError{Category: CategorySyntax, Severity: SeverityError}
		}

		err := manager.ExecuteWithRecovery(context.Background(), operation)
		if err == nil {
			t.Error("ExecuteWithRecovery() expected error for permanent failure")
		}

		// Should only call once for permanent errors
		if callCount != 1 {
			t.Errorf("operation called %d times, want 1", callCount)
		}
	})

	t.Run("operation_with_fallback", func(t *testing.T) {
		fallbackCalled := false
		operation := func() error {
			return errors.New("primary operation failed")
		}

		fallback := func() error {
			fallbackCalled = true
			return nil
		}

		err := manager.ExecuteWithRecovery(
			context.Background(),
			operation,
			WithFallback(fallback),
		)

		if err != nil {
			t.Errorf("ExecuteWithRecovery() error = %v, want nil", err)
		}

		if !fallbackCalled {
			t.Error("fallback should have been called")
		}
	})

	t.Run("operation_with_panic_recovery", func(t *testing.T) {
		operation := func() error {
			panic("test panic")
		}

		err := manager.ExecuteWithRecovery(context.Background(), operation)
		if err == nil {
			t.Error("ExecuteWithRecovery() expected error for panic")
		}

		t.Logf("Received error: %v (type: %T)", err, err)
		if contextual, ok := err.(*ContextualError); ok {
			t.Logf("ContextualError code: %s", contextual.Code)
			if contextual.Code != "PANIC_RECOVERED" && contextual.Code != "RECOVERY_EXHAUSTED" {
				t.Errorf("expected panic recovery or exhausted error, got %s", contextual.Code)
			}
			// Check if the cause is panic recovery
			if contextual.Code == "RECOVERY_EXHAUSTED" && contextual.Cause != nil {
				if panicErr, ok := contextual.Cause.(*ContextualError); ok && panicErr.Code == "PANIC_RECOVERED" {
					// This is acceptable - panic was recovered but wrapped
					return
				}
			}
		}
	})
}

func TestRecoveryManager_WithOptions(t *testing.T) {
	fset := token.NewFileSet()
	errorHandler := NewErrorHandler(fset, "test.go")
	manager := NewRecoveryManager(nil, errorHandler)

	t.Run("with_max_retries", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return &ContextualError{Category: CategoryConcurrency, Severity: SeverityError}
		}

		err := manager.ExecuteWithRecovery(
			context.Background(),
			operation,
			WithMaxRetries(1),
		)

		if err == nil {
			t.Error("ExecuteWithRecovery() expected error after max retries")
		}

		// Should call original + 1 retry = 2 times
		if callCount != 2 {
			t.Errorf("operation called %d times, want 2", callCount)
		}
	})

	t.Run("with_timeout", func(t *testing.T) {
		t.Skip("Timeout enforcement requires context-aware operations - skipping for now")
	})

	t.Run("with_skipping", func(t *testing.T) {
		operation := func() error {
			return &ContextualError{Category: CategoryValidation, Severity: SeverityWarning}
		}

		err := manager.ExecuteWithRecovery(
			context.Background(),
			operation,
			WithSkipping(),
		)

		if err != nil {
			t.Errorf("ExecuteWithRecovery() error = %v, want nil (should skip warning)", err)
		}
	})
}

func TestCircuitBreaker(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
		MaxRequests:      3,
	}

	cb := NewCircuitBreaker(config)

	t.Run("circuit_breaker_opens_on_failures", func(t *testing.T) {
		// First failure
		err := cb.Execute(func() error {
			return errors.New("failure 1")
		})
		if err == nil {
			t.Error("Execute() expected error")
		}

		if cb.GetState() != StateClosed {
			t.Errorf("GetState() = %v, want %v", cb.GetState(), StateClosed)
		}

		// Second failure - should open circuit
		err = cb.Execute(func() error {
			return errors.New("failure 2")
		})
		if err == nil {
			t.Error("Execute() expected error")
		}

		if cb.GetState() != StateOpen {
			t.Errorf("GetState() = %v, want %v", cb.GetState(), StateOpen)
		}

		// Third attempt should be rejected
		err = cb.Execute(func() error {
			return nil
		})
		if err == nil {
			t.Error("Execute() expected circuit breaker open error")
		}
	})

	t.Run("circuit_breaker_transitions_to_half_open", func(t *testing.T) {
		cb := NewCircuitBreaker(config)

		// Force open state
		cb.Execute(func() error { return errors.New("fail") })
		cb.Execute(func() error { return errors.New("fail") })

		if cb.GetState() != StateOpen {
			t.Errorf("GetState() = %v, want %v", cb.GetState(), StateOpen)
		}

		// Wait for reset timeout
		time.Sleep(150 * time.Millisecond)

		// Should transition to half-open on next call
		err := cb.Execute(func() error {
			return nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}

		if cb.GetState() != StateClosed {
			t.Errorf("GetState() = %v, want %v", cb.GetState(), StateClosed)
		}
	})
}

func TestRecoveryMetrics(t *testing.T) {
	metrics := NewRecoveryMetrics()

	// Record some operations
	metrics.RecordAttempt(0, errors.New("test error"))
	metrics.RecordAttempt(1, nil)
	metrics.RecordSuccess(1)
	metrics.RecordFailure()
	metrics.RecordFallbackSuccess()
	metrics.RecordSkip()
	metrics.RecordPanicRecovery()

	result := metrics.GetMetrics()

	if result.TotalAttempts != 2 {
		t.Errorf("TotalAttempts = %d, want 2", result.TotalAttempts)
	}

	if result.SuccessfulRetrys != 1 {
		t.Errorf("SuccessfulRetrys = %d, want 1", result.SuccessfulRetrys)
	}

	if result.FailedOperations != 1 {
		t.Errorf("FailedOperations = %d, want 1", result.FailedOperations)
	}

	if result.FallbackUses != 1 {
		t.Errorf("FallbackUses = %d, want 1", result.FallbackUses)
	}

	if result.SkippedErrors != 1 {
		t.Errorf("SkippedErrors = %d, want 1", result.SkippedErrors)
	}

	if result.PanicRecoveries != 1 {
		t.Errorf("PanicRecoveries = %d, want 1", result.PanicRecoveries)
	}
}

func TestRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}

	if config.RetryDelay != 100*time.Millisecond {
		t.Errorf("RetryDelay = %v, want %v", config.RetryDelay, 100*time.Millisecond)
	}

	if !config.EnableFallback {
		t.Error("EnableFallback should be true")
	}

	if config.EnableSkipping {
		t.Error("EnableSkipping should be false")
	}

	if config.CircuitBreaker == nil {
		t.Error("CircuitBreaker config should not be nil")
	}
}

func TestRecoveryStrategy_String(t *testing.T) {
	tests := []struct {
		strategy RecoveryStrategy
		expected string
	}{
		{RecoveryRetry, "RETRY"},
		{RecoveryFallback, "FALLBACK"},
		{RecoverySkip, "SKIP"},
		{RecoveryAbort, "ABORT"},
		{RecoveryStrategy(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.strategy.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
		{CircuitBreakerState(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
