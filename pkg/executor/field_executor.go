package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrFieldExecutionNil        = errors.New("field execution cannot be nil")
	ErrStrategyNotFound         = errors.New("strategy not found")
	ErrDirectAssignmentTypeReqs = errors.New("direct assignment requires valid source and destination types")
)

// FieldExecutor handles the execution of individual field mappings.
type FieldExecutor interface {
	// ExecuteField executes a single field mapping
	ExecuteField(ctx context.Context, field *FieldExecution) (*FieldResult, error)

	// ExecuteFieldWithRetry executes a field mapping with retry logic
	ExecuteFieldWithRetry(ctx context.Context, field *FieldExecution) (*FieldResult, error)

	// GetMetrics returns current field execution metrics
	GetMetrics() *FieldMetrics

	// Shutdown gracefully shuts down the field executor
	Shutdown(ctx context.Context) error
}

// ConcreteFieldExecutor implements FieldExecutor.
type ConcreteFieldExecutor struct {
	config   *Config
	logger   *zap.Logger
	eventBus events.EventBus
	metrics  *ExecutionMetrics

	// Strategy registry for different field mapping strategies
	strategies map[string]FieldMappingStrategy

	// Cache for reusable computations
	cache map[string]interface{}

	// State management
	shutdown chan struct{}
}

// FieldMappingStrategy defines how different types of field mappings are executed.
type FieldMappingStrategy interface {
	// Execute performs the actual field mapping transformation
	Execute(ctx context.Context, mapping *domain.FieldMapping, context *ExecutionContext) (interface{}, error)

	// EstimateComplexity returns a complexity estimate for the mapping
	EstimateComplexity(mapping *domain.FieldMapping) int

	// GetRequiredResources returns resource requirements for the mapping
	GetRequiredResources(mapping *domain.FieldMapping) *ResourceRequirement

	// Validate checks if the strategy can handle the given mapping
	Validate(mapping *domain.FieldMapping) error
}

// ResourceRequirement defines the resources needed for field execution.
type ResourceRequirement struct {
	MemoryMB     int  `json:"memory_mb"`
	CPUIntensive bool `json:"cpu_intensive"`
	IOOperations int  `json:"io_operations"`
}

// ExecutionContext provides context and utilities for field execution.
type ExecutionContext struct {
	FieldID       string                 `json:"field_id"`
	BatchID       string                 `json:"batch_id"`
	MethodName    string                 `json:"method_name"`
	StartTime     time.Time              `json:"start_time"`
	Timeout       time.Duration          `json:"timeout"`
	Data          map[string]interface{} `json:"data"`
	SourceValue   interface{}            `json:"source_value"`
	TargetType    domain.Type            `json:"target_type"`
	Configuration *Config                `json:"configuration"`
	Logger        *zap.Logger            `json:"-"`
	Cache         map[string]interface{} `json:"-"`
}

// NewFieldExecutor creates a new field executor.
func NewFieldExecutor(config *Config, logger *zap.Logger, eventBus events.EventBus, metrics *ExecutionMetrics) FieldExecutor {
	executor := &ConcreteFieldExecutor{
		config:     config,
		logger:     logger,
		eventBus:   eventBus,
		metrics:    metrics,
		strategies: make(map[string]FieldMappingStrategy),
		cache:      make(map[string]interface{}),
		shutdown:   make(chan struct{}),
	}

	// Register default strategies
	executor.registerDefaultStrategies()

	return executor
}

