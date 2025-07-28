package parser

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RecoveryStrategy defines different approaches to error recovery
type RecoveryStrategy int

const (
	RecoveryRetry RecoveryStrategy = iota
	RecoveryFallback
	RecoverySkip
	RecoveryAbort
)

// String returns the string representation of RecoveryStrategy
func (rs RecoveryStrategy) String() string {
	switch rs {
	case RecoveryRetry:
		return "RETRY"
	case RecoveryFallback:
		return "FALLBACK"
	case RecoverySkip:
		return "SKIP"
	case RecoveryAbort:
		return "ABORT"
	default:
		return "UNKNOWN"
	}
}

// RecoveryConfig contains configuration for error recovery
type RecoveryConfig struct {
	MaxRetries      int
	RetryDelay      time.Duration
	EnableFallback  bool
	EnableSkipping  bool
	TimeoutDuration time.Duration
	CircuitBreaker  *CircuitBreakerConfig
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	MaxRequests      int
}

// DefaultRecoveryConfig returns a sensible default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
		EnableFallback:  true,
		EnableSkipping:  false,
		TimeoutDuration: 30 * time.Second,
		CircuitBreaker: &CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     60 * time.Second,
			MaxRequests:      10,
		},
	}
}

// RecoveryManager manages error recovery operations
type RecoveryManager struct {
	config         *RecoveryConfig
	errorHandler   *ErrorHandler
	circuitBreaker *CircuitBreaker
	metrics        *RecoveryMetrics
	mutex          sync.RWMutex
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(config *RecoveryConfig, errorHandler *ErrorHandler) *RecoveryManager {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	return &RecoveryManager{
		config:         config,
		errorHandler:   errorHandler,
		circuitBreaker: NewCircuitBreaker(config.CircuitBreaker),
		metrics:        NewRecoveryMetrics(),
	}
}

// ExecuteWithRecovery executes an operation with comprehensive error recovery
func (rm *RecoveryManager) ExecuteWithRecovery(ctx context.Context, operation func() error, options ...RecoveryOption) error {
	// Apply options
	opts := &RecoveryOptions{
		MaxRetries:     rm.config.MaxRetries,
		RetryDelay:     rm.config.RetryDelay,
		EnableFallback: rm.config.EnableFallback,
		EnableSkipping: rm.config.EnableSkipping,
		Timeout:        rm.config.TimeoutDuration,
	}

	for _, option := range options {
		option(opts)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Execute with circuit breaker protection
	return rm.circuitBreaker.Execute(func() error {
		return rm.executeWithRetry(timeoutCtx, operation, opts)
	})
}

// executeWithRetry handles retry logic
func (rm *RecoveryManager) executeWithRetry(ctx context.Context, operation func() error, opts *RecoveryOptions) error {
	var lastError error

	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return rm.errorHandler.WrapWithContext(ctx, ctx.Err(), "operation timeout")
		default:
		}

		// Execute operation with panic recovery
		err := rm.executeWithPanicRecovery(operation)

		if err == nil {
			rm.metrics.RecordSuccess(attempt)
			return nil
		}

		lastError = err
		rm.metrics.RecordAttempt(attempt, err)

		// Check if we should retry (don't retry certain critical errors)
		if attempt < opts.MaxRetries && rm.shouldRetry(err, attempt) {
			// Wait before retry with exponential backoff
			delay := rm.calculateRetryDelay(attempt, opts.RetryDelay)

			select {
			case <-ctx.Done():
				return rm.errorHandler.WrapWithContext(ctx, ctx.Err(), "operation cancelled during retry")
			case <-time.After(delay):
				// Continue to next attempt
			}
			continue
		}

		// Try fallback if enabled and available
		if opts.EnableFallback && opts.FallbackFunc != nil {
			fallbackErr := rm.executeWithPanicRecovery(opts.FallbackFunc)
			if fallbackErr == nil {
				rm.metrics.RecordFallbackSuccess()
				return nil
			}
			rm.metrics.RecordFallbackFailure()
		}

		// Skip if enabled
		if opts.EnableSkipping && rm.canSkip(err) {
			rm.metrics.RecordSkip()
			return nil // Successfully skipped
		}

		break
	}

	rm.metrics.RecordFailure()
	return rm.errorHandler.Wrap(lastError, ErrorOptions{
		Message:  "operation failed after all recovery attempts",
		Code:     "RECOVERY_EXHAUSTED",
		Category: CategoryGeneral,
		Severity: SeverityError,
		Metadata: map[string]interface{}{
			"attempts":    opts.MaxRetries + 1,
			"final_error": lastError.Error(),
		},
	})
}

