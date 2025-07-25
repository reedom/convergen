package coordinator

import (
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewMetricsCollector(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	
	collector := NewMetricsCollector(logger, config)
	
	if collector == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}
	
	// Verify it implements the interface
	var _ MetricsCollector = collector
}

func TestMetricsCollectorRecordEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	duration := 100 * time.Millisecond
	metadata := map[string]interface{}{
		"source_count": 5,
		"success":      true,
	}
	
	collector.RecordEvent("pipeline_completed", duration, metadata)
	
	metrics := collector.GetMetrics()
	
	if metrics.PipelineExecutions != 1 {
		t.Errorf("Expected 1 pipeline execution, got %d", metrics.PipelineExecutions)
	}
	
	if metrics.TotalDuration != duration {
		t.Errorf("Expected total duration %v, got %v", duration, metrics.TotalDuration)
	}
	
	if metrics.AverageDuration != duration {
		t.Errorf("Expected average duration %v, got %v", duration, metrics.AverageDuration)
	}
	
	if metrics.SuccessRate != 1.0 {
		t.Errorf("Expected success rate 1.0, got %f", metrics.SuccessRate)
	}
	
	if metrics.EventCounts["pipeline_completed"] != 1 {
		t.Errorf("Expected event count 1, got %d", metrics.EventCounts["pipeline_completed"])
	}
}

func TestMetricsCollectorRecordMultipleEvents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	// Record successful pipeline
	collector.RecordEvent("pipeline_completed", 100*time.Millisecond, map[string]interface{}{
		"success": true,
	})
	
	// Record failed pipeline
	collector.RecordEvent("pipeline_failed", 50*time.Millisecond, map[string]interface{}{
		"success": false,
	})
	
	metrics := collector.GetMetrics()
	
	if metrics.PipelineExecutions != 2 {
		t.Errorf("Expected 2 pipeline executions, got %d", metrics.PipelineExecutions)
	}
	
	expectedTotal := 150 * time.Millisecond
	if metrics.TotalDuration != expectedTotal {
		t.Errorf("Expected total duration %v, got %v", expectedTotal, metrics.TotalDuration)
	}
	
	expectedAvg := 75 * time.Millisecond
	if metrics.AverageDuration != expectedAvg {
		t.Errorf("Expected average duration %v, got %v", expectedAvg, metrics.AverageDuration)
	}
	
	expectedSuccessRate := 0.5
	if metrics.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected success rate %f, got %f", expectedSuccessRate, metrics.SuccessRate)
	}
}

func TestMetricsCollectorRecordComponent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	componentMetrics := map[string]interface{}{
		"parsed_files": 10,
		"parse_time":   time.Second,
	}
	
	collector.RecordComponent("parser", componentMetrics)
	
	metrics := collector.GetMetrics()
	
	if len(metrics.ComponentMetrics) != 1 {
		t.Errorf("Expected 1 component metric, got %d", len(metrics.ComponentMetrics))
	}
	
	parserMetrics, exists := metrics.ComponentMetrics["parser"]
	if !exists {
		t.Error("Expected parser metrics to exist")
	}
	
	parserMap, ok := parserMetrics.(map[string]interface{})
	if !ok {
		t.Fatal("Expected parser metrics to be map[string]interface{}")
	}
	
	if parserMap["parsed_files"] != 10 {
		t.Errorf("Expected parsed_files=10, got %v", parserMap["parsed_files"])
	}
}

func TestMetricsCollectorRecordError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	collector.RecordError("parser", "syntax_error")
	collector.RecordError("parser", "syntax_error") // Duplicate
	collector.RecordError("executor", "conversion_error")
	
	metrics := collector.GetMetrics()
	
	if metrics.ErrorCounts["parser.syntax_error"] != 2 {
		t.Errorf("Expected parser.syntax_error count 2, got %d", 
			metrics.ErrorCounts["parser.syntax_error"])
	}
	
	if metrics.ErrorCounts["executor.conversion_error"] != 1 {
		t.Errorf("Expected executor.conversion_error count 1, got %d", 
			metrics.ErrorCounts["executor.conversion_error"])
	}
}