// ExecuteField executes a single field mapping with comprehensive error handling.
func (fe *ConcreteFieldExecutor) ExecuteField(ctx context.Context, field *FieldExecution) (*FieldResult, error) {
	if field == nil {
		return nil, ErrFieldExecutionNil
	}

	fe.logger.Debug("executing field",
		zap.String("field_id", field.ID),
		zap.String("batch_id", field.BatchID),
		zap.String("strategy", field.Mapping.StrategyName))

	startTime := time.Now()
	fieldMetrics := &FieldMetrics{
		ExecutionTime: 0,
	}

	result := &FieldResult{
		FieldID:      field.ID,
		StartTime:    startTime,
		Metrics:      fieldMetrics,
		StrategyUsed: field.Mapping.StrategyName,
	}

	// Create field-specific context with timeout
	fieldCtx, cancel := context.WithTimeout(ctx, field.Timeout)
	defer cancel()

	// Emit field started event
	if err := fe.emitFieldEvent(ctx, EventFieldStarted, field, nil); err != nil {
		fe.logger.Warn("failed to emit field started event", zap.Error(err))
	}

	// Create execution context
	execContext := &ExecutionContext{
		FieldID:       field.ID,
		BatchID:       field.BatchID,
		MethodName:    field.MethodName,
		StartTime:     startTime,
		Timeout:       field.Timeout,
		Data:          field.Context,
		TargetType:    field.Mapping.Dest.Type,
		Configuration: field.Configuration,
		Logger:        fe.logger.With(zap.String("field_id", field.ID)),
		Cache:         fe.cache,
	}

	// Get strategy for this field mapping
	strategy, err := fe.getStrategy(field.Mapping.StrategyName)
	if err != nil {
		result.Error = &ExecutionError{
			FieldID:   field.ID,
			Error:     fmt.Sprintf("strategy not found: %v", err),
			ErrorType: "strategy_not_found",
			Timestamp: time.Now(),
			Retryable: false,
		}
		result.Success = false
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		return result, err
	}

	// Validate strategy can handle this mapping
	if err := strategy.Validate(field.Mapping); err != nil {
		result.Error = &ExecutionError{
			FieldID:   field.ID,
			Error:     fmt.Sprintf("strategy validation failed: %v", err),
			ErrorType: "strategy_validation",
			Timestamp: time.Now(),
			Retryable: false,
		}
		result.Success = false
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		return result, fmt.Errorf("field mapping strategy validation failed: %w", err)
	}

	// Record strategy time start
	strategyStart := time.Now()

	// Execute the field mapping strategy
	resultValue, err := strategy.Execute(fieldCtx, field.Mapping, execContext)

	// Record strategy time
	fieldMetrics.StrategyTime = time.Since(strategyStart)

	if err != nil {
		result.Error = &ExecutionError{
			FieldID:   field.ID,
			Error:     err.Error(),
			ErrorType: "strategy_execution",
			Timestamp: time.Now(),
			Retryable: fe.isRetryableError(err),
		}
		result.Success = false
	} else {
		result.Result = resultValue
		result.Success = true
	}

	// Finalize result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	fieldMetrics.ExecutionTime = result.Duration

	// Update global metrics
	fe.metrics.RecordFieldExecution(result)

	// Emit field completed event
	eventType := EventFieldCompleted
	if !result.Success {
		eventType = EventFieldFailed
	}

	if err := fe.emitFieldEvent(ctx, eventType, field, result); err != nil {
		fe.logger.Warn("failed to emit field completed event", zap.Error(err))
	}

	fe.logger.Debug("field execution completed",
		zap.String("field_id", field.ID),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
		zap.String("strategy", result.StrategyUsed))

	return result, nil
}

// ExecuteFieldWithRetry executes a field with retry logic for transient failures.
func (fe *ConcreteFieldExecutor) ExecuteFieldWithRetry(ctx context.Context, field *FieldExecution) (*FieldResult, error) {
	var lastResult *FieldResult

	var lastError error

	for attempt := 0; attempt <= fe.config.RetryAttempts; attempt++ {
		if 0 < attempt {
			// Calculate backoff delay
			backoff := fe.calculateBackoff(attempt)

			fe.logger.Debug("retrying field execution",
				zap.String("field_id", field.ID),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff))

			// Emit retry event
			if err := fe.emitRetryEvent(ctx, field, attempt); err != nil {
				fe.logger.Warn("failed to emit retry event", zap.Error(err))
			}

			// Wait for backoff period
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return lastResult, fmt.Errorf("field execution context cancelled: %w", ctx.Err())
			}
		}

		result, err := fe.ExecuteField(ctx, field)
		lastResult = result
		lastError = err

		if err == nil && result.Success {
			if 0 < attempt {
				result.RetryCount = attempt
			}

			return result, nil
		}

		// Check if error is retryable
		if result != nil && result.Error != nil && !result.Error.Retryable {
			break
		}

		if attempt == fe.config.RetryAttempts {
			// Emit retry exhausted event
			if err := fe.emitRetryExhaustedEvent(ctx, field, attempt); err != nil {
				fe.logger.Warn("failed to emit retry exhausted event", zap.Error(err))
			}
		}
	}

	return lastResult, lastError
}

// GetMetrics returns current field execution metrics.
func (fe *ConcreteFieldExecutor) GetMetrics() *FieldMetrics {
	// Return aggregated metrics from the global metrics store
	return &FieldMetrics{
		// This would be populated from fe.metrics
	}
}

// Shutdown gracefully shuts down the field executor.
func (fe *ConcreteFieldExecutor) Shutdown(ctx context.Context) error {
	fe.logger.Info("shutting down field executor")
	close(fe.shutdown)

	return nil
}

// Strategy registration and management