// executeWithPanicRecovery executes a function with panic recovery
func (rm *RecoveryManager) executeWithPanicRecovery(operation func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = rm.errorHandler.RecoverFromPanic(recovered)
		}
	}()

	return operation()
}

// shouldRetry determines if an error is retryable
func (rm *RecoveryManager) shouldRetry(err error, attempt int) bool {
	// Don't retry panic errors
	if contextual, ok := err.(*ContextualError); ok {
		if contextual.Code == "PANIC_RECOVERED" {
			return false
		}
	}
	return rm.errorHandler.ShouldRetry(err, attempt, rm.config.MaxRetries)
}

// canSkip determines if an error can be safely skipped
func (rm *RecoveryManager) canSkip(err error) bool {
	if contextual, ok := err.(*ContextualError); ok {
		return contextual.Severity == SeverityWarning ||
			contextual.Category == CategoryValidation
	}
	return false
}

// calculateRetryDelay calculates the delay before retry with exponential backoff
func (rm *RecoveryManager) calculateRetryDelay(attempt int, baseDelay time.Duration) time.Duration {
	return rm.errorHandler.GetRetryDelay(attempt)
}

// RecoveryOptions contains options for recovery execution
type RecoveryOptions struct {
	MaxRetries     int
	RetryDelay     time.Duration
	EnableFallback bool
	EnableSkipping bool
	Timeout        time.Duration
	FallbackFunc   func() error
}

// RecoveryOption is a function that modifies RecoveryOptions
type RecoveryOption func(*RecoveryOptions)

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) RecoveryOption {
	return func(opts *RecoveryOptions) {
		opts.MaxRetries = maxRetries
	}
}

// WithRetryDelay sets the base retry delay
func WithRetryDelay(delay time.Duration) RecoveryOption {
	return func(opts *RecoveryOptions) {
		opts.RetryDelay = delay
	}
}

// WithFallback enables fallback with the specified fallback function
func WithFallback(fallbackFunc func() error) RecoveryOption {
	return func(opts *RecoveryOptions) {
		opts.EnableFallback = true
		opts.FallbackFunc = fallbackFunc
	}
}

// WithSkipping enables skipping of non-critical errors
func WithSkipping() RecoveryOption {
	return func(opts *RecoveryOptions) {
		opts.EnableSkipping = true
	}
}

// WithRecoveryTimeout sets the operation timeout
func WithRecoveryTimeout(timeout time.Duration) RecoveryOption {
	return func(opts *RecoveryOptions) {
		opts.Timeout = timeout
	}
}

// CircuitBreaker implements circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	config       *CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	requests     int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of CircuitBreakerState
func (cbs CircuitBreakerState) String() string {
	switch cbs {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(operation func() error) error {
	cb.mutex.Lock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			cb.requests = 0
		} else {
			cb.mutex.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}

	if cb.state == StateHalfOpen && cb.requests >= cb.config.MaxRequests {
		cb.mutex.Unlock()
		return fmt.Errorf("circuit breaker half-open request limit exceeded")
	}

	cb.requests++
	cb.mutex.Unlock()

	err := operation()

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}

		return err
	}

	// Success
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failures = 0
	}

	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// RecoveryMetrics tracks recovery-related metrics
type RecoveryMetrics struct {
	TotalAttempts    int64
	SuccessfulRetrys int64
	FailedOperations int64
	FallbackUses     int64
	SkippedErrors    int64
	PanicRecoveries  int64
	mutex            sync.RWMutex
}

// NewRecoveryMetrics creates new recovery metrics
func NewRecoveryMetrics() *RecoveryMetrics {
	return &RecoveryMetrics{}
}

// RecordAttempt records an operation attempt
func (rm *RecoveryMetrics) RecordAttempt(attempt int, err error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.TotalAttempts++
}

// RecordSuccess records a successful operation
func (rm *RecoveryMetrics) RecordSuccess(attempt int) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if attempt > 0 {
		rm.SuccessfulRetrys++
	}
}

// RecordFailure records a failed operation
func (rm *RecoveryMetrics) RecordFailure() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.FailedOperations++
}

// RecordFallbackSuccess records a successful fallback
func (rm *RecoveryMetrics) RecordFallbackSuccess() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.FallbackUses++
}

// RecordFallbackFailure records a failed fallback
func (rm *RecoveryMetrics) RecordFallbackFailure() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	// Could add separate metric for fallback failures if needed
}

// RecordSkip records a skipped error
func (rm *RecoveryMetrics) RecordSkip() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.SkippedErrors++
}

// RecordPanicRecovery records a panic recovery
func (rm *RecoveryMetrics) RecordPanicRecovery() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.PanicRecoveries++
}

// GetMetrics returns a copy of the current metrics
func (rm *RecoveryMetrics) GetMetrics() RecoveryMetrics {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return *rm
}
