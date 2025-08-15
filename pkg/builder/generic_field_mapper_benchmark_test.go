// Package builder provides enhanced benchmarking for the GenericFieldMapper performance optimizations
package builder

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// Benchmark data structures for testing
type BenchmarkSource struct {
	ID          int
	Name        string
	Email       string
	Age         int
	Active      bool
	Tags        []string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	NestedData  *NestedStruct
	SliceData   []NestedStruct
	ComplexData map[string][]NestedStruct
}

type BenchmarkDestination struct {
	ID          int
	Name        string
	Email       string
	Age         int
	Active      bool
	Tags        []string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	NestedData  *NestedStruct
	SliceData   []NestedStruct
	ComplexData map[string][]NestedStruct
}

type NestedStruct struct {
	Value   string
	Count   int
	Enabled bool
}

func createMockTypes() (domain.Type, domain.Type) {
	// Create mock source type
	sourceFields := []domain.Field{
		{Name: "ID", Type: domain.NewBasicType("int", reflect.Int), Exported: true},
		{Name: "Name", Type: domain.NewBasicType("string", reflect.String), Exported: true},
		{Name: "Email", Type: domain.NewBasicType("string", reflect.String), Exported: true},
		{Name: "Age", Type: domain.NewBasicType("int", reflect.Int), Exported: true},
		{Name: "Active", Type: domain.NewBasicType("bool", reflect.Bool), Exported: true},
		{Name: "Tags", Type: domain.NewSliceType(domain.NewBasicType("string", reflect.String), ""), Exported: true},
		{Name: "Metadata", Type: domain.NewBasicType("interface{}", reflect.Interface), Exported: true},
		{Name: "CreatedAt", Type: domain.NewBasicType("time.Time", reflect.Struct), Exported: true},
		{Name: "UpdatedAt", Type: domain.NewPointerType(domain.NewBasicType("time.Time", reflect.Struct), ""), Exported: true},
	}

	// Create mock destination type with identical fields
	destFields := make([]domain.Field, len(sourceFields))
	copy(destFields, sourceFields)

	sourceType := domain.NewStructType("BenchmarkSource", sourceFields, "")
	destType := domain.NewStructType("BenchmarkDestination", destFields, "")

	return sourceType, destType
}

// BenchmarkGenericFieldMapper_Sequential tests performance without optimizations
func BenchmarkGenericFieldMapper_Sequential(b *testing.B) {
	logger := zap.NewNop()

	// Create mapper with optimizations disabled
	config := &GenericFieldMapperConfig{
		EnableCaching:        false,
		EnableOptimization:   false,
		EnableTypeValidation: true,
		PerformanceMode:      false,
	}

	perfConfig := &PerformanceConfig{
		EnableFieldMappingCache: false,
		EnableParallelMapping:   false,
		EnableMemoryPooling:     false,
		PerformanceProfile:      "memory",
	}

	mapper := NewGenericFieldMapper(nil, nil, logger, config)
	mapper.ConfigurePerformance(perfConfig)

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}
	}
}

// BenchmarkGenericFieldMapper_WithCaching tests performance with caching enabled
func BenchmarkGenericFieldMapper_WithCaching(b *testing.B) {
	logger := zap.NewNop()

	config := &GenericFieldMapperConfig{
		EnableCaching:        true,
		EnableOptimization:   true,
		EnableTypeValidation: true,
		PerformanceMode:      true,
	}

	perfConfig := &PerformanceConfig{
		EnableFieldMappingCache: true,
		MaxCacheSize:            1000,
		CacheTTL:                1 * time.Hour,
		EnableParallelMapping:   false,
		EnableMemoryPooling:     true,
		PerformanceProfile:      "balanced",
	}

	mapper := NewGenericFieldMapper(nil, nil, logger, config)
	mapper.ConfigurePerformance(perfConfig)

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}
	}
}

// BenchmarkGenericFieldMapper_WithParallel tests performance with parallel processing
func BenchmarkGenericFieldMapper_WithParallel(b *testing.B) {
	logger := zap.NewNop()

	config := &GenericFieldMapperConfig{
		EnableCaching:        false,
		EnableOptimization:   true,
		EnableTypeValidation: true,
		PerformanceMode:      true,
	}

	perfConfig := &PerformanceConfig{
		EnableFieldMappingCache: false,
		EnableParallelMapping:   true,
		MaxParallelWorkers:      runtime.NumCPU(),
		EnableMemoryPooling:     true,
		PerformanceProfile:      "speed",
	}

	mapper := NewGenericFieldMapper(nil, nil, logger, config)
	mapper.ConfigurePerformance(perfConfig)

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}
	}
}

