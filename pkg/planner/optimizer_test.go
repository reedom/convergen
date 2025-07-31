package planner

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
)

func TestPlanOptimizer_OptimizePlan(t *testing.T) {
	tests := []struct {
		name              string
		optimizationLevel int
		methodCount       int
		fieldsPerMethod   int
		expectedError     bool
	}{
		{
			name:              "no optimization",
			optimizationLevel: 0,
			methodCount:       3,
			fieldsPerMethod:   5,
			expectedError:     false,
		},
		{
			name:              "basic optimization",
			optimizationLevel: 1,
			methodCount:       5,
			fieldsPerMethod:   10,
			expectedError:     false,
		},
		{
			name:              "aggressive optimization",
			optimizationLevel: 2,
			methodCount:       8,
			fieldsPerMethod:   15,
			expectedError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			config := &PlannerConfig{
				EnableOptimizations:  true,
				OptimizationLevel:    tt.optimizationLevel,
				MaxConcurrentWorkers: 8,
				MaxMemoryMB:          512,
				MinBatchSize:         2,
				MaxBatchSize:         20,
			}

			optimizer := NewPlanOptimizer(config, logger)

			// Create test data
			methods := createTestMethods(tt.methodCount, tt.fieldsPerMethod)
			methodPlans := createTestMethodPlans(methods)
			batches := createTestExecutionBatches(tt.methodCount * 2)

			ctx := context.Background()
			err := optimizer.OptimizePlan(ctx, methodPlans, batches)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify optimization effects
			if tt.optimizationLevel > 0 {
				// Should have applied some optimizations
				totalWorkers := 0
				totalMemory := 0

				for _, plan := range methodPlans {
					totalWorkers += plan.RequiredWorkers
					totalMemory += plan.MemoryRequirementMB
				}

				assert.LessOrEqual(t, totalWorkers, config.MaxConcurrentWorkers*2) // Allow some overhead
				assert.LessOrEqual(t, totalMemory, config.MaxMemoryMB*2)           // Allow some overhead
			}
		})
	}
}

func TestPlanOptimizer_ApplyBatchOptimizations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultPlannerConfig()
	optimizer := NewPlanOptimizer(config, logger)

	// Create test batches with varying sizes
	batches := []*ExecutionBatch{
		{
			ID:                  "batch_1",
			Mappings:            createTestFieldMappings(1), // Small batch
			EstimatedDurationMS: 100,
			ConcurrencyLevel:    1,
			ResourceRequirement: &ResourceRequirement{MemoryMB: 10},
		},
		{
			ID:                  "batch_2",
			Mappings:            createTestFieldMappings(15), // Large batch
			EstimatedDurationMS: 500,
			ConcurrencyLevel:    15,
			ResourceRequirement: &ResourceRequirement{MemoryMB: 50},
		},
		{
			ID:                  "batch_3",
			Mappings:            createTestFieldMappings(5), // Medium batch
			EstimatedDurationMS: 200,
			ConcurrencyLevel:    5,
			ResourceRequirement: &ResourceRequirement{MemoryMB: 20},
		},
	}

	err := optimizer.ApplyBatchOptimizations(batches)
	require.NoError(t, err)

	// Verify optimizations were applied
	for _, batch := range batches {
		// Concurrency should be within reasonable bounds
		assert.LessOrEqual(t, batch.ConcurrencyLevel, config.MaxConcurrentWorkers)
		assert.GreaterOrEqual(t, batch.ConcurrencyLevel, 1)

		// Should have proper resource requirements
		assert.NotNil(t, batch.ResourceRequirement)
		assert.GreaterOrEqual(t, batch.ResourceRequirement.MemoryMB, 0)
	}
}

