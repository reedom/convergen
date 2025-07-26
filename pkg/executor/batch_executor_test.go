package executor

import (
	"context" 
	"fmt"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewBatchExecutor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()
	
	config := DefaultExecutorConfig()
	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	assert.NotNil(t, batchExecutor)
}

func TestBatchExecutor_ExecuteBatch(t *testing.T) {
	tests := []struct {
		name          string
		fieldCount    int
		expectedError bool
	}{
		{
			name:          "empty batch",
			fieldCount:    0,
			expectedError: false,
		},
		{
			name:          "single field",
			fieldCount:    1,
			expectedError: false,
		},
		{
			name:          "small batch",
			fieldCount:    5,
			expectedError: false,
		},
		{
			name:          "medium batch",
			fieldCount:    20,
			expectedError: false,
		},
		{
			name:          "large batch",
			fieldCount:    100,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			eventBus := events.NewInMemoryEventBus(logger)
			defer eventBus.Close()

			config := DefaultExecutorConfig()
			config.BatchTimeout = 30 * time.Second
			config.EnableMetrics = true

			metrics := NewExecutionMetrics(true)
			resourcePool := NewResourcePool(config, logger, metrics)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				resourcePool.Shutdown(ctx)
			}()

			batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				batchExecutor.Shutdown(ctx)
			}()

			batch := createTestBatchExecution("test_batch", tt.fieldCount)
			ctx := context.Background()

			result, err := batchExecutor.ExecuteBatch(ctx, batch)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, batch.ID, result.BatchID)
			assert.NotZero(t, result.Duration)
			assert.Equal(t, tt.fieldCount, len(result.FieldResults))
			assert.NotNil(t, result.Metrics)
			assert.Equal(t, tt.fieldCount, result.Metrics.FieldsProcessed)
		})
	}
}

func TestBatchExecutor_ExecuteBatchWithNilBatch(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)

	ctx := context.Background()
	result, err := batchExecutor.ExecuteBatch(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "batch execution cannot be nil")
}

func TestBatchExecutor_ExecuteBatchWithDependencies(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	// Create dependency batch result
	dependencyResult := &BatchResult{
		BatchID:      "dependency_batch",
		Success:      true,
		StartTime:    time.Now().Add(-1 * time.Second),
		EndTime:      time.Now(),
		Duration:     1 * time.Second,
		FieldResults: map[string]interface{}{"dep_field": "value"},
		Metrics:      &BatchMetrics{FieldsProcessed: 1, FieldsSuccessful: 1},
	}

	dependencies := map[string]*BatchResult{
		"dependency_batch": dependencyResult,
	}

	// Create batch that depends on the dependency
	batch := createTestBatchExecution("dependent_batch", 3)
	batch.DependsOn = []string{"dependency_batch"}

	ctx := context.Background()
	result, err := batchExecutor.ExecuteBatchWithDependencies(ctx, batch, dependencies)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, batch.ID, result.BatchID)
	assert.True(t, result.Success)
}

func TestBatchExecutor_ConcurrentBatchExecution(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.MaxWorkers = 8
	config.EnableMetrics = true

	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	// Execute multiple batches concurrently
	batchCount := 5
	fieldCount := 10

	results := make(chan *BatchResult, batchCount)
	errors := make(chan error, batchCount)

	ctx := context.Background()
	for i := 0; i < batchCount; i++ {
		go func(batchID int) {
			batch := createTestBatchExecution(fmt.Sprintf("concurrent_batch_%d", batchID), fieldCount)
			result, err := batchExecutor.ExecuteBatch(ctx, batch)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < batchCount; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.True(t, result.Success)
			assert.Equal(t, fieldCount, len(result.FieldResults))
			successCount++
		case err := <-errors:
			assert.NoError(t, err)
			errorCount++
		case <-time.After(30 * time.Second):
			t.Fatal("Test timed out waiting for results")
		}
	}

	assert.Equal(t, batchCount, successCount)
	assert.Equal(t, 0, errorCount)
}

func TestBatchExecutor_GetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	// Execute a batch to generate metrics
	batch := createTestBatchExecution("metrics_test", 5)
	ctx := context.Background()
	
	_, err := batchExecutor.ExecuteBatch(ctx, batch)
	require.NoError(t, err)

	// Get metrics
	batchMetrics := batchExecutor.GetMetrics()
	assert.NotNil(t, batchMetrics)
	assert.GreaterOrEqual(t, batchMetrics.FieldsProcessed, 5)
}

func TestBatchExecutor_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.BatchTimeout = 1 * time.Minute // Long timeout
	
	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	batch := createTestBatchExecution("cancellation_test", 50) // Large batch
	
	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := batchExecutor.ExecuteBatch(ctx, batch)

	// Should complete or be cancelled gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	} else {
		assert.NotNil(t, result)
	}
}

func TestBatchExecutor_ResourceLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.MaxWorkers = 2          // Limited workers
	config.MaxMemoryMB = 64        // Limited memory
	config.EnableMetrics = true

	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	// Create large batch that would stress resource limits
	batch := createTestBatchExecution("resource_test", 50)
	ctx := context.Background()

	result, err := batchExecutor.ExecuteBatch(ctx, batch)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, result.WorkersUsed, config.MaxWorkers)
	assert.LessOrEqual(t, result.MemoryUsedMB, config.MaxMemoryMB)
}

func TestBatchExecutor_EmptyBatch(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	metrics := NewExecutionMetrics(true)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	batch := createTestBatchExecution("empty_batch", 0)
	ctx := context.Background()

	result, err := batchExecutor.ExecuteBatch(ctx, batch)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, batch.ID, result.BatchID)
	assert.True(t, result.Success)
	assert.Empty(t, result.FieldResults)
	assert.Empty(t, result.Errors)
}

func BenchmarkBatchExecutor_ExecuteBatch(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.EnableMetrics = false // Disable for accurate benchmarking

	metrics := NewExecutionMetrics(false)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	batch := createTestBatchExecution("benchmark_batch", 20)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := batchExecutor.ExecuteBatch(ctx, batch)
		require.NoError(b, err)
	}
}

func BenchmarkBatchExecutor_ConcurrentBatches(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.MaxWorkers = 8
	config.EnableMetrics = false

	metrics := NewExecutionMetrics(false)
	resourcePool := NewResourcePool(config, logger, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resourcePool.Shutdown(ctx)
	}()

	batchExecutor := NewBatchExecutor(config, logger, eventBus, resourcePool, metrics)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		batchExecutor.Shutdown(ctx)
	}()

	batches := make([]*BatchExecution, 4)
	for i := 0; i < 4; i++ {
		batches[i] = createTestBatchExecution(fmt.Sprintf("bench_batch_%d", i), 10)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Execute batches concurrently
		results := make(chan *BatchResult, len(batches))
		errors := make(chan error, len(batches))

		for _, batch := range batches {
			go func(b *BatchExecution) {
				result, err := batchExecutor.ExecuteBatch(ctx, b)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(batch)
		}

		// Wait for all to complete
		for j := 0; j < len(batches); j++ {
			select {
			case <-results:
			case err := <-errors:
				require.NoError(b, err)
			}
		}
	}
}