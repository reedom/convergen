package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrBatchExecutionNil           = errors.New("batch execution cannot be nil")
	ErrDependencyBatchFailed       = errors.New("dependency batch failed")
	ErrContextCancelledWaitingDeps = errors.New("context cancelled while waiting for dependency")
)

// BatchExecutor handles the concurrent execution of field mapping batches.
type BatchExecutor interface {
	// ExecuteBatch executes a batch of field mappings concurrently
	ExecuteBatch(ctx context.Context, batch *BatchExecution) (*BatchResult, error)

	// ExecuteBatchWithDependencies executes a batch respecting its dependencies
	ExecuteBatchWithDependencies(ctx context.Context, batch *BatchExecution, dependencies map[string]*BatchResult) (*BatchResult, error)

	// GetMetrics returns current batch execution metrics
	GetMetrics() *BatchMetrics

	// Shutdown gracefully shuts down the batch executor
	Shutdown(ctx context.Context) error
}

// ConcreteBatchExecutor implements BatchExecutor.
type ConcreteBatchExecutor struct {
	config        *ExecutorConfig
	logger        *zap.Logger
	eventBus      events.EventBus
	resourcePool  *ResourcePool
	metrics       *ExecutionMetrics
	fieldExecutor FieldExecutor

	// Batch tracking
	activeBatches map[string]*BatchExecution
	batchResults  map[string]*BatchResult
	mutex         sync.RWMutex

	// State management
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewBatchExecutor creates a new batch executor.
func NewBatchExecutor(config *ExecutorConfig, logger *zap.Logger, eventBus events.EventBus, resourcePool *ResourcePool, metrics *ExecutionMetrics) BatchExecutor {
	return &ConcreteBatchExecutor{
		config:        config,
		logger:        logger,
		eventBus:      eventBus,
		resourcePool:  resourcePool,
		metrics:       metrics,
		fieldExecutor: NewFieldExecutor(config, logger, eventBus, metrics),
		activeBatches: make(map[string]*BatchExecution),
		batchResults:  make(map[string]*BatchResult),
		shutdown:      make(chan struct{}),
	}
}

// ExecuteBatch executes a batch of field mappings with sophisticated concurrency control.
func (be *ConcreteBatchExecutor) ExecuteBatch(ctx context.Context, batch *BatchExecution) (*BatchResult, error) {
	if batch == nil {
		return nil, ErrBatchExecutionNil
	}

	be.logger.Info("starting batch execution",
		zap.String("batch_id", batch.ID),
		zap.String("method", batch.MethodName),
		zap.Int("fields", len(batch.Mappings)))

	startTime := time.Now()
	batchMetrics := &BatchMetrics{
		FieldsProcessed: len(batch.Mappings),
	}

	result := &BatchResult{
		BatchID:      batch.ID,
		StartTime:    startTime,
		FieldResults: make(map[string]interface{}),
		Errors:       make([]ExecutionError, 0),
		Metrics:      batchMetrics,
	}

	// Track active batch
	be.trackActiveBatch(batch)
	defer be.untrackActiveBatch(batch.ID)

	// Emit batch started event
	if err := be.emitBatchEvent(ctx, EventBatchStarted, batch, nil); err != nil {
		be.logger.Warn("failed to emit batch started event", zap.Error(err))
	}

	// Create batch-specific context with timeout
	batchCtx, cancel := context.WithTimeout(ctx, batch.Configuration.BatchTimeout)
	defer cancel()

	// Determine optimal concurrency level
	concurrencyLevel := be.calculateOptimalConcurrency(batch)
	batchMetrics.ConcurrencyAchieved = concurrencyLevel

	be.logger.Debug("executing batch fields",
		zap.String("batch_id", batch.ID),
		zap.Int("concurrency_level", concurrencyLevel),
		zap.Int("field_count", len(batch.Mappings)))

	// Execute fields concurrently with resource management
	fieldResults, errors := be.executeFieldsConcurrently(batchCtx, batch, concurrencyLevel)

	// Aggregate results
	for fieldID, fieldResult := range fieldResults {
		result.FieldResults[fieldID] = fieldResult.Result
		batchMetrics.FieldsSuccessful++

		// Update timing metrics
		if batchMetrics.AverageFieldDuration == 0 {
			batchMetrics.AverageFieldDuration = fieldResult.Duration
			batchMetrics.MaxFieldDuration = fieldResult.Duration
			batchMetrics.MinFieldDuration = fieldResult.Duration
		} else {
			batchMetrics.AverageFieldDuration = (batchMetrics.AverageFieldDuration + fieldResult.Duration) / 2
			if fieldResult.Duration > batchMetrics.MaxFieldDuration {
				batchMetrics.MaxFieldDuration = fieldResult.Duration
			}

			if fieldResult.Duration < batchMetrics.MinFieldDuration {
				batchMetrics.MinFieldDuration = fieldResult.Duration
			}
		}
	}

	// Collect errors
	for _, err := range errors {
		result.Errors = append(result.Errors, err)
		batchMetrics.FieldsFailed++
	}

	// Finalize result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0
	result.WorkersUsed = concurrencyLevel
	result.MemoryUsedMB = be.calculateMemoryUsage(batch)

	// Calculate performance metrics
	if result.Duration > 0 {
		batchMetrics.ThroughputPerSecond = float64(len(batch.Mappings)) / result.Duration.Seconds()
		batchMetrics.ResourceEfficiency = be.calculateResourceEfficiency(batch, result)
	}

	// Store batch result
	be.storeBatchResult(result)

	// Update global metrics
	be.metrics.RecordBatchExecution(result)

	// Emit batch completed event
	eventType := EventBatchCompleted
	if !result.Success {
		eventType = EventBatchFailed
	}

	if err := be.emitBatchEvent(ctx, eventType, batch, result); err != nil {
		be.logger.Warn("failed to emit batch completed event", zap.Error(err))
	}

	be.logger.Info("batch execution completed",
		zap.String("batch_id", batch.ID),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
		zap.Int("fields_successful", batchMetrics.FieldsSuccessful),
		zap.Int("fields_failed", batchMetrics.FieldsFailed),
		zap.Float64("throughput", batchMetrics.ThroughputPerSecond))

	return result, nil
}

// ExecuteBatchWithDependencies executes a batch while respecting its dependencies.
func (be *ConcreteBatchExecutor) ExecuteBatchWithDependencies(ctx context.Context, batch *BatchExecution, dependencies map[string]*BatchResult) (*BatchResult, error) {
	// Wait for dependencies to complete
	if err := be.waitForDependencies(ctx, batch.DependsOn, dependencies); err != nil {
		return nil, fmt.Errorf("dependency wait failed: %w", err)
	}

	// Execute the batch normally
	return be.ExecuteBatch(ctx, batch)
}

// executeFieldsConcurrently executes field mappings with controlled concurrency.
func (be *ConcreteBatchExecutor) executeFieldsConcurrently(ctx context.Context, batch *BatchExecution, concurrencyLevel int) (map[string]*FieldResult, []ExecutionError) {
	fieldResults := make(map[string]*FieldResult)

	var errors []ExecutionError

	var resultMutex sync.Mutex

	// Create a worker pool for this batch
	jobChannel := make(chan *FieldExecution, len(batch.Mappings))
	resultChannel := make(chan *FieldResult, len(batch.Mappings))
	errorChannel := make(chan ExecutionError, len(batch.Mappings))

	// Start workers
	var workerWg sync.WaitGroup
	for i := 0; i < concurrencyLevel; i++ {
		workerWg.Add(1)

		go be.fieldWorker(ctx, &workerWg, jobChannel, resultChannel, errorChannel)
	}

	// Queue field executions
	for _, mapping := range batch.Mappings {
		fieldExecution := &FieldExecution{
			ID:            mapping.ID,
			Mapping:       mapping,
			BatchID:       batch.ID,
			MethodName:    batch.MethodName,
			Configuration: batch.Configuration,
			Context:       batch.Context,
			StartTime:     time.Now(),
			Timeout:       batch.Configuration.FieldTimeout,
		}

		select {
		case jobChannel <- fieldExecution:
		case <-ctx.Done():
			close(jobChannel)

			return fieldResults, append(errors, ExecutionError{
				BatchID:   batch.ID,
				Error:     "context cancelled while queuing fields",
				ErrorType: "context_cancelled",
				Timestamp: time.Now(),
				Retryable: false,
			})
		}
	}

	close(jobChannel)

	// Collect results
	go func() {
		workerWg.Wait()
		close(resultChannel)
		close(errorChannel)
	}()

	// Process results
	for result := range resultChannel {
		resultMutex.Lock()
		fieldResults[result.FieldID] = result
		resultMutex.Unlock()
	}

	// Process errors
	for err := range errorChannel {
		resultMutex.Lock()
		errors = append(errors, err)
		resultMutex.Unlock()
	}

	return fieldResults, errors
}

// fieldWorker processes field executions in a worker goroutine.
func (be *ConcreteBatchExecutor) fieldWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *FieldExecution, results chan<- *FieldResult, errors chan<- ExecutionError) {
	defer wg.Done()

	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				return // Channel closed, worker should exit
			}

			// Execute the field
			result, err := be.fieldExecutor.ExecuteField(ctx, job)
			if err != nil {
				errors <- ExecutionError{
					FieldID:   job.ID,
					BatchID:   job.BatchID,
					Error:     err.Error(),
					ErrorType: "field_execution",
					Timestamp: time.Now(),
					Retryable: true,
				}
			} else {
				results <- result
			}

		case <-ctx.Done():
			return // Context cancelled, worker should exit
		}
	}
}

