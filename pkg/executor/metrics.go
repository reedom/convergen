package executor

import (
	"sync"
	"time"
)

// ExecutionMetrics provides comprehensive metrics collection for the execution system
type ExecutionMetrics struct {
	enabled bool
	mutex   sync.RWMutex

	// Plan-level metrics
	PlansExecuted         int64         `json:"plans_executed"`
	PlansSuccessful       int64         `json:"plans_successful"`
	PlansFailed           int64         `json:"plans_failed"`
	TotalPlanDuration     time.Duration `json:"total_plan_duration"`
	AveragePlanDuration   time.Duration `json:"average_plan_duration"`
	ParallelizationRatio  float64       `json:"parallelization_ratio"`
	EstimatedSpeedupRatio float64       `json:"estimated_speedup_ratio"`

	// Batch-level metrics
	BatchesExecuted       int64         `json:"batches_executed"`
	BatchesSuccessful     int64         `json:"batches_successful"`
	BatchesFailed         int64         `json:"batches_failed"`
	TotalBatchDuration    time.Duration `json:"total_batch_duration"`
	AverageBatchDuration  time.Duration `json:"average_batch_duration"`
	AverageBatchSize      float64       `json:"average_batch_size"`
	MaxConcurrentBatches  int           `json:"max_concurrent_batches"`

	// Field-level metrics
	FieldsProcessed       int64         `json:"fields_processed"`
	FieldsSuccessful      int64         `json:"fields_successful"`
	FieldsFailed          int64         `json:"fields_failed"`
	TotalFieldDuration    time.Duration `json:"total_field_duration"`
	AverageFieldDuration  time.Duration `json:"average_field_duration"`
	ThroughputPerSecond   float64       `json:"throughput_per_second"`
	MaxConcurrentFields   int           `json:"max_concurrent_fields"`

	// Error metrics
	TotalErrors           int64                    `json:"total_errors"`
	ErrorsByType          map[string]int64         `json:"errors_by_type"`
	RetryAttempts         int64                    `json:"retry_attempts"`
	RetriesSuccessful     int64                    `json:"retries_successful"`
	RetriesFailed         int64                    `json:"retries_failed"`

	// Resource metrics
	PeakMemoryUsageMB     int                      `json:"peak_memory_usage_mb"`
	AverageMemoryUsageMB  int                      `json:"average_memory_usage_mb"`
	PeakWorkerCount       int                      `json:"peak_worker_count"`
	AverageWorkerCount    float64                  `json:"average_worker_count"`
	ResourceEfficiency    float64                  `json:"resource_efficiency"`

	// Performance metrics
	CPUTimeTotal          time.Duration            `json:"cpu_time_total"`
	WaitTimeTotal         time.Duration            `json:"wait_time_total"`
	IdleTimeTotal         time.Duration            `json:"idle_time_total"`
	QueueWaitTimeTotal    time.Duration            `json:"queue_wait_time_total"`

	// Strategy metrics
	StrategyUsage         map[string]int64         `json:"strategy_usage"`
	StrategyPerformance   map[string]time.Duration `json:"strategy_performance"`
	StrategyErrors        map[string]int64         `json:"strategy_errors"`

	// Time-series data for trending
	MetricsHistory        []MetricsSnapshot        `json:"metrics_history"`
	StartTime             time.Time                `json:"start_time"`
	LastUpdated           time.Time                `json:"last_updated"`
}

// MetricsSnapshot captures metrics at a specific point in time
type MetricsSnapshot struct {
	Timestamp           time.Time `json:"timestamp"`
	FieldsPerSecond     float64   `json:"fields_per_second"`
	BatchesPerSecond    float64   `json:"batches_per_second"`
	ErrorRate           float64   `json:"error_rate"`
	MemoryUsageMB       int       `json:"memory_usage_mb"`
	ActiveWorkers       int       `json:"active_workers"`
	QueueSize           int       `json:"queue_size"`
	MemoryPressure      float64   `json:"memory_pressure"`
	CPUUsagePercent     float64   `json:"cpu_usage_percent"`
}

// NewExecutionMetrics creates a new metrics collector
func NewExecutionMetrics(enabled bool) *ExecutionMetrics {
	return &ExecutionMetrics{
		enabled:             enabled,
		ErrorsByType:        make(map[string]int64),
		StrategyUsage:       make(map[string]int64),
		StrategyPerformance: make(map[string]time.Duration),
		StrategyErrors:      make(map[string]int64),
		MetricsHistory:      make([]MetricsSnapshot, 0, 1000),
		StartTime:           time.Now(),
		LastUpdated:         time.Now(),
	}
}

