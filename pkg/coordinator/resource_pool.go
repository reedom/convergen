package coordinator

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// ResourcePool manages shared resources across the pipeline
type ResourcePool interface {
	// Get worker pool for concurrent processing
	GetWorkerPool(ctx context.Context, size int) (*WorkerPool, error)

	// Get memory buffer pool
	GetBufferPool() *BufferPool

	// Get channel pool for event communication
	GetChannelPool() *ChannelPool

	// Release all resources
	Release(ctx context.Context) error

	// Get resource usage statistics
	GetResourceUsage() *ResourceUsage

	// Force garbage collection
	ForceGC()
}

// ConcreteResourcePool implements ResourcePool
type ConcreteResourcePool struct {
	logger *zap.Logger
	config *Config

	// Resource pools
	mutex       sync.RWMutex
	workerPools map[string]*WorkerPool
	bufferPool  *BufferPool
	channelPool *ChannelPool

	// Resource tracking
	resourceUsage *ResourceUsage
	gcStats       *GCStatistics

	// Lifecycle management
	shutdown chan struct{}
	released bool
}

// NewResourcePool creates a new resource pool
func NewResourcePool(logger *zap.Logger, config *Config) ResourcePool {
	pool := &ConcreteResourcePool{
		logger:      logger,
		config:      config,
		workerPools: make(map[string]*WorkerPool),
		shutdown:    make(chan struct{}),
		resourceUsage: &ResourceUsage{
			GoroutineCount: 0,
		},
		gcStats: &GCStatistics{},
	}

	// Initialize buffer pool
	pool.bufferPool = pool.createBufferPool()

	// Initialize channel pool
	pool.channelPool = pool.createChannelPool()

	// Start resource monitoring
	go pool.monitorResources()

	logger.Info("resource pool initialized",
		zap.Int("worker_pool_size", config.WorkerPoolSize),
		zap.Int("buffer_pool_size", config.BufferPoolSize),
		zap.Int("channel_pool_size", config.ChannelPoolSize))

	return pool
}

// GetWorkerPool retrieves or creates a worker pool
func (r *ConcreteResourcePool) GetWorkerPool(ctx context.Context, size int) (*WorkerPool, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.released {
		return nil, fmt.Errorf("resource pool has been released")
	}

	// Create unique key for worker pool
	key := fmt.Sprintf("worker_%d", size)

	// Return existing pool if available
	if pool, exists := r.workerPools[key]; exists {
		return pool, nil
	}

	// Create new worker pool
	pool := &WorkerPool{
		Size:      size,
		Workers:   make(chan struct{}, size),
		Tasks:     make(chan func(), r.config.WorkerPoolSize*2),
		Done:      make(chan struct{}),
		Error:     make(chan error, 10),
		Active:    0,
		Processed: 0,
	}

	// Start worker goroutines
	for i := 0; i < size; i++ {
		go r.startWorker(pool, i)
		pool.Workers <- struct{}{} // Initialize worker slots
	}

	r.workerPools[key] = pool
	atomic.AddInt64(&r.resourceUsage.GoroutineCount, int64(size))

	r.logger.Debug("worker pool created",
		zap.String("key", key),
		zap.Int("size", size))

	return pool, nil
}

// GetBufferPool returns the shared buffer pool
func (r *ConcreteResourcePool) GetBufferPool() *BufferPool {
	return r.bufferPool
}

// GetChannelPool returns the shared channel pool
func (r *ConcreteResourcePool) GetChannelPool() *ChannelPool {
	return r.channelPool
}

// Release shuts down all resources
func (r *ConcreteResourcePool) Release(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.released {
		return nil
	}

	r.logger.Info("releasing resource pool")

	// Signal shutdown
	close(r.shutdown)

	// Release worker pools
	for key, pool := range r.workerPools {
		close(pool.Done)
		r.logger.Debug("worker pool released", zap.String("key", key))
	}

	// Release buffer pool
	r.releaseBufferPool()

	// Release channel pool
	r.releaseChannelPool()

	r.released = true
	r.logger.Info("resource pool released successfully")

	return nil
}

// GetResourceUsage returns current resource usage statistics
func (r *ConcreteResourcePool) GetResourceUsage() *ResourceUsage {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Update current statistics
	r.updateResourceUsage()

	// Return a copy
	usage := *r.resourceUsage
	return &usage
}

// ForceGC triggers garbage collection
func (r *ConcreteResourcePool) ForceGC() {
	// This would typically call runtime.GC() but we'll simulate it
	r.logger.Debug("forcing garbage collection")

	r.mutex.Lock()
	r.gcStats.NumGC++
	r.gcStats.LastGC = time.Now()
	r.mutex.Unlock()
}

// Private methods

func (r *ConcreteResourcePool) createBufferPool() *BufferPool {
	pool := &BufferPool{
		pool:    make(chan []byte, r.config.BufferPoolSize),
		size:    r.config.BufferPoolSize,
		bufSize: 4096, // 4KB buffers
		created: 0,
		reused:  0,
	}

	// Pre-populate with buffers
	for i := 0; i < r.config.BufferPoolSize; i++ {
		pool.pool <- make([]byte, pool.bufSize)
		atomic.AddInt64(&pool.created, 1)
	}

	return pool
}