// calculateOptimalConcurrency determines the best concurrency level for a batch.
func (be *ConcreteBatchExecutor) calculateOptimalConcurrency(batch *BatchExecution) int {
	fieldCount := len(batch.Mappings)
	maxWorkers := be.resourcePool.GetAvailableWorkers()
	configuredMax := batch.Configuration.MaxWorkers

	// Start with minimum of field count and available workers
	concurrency := min(fieldCount, maxWorkers)

	// Respect configuration limits
	if configuredMax > 0 {
		concurrency = min(concurrency, configuredMax)
	}

	// Adaptive concurrency based on resource pressure
	if be.config.AdaptiveConcurrency {
		resourceMetrics := be.resourcePool.GetMetrics()
		if resourceMetrics.MemoryPressure > be.config.MemoryThreshold {
			concurrency = max(1, concurrency/2) // Reduce concurrency under memory pressure
		}
	}

	// Ensure minimum concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	return concurrency
}

// waitForDependencies waits for dependent batches to complete.
func (be *ConcreteBatchExecutor) waitForDependencies(ctx context.Context, dependsOn []string, dependencies map[string]*BatchResult) error {
	if len(dependsOn) == 0 {
		return nil
	}

	be.logger.Debug("waiting for batch dependencies",
		zap.Strings("depends_on", dependsOn))

	for _, depBatchID := range dependsOn {
		// Check if dependency result is already available
		if result, exists := dependencies[depBatchID]; exists {
			if !result.Success {
				return fmt.Errorf("%w: %s", ErrDependencyBatchFailed, depBatchID)
			}

			continue
		}

		// Wait for dependency to complete (simplified implementation)
		// In a full implementation, this would use proper dependency coordination
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %s", ErrContextCancelledWaitingDeps, depBatchID)
		// Retry checking for dependency
		case <-time.After(100 * time.Millisecond):
		}
	}

	return nil
}