// BenchmarkGenericFieldMapper_AllOptimizations tests performance with all optimizations enabled
func BenchmarkGenericFieldMapper_AllOptimizations(b *testing.B) {
	logger := zap.NewNop()

	config := &GenericFieldMapperConfig{
		EnableCaching:        true,
		EnableOptimization:   true,
		EnableTypeValidation: true,
		PerformanceMode:      true,
	}

	perfConfig := &PerformanceConfig{
		EnableFieldMappingCache: true,
		MaxCacheSize:            10000,
		CacheTTL:                2 * time.Hour,
		EnableParallelMapping:   true,
		MaxParallelWorkers:      runtime.NumCPU(),
		EnableMemoryPooling:     true,
		PerformanceProfile:      "speed",
		AutoTune:                true,
	}

	mapper := NewGenericFieldMapper(nil, nil, logger, config)
	mapper.ConfigurePerformance(perfConfig)

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}
	}
}

// BenchmarkGenericFieldMapper_MemoryUsage tests memory efficiency
func BenchmarkGenericFieldMapper_MemoryUsage(b *testing.B) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("memory")

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	// Track memory usage
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}

		if i%100 == 0 {
			mapper.OptimizeMemoryUsage()
		}
	}

	b.StopTimer()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	memUsed := m2.TotalAlloc - m1.TotalAlloc
	b.ReportMetric(float64(memUsed)/float64(b.N), "bytes/op")
}

// BenchmarkGenericFieldMapper_ConcurrentAccess tests thread safety under concurrent load
func BenchmarkGenericFieldMapper_ConcurrentAccess(b *testing.B) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("speed")

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
			if err != nil {
				b.Fatalf("Concurrent mapping failed: %v", err)
			}
		}
	})
}

// BenchmarkGenericFieldMapper_CacheHitRate measures cache effectiveness
func BenchmarkGenericFieldMapper_CacheHitRate(b *testing.B) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("speed")

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	// Warm up cache
	for i := 0; i < 10; i++ {
		mapper.MapGenericFields(srcType, dstType, nil, options)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}
	}

	metrics := mapper.GetEnhancedMetrics()
	perfMetrics := metrics["performance_metrics"].(map[string]interface{})
	cacheHits := perfMetrics["cache_hits"].(int64)
	cacheMisses := perfMetrics["cache_misses"].(int64)

	if cacheHits+cacheMisses > 0 {
		hitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
		b.ReportMetric(hitRate, "cache_hit_%")
	}
}

// BenchmarkGenericFieldMapper_LargeStructs tests performance with complex nested structures
func BenchmarkGenericFieldMapper_LargeStructs(b *testing.B) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("balanced")

	// Create larger, more complex types
	var sourceFields []domain.Field
	var destFields []domain.Field

	// Add 50 fields to simulate a large struct
	for i := 0; i < 50; i++ {
		field := domain.Field{
			Name:     fmt.Sprintf("Field%d", i),
			Type:     domain.NewBasicType("string", reflect.String),
			Exported: true,
		}
		sourceFields = append(sourceFields, field)
		destFields = append(destFields, field)
	}

	sourceType := domain.NewStructType("LargeSource", sourceFields, "")
	destType := domain.NewStructType("LargeDestination", destFields, "")

	options := DefaultFieldMappingOptions()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapGenericFields(sourceType, destType, nil, options)
		if err != nil {
			b.Fatalf("Large struct mapping failed: %v", err)
		}
	}
}

