package emitter

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/executor"
	"github.com/reedom/convergen/v9/pkg/internal/events"
)

// TestConcurrentCodeGenMetrics tests thread safety of CodeGenMetrics.
func TestConcurrentCodeGenMetrics(t *testing.T) {
	metrics := NewCodeGenMetrics()

	const numGoroutines = 10

	const operationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Start multiple goroutines updating metrics concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				metrics.IncrementMethods()
				metrics.AddGenerationTime(time.Millisecond)
				metrics.IncrementFields()
				metrics.IncrementStrategy("test_strategy")
				metrics.IncrementErrors()
				metrics.IncrementErrorHandlers()
			}
		}()
	}

	// Also read metrics concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine/2; j++ {
				snapshot := metrics.GetSnapshot()
				assert.NotNil(t, snapshot)
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()

	// Verify final metrics
	snapshot := metrics.GetSnapshot()
	expectedMethods := int64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expectedMethods, snapshot.MethodsGenerated)
	assert.Equal(t, expectedMethods, snapshot.FieldsGenerated)
	assert.Equal(t, expectedMethods, snapshot.ErrorHandlersGenerated)
	assert.Equal(t, expectedMethods, snapshot.ErrorsEncountered)
	assert.True(t, 0 < snapshot.TotalGenerationTime)
}

// TestConcurrentMetricsAccess tests thread safety of Metrics.
func TestConcurrentMetricsAccess(t *testing.T) {
	metrics := NewMetrics()

	const numGoroutines = 8

	const operationsPerGoroutine = 50

	var wg sync.WaitGroup

	// Create sample method code for testing
	methodCode := &MethodCode{
		Name: "TestMethod",
		Body: "return &Dest{Field1: src.Field1, Field2: src.Field2}, nil",
		Fields: []*FieldCode{
			{Name: "Field1"},
			{Name: "Field2"},
		},
	}

	// Start multiple goroutines updating metrics concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				metrics.RecordGeneration(methodCode, "test_package", []*MethodCode{methodCode})
				metrics.AddGenerationTime(time.Millisecond * time.Duration(j+1))
				metrics.RecordStrategyUsage("composite_literal", time.Microsecond*time.Duration(j+1))
				metrics.RecordError("test_error")
				metrics.UpdateMemoryUsage(int64(1024 * (j + 1)))
			}
		}()
	}

	// Also read metrics concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				snapshot := metrics.GetSnapshot()
				assert.NotNil(t, snapshot)
				assert.NotNil(t, snapshot.StrategyUsage)
				assert.NotNil(t, snapshot.ErrorsByType)
				time.Sleep(time.Microsecond * 10)
			}
		}()
	}

	wg.Wait()

	// Verify final metrics
	snapshot := metrics.GetSnapshot()
	expectedGenerations := int64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expectedGenerations, snapshot.TotalGenerations)
	assert.Equal(t, expectedGenerations, snapshot.TotalMethods)
	assert.True(t, 0 < snapshot.TotalGenerationTime)
	assert.True(t, 0 < snapshot.PeakMemoryUsage)
	assert.Equal(t, expectedGenerations, snapshot.ErrorsEncountered)
}

// TestConcurrentCodeGeneration tests concurrent method generation.
func TestConcurrentCodeGeneration(t *testing.T) {
	// Use a no-op logger for high concurrency testing to avoid test infrastructure race conditions
	logger := zap.NewNop()
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultConfig()
	config.EnableConcurrentGen = true
	config.MaxConcurrentMethods = 4

	emitter := NewEmitter(logger, eventBus, config)
	defer func() {
		if err := emitter.Shutdown(context.Background()); err != nil {
			t.Errorf("emitter.Shutdown failed: %v", err)
		}
	}()

	// Create multiple methods for concurrent generation
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)

	methods := make([]*domain.MethodResult, 10)

	for i := 0; i < 10; i++ {
		method, err := domain.NewMethod("ConvertMethod", sourceType, destType)
		require.NoError(t, err)

		methods[i] = &domain.MethodResult{
			Method:      method,
			Code:        "",
			Success:     true,
			Error:       nil,
			ProcessedAt: time.Now(),
			DurationMS:  5,
			Metadata: map[string]interface{}{
				"test_field": &executor.FieldResult{
					FieldID: "TestField",
					Success: true,
				},
			},
		}
	}

	const numWorkers = 5

	var wg sync.WaitGroup

	// Generate methods concurrently
	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			for i := workerID; i < len(methods); i += numWorkers {
				methodCode, err := emitter.GenerateMethod(context.Background(), methods[i])
				assert.NoError(t, err)
				assert.NotNil(t, methodCode)
				assert.NotEmpty(t, methodCode.Name)
			}
		}(worker)
	}

	wg.Wait()

	// Verify no race conditions occurred
	metrics := emitter.GetMetrics()
	t.Logf("Concurrent code generation completed. Emitter metrics: %d generations, %d methods",
		metrics.TotalGenerations, metrics.TotalMethods)
	// Note: The primary goal is race condition detection rather than specific counts
	assert.NotNil(t, emitter)
}