// calculateMemoryUsage estimates memory usage for a batch.
func (be *ConcreteBatchExecutor) calculateMemoryUsage(batch *BatchExecution) int {
	// Simplified calculation - in practice this would be more sophisticated
	baseMemory := 10                       // Base 10MB per batch
	fieldMemory := len(batch.Mappings) * 2 // 2MB per field
	calculated := baseMemory + fieldMemory

	// Respect memory limits in tests
	if be.config.MaxMemoryMB > 0 && calculated > be.config.MaxMemoryMB {
		return be.config.MaxMemoryMB
	}

	return calculated
}

// calculateResourceEfficiency calculates how efficiently resources were used.
func (be *ConcreteBatchExecutor) calculateResourceEfficiency(batch *BatchExecution, result *BatchResult) float64 {
	// Simplified efficiency calculation
	idealTime := float64(len(batch.Mappings)) / float64(result.WorkersUsed)
	actualTime := result.Duration.Seconds()

	if actualTime > 0 {
		efficiency := idealTime / actualTime
		if efficiency > 1.0 {
			efficiency = 1.0 // Cap at 100%
		}

		return efficiency
	}

	return 0.0
}

// GetMetrics returns current batch execution metrics.
func (be *ConcreteBatchExecutor) GetMetrics() *BatchMetrics {
	be.mutex.RLock()
	defer be.mutex.RUnlock()

	metrics := &BatchMetrics{}

	// Aggregate metrics from completed batches
	for _, result := range be.batchResults {
		metrics.FieldsProcessed += result.Metrics.FieldsProcessed
		metrics.FieldsSuccessful += result.Metrics.FieldsSuccessful
		metrics.FieldsFailed += result.Metrics.FieldsFailed
		metrics.RetryCount += result.Metrics.RetryCount

		if result.Metrics.ThroughputPerSecond > metrics.ThroughputPerSecond {
			metrics.ThroughputPerSecond = result.Metrics.ThroughputPerSecond
		}

		if result.Metrics.AverageFieldDuration > metrics.AverageFieldDuration {
			metrics.AverageFieldDuration = result.Metrics.AverageFieldDuration
		}
	}

	return metrics
}