// BenchmarkGenericFieldMapper_ParallelVsSequential compares parallel vs sequential performance
func BenchmarkGenericFieldMapper_ParallelVsSequential(b *testing.B) {
	logger := zap.NewNop()

	// Test sequential
	b.Run("Sequential", func(b *testing.B) {
		mapper := NewGenericFieldMapper(nil, nil, logger, nil)
		perfConfig := &PerformanceConfig{
			EnableParallelMapping: false,
			EnableMemoryPooling:   false,
		}
		mapper.ConfigurePerformance(perfConfig)

		srcType, dstType := createMockTypes()
		options := DefaultFieldMappingOptions()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
			if err != nil {
				b.Fatalf("Sequential mapping failed: %v", err)
			}
		}
	})

	// Test parallel
	b.Run("Parallel", func(b *testing.B) {
		mapper := NewGenericFieldMapper(nil, nil, logger, nil)
		perfConfig := &PerformanceConfig{
			EnableParallelMapping: true,
			MaxParallelWorkers:    runtime.NumCPU(),
			EnableMemoryPooling:   true,
		}
		mapper.ConfigurePerformance(perfConfig)

		srcType, dstType := createMockTypes()
		options := DefaultFieldMappingOptions()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
			if err != nil {
				b.Fatalf("Parallel mapping failed: %v", err)
			}
		}
	})
}

// BenchmarkGenericFieldMapper_MapFieldsInterface tests the new interface method
func BenchmarkGenericFieldMapper_MapFieldsInterface(b *testing.B) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("speed")

	srcType, dstType := createMockTypes()
	annotations := map[string]string{
		"map:Name":      "Name",
		"skip:Internal": "true",
		"converter:Age": "string",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mapper.MapFields(srcType, dstType, annotations)
		if err != nil {
			b.Fatalf("Interface mapping failed: %v", err)
		}
	}
}

// TestGenericFieldMapper_PerformanceMetrics validates performance metrics collection
func TestGenericFieldMapper_PerformanceMetrics(t *testing.T) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("speed")

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	// Perform several mappings
	for i := 0; i < 5; i++ {
		_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
		if err != nil {
			t.Fatalf("Mapping failed: %v", err)
		}
	}

	// Check metrics
	metrics := mapper.GetEnhancedMetrics()

	basicMetrics := metrics["basic_metrics"].(map[string]interface{})
	if totalMappings := basicMetrics["total_mappings"].(int64); totalMappings < 1 {
		t.Errorf("Expected at least 1 mapping, got %d", totalMappings)
	}

	perfMetrics := metrics["performance_metrics"].(map[string]interface{})
	if cacheHits := perfMetrics["cache_hits"].(int64); cacheHits < 0 {
		t.Errorf("Cache hits should be non-negative, got %d", cacheHits)
	}

	config := metrics["configuration"].(map[string]interface{})
	if cacheEnabled := config["cache_enabled"].(bool); !cacheEnabled {
		t.Error("Cache should be enabled in speed profile")
	}
}

// TestGenericFieldMapper_MemoryOptimization validates memory optimization features
func TestGenericFieldMapper_MemoryOptimization(t *testing.T) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)

	// Test memory profile
	mapper.OptimizeForProfile("memory")
	config := mapper.GetPerformanceProfile()

	if config.MaxCacheSize != 1000 {
		t.Errorf("Expected MaxCacheSize=1000 for memory profile, got %d", config.MaxCacheSize)
	}

	if config.EnableParallelMapping {
		t.Error("Parallel mapping should be disabled in memory profile")
	}

	// Test manual memory optimization
	mapper.OptimizeMemoryUsage() // Should not panic

	// Reset metrics
	mapper.ResetPerformanceMetrics()
	metrics := mapper.GetEnhancedMetrics()
	basicMetrics := metrics["basic_metrics"].(map[string]interface{})

	if totalMappings := basicMetrics["total_mappings"].(int64); totalMappings != 0 {
		t.Errorf("Expected 0 mappings after reset, got %d", totalMappings)
	}
}

// TestGenericFieldMapper_ThreadSafety validates thread safety
func TestGenericFieldMapper_ThreadSafety(t *testing.T) {
	logger := zap.NewNop()
	mapper := NewGenericFieldMapper(nil, nil, logger, nil)
	mapper.OptimizeForProfile("speed")

	srcType, dstType := createMockTypes()
	options := DefaultFieldMappingOptions()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := mapper.MapGenericFields(srcType, dstType, nil, options)
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify metrics consistency
	metrics := mapper.GetEnhancedMetrics()
	basicMetrics := metrics["basic_metrics"].(map[string]interface{})
	totalMappings := basicMetrics["total_mappings"].(int64)
	successfulMappings := basicMetrics["successful_mappings"].(int64)

	if totalMappings != successfulMappings {
		t.Errorf("Mapping count mismatch: total=%d, successful=%d", totalMappings, successfulMappings)
	}
}