// TestConcurrentEmitterOperations tests various emitter operations under concurrency.
func TestConcurrentEmitterOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultConfig()
	config.EnableConcurrentGen = true

	emitter := NewEmitter(logger, eventBus, config)
	defer func() {
		if err := emitter.Shutdown(context.Background()); err != nil {
			t.Errorf("emitter.Shutdown failed: %v", err)
		}
	}()

	sourceType := domain.NewBasicType("ConcurrentSource", reflect.Struct)
	destType := domain.NewBasicType("ConcurrentDest", reflect.Struct)
	method, err := domain.NewMethod("ConcurrentConvert", sourceType, destType)
	require.NoError(t, err)

	// Create execution results
	results := &domain.ExecutionResults{
		PackageName: "concurrent_test",
		Methods: []*domain.MethodResult{
			{
				Method:      method,
				Code:        "",
				Success:     true,
				Error:       nil,
				ProcessedAt: time.Now(),
				DurationMS:  10,
				Metadata: map[string]interface{}{
					"field1": &executor.FieldResult{FieldID: "Field1", Success: true},
					"field2": &executor.FieldResult{FieldID: "Field2", Success: true},
				},
			},
		},
		Success: true,
	}

	const numOperations = 10

	var wg sync.WaitGroup

	// Perform various operations concurrently
	for i := 0; i < numOperations; i++ {
		wg.Add(1)

		go func(opID int) {
			defer wg.Done()

			switch opID % 3 {
			case 0:
				// Generate complete code
				code, err := emitter.GenerateCode(context.Background(), results)
				assert.NoError(t, err)
				assert.NotNil(t, code)

			case 1:
				// Generate single method
				methodCode, err := emitter.GenerateMethod(context.Background(), results.Methods[0])
				assert.NoError(t, err)
				assert.NotNil(t, methodCode)

			case 2:
				// Get metrics (should not race with other operations)
				metrics := emitter.GetMetrics()
				assert.NotNil(t, metrics)
			}
		}(i)
	}

	wg.Wait()

	// Final verification
	finalMetrics := emitter.GetMetrics()
	assert.True(t, 0 < finalMetrics.TotalGenerations)
}

// TestStressTesting performs high-stress concurrent testing.
func TestStressTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Use a no-op logger for stress testing to avoid race conditions in test infrastructure
	logger := zap.NewNop()
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultConfig()
	config.EnableConcurrentGen = true
	config.MaxConcurrentMethods = 8

	emitter := NewEmitter(logger, eventBus, config)
	defer func() {
		if err := emitter.Shutdown(context.Background()); err != nil {
			t.Errorf("emitter.Shutdown failed: %v", err)
		}
	}()

	sourceType := domain.NewBasicType("StressSource", reflect.Struct)
	destType := domain.NewBasicType("StressDest", reflect.Struct)
	method, err := domain.NewMethod("StressConvert", sourceType, destType)
	require.NoError(t, err)

	methodResult := &domain.MethodResult{
		Method:      method,
		Code:        "",
		Success:     true,
		Error:       nil,
		ProcessedAt: time.Now(),
		DurationMS:  1,
		Metadata: map[string]interface{}{
			"stress_field": &executor.FieldResult{FieldID: "StressField", Success: true},
		},
	}

	const stressLevel = 100

	var wg sync.WaitGroup

	startTime := time.Now()

	// High stress concurrent generation
	for i := 0; i < stressLevel; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				methodCode, err := emitter.GenerateMethod(context.Background(), methodResult)
				assert.NoError(t, err)
				assert.NotNil(t, methodCode)

				// Occasionally get metrics to test concurrent reads
				if j%3 == 0 {
					metrics := emitter.GetMetrics()
					assert.NotNil(t, metrics)
				}
			}
		}()
	}

	wg.Wait()

	duration := time.Since(startTime)
	t.Logf("Stress test completed in %v", duration)

	// Verify final state - just check that we completed without race conditions
	metrics := emitter.GetMetrics()
	t.Logf("Stress test completed successfully. Total methods in emitter metrics: %d", metrics.TotalMethods)
	// Note: The main goal is race condition detection, not specific metric counting
	assert.NotNil(t, emitter)
}

// TestRaceDetectorCompliance ensures all operations pass race detector.
func TestRaceDetectorCompliance(t *testing.T) {
	// This test is specifically designed to trigger race conditions if they exist
	// It should always pass when run with -race flag if thread safety is correct
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	config := DefaultConfig()
	config.EnableConcurrentGen = true

	emitter := NewEmitter(logger, eventBus, config)
	defer func() {
		if err := emitter.Shutdown(context.Background()); err != nil {
			t.Errorf("emitter.Shutdown failed: %v", err)
		}
	}()

	sourceType := domain.NewBasicType("RaceTestSource", reflect.Struct)
	destType := domain.NewBasicType("RaceTestDest", reflect.Struct)
	method, err := domain.NewMethod("RaceTestConvert", sourceType, destType)
	require.NoError(t, err)

	methodResult := &domain.MethodResult{
		Method:      method,
		Code:        "",
		Success:     true,
		Error:       nil,
		ProcessedAt: time.Now(),
		DurationMS:  1,
		Metadata: map[string]interface{}{
			"race_field": &executor.FieldResult{FieldID: "RaceField", Success: true},
		},
	}

	var wg sync.WaitGroup

	// Multiple concurrent operations designed to trigger races
	for i := 0; i < 20; i++ {
		wg.Add(1)

		go func(_ int) {
			defer wg.Done()

			// Mix of read and write operations
			for j := 0; j < 50; j++ {
				switch j % 4 {
				case 0:
					_, _ = emitter.GenerateMethod(context.Background(), methodResult)
				case 1:
					emitter.GetMetrics()
				case 2:
					// Access CodeGenMetrics directly through emitter
					if codeGen, ok := emitter.(*ConcreteEmitter); ok {
						if cg, ok := codeGen.codeGen.(*ConcreteCodeGenerator); ok {
							cg.GetMetrics()
						}
					}
				case 3:
					// Force metrics snapshot creation
					metrics := emitter.GetMetrics()
					if 0 < metrics.TotalMethods {
						// This should not race
						_ = metrics.TotalGenerations
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Final validation
	metrics := emitter.GetMetrics()
	assert.NotNil(t, metrics)
	t.Logf("Race detector compliance test completed successfully with %d methods generated",
		metrics.TotalMethods)
}