func TestMetricsCollectorRecordRetry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	// Record successful retry
	collector.RecordRetry("parser", true, 100*time.Millisecond)
	
	// Record failed retry
	collector.RecordRetry("executor", false, 200*time.Millisecond)
	
	metrics := collector.GetMetrics()
	
	if metrics.RetryStats.TotalRetries != 2 {
		t.Errorf("Expected 2 total retries, got %d", metrics.RetryStats.TotalRetries)
	}
	
	if metrics.RetryStats.SuccessfulRetries != 1 {
		t.Errorf("Expected 1 successful retry, got %d", metrics.RetryStats.SuccessfulRetries)
	}
	
	if metrics.RetryStats.FailedRetries != 1 {
		t.Errorf("Expected 1 failed retry, got %d", metrics.RetryStats.FailedRetries)
	}
	
	if metrics.RetryStats.RetrysByComponent["parser"] != 1 {
		t.Errorf("Expected 1 parser retry, got %d", metrics.RetryStats.RetrysByComponent["parser"])
	}
	
	if metrics.RetryStats.RetrysByComponent["executor"] != 1 {
		t.Errorf("Expected 1 executor retry, got %d", metrics.RetryStats.RetrysByComponent["executor"])
	}
	
	expectedAvgDelay := 150 * time.Millisecond
	if metrics.RetryStats.AverageRetryDelay != expectedAvgDelay {
		t.Errorf("Expected average retry delay %v, got %v", 
			expectedAvgDelay, metrics.RetryStats.AverageRetryDelay)
	}
}

func TestMetricsCollectorUpdateResourceUsage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	usage := &ResourceUsage{
		PeakMemoryUsage:    1024 * 1024, // 1MB
		CurrentMemoryUsage: 512 * 1024,  // 512KB
		GoroutineCount:     10,
		CPUUsage:           0.75,
	}
	
	collector.UpdateResourceUsage(usage)
	
	metrics := collector.GetMetrics()
	
	if metrics.ResourceUsage.PeakMemoryUsage != usage.PeakMemoryUsage {
		t.Errorf("Expected peak memory %d, got %d", 
			usage.PeakMemoryUsage, metrics.ResourceUsage.PeakMemoryUsage)
	}
	
	if metrics.ResourceUsage.GoroutineCount != usage.GoroutineCount {
		t.Errorf("Expected goroutine count %d, got %d", 
			usage.GoroutineCount, metrics.ResourceUsage.GoroutineCount)
	}
}

func TestMetricsCollectorRecordThroughput(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	collector.RecordThroughput(10.5)
	collector.RecordThroughput(12.3)
	
	// Access the concrete type to check throughput history
	concreteCollector := collector.(*ConcreteMetricsCollector)
	
	if len(concreteCollector.throughputHistory) != 2 {
		t.Errorf("Expected 2 throughput measurements, got %d", 
			len(concreteCollector.throughputHistory))
	}
	
	if concreteCollector.throughputHistory[0] != 10.5 {
		t.Errorf("Expected first throughput 10.5, got %f", 
			concreteCollector.throughputHistory[0])
	}
}

func TestMetricsCollectorReset(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	// Add some metrics
	collector.RecordEvent("pipeline_completed", time.Second, nil)
	collector.RecordError("parser", "error")
	collector.RecordRetry("executor", true, time.Millisecond)
	
	// Verify metrics exist
	metrics := collector.GetMetrics()
	if metrics.PipelineExecutions == 0 {
		t.Error("Expected metrics before reset")
	}
	
	// Reset
	collector.Reset()
	
	// Verify metrics are cleared
	metrics = collector.GetMetrics()
	if metrics.PipelineExecutions != 0 {
		t.Errorf("Expected 0 pipeline executions after reset, got %d", 
			metrics.PipelineExecutions)
	}
	
	if len(metrics.ErrorCounts) != 0 {
		t.Errorf("Expected 0 error counts after reset, got %d", len(metrics.ErrorCounts))
	}
	
	if metrics.RetryStats.TotalRetries != 0 {
		t.Errorf("Expected 0 total retries after reset, got %d", 
			metrics.RetryStats.TotalRetries)
	}
}

func TestMetricsCollectorWithDisabledMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	config.EnableMetrics = false
	collector := NewMetricsCollector(logger, config)
	
	// Record events - should be ignored
	collector.RecordEvent("pipeline_completed", time.Second, nil)
	collector.RecordError("parser", "error")
	collector.RecordRetry("executor", true, time.Millisecond)
	
	metrics := collector.GetMetrics()
	
	// Metrics should remain at zero
	if metrics.PipelineExecutions != 0 {
		t.Errorf("Expected 0 pipeline executions with disabled metrics, got %d", 
			metrics.PipelineExecutions)
	}
	
	if len(metrics.ErrorCounts) != 0 {
		t.Errorf("Expected 0 error counts with disabled metrics, got %d", 
			len(metrics.ErrorCounts))
	}
}

func TestMetricsCollectorLatencyCalculation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	// Record several pipeline executions with different durations
	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		150 * time.Millisecond,
		300 * time.Millisecond,
		250 * time.Millisecond,
	}
	
	for _, duration := range durations {
		collector.RecordEvent("pipeline_completed", duration, nil)
	}
	
	metrics := collector.GetMetrics()
	
	// Check latency metrics
	if metrics.Latency.Min != 100*time.Millisecond {
		t.Errorf("Expected min latency 100ms, got %v", metrics.Latency.Min)
	}
	
	if metrics.Latency.Max != 300*time.Millisecond {
		t.Errorf("Expected max latency 300ms, got %v", metrics.Latency.Max)
	}
	
	// P50 should be 200ms (middle value when sorted: 100, 150, 200, 250, 300)
	if metrics.Latency.P50 != 200*time.Millisecond {
		t.Errorf("Expected P50 latency 200ms, got %v", metrics.Latency.P50)
	}
	
	// P90 should be 250ms
	if metrics.Latency.P90 != 250*time.Millisecond {
		t.Errorf("Expected P90 latency 250ms, got %v", metrics.Latency.P90)
	}
	
	// Mean should be 200ms ((100+200+150+300+250)/5)
	expectedMean := 200 * time.Millisecond
	if metrics.Latency.Mean != expectedMean {
		t.Errorf("Expected mean latency %v, got %v", expectedMean, metrics.Latency.Mean)
	}
}

func TestMetricsCollectorThroughputCalculation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	// Simulate some pipeline executions over time
	start := time.Now()
	
	// Record events
	collector.RecordEvent("pipeline_completed", 100*time.Millisecond, nil)
	collector.RecordEvent("pipeline_completed", 200*time.Millisecond, nil)
	
	// Get metrics - throughput should be calculated based on elapsed time
	metrics := collector.GetMetrics()
	
	// Throughput should be > 0 since we have executions and time has passed
	if metrics.Throughput <= 0 {
		t.Errorf("Expected throughput > 0, got %f", metrics.Throughput)
	}
	
	// Verify it's reasonable (should be around 2 executions per elapsed seconds)
	elapsed := time.Since(start).Seconds()
	expectedThroughput := float64(2) / elapsed
	
	// Allow some tolerance due to timing variations
	tolerance := expectedThroughput * 0.5
	if metrics.Throughput < expectedThroughput-tolerance || 
	   metrics.Throughput > expectedThroughput+tolerance {
		t.Logf("Throughput %f is outside expected range %f ± %f", 
			metrics.Throughput, expectedThroughput, tolerance)
	}
}

// Test utility methods

func TestMetricsCollectorGetAverageEventProcessingTime(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	concreteCollector := collector.(*ConcreteMetricsCollector)
	
	// Record multiple events of same type
	collector.RecordEvent("test_event", 100*time.Millisecond, nil)
	collector.RecordEvent("test_event", 200*time.Millisecond, nil)
	
	avgTime := concreteCollector.GetAverageEventProcessingTime("test_event")
	expected := 150 * time.Millisecond
	
	if avgTime != expected {
		t.Errorf("Expected average time %v, got %v", expected, avgTime)
	}
	
	// Test non-existent event
	avgTime = concreteCollector.GetAverageEventProcessingTime("non_existent")
	if avgTime != 0 {
		t.Errorf("Expected 0 for non-existent event, got %v", avgTime)
	}
}

func TestMetricsCollectorGetTopErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	concreteCollector := collector.(*ConcreteMetricsCollector)
	
	// Record errors with different frequencies
	for i := 0; i < 5; i++ {
		collector.RecordError("parser", "syntax_error")
	}
	for i := 0; i < 3; i++ {
		collector.RecordError("executor", "conversion_error")
	}
	for i := 0; i < 1; i++ {
		collector.RecordError("planner", "planning_error")
	}
	
	topErrors := concreteCollector.GetTopErrors(2)
	
	if len(topErrors) != 2 {
		t.Errorf("Expected 2 top errors, got %d", len(topErrors))
	}
	
	// Should be sorted by frequency (descending)
	if topErrors[0] != "parser.syntax_error" {
		t.Errorf("Expected first error 'parser.syntax_error', got %q", topErrors[0])
	}
	
	if topErrors[1] != "executor.conversion_error" {
		t.Errorf("Expected second error 'executor.conversion_error', got %q", topErrors[1])
	}
}

func TestMetricsCollectorGetHealthScore(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	concreteCollector := collector.(*ConcreteMetricsCollector)
	
	// Test perfect health (no executions)
	score := concreteCollector.GetHealthScore()
	if score != 1.0 {
		t.Errorf("Expected health score 1.0 with no executions, got %f", score)
	}
	
	// Test perfect health with successful executions
	collector.RecordEvent("pipeline_completed", time.Millisecond, nil)
	collector.RecordEvent("pipeline_completed", time.Millisecond, nil)
	
	score = concreteCollector.GetHealthScore()
	if score != 1.0 {
		t.Errorf("Expected health score 1.0 with all successes, got %f", score)
	}
	
	// Test degraded health with failures
	collector.RecordEvent("pipeline_failed", time.Millisecond, nil)
	
	score = concreteCollector.GetHealthScore()
	expectedScore := 2.0 / 3.0 // 2 successes out of 3 total
	if score != expectedScore {
		t.Errorf("Expected health score %f, got %f", expectedScore, score)
	}
	
	// Test health with high error rate
	for i := 0; i < 10; i++ {
		collector.RecordError("component", "error")
	}
	
	score = concreteCollector.GetHealthScore()
	// Score should be reduced due to high error rate
	if score >= expectedScore {
		t.Errorf("Expected health score to decrease due to errors, got %f", score)
	}
}

// Concurrency tests

func TestMetricsCollectorConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	done := make(chan bool, 10)
	
	// Concurrent metric recording
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				collector.RecordEvent("test_event", time.Millisecond, nil)
				collector.RecordError("component", "error")
				collector.RecordRetry("component", true, time.Millisecond)
				_ = collector.GetMetrics()
			}
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify final metrics are consistent
	metrics := collector.GetMetrics()
	if metrics.PipelineExecutions != 1000 {
		t.Errorf("Expected 1000 pipeline executions, got %d", metrics.PipelineExecutions)
	}
	
	if metrics.RetryStats.TotalRetries != 1000 {
		t.Errorf("Expected 1000 retries, got %d", metrics.RetryStats.TotalRetries)
	}
}

// Benchmark tests

func BenchmarkMetricsCollectorRecordEvent(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	duration := time.Millisecond
	metadata := map[string]interface{}{"test": true}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		collector.RecordEvent("test_event", duration, metadata)
	}
}

func BenchmarkMetricsCollectorGetMetrics(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	
	// Pre-populate with some metrics
	for i := 0; i < 100; i++ {
		collector.RecordEvent("test_event", time.Millisecond, nil)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = collector.GetMetrics()
	}
}

func BenchmarkMetricsCollectorConcurrency(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()
	collector := NewMetricsCollector(logger, config)
	concreteCollector := collector.(*ConcreteMetricsCollector)
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			concreteCollector.IncrementConcurrency()
			concreteCollector.DecrementConcurrency()
		}
	})
}