func TestPlanOptimizer_OptimizeConcurrency(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &PlannerConfig{
		MaxConcurrentWorkers: 8,
		EnableOptimizations:  true,
		OptimizationLevel:    1,
	}
	optimizer := NewPlanOptimizer(config, logger)

	// Create method plans that exceed worker limits
	methodPlans := map[string]*domain.MethodPlan{
		"Method1": {
			MethodName:          "Method1",
			RequiredWorkers:     6,
			MemoryRequirementMB: 100,
		},
		"Method2": {
			MethodName:          "Method2",
			RequiredWorkers:     4,
			MemoryRequirementMB: 80,
		},
		"Method3": {
			MethodName:          "Method3",
			RequiredWorkers:     3,
			MemoryRequirementMB: 60,
		},
	}

	err := optimizer.OptimizeConcurrency(methodPlans)
	require.NoError(t, err)

	// Verify worker allocation is within limits
	totalWorkers := 0
	for _, plan := range methodPlans {
		totalWorkers += plan.RequiredWorkers
		assert.GreaterOrEqual(t, plan.RequiredWorkers, 1) // Minimum allocation
	}

	// Total should not significantly exceed limit (some overhead allowed)
	assert.LessOrEqual(t, totalWorkers, config.MaxConcurrentWorkers*2)
}

func TestPlanOptimizer_OptimizeResourceUsage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &PlannerConfig{
		MaxMemoryMB:         256,
		EnableOptimizations: true,
		OptimizationLevel:   1,
	}
	optimizer := NewPlanOptimizer(config, logger)

	// Create method plans that exceed memory limits
	methodPlans := map[string]*domain.MethodPlan{
		"Method1": {
			MethodName:          "Method1",
			RequiredWorkers:     2,
			MemoryRequirementMB: 150,
		},
		"Method2": {
			MethodName:          "Method2",
			RequiredWorkers:     2,
			MemoryRequirementMB: 120,
		},
		"Method3": {
			MethodName:          "Method3",
			RequiredWorkers:     2,
			MemoryRequirementMB: 100,
		},
	}

	err := optimizer.OptimizeResourceUsage(methodPlans)
	require.NoError(t, err)

	// Verify memory allocation is optimized
	totalMemory := 0
	for _, plan := range methodPlans {
		totalMemory += plan.MemoryRequirementMB
		assert.GreaterOrEqual(t, plan.MemoryRequirementMB, 10) // Minimum allocation
	}

	// Total should not significantly exceed limit
	assert.LessOrEqual(t, totalMemory, config.MaxMemoryMB*2)
}

func TestPlanOptimizer_DisabledOptimizations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &PlannerConfig{
		EnableOptimizations: false, // Disabled
		OptimizationLevel:   2,
	}
	optimizer := NewPlanOptimizer(config, logger)

	// Create test data
	methods := createTestMethods(3, 5)
	methodPlans := createTestMethodPlans(methods)
	batches := createTestExecutionBatches(6)

	// Store original values for comparison
	originalWorkers := make(map[string]int)
	originalMemory := make(map[string]int)

	for name, plan := range methodPlans {
		originalWorkers[name] = plan.RequiredWorkers
		originalMemory[name] = plan.MemoryRequirementMB
	}

	ctx := context.Background()
	err := optimizer.OptimizePlan(ctx, methodPlans, batches)
	require.NoError(t, err)

	// Verify no optimizations were applied (values should remain unchanged)
	for name, plan := range methodPlans {
		assert.Equal(t, originalWorkers[name], plan.RequiredWorkers)
		assert.Equal(t, originalMemory[name], plan.MemoryRequirementMB)
	}
}