func (r *ConcreteResourcePool) createChannelPool() *ChannelPool {
	pool := &ChannelPool{
		eventChans: make(chan chan events.Event, r.config.ChannelPoolSize),
		size:       r.config.ChannelPoolSize,
		created:    0,
		reused:     0,
	}

	// Pre-populate with channels
	for i := 0; i < r.config.ChannelPoolSize; i++ {
		pool.eventChans <- make(chan events.Event, 10)
		atomic.AddInt64(&pool.created, 1)
	}

	return pool
}

func (r *ConcreteResourcePool) startWorker(pool *WorkerPool, workerID int) {
	r.logger.Debug("starting worker", zap.Int("worker_id", workerID))

	for {
		select {
		case task := <-pool.Tasks:
			atomic.AddInt32(&pool.Active, 1)

			// Execute task with recovery
			func() {
				defer func() {
					if err := recover(); err != nil {
						r.logger.Error("worker task panic",
							zap.Int("worker_id", workerID),
							zap.Any("panic", err))

						select {
						case pool.Error <- fmt.Errorf("worker panic: %v", err):
						default:
							// Error channel full, log and continue
						}
					}

					atomic.AddInt32(&pool.Active, -1)
					atomic.AddInt64(&pool.Processed, 1)
				}()

				task()
			}()

		case <-pool.Done:
			r.logger.Debug("worker stopping", zap.Int("worker_id", workerID))
			return

		case <-r.shutdown:
			r.logger.Debug("worker stopping due to shutdown", zap.Int("worker_id", workerID))
			return
		}
	}
}

func (r *ConcreteResourcePool) monitorResources() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.updateResourceUsage()

		case <-r.shutdown:
			return
		}
	}
}

func (r *ConcreteResourcePool) updateResourceUsage() {
	// Update goroutine count
	var totalGoroutines int32
	for _, pool := range r.workerPools {
		totalGoroutines += pool.Active
	}

	r.resourceUsage.GoroutineCount = int64(totalGoroutines)

	// Update memory usage (simulated)
	r.resourceUsage.CurrentMemoryUsage = int64(len(r.workerPools)) * 1024 * 1024 // 1MB per pool
	if r.resourceUsage.CurrentMemoryUsage > r.resourceUsage.PeakMemoryUsage {
		r.resourceUsage.PeakMemoryUsage = r.resourceUsage.CurrentMemoryUsage
	}

	// Update GC stats
	r.resourceUsage.GCStats = r.gcStats
}

func (r *ConcreteResourcePool) releaseBufferPool() {
	if r.bufferPool == nil {
		return
	}

	// Drain the buffer pool
	close(r.bufferPool.pool)
	for range r.bufferPool.pool {
		// Drain remaining buffers
	}

	r.logger.Debug("buffer pool released",
		zap.Int64("created", r.bufferPool.created),
		zap.Int64("reused", r.bufferPool.reused))
}

func (r *ConcreteResourcePool) releaseChannelPool() {
	if r.channelPool == nil {
		return
	}

	// Drain the channel pool
	close(r.channelPool.eventChans)
	for ch := range r.channelPool.eventChans {
		close(ch)
	}

	r.logger.Debug("channel pool released",
		zap.Int64("created", r.channelPool.created),
		zap.Int64("reused", r.channelPool.reused))
}

// Buffer pool methods

// GetBuffer retrieves a buffer from the pool
func (b *BufferPool) GetBuffer() []byte {
	select {
	case buf := <-b.pool:
		atomic.AddInt64(&b.reused, 1)
		return buf[:0] // Reset length but keep capacity
	default:
		// Pool empty, create new buffer
		atomic.AddInt64(&b.created, 1)
		return make([]byte, 0, b.bufSize)
	}
}

// PutBuffer returns a buffer to the pool
func (b *BufferPool) PutBuffer(buf []byte) {
	if cap(buf) != b.bufSize {
		return // Wrong size, don't return to pool
	}

	select {
	case b.pool <- buf:
		// Successfully returned to pool
	default:
		// Pool full, buffer will be garbage collected
	}
}

// Channel pool methods

// GetEventChannel retrieves an event channel from the pool
func (c *ChannelPool) GetEventChannel() chan events.Event {
	select {
	case ch := <-c.eventChans:
		atomic.AddInt64(&c.reused, 1)
		return ch
	default:
		// Pool empty, create new channel
		atomic.AddInt64(&c.created, 1)
		return make(chan events.Event, 10)
	}
}

// PutEventChannel returns an event channel to the pool
func (c *ChannelPool) PutEventChannel(ch chan events.Event) {
	// Drain the channel before returning
	for {
		select {
		case <-ch:
			// Drain remaining events
		default:
			// Channel empty, safe to return
			select {
			case c.eventChans <- ch:
				// Successfully returned to pool
			default:
				// Pool full, close channel
				close(ch)
			}
			return
		}
	}
}

// Worker pool methods

// SubmitTask submits a task to the worker pool
func (w *WorkerPool) SubmitTask(task func()) error {
	select {
	case w.Tasks <- task:
		return nil
	default:
		return fmt.Errorf("worker pool task queue full")
	}
}

// GetStats returns worker pool statistics
func (w *WorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"size":      w.Size,
		"active":    atomic.LoadInt32(&w.Active),
		"processed": atomic.LoadInt64(&w.Processed),
		"queue_len": len(w.Tasks),
	}
}
