package executor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourcePool manages worker pools, memory allocation, and resource throttling
type ResourcePool struct {
	config  *ExecutorConfig
	logger  *zap.Logger
	metrics *ExecutionMetrics

	// Worker management
	workers        map[string]*Worker
	availableWorkers chan *Worker
	workerQueue    chan *FieldExecution
	
	// Resource tracking
	memoryUsedMB    int
	memoryLimitMB   int
	activeJobs      int
	maxJobs         int
	
	// State management
	shutdown  chan struct{}
	wg        sync.WaitGroup
	mutex     sync.RWMutex
	
	// Adaptive management
	lastAdjustment  time.Time
	performanceHistory []float64
}

// NewResourcePool creates a new resource pool
func NewResourcePool(config *ExecutorConfig, logger *zap.Logger, metrics *ExecutionMetrics) *ResourcePool {
	pool := &ResourcePool{
		config:          config,
		logger:          logger,
		metrics:         metrics,
		workers:         make(map[string]*Worker),
		availableWorkers: make(chan *Worker, config.MaxWorkers),
		workerQueue:     make(chan *FieldExecution, config.PipelineBufferSize),
		memoryLimitMB:   config.MaxMemoryMB,
		maxJobs:         config.MaxConcurrentJobs,
		shutdown:        make(chan struct{}),
		lastAdjustment:  time.Now(),
		performanceHistory: make([]float64, 0, 100),
	}

	// Start with minimum workers
	pool.startInitialWorkers()
	
	// Start background management tasks
	pool.startResourceMonitoring()
	pool.startAdaptiveManagement()
	
	logger.Info("resource pool initialized",
		zap.Int("initial_workers", config.MinWorkers),
		zap.Int("max_workers", config.MaxWorkers),
		zap.Int("memory_limit_mb", config.MaxMemoryMB))

	return pool
}

// SetLimits updates resource limits dynamically
func (rp *ResourcePool) SetLimits(maxWorkers, maxMemoryMB int) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()
	
	oldMaxWorkers := rp.config.MaxWorkers
	rp.config.MaxWorkers = maxWorkers
	rp.memoryLimitMB = maxMemoryMB
	
	rp.logger.Info("resource limits updated",
		zap.Int("old_max_workers", oldMaxWorkers),
		zap.Int("new_max_workers", maxWorkers),
		zap.Int("memory_limit_mb", maxMemoryMB))
	
	// Adjust worker pool if necessary
	if maxWorkers < oldMaxWorkers {
		rp.scaleDownWorkers(maxWorkers)
	}
}

// SubmitJob submits a field execution job to the resource pool
func (rp *ResourcePool) SubmitJob(ctx context.Context, job *FieldExecution) error {
	// Check resource limits
	if err := rp.checkResourceLimits(); err != nil {
		return fmt.Errorf("resource limit exceeded: %w", err)
	}
	
	select {
	case rp.workerQueue <- job:
		rp.mutex.Lock()
		rp.activeJobs++
		rp.mutex.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("worker queue is full")
	}
}

// GetAvailableWorkers returns the number of available workers
func (rp *ResourcePool) GetAvailableWorkers() int {
	return len(rp.availableWorkers)
}

// GetMetrics returns current resource pool metrics
func (rp *ResourcePool) GetMetrics() *ResourceMetrics {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	totalWorkers := len(rp.workers)
	availableWorkers := len(rp.availableWorkers)
	activeWorkers := totalWorkers - availableWorkers
	
	memoryPressure := float64(rp.memoryUsedMB) / float64(rp.memoryLimitMB)
	if memoryPressure > 1.0 {
		memoryPressure = 1.0
	}
	
	return &ResourceMetrics{
		WorkersActive:     activeWorkers,
		WorkersIdle:       availableWorkers,
		WorkersTotal:      totalWorkers,
		MemoryUsedMB:      int(m.Alloc / 1024 / 1024),
		MemoryAvailableMB: rp.memoryLimitMB - rp.memoryUsedMB,
		MemoryPressure:    memoryPressure,
		QueuedJobs:        len(rp.workerQueue),
		LastUpdated:       time.Now(),
	}
}

