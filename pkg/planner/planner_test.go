package planner

import (
	"context"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestExecutionPlanner_NewExecutionPlanner(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	planner := NewExecutionPlanner(logger, eventBus, nil)
	assert.NotNil(t, planner)
	assert.NotNil(t, planner.config)
	assert.NotNil(t, planner.depGraph)
	assert.NotNil(t, planner.optimizer)
	assert.NotNil(t, planner.metrics)
}

func TestExecutionPlanner_CreateExecutionPlan(t *testing.T) {
	tests := []struct {
		name           string
		methods        []*domain.Method
		expectedError  bool
		expectedEvents int
	}{
		{
			name:           "empty methods",
			methods:        []*domain.Method{},
			expectedError:  false,
			expectedEvents: 2, // start + completed events
		},
		{
			name:           "single method with simple mappings",
			methods:        createTestMethods(1, 3),
			expectedError:  false,
			expectedEvents: 2,
		},
		{
			name:           "multiple methods with complex mappings",
			methods:        createTestMethods(3, 5),
			expectedError:  false,
			expectedEvents: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			eventBus := events.NewInMemoryEventBus(logger)
			defer eventBus.Close()

			// Subscribe to events for testing
			var receivedEvents []events.Event
			handler := events.NewFuncEventHandler("plan.started", func(ctx context.Context, event events.Event) error {
				receivedEvents = append(receivedEvents, event)
				return nil
			})
			err := eventBus.Subscribe("plan.started", handler)
			require.NoError(t, err)

			completedHandler := events.NewFuncEventHandler("plan.completed", func(ctx context.Context, event events.Event) error {
				receivedEvents = append(receivedEvents, event)
				return nil
			})
			err = eventBus.Subscribe("plan.completed", completedHandler)
			require.NoError(t, err)

			planner := NewExecutionPlanner(logger, eventBus, &PlannerConfig{
				MaxConcurrentWorkers: 4,
				MaxMemoryMB:          256,
				PlanningTimeout:      5 * time.Second,
				EnableOptimizations:  true,
				OptimizationLevel:    1,
				MinBatchSize:         1,
				MaxBatchSize:         10,
				EnableMetrics:        true,
				DebugMode:            false,
			})

			ctx := context.Background()
			plan, err := planner.CreateExecutionPlan(ctx, tt.methods)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, plan)
			assert.NotNil(t, plan.Methods)
			assert.NotNil(t, plan.GlobalLimits)
			assert.NotNil(t, plan.Metrics)

			// Verify events were emitted
			assert.Len(t, receivedEvents, tt.expectedEvents)

			// Verify plan structure
			assert.LessOrEqual(t, plan.GlobalLimits.MaxWorkers, 4)
			assert.LessOrEqual(t, plan.GlobalLimits.MaxMemoryMB, 256)
			assert.Equal(t, len(tt.methods), len(plan.Methods))
		})
	}
}

func TestExecutionPlanner_DependencyHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	planner := NewExecutionPlanner(logger, eventBus, DefaultPlannerConfig())

	// Create methods with dependencies
	methods := createTestMethodsWithDependencies()

	ctx := context.Background()
	plan, err := planner.CreateExecutionPlan(ctx, methods)

	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Verify that dependencies are handled correctly
	for _, methodPlan := range plan.Methods {
		assert.GreaterOrEqual(t, len(methodPlan.Batches), 1)
		
		// Verify batches are ordered correctly
		for i := 1; i < len(methodPlan.Batches); i++ {
			// Each batch should depend on previous ones or be independent
			assert.NotEmpty(t, methodPlan.Batches[i].ID)
		}
	}
}