func (fe *ConcreteFieldExecutor) registerDefaultStrategies() {
	fe.strategies["direct"] = NewDirectAssignmentStrategy()
	fe.strategies["converter"] = NewConverterStrategy()
	fe.strategies["literal"] = NewLiteralStrategy()
	fe.strategies["expression"] = NewExpressionStrategy()
	fe.strategies["custom"] = NewCustomStrategy()

	fe.logger.Debug("registered field mapping strategies",
		zap.Int("strategy_count", len(fe.strategies)))
}

func (fe *ConcreteFieldExecutor) getStrategy(strategyName string) (FieldMappingStrategy, error) {
	fe.logger.Debug("looking for strategy",
		zap.String("strategy_name", strategyName),
		zap.Int("available_strategies", len(fe.strategies)))

	strategy, exists := fe.strategies[strategyName]
	if !exists {
		// Log available strategies for debugging
		availableStrategies := make([]string, 0, len(fe.strategies))
		for name := range fe.strategies {
			availableStrategies = append(availableStrategies, name)
		}

		fe.logger.Error("strategy not found",
			zap.String("requested_strategy", strategyName),
			zap.Strings("available_strategies", availableStrategies))

		return nil, fmt.Errorf("%w: '%s'", ErrStrategyNotFound, strategyName)
	}

	return strategy, nil
}

// Helper methods