// Shutdown gracefully shuts down the resource pool
func (rp *ResourcePool) Shutdown(ctx context.Context) error {
	rp.logger.Info("shutting down resource pool")
	
	// Signal shutdown
	close(rp.shutdown)
	
	// Stop accepting new jobs
	close(rp.workerQueue)
	
	// Shutdown all workers
	rp.mutex.Lock()
	for _, worker := range rp.workers {
		close(worker.Done)
	}
	rp.mutex.Unlock()
	
	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		rp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		rp.logger.Info("resource pool shutdown completed")
		return nil
	case <-ctx.Done():
		rp.logger.Warn("resource pool shutdown timed out")
		return ctx.Err()
	}
}

// Worker management

func (rp *ResourcePool) startInitialWorkers() {
	for i := 0; i < rp.config.MinWorkers; i++ {
		worker := rp.createWorker()
		rp.startWorker(worker)
	}
}

func (rp *ResourcePool) createWorker() *Worker {
	workerID := fmt.Sprintf("worker_%d_%d", time.Now().Unix(), len(rp.workers))
	
	worker := &Worker{
		ID:        workerID,
		State:     WorkerStateIdle,
		StartTime: time.Now(),
		LastActivity: time.Now(),
		TasksHandled: 0,
		Metrics:   &WorkerMetrics{
			WorkerID: workerID,
		},
		Channel: make(chan *FieldExecution, 1),
		Done:    make(chan struct{}),
	}
	
	rp.mutex.Lock()
	rp.workers[workerID] = worker
	rp.mutex.Unlock()
	
	return worker
}

func (rp *ResourcePool) startWorker(worker *Worker) {
	rp.wg.Add(1)
	go rp.workerLoop(worker)
}

func (rp *ResourcePool) workerLoop(worker *Worker) {
	defer rp.wg.Done()
	defer rp.removeWorker(worker.ID)
	
	rp.logger.Debug("worker started", zap.String("worker_id", worker.ID))
	
	idleTimer := time.NewTimer(rp.config.WorkerIdleTimeout)
	defer idleTimer.Stop()
	
	for {
		// Make worker available for work
		select {
		case rp.availableWorkers <- worker:
		case <-worker.Done:
			return
		case <-rp.shutdown:
			return
		}
		
		// Wait for job or shutdown
		select {
		case job := <-rp.workerQueue:
			if job == nil {
				return // Channel closed
			}
			
			// Reset idle timer
			if !idleTimer.Stop() {
				<-idleTimer.C
			}
			idleTimer.Reset(rp.config.WorkerIdleTimeout)
			
			// Execute job
			rp.executeJob(worker, job)
			
		case <-idleTimer.C:
			// Worker has been idle too long
			if rp.shouldRemoveIdleWorker() {
				rp.logger.Debug("removing idle worker", zap.String("worker_id", worker.ID))
				return
			}
			idleTimer.Reset(rp.config.WorkerIdleTimeout)
			
		case <-worker.Done:
			return
			
		case <-rp.shutdown:
			return
		}
	}
}

func (rp *ResourcePool) executeJob(worker *Worker, job *FieldExecution) {
	startTime := time.Now()
	
	// Update worker state
	worker.State = WorkerStateActive
	worker.CurrentTask = job
	worker.LastActivity = startTime
	
	rp.logger.Debug("worker executing job",
		zap.String("worker_id", worker.ID),
		zap.String("field_id", job.ID))
	
	// Track memory usage
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	
	// Execute the job (this would integrate with field executor)
	// For now, simulate work
	time.Sleep(10 * time.Millisecond)
	
	// Track memory usage after execution
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	
	// Update worker metrics
	duration := time.Since(startTime)
	worker.TasksHandled++
	worker.Metrics.TasksCompleted++
	worker.Metrics.TotalActiveTime += duration
	worker.Metrics.LastTaskTime = time.Now()
	worker.Metrics.AverageTaskTime = worker.Metrics.TotalActiveTime / time.Duration(worker.TasksHandled)
	
	memUsedMB := int((endMem.Alloc - startMem.Alloc) / 1024 / 1024)
	worker.Metrics.CurrentMemoryMB = memUsedMB
	if memUsedMB > worker.Metrics.PeakMemoryMB {
		worker.Metrics.PeakMemoryMB = memUsedMB
	}
	
	// Update resource usage
	rp.mutex.Lock()
	rp.activeJobs--
	rp.memoryUsedMB += memUsedMB
	rp.mutex.Unlock()
	
	// Reset worker state
	worker.State = WorkerStateIdle
	worker.CurrentTask = nil
	
	rp.logger.Debug("worker completed job",
		zap.String("worker_id", worker.ID),
		zap.String("field_id", job.ID),
		zap.Duration("duration", duration))
}