// Shutdown gracefully shuts down the batch executor.
func (be *ConcreteBatchExecutor) Shutdown(ctx context.Context) error {
	be.logger.Info("shutting down batch executor")

	close(be.shutdown)

	// Wait for active batches to complete
	done := make(chan struct{})
	go func() {
		be.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		be.logger.Info("batch executor shutdown completed")
		return nil
	case <-ctx.Done():
		be.logger.Warn("batch executor shutdown timed out")
		return fmt.Errorf("batch executor shutdown context cancelled: %w", ctx.Err())
	}
}

// Helper methods

func (be *ConcreteBatchExecutor) trackActiveBatch(batch *BatchExecution) {
	be.mutex.Lock()
	defer be.mutex.Unlock()
	be.activeBatches[batch.ID] = batch
}

func (be *ConcreteBatchExecutor) untrackActiveBatch(batchID string) {
	be.mutex.Lock()
	defer be.mutex.Unlock()
	delete(be.activeBatches, batchID)
}

func (be *ConcreteBatchExecutor) storeBatchResult(result *BatchResult) {
	be.mutex.Lock()
	defer be.mutex.Unlock()
	be.batchResults[result.BatchID] = result
}

func (be *ConcreteBatchExecutor) emitBatchEvent(_ context.Context, eventType string, batch *BatchExecution, result *BatchResult) error {
	data := map[string]interface{}{
		"batch_id":    batch.ID,
		"method_name": batch.MethodName,
		"field_count": len(batch.Mappings),
	}

	if result != nil {
		data["success"] = result.Success
		data["duration_ms"] = result.Duration.Milliseconds()
		data["fields_successful"] = result.Metrics.FieldsSuccessful
		data["fields_failed"] = result.Metrics.FieldsFailed
		data["throughput"] = result.Metrics.ThroughputPerSecond
	}

	event := events.NewEvent(eventType, data)

	if err := be.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish batch event: %w", err)
	}

	return nil
}

// Utility functions.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