func (fe *ConcreteFieldExecutor) isRetryableError(err error) bool {
	// Determine if an error is retryable based on error type
	// This is a simplified implementation
	errorStr := err.Error()

	// Common retryable error patterns
	retryablePatterns := []string{
		"timeout",
		"temporary",
		"connection",
		"network",
		"resource temporarily unavailable",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

func (fe *ConcreteFieldExecutor) calculateBackoff(attempt int) time.Duration {
	baseDelay := fe.config.RetryBackoffBase
	maxDelay := fe.config.RetryBackoffMax

	// Exponential backoff with jitter
	delay := time.Duration(int64(baseDelay) * int64(1<<(attempt-1)))

	if maxDelay < delay {
		delay = maxDelay
	}

	// Add jitter (up to 25% of delay)
	jitter := time.Duration(float64(delay) * 0.25)
	if 0 < jitter {
		delay += time.Duration(int64(jitter) * int64(attempt) / int64(fe.config.RetryAttempts))
	}

	return delay
}

func (fe *ConcreteFieldExecutor) emitFieldEvent(ctx context.Context, eventType string, field *FieldExecution, result *FieldResult) error {
	data := map[string]interface{}{
		"field_id":    field.ID,
		"batch_id":    field.BatchID,
		"method_name": field.MethodName,
		"strategy":    field.Mapping.StrategyName,
	}

	if result != nil {
		data["success"] = result.Success
		data["duration_ms"] = result.Duration.Milliseconds()
		data["retry_count"] = result.RetryCount
	}

	event := events.NewEvent(eventType, data)

	if err := fe.eventBus.Emit(ctx, event); err != nil {
		return fmt.Errorf("failed to emit field event: %w", err)
	}

	return nil
}

func (fe *ConcreteFieldExecutor) emitRetryEvent(ctx context.Context, field *FieldExecution, attempt int) error {
	data := map[string]interface{}{
		"field_id":     field.ID,
		"batch_id":     field.BatchID,
		"attempt":      attempt,
		"max_attempts": fe.config.RetryAttempts,
	}

	event := events.NewEvent(EventRetryAttempt, data)

	if err := fe.eventBus.Emit(ctx, event); err != nil {
		return fmt.Errorf("failed to emit field event: %w", err)
	}

	return nil
}

func (fe *ConcreteFieldExecutor) emitRetryExhaustedEvent(ctx context.Context, field *FieldExecution, attempts int) error {
	data := map[string]interface{}{
		"field_id": field.ID,
		"batch_id": field.BatchID,
		"attempts": attempts,
	}

	event := events.NewEvent(EventRetryExhausted, data)

	if err := fe.eventBus.Emit(ctx, event); err != nil {
		return fmt.Errorf("failed to emit field event: %w", err)
	}

	return nil
}

// Strategy implementations

// DirectAssignmentStrategy handles direct field assignments.
type DirectAssignmentStrategy struct{}

func NewDirectAssignmentStrategy() FieldMappingStrategy {
	return &DirectAssignmentStrategy{}
}

func (s *DirectAssignmentStrategy) Execute(ctx context.Context, mapping *domain.FieldMapping, execCtx *ExecutionContext) (interface{}, error) {
	// Simplified direct assignment implementation
	return execCtx.SourceValue, nil
}

func (s *DirectAssignmentStrategy) EstimateComplexity(mapping *domain.FieldMapping) int {
	return 1 // Lowest complexity
}

func (s *DirectAssignmentStrategy) GetRequiredResources(mapping *domain.FieldMapping) *ResourceRequirement {
	return &ResourceRequirement{
		MemoryMB:     1,
		CPUIntensive: false,
		IOOperations: 0,
	}
}

func (s *DirectAssignmentStrategy) Validate(mapping *domain.FieldMapping) error {
	// For now, be permissive for testing - in production this would check type compatibility
	if mapping.Source.Type == nil || mapping.Dest.Type == nil {
		return ErrDirectAssignmentTypeReqs
	}

	return nil
}

// ConverterStrategy handles conversions between different types.
type ConverterStrategy struct{}

func NewConverterStrategy() FieldMappingStrategy {
	return &ConverterStrategy{}
}

func (s *ConverterStrategy) Execute(ctx context.Context, mapping *domain.FieldMapping, execCtx *ExecutionContext) (interface{}, error) {
	// Simplified converter implementation
	// In practice, this would handle type conversions
	return execCtx.SourceValue, nil
}

func (s *ConverterStrategy) EstimateComplexity(mapping *domain.FieldMapping) int {
	return 3 // Medium complexity
}

func (s *ConverterStrategy) GetRequiredResources(mapping *domain.FieldMapping) *ResourceRequirement {
	return &ResourceRequirement{
		MemoryMB:     2,
		CPUIntensive: true,
		IOOperations: 0,
	}
}

func (s *ConverterStrategy) Validate(mapping *domain.FieldMapping) error {
	// Validation logic for converter strategy
	return nil
}

// LiteralStrategy handles literal value assignments.
type LiteralStrategy struct{}

func NewLiteralStrategy() FieldMappingStrategy {
	return &LiteralStrategy{}
}

func (s *LiteralStrategy) Execute(ctx context.Context, mapping *domain.FieldMapping, execCtx *ExecutionContext) (interface{}, error) {
	// Return the literal value specified in the mapping
	return mapping.Source.Path[0], nil // Simplified - use first path element
}

func (s *LiteralStrategy) EstimateComplexity(mapping *domain.FieldMapping) int {
	return 1 // Lowest complexity
}

func (s *LiteralStrategy) GetRequiredResources(mapping *domain.FieldMapping) *ResourceRequirement {
	return &ResourceRequirement{
		MemoryMB:     1,
		CPUIntensive: false,
		IOOperations: 0,
	}
}

func (s *LiteralStrategy) Validate(mapping *domain.FieldMapping) error {
	return nil
}

// ExpressionStrategy handles expression evaluations.
type ExpressionStrategy struct{}

func NewExpressionStrategy() FieldMappingStrategy {
	return &ExpressionStrategy{}
}

func (s *ExpressionStrategy) Execute(ctx context.Context, mapping *domain.FieldMapping, execCtx *ExecutionContext) (interface{}, error) {
	// Simplified expression evaluation
	// In practice, this would evaluate complex expressions
	return execCtx.SourceValue, nil
}

func (s *ExpressionStrategy) EstimateComplexity(mapping *domain.FieldMapping) int {
	return 5 // High complexity
}

func (s *ExpressionStrategy) GetRequiredResources(mapping *domain.FieldMapping) *ResourceRequirement {
	return &ResourceRequirement{
		MemoryMB:     3,
		CPUIntensive: true,
		IOOperations: 0,
	}
}

func (s *ExpressionStrategy) Validate(mapping *domain.FieldMapping) error {
	return nil
}

// CustomStrategy handles custom transformation logic.
type CustomStrategy struct{}

func NewCustomStrategy() FieldMappingStrategy {
	return &CustomStrategy{}
}

func (s *CustomStrategy) Execute(ctx context.Context, mapping *domain.FieldMapping, execCtx *ExecutionContext) (interface{}, error) {
	// Custom strategy implementation
	return execCtx.SourceValue, nil
}

func (s *CustomStrategy) EstimateComplexity(mapping *domain.FieldMapping) int {
	return 4 // Medium-high complexity
}

func (s *CustomStrategy) GetRequiredResources(mapping *domain.FieldMapping) *ResourceRequirement {
	return &ResourceRequirement{
		MemoryMB:     2,
		CPUIntensive: true,
		IOOperations: 1,
	}
}

func (s *CustomStrategy) Validate(mapping *domain.FieldMapping) error {
	return nil
}

// Utility function.
func contains(s, substr string) bool {
	return len(substr) <= len(s) && (substr == s || s[len(s)-len(substr):] == substr || s[:len(substr)] == substr)
}