func (rp *ResourcePool) shouldRemoveIdleWorker() bool {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()
	
	// Don't remove if we're at minimum workers
	if len(rp.workers) <= rp.config.MinWorkers {
		return false
	}
	
	// Remove if we have low job queue utilization
	queueUtilization := float64(len(rp.workerQueue)) / float64(cap(rp.workerQueue))
	return queueUtilization < 0.1
}

func (rp *ResourcePool) removeWorker(workerID string) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()
	delete(rp.workers, workerID)
}

func (rp *ResourcePool) scaleDownWorkers(targetCount int) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()
	
	currentCount := len(rp.workers)
	if currentCount <= targetCount {
		return
	}
	
	workersToRemove := currentCount - targetCount
	removedCount := 0
	
	for workerID, worker := range rp.workers {
		if removedCount >= workersToRemove {
			break
		}
		
		if worker.State == WorkerStateIdle {
			close(worker.Done)
			removedCount++
			rp.logger.Debug("scaling down worker", zap.String("worker_id", workerID))
		}
	}
}

// Resource monitoring and management

func (rp *ResourcePool) startResourceMonitoring() {
	rp.wg.Add(1)
	go func() {
		defer rp.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				rp.monitorResources()
			case <-rp.shutdown:
				return
			}
		}
	}()
}

func (rp *ResourcePool) monitorResources() {
	metrics := rp.GetMetrics()
	
	// Check memory pressure
	if metrics.MemoryPressure > rp.config.MemoryThreshold {
		rp.logger.Warn("high memory pressure detected",
			zap.Float64("pressure", metrics.MemoryPressure),
			zap.Int("memory_used_mb", metrics.MemoryUsedMB),
			zap.Int("memory_limit_mb", rp.memoryLimitMB))
		
		// Trigger memory pressure response
		rp.handleMemoryPressure()
	}
	
	// Update metrics
	rp.metrics.UpdateResourceMetrics(metrics)
}

func (rp *ResourcePool) handleMemoryPressure() {
	// Force garbage collection
	runtime.GC()
	
	// Scale down workers if possible
	rp.mutex.RLock()
	workerCount := len(rp.workers)
	minWorkers := rp.config.MinWorkers
	rp.mutex.RUnlock()
	
	if workerCount > minWorkers {
		targetWorkers := max(minWorkers, workerCount*3/4)
		rp.scaleDownWorkers(targetWorkers)
	}
}

func (rp *ResourcePool) startAdaptiveManagement() {
	if !rp.config.AdaptiveConcurrency {
		return
	}
	
	rp.wg.Add(1)
	go func() {
		defer rp.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				rp.adaptiveAdjustment()
			case <-rp.shutdown:
				return
			}
		}
	}()
}

func (rp *ResourcePool) adaptiveAdjustment() {
	// Collect performance data
	metrics := rp.GetMetrics()
	queueUtilization := float64(metrics.QueuedJobs) / float64(rp.config.PipelineBufferSize)
	
	// Track performance history
	rp.performanceHistory = append(rp.performanceHistory, queueUtilization)
	if len(rp.performanceHistory) > 100 {
		rp.performanceHistory = rp.performanceHistory[1:]
	}
	
	// Decide on scaling action
	if queueUtilization > 0.8 && len(rp.workers) < rp.config.MaxWorkers {
		// Scale up
		newWorker := rp.createWorker()
		rp.startWorker(newWorker)
		rp.logger.Debug("adaptive scaling up",
			zap.Float64("queue_utilization", queueUtilization),
			zap.Int("new_worker_count", len(rp.workers)))
	} else if queueUtilization < 0.2 && len(rp.workers) > rp.config.MinWorkers {
		// Scale down
		rp.scaleDownWorkers(len(rp.workers) - 1)
		rp.logger.Debug("adaptive scaling down",
			zap.Float64("queue_utilization", queueUtilization),
			zap.Int("new_worker_count", len(rp.workers)))
	}
	
	rp.lastAdjustment = time.Now()
}

func (rp *ResourcePool) checkResourceLimits() error {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()
	
	if rp.activeJobs >= rp.maxJobs {
		return fmt.Errorf("maximum concurrent jobs limit reached: %d", rp.maxJobs)
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	currentMemoryMB := int(m.Alloc / 1024 / 1024)
	
	if currentMemoryMB >= rp.memoryLimitMB {
		return fmt.Errorf("memory limit reached: %dMB", currentMemoryMB)
	}
	
	return nil
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}