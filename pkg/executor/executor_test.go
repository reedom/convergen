package executor

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewExecutor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	executor := NewExecutor(logger, eventBus, nil)
	assert.NotNil(t, executor)

	status := executor.GetStatus()
	assert.Equal(t, ExecutorStateIdle, status.State)
	assert.NotNil(t, status.StartTime)
}

func TestExecutor_ExecutePlan(t *testing.T) {
	tests := []struct {
		name          string
		plan          *domain.ExecutionPlan
		expectedError bool
		expectedSuccess bool
	}{
		{
			name:            "nil plan",
			plan:            nil,
			expectedError:   true,
			expectedSuccess: false,
		},
		{
			name:            "empty plan",
			plan:            createTestExecutionPlan("empty", 0, 0),
			expectedError:   false,
			expectedSuccess: true,
		},
		{
			name:            "simple plan",
			plan:            createTestExecutionPlan("simple", 2, 3),
			expectedError:   false,
			expectedSuccess: true,
		},
		{
			name:            "complex plan",
			plan:            createTestExecutionPlan("complex", 5, 10),
			expectedError:   false,
			expectedSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			eventBus := events.NewInMemoryEventBus(logger)
			defer eventBus.Close()

			config := DefaultExecutorConfig()
			config.ExecutionTimeout = 10 * time.Second
			config.EnableMetrics = true

			executor := NewExecutor(logger, eventBus, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				executor.Shutdown(ctx)
			}()

			ctx := context.Background()
			result, err := executor.ExecutePlan(ctx, tt.plan)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedSuccess, result.Success)
			
			if tt.plan != nil {
				assert.Equal(t, tt.plan.ID, result.PlanID)
				assert.NotZero(t, result.Duration)
				assert.NotNil(t, result.Metrics)
			}
		})
	}
}

func TestExecutor_ExecuteBatch(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	executor := NewExecutor(logger, eventBus, DefaultExecutorConfig())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	batch := createTestBatchExecution("test_batch", 3)
	ctx := context.Background()

	result, err := executor.ExecuteBatch(ctx, batch)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, batch.ID, result.BatchID)
	assert.NotZero(t, result.Duration)
	assert.Equal(t, len(batch.Mappings), len(result.FieldResults))
}

func TestExecutor_ExecuteField(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	executor := NewExecutor(logger, eventBus, DefaultExecutorConfig())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	field := createTestFieldExecution("test_field", "direct")
	ctx := context.Background()

	result, err := executor.ExecuteField(ctx, field)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, field.ID, result.FieldID)
	assert.NotZero(t, result.Duration)
	assert.Equal(t, field.Mapping.StrategyName, result.StrategyUsed)
}

func TestExecutor_GetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.EnableMetrics = true

	executor := NewExecutor(logger, eventBus, config)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	metrics := executor.GetMetrics()
	assert.NotNil(t, metrics)
	assert.True(t, metrics.enabled)
	assert.NotZero(t, metrics.StartTime)
}

func TestExecutor_GetStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	executor := NewExecutor(logger, eventBus, DefaultExecutorConfig())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	status := executor.GetStatus()
	assert.NotNil(t, status)
	assert.Equal(t, ExecutorStateIdle, status.State)
	assert.NotNil(t, status.ActiveBatches)
	assert.NotNil(t, status.CompletedBatches)
	assert.NotNil(t, status.QueuedBatches)
}

func TestExecutor_ConcurrentExecution(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.MaxWorkers = 8
	config.EnableMetrics = true

	executor := NewExecutor(logger, eventBus, config)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	// Execute multiple plans concurrently
	planCount := 5
	results := make(chan *ExecutionResult, planCount)
	errors := make(chan error, planCount)

	ctx := context.Background()
	for i := 0; i < planCount; i++ {
		go func(id int) {
			plan := createTestExecutionPlan(fmt.Sprintf("concurrent_%d", id), 3, 5)
			result, err := executor.ExecutePlan(ctx, plan)
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
	
	for i := 0; i < planCount; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.True(t, result.Success)
			successCount++
		case err := <-errors:
			assert.NoError(t, err)
			errorCount++
		case <-time.After(30 * time.Second):
			t.Fatal("Test timed out waiting for results")
		}
	}

	assert.Equal(t, planCount, successCount)
	assert.Equal(t, 0, errorCount)

	// Verify metrics
	metrics := executor.GetMetrics()
	assert.GreaterOrEqual(t, metrics.PlansExecuted, int64(planCount))
	assert.GreaterOrEqual(t, metrics.PlansSuccessful, int64(planCount))
}

func TestExecutor_ResourceLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.MaxWorkers = 2          // Low worker limit
	config.MaxMemoryMB = 64        // Low memory limit
	config.MaxConcurrentJobs = 5   // Low job limit

	executor := NewExecutor(logger, eventBus, config)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	// Create a plan that would stress resources
	plan := createTestExecutionPlan("resource_test", 10, 20)
	ctx := context.Background()

	result, err := executor.ExecutePlan(ctx, plan)

	// Should complete successfully despite resource limits
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Verify that resource limits were respected
	status := executor.GetStatus()
	assert.LessOrEqual(t, len(status.ActiveBatches), config.MaxConcurrentJobs)
}

