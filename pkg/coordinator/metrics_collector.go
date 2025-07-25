package coordinator

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// MetricsCollector collects and aggregates metrics from all pipeline components
type MetricsCollector interface {
	// Record pipeline event with duration and metadata
	RecordEvent(event string, duration time.Duration, metadata map[string]interface{})
	
	// Record component metrics
	RecordComponent(component string, metrics interface{})
	
	// Get aggregated metrics
	GetMetrics() *CoordinatorMetrics
	
	// Reset all metrics
	Reset()
	
	// Record error for statistics
	RecordError(component string, errorType string)
	
	// Record retry attempt
	RecordRetry(component string, success bool, delay time.Duration)
	
	// Update resource usage metrics
	UpdateResourceUsage(usage *ResourceUsage)
	
	// Record throughput measurement
	RecordThroughput(pipelinesPerSecond float64)
}

// ConcreteMetricsCollector implements MetricsCollector
type ConcreteMetricsCollector struct {
	logger *zap.Logger
	config *Config
	
	// Metrics storage
	mutex                sync.RWMutex
	pipelineExecutions   int64
	totalDuration        int64 // nanoseconds
	successCount         int64
	failureCount         int64
	
	// Component metrics
	componentMetrics map[string]interface{}
	
	// Event metrics
	eventCounts         map[string]int64
	eventProcessingTime map[string]int64 // nanoseconds
	
	// Error metrics
	errorCounts map[string]int64
	
	// Retry statistics
	retryStats *RetryStatistics
	
	// Performance metrics
	latencyMeasurements []time.Duration
	throughputHistory   []float64
	resourceUsage       *ResourceUsage
	
	// Real-time metrics
	startTime           time.Time
	lastUpdateTime      time.Time
	currentConcurrency  int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *zap.Logger, config *Config) MetricsCollector {
	collector := &ConcreteMetricsCollector{
		logger:              logger,
		config:              config,
		componentMetrics:    make(map[string]interface{}),
		eventCounts:         make(map[string]int64),
		eventProcessingTime: make(map[string]int64),
		errorCounts:         make(map[string]int64),
		retryStats: &RetryStatistics{
			RetrysByComponent: make(map[string]int64),
		},
		latencyMeasurements: make([]time.Duration, 0, 1000),
		throughputHistory:   make([]float64, 0, 100),
		resourceUsage:       &ResourceUsage{},
		startTime:           time.Now(),
		lastUpdateTime:      time.Now(),
	}
	
	if config.EnableMetrics {
		logger.Info("metrics collection enabled")
	} else {
		logger.Info("metrics collection disabled")
	}
	
	return collector
}