// RecordPlanExecution records metrics for a completed plan execution
func (em *ExecutionMetrics) RecordPlanExecution(result *ExecutionResult) {
	if !em.enabled {
		return
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.PlansExecuted++
	if result.Success {
		em.PlansSuccessful++
	} else {
		em.PlansFailed++
	}

	em.TotalPlanDuration += result.Duration
	em.AveragePlanDuration = em.TotalPlanDuration / time.Duration(em.PlansExecuted)

	// Calculate parallelization metrics
	if result.Metrics != nil {
		em.ParallelizationRatio = result.Metrics.ParallelizationRatio
		em.EstimatedSpeedupRatio = result.Metrics.EstimatedSpeedupRatio
	}

	// Record errors
	for _, execError := range result.Errors {
		em.TotalErrors++
		em.ErrorsByType[execError.ErrorType]++
	}

	em.LastUpdated = time.Now()
}

// RecordBatchExecution records metrics for a completed batch execution
func (em *ExecutionMetrics) RecordBatchExecution(result *BatchResult) {
	if !em.enabled {
		return
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.BatchesExecuted++
	if result.Success {
		em.BatchesSuccessful++
	} else {
		em.BatchesFailed++
	}

	em.TotalBatchDuration += result.Duration
	em.AverageBatchDuration = em.TotalBatchDuration / time.Duration(em.BatchesExecuted)

	// Update batch size metrics
	batchSize := float64(len(result.FieldResults))
	if em.BatchesExecuted == 1 {
		em.AverageBatchSize = batchSize
	} else {
		em.AverageBatchSize = (em.AverageBatchSize*float64(em.BatchesExecuted-1) + batchSize) / float64(em.BatchesExecuted)
	}

	// Track peak concurrent batches
	if result.WorkersUsed > em.MaxConcurrentBatches {
		em.MaxConcurrentBatches = result.WorkersUsed
	}

	// Update memory metrics
	if result.MemoryUsedMB > em.PeakMemoryUsageMB {
		em.PeakMemoryUsageMB = result.MemoryUsedMB
	}

	// Record errors
	for _, execError := range result.Errors {
		em.TotalErrors++
		em.ErrorsByType[execError.ErrorType]++
	}

	em.LastUpdated = time.Now()
}

// RecordFieldExecution records metrics for a completed field execution
func (em *ExecutionMetrics) RecordFieldExecution(result *FieldResult) {
	if !em.enabled {
		return
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.FieldsProcessed++
	if result.Success {
		em.FieldsSuccessful++
	} else {
		em.FieldsFailed++
	}

	em.TotalFieldDuration += result.Duration
	em.AverageFieldDuration = em.TotalFieldDuration / time.Duration(em.FieldsProcessed)

	// Calculate throughput
	if em.FieldsProcessed > 0 {
		totalTimeSeconds := time.Since(em.StartTime).Seconds()
		if totalTimeSeconds > 0 {
			em.ThroughputPerSecond = float64(em.FieldsProcessed) / totalTimeSeconds
		}
	}

	// Update strategy metrics
	strategy := result.StrategyUsed
	em.StrategyUsage[strategy]++
	em.StrategyPerformance[strategy] += result.Duration

	// Record errors
	if result.Error != nil {
		em.TotalErrors++
		em.ErrorsByType[result.Error.ErrorType]++
		em.StrategyErrors[strategy]++
	}

	// Record retry metrics
	if result.RetryCount > 0 {
		em.RetryAttempts += int64(result.RetryCount)
		if result.Success {
			em.RetriesSuccessful++
		} else {
			em.RetriesFailed++
		}
	}

	em.LastUpdated = time.Now()
}

// UpdateResourceMetrics updates resource-related metrics
func (em *ExecutionMetrics) UpdateResourceMetrics(resourceMetrics *ResourceMetrics) {
	if !em.enabled {
		return
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Update peak values
	if resourceMetrics.MemoryUsedMB > em.PeakMemoryUsageMB {
		em.PeakMemoryUsageMB = resourceMetrics.MemoryUsedMB
	}

	if resourceMetrics.WorkersTotal > em.PeakWorkerCount {
		em.PeakWorkerCount = resourceMetrics.WorkersTotal
	}

	// Update averages
	if em.AverageMemoryUsageMB == 0 {
		em.AverageMemoryUsageMB = resourceMetrics.MemoryUsedMB
	} else {
		em.AverageMemoryUsageMB = (em.AverageMemoryUsageMB + resourceMetrics.MemoryUsedMB) / 2
	}

	if em.AverageWorkerCount == 0 {
		em.AverageWorkerCount = float64(resourceMetrics.WorkersTotal)
	} else {
		em.AverageWorkerCount = (em.AverageWorkerCount + float64(resourceMetrics.WorkersTotal)) / 2
	}

	em.LastUpdated = time.Now()
}

// RecordExecutorStatus records executor status metrics
func (em *ExecutionMetrics) RecordExecutorStatus(status *ExecutorStatus) {
	if !em.enabled {
		return
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Create metrics snapshot
	snapshot := MetricsSnapshot{
		Timestamp:        time.Now(),
		FieldsPerSecond:  em.ThroughputPerSecond,
		BatchesPerSecond: em.calculateBatchesPerSecond(),
		ErrorRate:        em.calculateErrorRate(),
		ActiveWorkers:    len(status.ActiveBatches),
		QueueSize:        len(status.QueuedBatches),
	}

	// Add to history
	em.MetricsHistory = append(em.MetricsHistory, snapshot)
	
	// Keep only last 1000 snapshots
	if len(em.MetricsHistory) > 1000 {
		em.MetricsHistory = em.MetricsHistory[1:]
	}

	em.LastUpdated = time.Now()
}

// GetSnapshot returns a copy of current metrics
func (em *ExecutionMetrics) GetSnapshot() *ExecutionMetrics {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	// Create a copy of the metrics
	snapshot := &ExecutionMetrics{
		enabled:               em.enabled,
		PlansExecuted:         em.PlansExecuted,
		PlansSuccessful:       em.PlansSuccessful,
		PlansFailed:           em.PlansFailed,
		TotalPlanDuration:     em.TotalPlanDuration,
		AveragePlanDuration:   em.AveragePlanDuration,
		ParallelizationRatio:  em.ParallelizationRatio,
		EstimatedSpeedupRatio: em.EstimatedSpeedupRatio,
		BatchesExecuted:       em.BatchesExecuted,
		BatchesSuccessful:     em.BatchesSuccessful,
		BatchesFailed:         em.BatchesFailed,
		TotalBatchDuration:    em.TotalBatchDuration,
		AverageBatchDuration:  em.AverageBatchDuration,
		AverageBatchSize:      em.AverageBatchSize,
		MaxConcurrentBatches:  em.MaxConcurrentBatches,
		FieldsProcessed:       em.FieldsProcessed,
		FieldsSuccessful:      em.FieldsSuccessful,
		FieldsFailed:          em.FieldsFailed,
		TotalFieldDuration:    em.TotalFieldDuration,
		AverageFieldDuration:  em.AverageFieldDuration,
		ThroughputPerSecond:   em.ThroughputPerSecond,
		MaxConcurrentFields:   em.MaxConcurrentFields,
		TotalErrors:           em.TotalErrors,
		RetryAttempts:         em.RetryAttempts,
		RetriesSuccessful:     em.RetriesSuccessful,
		RetriesFailed:         em.RetriesFailed,
		PeakMemoryUsageMB:     em.PeakMemoryUsageMB,
		AverageMemoryUsageMB:  em.AverageMemoryUsageMB,
		PeakWorkerCount:       em.PeakWorkerCount,
		AverageWorkerCount:    em.AverageWorkerCount,
		ResourceEfficiency:    em.ResourceEfficiency,
		CPUTimeTotal:          em.CPUTimeTotal,
		WaitTimeTotal:         em.WaitTimeTotal,
		IdleTimeTotal:         em.IdleTimeTotal,
		QueueWaitTimeTotal:    em.QueueWaitTimeTotal,
		StartTime:             em.StartTime,
		LastUpdated:           em.LastUpdated,
	}

	// Copy maps
	snapshot.ErrorsByType = make(map[string]int64)
	for k, v := range em.ErrorsByType {
		snapshot.ErrorsByType[k] = v
	}

	snapshot.StrategyUsage = make(map[string]int64)
	for k, v := range em.StrategyUsage {
		snapshot.StrategyUsage[k] = v
	}

	snapshot.StrategyPerformance = make(map[string]time.Duration)
	for k, v := range em.StrategyPerformance {
		snapshot.StrategyPerformance[k] = v
	}

	snapshot.StrategyErrors = make(map[string]int64)
	for k, v := range em.StrategyErrors {
		snapshot.StrategyErrors[k] = v
	}

	// Copy history
	snapshot.MetricsHistory = make([]MetricsSnapshot, len(em.MetricsHistory))
	copy(snapshot.MetricsHistory, em.MetricsHistory)

	return snapshot
}

// Clone creates a deep copy of the metrics
func (em *ExecutionMetrics) Clone() *ExecutionMetrics {
	return em.GetSnapshot()
}

// Reset resets all metrics to zero
func (em *ExecutionMetrics) Reset() {
	if !em.enabled {
		return
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Reset all counters and metrics
	em.PlansExecuted = 0
	em.PlansSuccessful = 0
	em.PlansFailed = 0
	em.TotalPlanDuration = 0
	em.AveragePlanDuration = 0
	em.ParallelizationRatio = 0
	em.EstimatedSpeedupRatio = 0
	em.BatchesExecuted = 0
	em.BatchesSuccessful = 0
	em.BatchesFailed = 0
	em.TotalBatchDuration = 0
	em.AverageBatchDuration = 0
	em.AverageBatchSize = 0
	em.MaxConcurrentBatches = 0
	em.FieldsProcessed = 0
	em.FieldsSuccessful = 0
	em.FieldsFailed = 0
	em.TotalFieldDuration = 0
	em.AverageFieldDuration = 0
	em.ThroughputPerSecond = 0
	em.MaxConcurrentFields = 0
	em.TotalErrors = 0
	em.RetryAttempts = 0
	em.RetriesSuccessful = 0
	em.RetriesFailed = 0
	em.PeakMemoryUsageMB = 0
	em.AverageMemoryUsageMB = 0
	em.PeakWorkerCount = 0
	em.AverageWorkerCount = 0
	em.ResourceEfficiency = 0
	em.CPUTimeTotal = 0
	em.WaitTimeTotal = 0
	em.IdleTimeTotal = 0
	em.QueueWaitTimeTotal = 0

	// Clear maps
	em.ErrorsByType = make(map[string]int64)
	em.StrategyUsage = make(map[string]int64)
	em.StrategyPerformance = make(map[string]time.Duration)
	em.StrategyErrors = make(map[string]int64)
	
	// Clear history
	em.MetricsHistory = make([]MetricsSnapshot, 0, 1000)
	
	em.StartTime = time.Now()
	em.LastUpdated = time.Now()
}

// Helper methods

func (em *ExecutionMetrics) calculateBatchesPerSecond() float64 {
	if em.BatchesExecuted == 0 {
		return 0
	}
	
	totalTimeSeconds := time.Since(em.StartTime).Seconds()
	if totalTimeSeconds > 0 {
		return float64(em.BatchesExecuted) / totalTimeSeconds
	}
	
	return 0
}

func (em *ExecutionMetrics) calculateErrorRate() float64 {
	if em.FieldsProcessed == 0 {
		return 0
	}
	
	return float64(em.FieldsFailed) / float64(em.FieldsProcessed)
}

// GetPerformanceSummary returns a summary of key performance metrics
func (em *ExecutionMetrics) GetPerformanceSummary() map[string]interface{} {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	return map[string]interface{}{
		"uptime_seconds":         time.Since(em.StartTime).Seconds(),
		"plans_executed":         em.PlansExecuted,
		"success_rate":          em.calculateSuccessRate(),
		"throughput_per_second":  em.ThroughputPerSecond,
		"average_field_duration_ms": em.AverageFieldDuration.Milliseconds(),
		"error_rate":            em.calculateErrorRate(),
		"peak_memory_mb":        em.PeakMemoryUsageMB,
		"peak_workers":          em.PeakWorkerCount,
		"total_errors":          em.TotalErrors,
		"retry_success_rate":    em.calculateRetrySuccessRate(),
	}
}

func (em *ExecutionMetrics) calculateSuccessRate() float64 {
	if em.FieldsProcessed == 0 {
		return 0
	}
	return float64(em.FieldsSuccessful) / float64(em.FieldsProcessed)
}

func (em *ExecutionMetrics) calculateRetrySuccessRate() float64 {
	totalRetries := em.RetriesSuccessful + em.RetriesFailed
	if totalRetries == 0 {
		return 0
	}
	return float64(em.RetriesSuccessful) / float64(totalRetries)
}