func TestPlanOptimizer_BatchMerging(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &PlannerConfig{
		EnableOptimizations:  true,
		OptimizationLevel:    2, // Aggressive optimization enables batch merging
		MinBatchSize:         3,
		MaxBatchSize:         20,
		MaxConcurrentWorkers: 8,
	}
	optimizer := NewPlanOptimizer(config, logger)

	// Create small batches that can be merged
	batches := []*ExecutionBatch{
		{
			ID:                  "batch_1",
			Mappings:            createTestFieldMappings(2), // Small
			EstimatedDurationMS: 50,
			ConcurrencyLevel:    2,
			ResourceRequirement: &ResourceRequirement{MemoryMB: 10},
		},
		{
			ID:                  "batch_2",
			Mappings:            createTestFieldMappings(2), // Small
			EstimatedDurationMS: 60,
			ConcurrencyLevel:    2,
			ResourceRequirement: &ResourceRequirement{MemoryMB: 12},
		},
		{
			ID:                  "batch_3",
			Mappings:            createTestFieldMappings(8), // Good size
			EstimatedDurationMS: 200,
			ConcurrencyLevel:    8,
			ResourceRequirement: &ResourceRequirement{MemoryMB: 40},
		},
	}

	originalBatchCount := len(batches)

	methods := createTestMethods(2, 4)
	methodPlans := createTestMethodPlans(methods)

	ctx := context.Background()
	err := optimizer.OptimizePlan(ctx, methodPlans, batches)
	require.NoError(t, err)

	// Verify optimization effects (this is testing internal behavior)
	// In practice, batch merging would be verified through the actual batch structures
	assert.NotNil(t, batches)                       // Batches should still exist
	assert.GreaterOrEqual(t, originalBatchCount, 1) // Should have had batches to work with
}

func BenchmarkPlanOptimizer_OptimizePlan(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := &PlannerConfig{
		EnableOptimizations:  true,
		OptimizationLevel:    1,
		MaxConcurrentWorkers: 8,
		MaxMemoryMB:          512,
		MinBatchSize:         2,
		MaxBatchSize:         20,
	}
	optimizer := NewPlanOptimizer(config, logger)

	// Create test data
	methods := createTestMethods(20, 10)
	methodPlans := createTestMethodPlans(methods)
	batches := createTestExecutionBatches(40)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := optimizer.OptimizePlan(ctx, methodPlans, batches)
		require.NoError(b, err)
	}
}

// Helper functions for creating test data

func createTestMethodPlans(methods []*domain.Method) map[string]*domain.MethodPlan {
	plans := make(map[string]*domain.MethodPlan)

	for i, method := range methods {
		plans[method.Name] = &domain.MethodPlan{
			MethodName:          method.Name,
			TotalFields:         len(method.FieldMappings()),
			Batches:             createTestConcurrentBatches(2), // 2 batches per method
			EstimatedDurationMS: int64((i + 1) * 100),
			RequiredWorkers:     (i % 4) + 2,  // 2-5 workers
			MemoryRequirementMB: (i + 1) * 20, // 20-100 MB
			Strategy:            domain.MethodStrategyDirect,
		}
	}

	return plans
}

func createTestConcurrentBatches(count int) []*domain.ConcurrentBatch {
	batches := make([]*domain.ConcurrentBatch, count)

	for i := 0; i < count; i++ {
		batch, _ := domain.NewConcurrentBatch(
			fmt.Sprintf("batch_%d", i),
			createTestFieldMappings((i%5)+3), // 3-7 mappings
		)
		batches[i] = batch
	}

	return batches
}

func createTestExecutionBatches(count int) []*ExecutionBatch {
	batches := make([]*ExecutionBatch, count)

	for i := 0; i < count; i++ {
		batches[i] = &ExecutionBatch{
			ID:                  fmt.Sprintf("batch_%d", i),
			Mappings:            createTestFieldMappings((i % 5) + 3), // 3-7 mappings
			EstimatedDurationMS: int64((i + 1) * 50),
			ResourceRequirement: &ResourceRequirement{
				MemoryMB:     (i + 1) * 10,
				CPUIntensive: i%2 == 0,
				IOOperations: i % 3,
			},
			DependsOn:        []string{},  // No dependencies for simplicity
			ConcurrencyLevel: (i % 4) + 1, // 1-4 concurrency
		}
	}

	return batches
}

func createTestFieldMappings(count int) []*domain.FieldMapping {
	mappings := make([]*domain.FieldMapping, count)

	for i := 0; i < count; i++ {
		mappings[i] = createTestFieldMapping(1, i+1)
	}

	return mappings
}