// RecordEvent records a pipeline event with timing and metadata
func (m *ConcreteMetricsCollector) RecordEvent(event string, duration time.Duration, metadata map[string]interface{}) {
	if !m.config.EnableMetrics {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Update event counts
	m.eventCounts[event]++
	
	// Update processing time
	m.eventProcessingTime[event] += duration.Nanoseconds()
	
	// Record pipeline execution
	if event == "pipeline_completed" || event == "pipeline_failed" {
		atomic.AddInt64(&m.pipelineExecutions, 1)
		atomic.AddInt64(&m.totalDuration, duration.Nanoseconds())
		
		// Record latency
		if len(m.latencyMeasurements) < cap(m.latencyMeasurements) {
			m.latencyMeasurements = append(m.latencyMeasurements, duration)
		} else {
			// Rotate measurements (keep most recent)
			copy(m.latencyMeasurements, m.latencyMeasurements[1:])
			m.latencyMeasurements[len(m.latencyMeasurements)-1] = duration
		}
		
		if event == "pipeline_completed" {
			atomic.AddInt64(&m.successCount, 1)
		} else {
			atomic.AddInt64(&m.failureCount, 1)
		}
	}
	
	m.lastUpdateTime = time.Now()
	
	m.logger.Debug("event recorded",
		zap.String("event", event),
		zap.Duration("duration", duration),
		zap.Any("metadata", metadata))
}

// RecordComponent records metrics from a specific component
func (m *ConcreteMetricsCollector) RecordComponent(component string, metrics interface{}) {
	if !m.config.EnableMetrics {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.componentMetrics[component] = metrics
	m.lastUpdateTime = time.Now()
	
	m.logger.Debug("component metrics recorded",
		zap.String("component", component),
		zap.Any("metrics", metrics))
}

// GetMetrics returns the aggregated coordinator metrics
func (m *ConcreteMetricsCollector) GetMetrics() *CoordinatorMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	executions := atomic.LoadInt64(&m.pipelineExecutions)
	totalDur := atomic.LoadInt64(&m.totalDuration)
	successes := atomic.LoadInt64(&m.successCount)
	
	var avgDuration time.Duration
	var successRate float64
	var throughput float64
	
	if executions > 0 {
		avgDuration = time.Duration(totalDur / executions)
		successRate = float64(successes) / float64(executions)
		
		// Calculate throughput (pipelines per second)
		elapsed := time.Since(m.startTime)
		if elapsed.Seconds() > 0 {
			throughput = float64(executions) / elapsed.Seconds()
		}
	}
	
	metrics := &CoordinatorMetrics{
		PipelineExecutions: executions,
		TotalDuration:      time.Duration(totalDur),
		AverageDuration:    avgDuration,
		SuccessRate:        successRate,
		ComponentMetrics:   m.copyComponentMetrics(),
		ResourceUsage:      m.copyResourceUsage(),
		EventCounts:        m.copyEventCounts(),
		EventProcessingTime: m.copyEventProcessingTime(),
		ErrorCounts:        m.copyErrorCounts(),
		RetryStats:         m.copyRetryStats(),
		Throughput:         throughput,
		Latency:            m.calculateLatencyMetrics(),
		ConcurrencyLevel:   float64(atomic.LoadInt64(&m.currentConcurrency)),
	}
	
	return metrics
}

// Reset clears all metrics
func (m *ConcreteMetricsCollector) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	atomic.StoreInt64(&m.pipelineExecutions, 0)
	atomic.StoreInt64(&m.totalDuration, 0)
	atomic.StoreInt64(&m.successCount, 0)
	atomic.StoreInt64(&m.failureCount, 0)
	atomic.StoreInt64(&m.currentConcurrency, 0)
	
	// Clear maps
	for event := range m.eventCounts {
		m.eventCounts[event] = 0
	}
	
	for event := range m.eventProcessingTime {
		m.eventProcessingTime[event] = 0
	}
	
	for errorType := range m.errorCounts {
		m.errorCounts[errorType] = 0
	}
	
	// Clear component metrics
	m.componentMetrics = make(map[string]interface{})
	
	// Reset retry stats
	m.retryStats = &RetryStatistics{
		RetrysByComponent: make(map[string]int64),
	}
	
	// Clear history
	m.latencyMeasurements = m.latencyMeasurements[:0]
	m.throughputHistory = m.throughputHistory[:0]
	
	m.startTime = time.Now()
	m.lastUpdateTime = time.Now()
	
	m.logger.Debug("metrics collector reset")
}