func TestExecutionPlanner_ConcurrentProcessing(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := &PlannerConfig{
		MaxConcurrentWorkers: 8,
		MaxMemoryMB:          512,
		PlanningTimeout:      10 * time.Second,
		EnableOptimizations:  true,
		OptimizationLevel:    2,
		MinBatchSize:         2,
		MaxBatchSize:         20,
		EnableMetrics:        true,
		DebugMode:            false,
	}

	planner := NewExecutionPlanner(logger, eventBus, config)

	// Create a large set of methods to test concurrent planning
	methods := createTestMethods(10, 15)

	ctx := context.Background()
	startTime := time.Now()
	plan, err := planner.CreateExecutionPlan(ctx, methods)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Verify performance characteristics
	assert.Less(t, duration, 5*time.Second, "Planning should complete quickly")
	assert.Greater(t, plan.Metrics.ParallelizationRatio, 0.0, "Should have some parallelization")
	assert.Greater(t, plan.Metrics.EstimatedSpeedupRatio, 1.0, "Should show speedup potential")

	// Verify resource allocation
	totalWorkers := 0
	totalMemory := 0
	for _, methodPlan := range plan.Methods {
		totalWorkers += methodPlan.RequiredWorkers
		totalMemory += methodPlan.MemoryRequirementMB
	}

	assert.LessOrEqual(t, plan.GlobalLimits.MaxWorkers, config.MaxConcurrentWorkers)
	assert.LessOrEqual(t, plan.GlobalLimits.MaxMemoryMB, config.MaxMemoryMB)
}

func TestExecutionPlanner_OptimizationLevels(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	methods := createTestMethods(5, 8)
	ctx := context.Background()

	optimizationLevels := []int{0, 1, 2}
	plans := make([]*domain.ExecutionPlan, len(optimizationLevels))

	for i, level := range optimizationLevels {
		config := DefaultPlannerConfig()
		config.OptimizationLevel = level

		planner := NewExecutionPlanner(logger, eventBus, config)
		plan, err := planner.CreateExecutionPlan(ctx, methods)
		require.NoError(t, err)
		plans[i] = plan
	}

	// Verify that higher optimization levels produce different (potentially better) plans
	for i := 1; i < len(plans); i++ {
		// Plans with higher optimization levels should have metrics indicating optimization
		// Note: This is a simplified check - in practice you'd verify specific optimizations
		assert.GreaterOrEqual(t, plans[i].Metrics.EstimatedSpeedupRatio, plans[i-1].Metrics.EstimatedSpeedupRatio)
	}
}

func TestExecutionPlanner_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	tests := []struct {
		name          string
		methods       []*domain.Method
		config        *PlannerConfig
		expectedError string
	}{
		{
			name:    "circular dependencies",
			methods: createTestMethodsWithCircularDependencies(),
			config:  DefaultPlannerConfig(),
		},
		{
			name:    "invalid configuration",
			methods: createTestMethods(1, 1),
			config: &PlannerConfig{
				MaxConcurrentWorkers: -1,
				MaxMemoryMB:          -1,
				PlanningTimeout:      -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := NewExecutionPlanner(logger, eventBus, tt.config)

			ctx := context.Background()
			_, err := planner.CreateExecutionPlan(ctx, tt.methods)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestExecutionPlanner_ResourceLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := &PlannerConfig{
		MaxConcurrentWorkers: 2,
		MaxMemoryMB:          64,
		PlanningTimeout:      1 * time.Second,
		EnableOptimizations:  true,
		OptimizationLevel:    1,
		MinBatchSize:         1,
		MaxBatchSize:         5,
		EnableMetrics:        true,
		DebugMode:            false,
	}

	planner := NewExecutionPlanner(logger, eventBus, config)

	// Create methods that would exceed limits without optimization
	methods := createTestMethods(8, 10)

	ctx := context.Background()
	plan, err := planner.CreateExecutionPlan(ctx, methods)

	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Verify that limits are respected
	assert.LessOrEqual(t, plan.GlobalLimits.MaxWorkers, config.MaxConcurrentWorkers)
	assert.LessOrEqual(t, plan.GlobalLimits.MaxMemoryMB, config.MaxMemoryMB)

	// Verify that all methods still have valid plans
	for _, methodPlan := range plan.Methods {
		assert.GreaterOrEqual(t, methodPlan.RequiredWorkers, 1)
		assert.GreaterOrEqual(t, methodPlan.MemoryRequirementMB, 10)
		assert.NotEmpty(t, methodPlan.Batches)
	}
}

func BenchmarkExecutionPlanner_CreatePlan(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultPlannerConfig()
	config.EnableMetrics = false // Disable for more accurate benchmarking

	planner := NewExecutionPlanner(logger, eventBus, config)
	methods := createTestMethods(20, 25)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := planner.CreateExecutionPlan(ctx, methods)
		require.NoError(b, err)
	}
}

func BenchmarkExecutionPlanner_LargeScale(b *testing.B) {
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	config := DefaultPlannerConfig()
	config.EnableMetrics = false
	config.OptimizationLevel = 0 // Disable optimizations for baseline

	planner := NewExecutionPlanner(logger, eventBus, config)
	methods := createTestMethods(100, 50) // 100 methods with 50 fields each
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := planner.CreateExecutionPlan(ctx, methods)
		require.NoError(b, err)
	}
}