func TestExecutor_Shutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	executor := NewExecutor(logger, eventBus, DefaultExecutorConfig())

	// Start a long-running plan
	plan := createTestExecutionPlan("shutdown_test", 3, 5)
	ctx := context.Background()

	go func() {
		executor.ExecutePlan(ctx, plan)
	}()

	// Allow some execution time
	time.Sleep(100 * time.Millisecond)

	// Shutdown should complete gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := executor.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func BenchmarkExecutor_ExecutePlan(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultExecutorConfig()
	config.EnableMetrics = false // Disable for accurate benchmarking

	executor := NewExecutor(logger, eventBus, config)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		executor.Shutdown(ctx)
	}()

	plan := createTestExecutionPlan("benchmark", 5, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecutePlan(ctx, plan)
		require.NoError(b, err)
	}
}

// Helper functions for creating test data

func createTestExecutionPlan(id string, methodCount, fieldsPerMethod int) *domain.ExecutionPlan {
	plan := &domain.ExecutionPlan{
		ID:      id,
		Methods: make(map[string]*domain.MethodPlan),
		GlobalLimits: &domain.ResourceLimits{
			MaxWorkers:  8,
			MaxMemoryMB: 256,
		},
		Metrics: &domain.PlanMetrics{
			PlanningDurationMS:    100,
			MethodsPlanned:        methodCount,
			TotalFields:           methodCount * fieldsPerMethod,
			ConcurrentBatches:     methodCount,
			ParallelizationRatio:  0.8,
			EstimatedSpeedupRatio: 2.5,
		},
	}

	for i := 0; i < methodCount; i++ {
		methodName := fmt.Sprintf("Method%d", i)
		methodPlan := &domain.MethodPlan{
			MethodName:          methodName,
			TotalFields:         fieldsPerMethod,
			Batches:             createTestMethodBatches(fieldsPerMethod),
			EstimatedDurationMS: int64((i + 1) * 100),
			RequiredWorkers:     2,
			MemoryRequirementMB: 32,
			Strategy:            domain.MethodStrategyDirect,
		}
		plan.Methods[methodName] = methodPlan
	}

	return plan
}

func createTestMethodBatches(fieldCount int) []*domain.ConcurrentBatch {
	batchSize := 5
	batchCount := (fieldCount + batchSize - 1) / batchSize

	batches := make([]*domain.ConcurrentBatch, batchCount)
	for i := 0; i < batchCount; i++ {
		startIdx := i * batchSize
		endIdx := min(startIdx+batchSize, fieldCount)
		
		mappings := make([]*domain.FieldMapping, endIdx-startIdx)
		for j := startIdx; j < endIdx; j++ {
			mappings[j-startIdx] = createTestFieldMapping(fmt.Sprintf("field_%d", j))
		}

		batches[i] = &domain.ConcurrentBatch{
			ID:     fmt.Sprintf("batch_%d", i),
			Fields: mappings,
		}
	}

	return batches
}

func createTestBatchExecution(id string, fieldCount int) *BatchExecution {
	mappings := make([]*domain.FieldMapping, fieldCount)
	for i := 0; i < fieldCount; i++ {
		mappings[i] = createTestFieldMapping(fmt.Sprintf("field_%d", i))
	}

	return &BatchExecution{
		ID:            id,
		Mappings:      mappings,
		MethodName:    "TestMethod",
		BatchIndex:    0,
		DependsOn:     []string{},
		Configuration: DefaultExecutorConfig(),
		StartTime:     time.Now(),
		Context:       make(map[string]interface{}),
	}
}

func createTestFieldExecution(id, strategy string) *FieldExecution {
	mapping := createTestFieldMapping(id)
	mapping.StrategyName = strategy

	return &FieldExecution{
		ID:            id,
		Mapping:       mapping,
		BatchID:       "test_batch",
		MethodName:    "TestMethod",
		Configuration: DefaultExecutorConfig(),
		Context:       make(map[string]interface{}),
		StartTime:     time.Now(),
		Timeout:       5 * time.Second,
	}
}

func createTestFieldMapping(id string) *domain.FieldMapping {
	sourceField := &domain.Field{
		Name:     "SourceField",
		Type:     createTestType("string"),
		Exported: true,
		Position: 0,
	}

	destField := &domain.Field{
		Name:     "DestField",
		Type:     createTestType("string"),
		Exported: true,
		Position: 0,
	}

	sourceSpec, _ := domain.NewFieldSpec([]string{sourceField.Name}, sourceField.Type)
	destSpec, _ := domain.NewFieldSpec([]string{destField.Name}, destField.Type)

	strategy := &domain.DirectAssignmentStrategy{}
	mapping, _ := domain.NewFieldMapping(id, sourceSpec, destSpec, strategy)
	
	return mapping
}

func createTestType(name string) domain.Type {
	return domain.NewBasicType(name, reflect.String)
}