// RecordError records an error for statistics
func (m *ConcreteMetricsCollector) RecordError(component string, errorType string) {
	if !m.config.EnableMetrics {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	key := component + "." + errorType
	m.errorCounts[key]++
	m.lastUpdateTime = time.Now()
}

// RecordRetry records a retry attempt
func (m *ConcreteMetricsCollector) RecordRetry(component string, success bool, delay time.Duration) {
	if !m.config.EnableMetrics {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.retryStats.TotalRetries++
	m.retryStats.RetrysByComponent[component]++
	
	if success {
		m.retryStats.SuccessfulRetries++
	} else {
		m.retryStats.FailedRetries++
	}
	
	// Update average retry delay
	if m.retryStats.TotalRetries > 0 {
		totalDelay := time.Duration(m.retryStats.AverageRetryDelay.Nanoseconds()*int64(m.retryStats.TotalRetries-1)) + delay
		m.retryStats.AverageRetryDelay = time.Duration(totalDelay.Nanoseconds() / int64(m.retryStats.TotalRetries))
	}
	
	m.lastUpdateTime = time.Now()
}

// UpdateResourceUsage updates resource usage metrics
func (m *ConcreteMetricsCollector) UpdateResourceUsage(usage *ResourceUsage) {
	if !m.config.EnableMetrics {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.resourceUsage = usage
	m.lastUpdateTime = time.Now()
}

// RecordThroughput records a throughput measurement
func (m *ConcreteMetricsCollector) RecordThroughput(pipelinesPerSecond float64) {
	if !m.config.EnableMetrics {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if len(m.throughputHistory) < cap(m.throughputHistory) {
		m.throughputHistory = append(m.throughputHistory, pipelinesPerSecond)
	} else {
		// Rotate history (keep most recent)
		copy(m.throughputHistory, m.throughputHistory[1:])
		m.throughputHistory[len(m.throughputHistory)-1] = pipelinesPerSecond
	}
	
	m.lastUpdateTime = time.Now()
}

// Private methods

func (m *ConcreteMetricsCollector) copyComponentMetrics() map[string]interface{} {
	result := make(map[string]interface{})
	for component, metrics := range m.componentMetrics {
		result[component] = metrics
	}
	return result
}

func (m *ConcreteMetricsCollector) copyResourceUsage() *ResourceUsage {
	if m.resourceUsage == nil {
		return &ResourceUsage{}
	}
	
	usage := *m.resourceUsage
	return &usage
}

func (m *ConcreteMetricsCollector) copyEventCounts() map[string]int64 {
	result := make(map[string]int64)
	for event, count := range m.eventCounts {
		result[event] = count
	}
	return result
}

func (m *ConcreteMetricsCollector) copyEventProcessingTime() map[string]time.Duration {
	result := make(map[string]time.Duration)
	for event, timeNs := range m.eventProcessingTime {
		result[event] = time.Duration(timeNs)
	}
	return result
}

func (m *ConcreteMetricsCollector) copyErrorCounts() map[string]int64 {
	result := make(map[string]int64)
	for errorType, count := range m.errorCounts {
		result[errorType] = count
	}
	return result
}

func (m *ConcreteMetricsCollector) copyRetryStats() *RetryStatistics {
	stats := &RetryStatistics{
		TotalRetries:      m.retryStats.TotalRetries,
		SuccessfulRetries: m.retryStats.SuccessfulRetries,
		FailedRetries:     m.retryStats.FailedRetries,
		AverageRetryDelay: m.retryStats.AverageRetryDelay,
		RetrysByComponent: make(map[string]int64),
	}
	
	for component, retries := range m.retryStats.RetrysByComponent {
		stats.RetrysByComponent[component] = retries
	}
	
	return stats
}

func (m *ConcreteMetricsCollector) calculateLatencyMetrics() *LatencyMetrics {
	if len(m.latencyMeasurements) == 0 {
		return &LatencyMetrics{}
	}
	
	// Sort measurements for percentile calculation
	measurements := make([]time.Duration, len(m.latencyMeasurements))
	copy(measurements, m.latencyMeasurements)
	
	// Simple bubble sort for demonstration (use sort.Slice in production)
	for i := 0; i < len(measurements); i++ {
		for j := i + 1; j < len(measurements); j++ {
			if measurements[i] > measurements[j] {
				measurements[i], measurements[j] = measurements[j], measurements[i]
			}
		}
	}
	
	n := len(measurements)
	
	latency := &LatencyMetrics{
		Min:  measurements[0],
		Max:  measurements[n-1],
		P50:  measurements[n*50/100],
		P90:  measurements[n*90/100],
		P95:  measurements[n*95/100],
		P99:  measurements[n*99/100],
	}
	
	// Calculate mean
	var total int64
	for _, duration := range measurements {
		total += duration.Nanoseconds()
	}
	latency.Mean = time.Duration(total / int64(n))
	
	return latency
}

// Concurrency tracking methods

// IncrementConcurrency increments the current concurrency level
func (m *ConcreteMetricsCollector) IncrementConcurrency() {
	atomic.AddInt64(&m.currentConcurrency, 1)
}

// DecrementConcurrency decrements the current concurrency level  
func (m *ConcreteMetricsCollector) DecrementConcurrency() {
	atomic.AddInt64(&m.currentConcurrency, -1)
}

// SetConcurrency sets the current concurrency level
func (m *ConcreteMetricsCollector) SetConcurrency(level int64) {
	atomic.StoreInt64(&m.currentConcurrency, level)
}

// Utility methods for metrics analysis

// GetAverageEventProcessingTime returns average processing time for an event
func (m *ConcreteMetricsCollector) GetAverageEventProcessingTime(event string) time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	count := m.eventCounts[event]
	totalTime := m.eventProcessingTime[event]
	
	if count == 0 {
		return 0
	}
	
	return time.Duration(totalTime / count)
}

// GetTopErrors returns the most frequent error types
func (m *ConcreteMetricsCollector) GetTopErrors(limit int) []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Create sorted list of errors by count
	type errorCount struct {
		errorType string
		count     int64
	}
	
	var errors []errorCount
	for errorType, count := range m.errorCounts {
		errors = append(errors, errorCount{errorType, count})
	}
	
	// Simple sort by count (descending)
	for i := 0; i < len(errors); i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i].count < errors[j].count {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}
	
	// Return top errors
	var result []string
	for i := 0; i < limit && i < len(errors); i++ {
		result = append(result, errors[i].errorType)
	}
	
	return result
}

// GetHealthScore calculates a health score based on success rate and error rate
func (m *ConcreteMetricsCollector) GetHealthScore() float64 {
	metrics := m.GetMetrics()
	
	if metrics.PipelineExecutions == 0 {
		return 1.0 // Perfect score with no executions
	}
	
	// Base score on success rate
	healthScore := metrics.SuccessRate
	
	// Penalize for high error rates
	totalErrors := int64(0)
	for _, count := range metrics.ErrorCounts {
		totalErrors += count
	}
	
	errorRate := float64(totalErrors) / float64(metrics.PipelineExecutions)
	if errorRate > 0.1 { // More than 10% error rate
		healthScore *= (1.0 - errorRate)
	}
	
	// Ensure score is between 0 and 1
	if healthScore < 0 {
		healthScore = 0
	}
	if healthScore > 1 {
		healthScore = 1
	}
	
	return healthScore
}