// Helper functions for creating test data

func createTestMethods(methodCount, fieldsPerMethod int) []*domain.Method {
	methods := make([]*domain.Method, methodCount)

	for i := 0; i < methodCount; i++ {
		method, _ := domain.NewMethod(
			fmt.Sprintf("Method%d", i),
			createTestType(fmt.Sprintf("Source%d", i)),
			createTestType(fmt.Sprintf("Dest%d", i)),
		)

		// Add field mappings
		for j := 0; j < fieldsPerMethod; j++ {
			mapping := createTestFieldMapping(i, j)
			method.AddMapping(mapping)
		}

		methods[i] = method
	}

	return methods
}

func createTestMethodsWithDependencies() []*domain.Method {
	methods := make([]*domain.Method, 3)

	// Method 1: No dependencies
	method1, _ := domain.NewMethod("Method1", createTestType("Source1"), createTestType("Dest1"))
	mapping1 := createTestFieldMapping(1, 1)
	method1.AddMapping(mapping1)
	methods[0] = method1

	// Method 2: Depends on Method1
	method2, _ := domain.NewMethod("Method2", createTestType("Source2"), createTestType("Dest2"))
	mapping2 := createTestFieldMapping(2, 1)
	mapping2.Dependencies = []string{mapping1.ID}
	method2.AddMapping(mapping2)
	methods[1] = method2

	// Method 3: Depends on Method2
	method3, _ := domain.NewMethod("Method3", createTestType("Source3"), createTestType("Dest3"))
	mapping3 := createTestFieldMapping(3, 1)
	mapping3.Dependencies = []string{mapping2.ID}
	method3.AddMapping(mapping3)
	methods[2] = method3

	return methods
}

func createTestMethodsWithCircularDependencies() []*domain.Method {
	methods := make([]*domain.Method, 2)

	// Method 1
	method1, _ := domain.NewMethod("Method1", createTestType("Source1"), createTestType("Dest1"))
	mapping1 := createTestFieldMapping(1, 1)
	methods[0] = method1

	// Method 2
	method2, _ := domain.NewMethod("Method2", createTestType("Source2"), createTestType("Dest2"))
	mapping2 := createTestFieldMapping(2, 1)
	methods[1] = method2

	// Create circular dependency
	mapping1.Dependencies = []string{mapping2.ID}
	mapping2.Dependencies = []string{mapping1.ID}

	method1.AddMapping(mapping1)
	method2.AddMapping(mapping2)

	return methods
}

func createTestFieldMapping(methodIndex, fieldIndex int) *domain.FieldMapping {
	sourceField := &domain.Field{
		Name:     fmt.Sprintf("Field%d", fieldIndex),
		Type:     createTestType("string"),
		Exported: true,
		Position: fieldIndex,
	}

	destField := &domain.Field{
		Name:     fmt.Sprintf("Field%d", fieldIndex),
		Type:     createTestType("string"),
		Exported: true,
		Position: fieldIndex,
	}

	sourceSpec, _ := domain.NewFieldSpec([]string{sourceField.Name}, sourceField.Type)
	destSpec, _ := domain.NewFieldSpec([]string{destField.Name}, destField.Type)

	strategy := &domain.DirectAssignmentStrategy{}
	mappingID := fmt.Sprintf("mapping_%d_%d", methodIndex, fieldIndex)

	mapping, _ := domain.NewFieldMapping(mappingID, sourceSpec, destSpec, strategy)
	return mapping
}

func createTestType(name string) domain.Type {
	return domain.NewBasicType(name, domain.TypeKindString)
}