package coordinator

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewResourcePool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)

	if pool == nil {
		t.Fatal("NewResourcePool returned nil")
	}

	// Verify it implements the interface
	var _ ResourcePool = pool
}

func TestResourcePoolGetWorkerPool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()
	workerPool, err := pool.GetWorkerPool(ctx, 3)

	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}

	if workerPool == nil {
		t.Fatal("GetWorkerPool returned nil")
	}

	if workerPool.Size != 3 {
		t.Errorf("Expected worker pool size 3, got %d", workerPool.Size)
	}

	if len(workerPool.Workers) != 3 {
		t.Errorf("Expected 3 workers in channel, got %d", len(workerPool.Workers))
	}
}

func TestResourcePoolGetWorkerPoolSameSize(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()

	// Get first worker pool
	pool1, err := pool.GetWorkerPool(ctx, 2)
	if err != nil {
		t.Fatalf("First GetWorkerPool failed: %v", err)
	}

	// Get second worker pool with same size - should return the same instance
	pool2, err := pool.GetWorkerPool(ctx, 2)
	if err != nil {
		t.Fatalf("Second GetWorkerPool failed: %v", err)
	}

	if pool1 != pool2 {
		t.Error("Expected same worker pool instance for same size")
	}
}

func TestResourcePoolGetWorkerPoolAfterRelease(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	pool := NewResourcePool(logger, config)

	ctx := context.Background()

	// Get worker pool
	_, err := pool.GetWorkerPool(ctx, 2)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}

	// Release pool
	err = pool.Release(ctx)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Try to get worker pool after release - should fail
	_, err = pool.GetWorkerPool(ctx, 2)
	if err == nil {
		t.Error("Expected error when getting worker pool after release")
	}

	expectedMsg := "resource pool has been released"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestResourcePoolGetBufferPool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	bufferPool := pool.GetBufferPool()

	if bufferPool == nil {
		t.Fatal("GetBufferPool returned nil")
	}

	if bufferPool.size != config.BufferPoolSize {
		t.Errorf("Expected buffer pool size %d, got %d",
			config.BufferPoolSize, bufferPool.size)
	}

	// Test getting and putting buffers
	buf := bufferPool.GetBuffer()
	if buf == nil {
		t.Error("GetBuffer returned nil")
	}

	if cap(buf) < bufferPool.bufSize {
		t.Errorf("Expected buffer capacity >= %d, got %d",
			bufferPool.bufSize, cap(buf))
	}

	bufferPool.PutBuffer(buf)

	// Should be able to get the buffer back
	buf2 := bufferPool.GetBuffer()
	if buf2 == nil {
		t.Error("GetBuffer returned nil after put")
	}
}

func TestResourcePoolGetChannelPool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	channelPool := pool.GetChannelPool()

	if channelPool == nil {
		t.Fatal("GetChannelPool returned nil")
	}

	if channelPool.size != config.ChannelPoolSize {
		t.Errorf("Expected channel pool size %d, got %d",
			config.ChannelPoolSize, channelPool.size)
	}

	// Test getting and putting channels
	ch := channelPool.GetEventChannel()
	if ch == nil {
		t.Error("GetEventChannel returned nil")
	}

	channelPool.PutEventChannel(ch)

	// Should be able to get a channel back
	ch2 := channelPool.GetEventChannel()
	if ch2 == nil {
		t.Error("GetEventChannel returned nil after put")
	}
}

func TestResourcePoolGetResourceUsage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	usage := pool.GetResourceUsage()

	if usage == nil {
		t.Fatal("GetResourceUsage returned nil")
	}

	// Should have some goroutines from worker pools
	if usage.GoroutineCount < 0 {
		t.Errorf("Expected non-negative goroutine count, got %d", usage.GoroutineCount)
	}

	if usage.GCStats == nil {
		t.Error("Expected GC stats to be initialized")
	}
}

func TestResourcePoolForceGC(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	// Get initial GC stats
	usage1 := pool.GetResourceUsage()
	initialGC := usage1.GCStats.NumGC

	// Force GC
	pool.ForceGC()

	// Get updated stats
	usage2 := pool.GetResourceUsage()

	if usage2.GCStats.NumGC <= initialGC {
		t.Errorf("Expected GC count to increase from %d, got %d",
			initialGC, usage2.GCStats.NumGC)
	}
}

func TestResourcePoolRelease(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	pool := NewResourcePool(logger, config)

	ctx := context.Background()

	// Create some worker pools
	_, err := pool.GetWorkerPool(ctx, 2)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}

	// Release should succeed
	err = pool.Release(ctx)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Second release should be idempotent
	err = pool.Release(ctx)
	if err != nil {
		t.Errorf("Second release failed: %v", err)
	}
}

// Test BufferPool methods

func TestBufferPoolGetPutBuffer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	bufferPool := pool.bufferPool

	// Get buffer
	buf := bufferPool.GetBuffer()
	if buf == nil {
		t.Fatal("GetBuffer returned nil")
	}

	originalLen := len(buf)
	originalCap := cap(buf)

	// Write some data
	buf = append(buf, []byte("test data")...)

	// Put buffer back
	bufferPool.PutBuffer(buf)

	// Get buffer again
	buf2 := bufferPool.GetBuffer()
	if buf2 == nil {
		t.Fatal("GetBuffer returned nil after put")
	}

	// Should be reset to zero length but same capacity
	if len(buf2) != originalLen {
		t.Errorf("Expected buffer length %d after reset, got %d",
			originalLen, len(buf2))
	}

	if cap(buf2) != originalCap {
		t.Errorf("Expected buffer capacity %d, got %d", originalCap, cap(buf2))
	}
}

func TestBufferPoolWrongSizeBuffer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	bufferPool := pool.bufferPool

	// Create buffer with wrong size
	wrongSizeBuf := make([]byte, bufferPool.bufSize*2)

	// Put wrong size buffer - should be ignored
	bufferPool.PutBuffer(wrongSizeBuf)

	// Pool should still work normally
	buf := bufferPool.GetBuffer()
	if buf == nil {
		t.Error("GetBuffer failed after putting wrong size buffer")
	}

	if cap(buf) != bufferPool.bufSize {
		t.Errorf("Expected buffer capacity %d, got %d", bufferPool.bufSize, cap(buf))
	}
}

func TestBufferPoolExhaustion(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()
	config.BufferPoolSize = 2 // Small pool for testing

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	bufferPool := pool.bufferPool

	// Get all buffers from pool
	buf1 := bufferPool.GetBuffer()
	buf2 := bufferPool.GetBuffer()

	// Pool should be empty now, but GetBuffer should still work (creates new)
	buf3 := bufferPool.GetBuffer()
	if buf3 == nil {
		t.Error("GetBuffer should create new buffer when pool is empty")
	}

	// Return buffers
	bufferPool.PutBuffer(buf1)
	bufferPool.PutBuffer(buf2)
	bufferPool.PutBuffer(buf3) // This might be ignored if pool is full
}

// Test ChannelPool methods

func TestChannelPoolGetPutChannel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	channelPool := pool.channelPool

	// Get channel
	ch := channelPool.GetEventChannel()
	if ch == nil {
		t.Fatal("GetEventChannel returned nil")
	}

	// Put channel back
	channelPool.PutEventChannel(ch)

	// Get channel again
	ch2 := channelPool.GetEventChannel()
	if ch2 == nil {
		t.Fatal("GetEventChannel returned nil after put")
	}
}

func TestChannelPoolDrainChannel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	channelPool := pool.channelPool

	// Get channel and add some events
	ch := channelPool.GetEventChannel()

	// Add test events (using simple test events)
	for i := 0; i < 3; i++ {
		select {
		case ch <- nil: // Simple nil events for testing
		// Channel full, skip
		default:
		}
	}

	// Put channel back - should drain events
	channelPool.PutEventChannel(ch)

	// Get channel again - should be drained
	ch2 := channelPool.GetEventChannel()

	// Channel should be empty
	select {
	case <-ch2:
		t.Error("Expected channel to be drained")
	// Channel is empty as expected
	default:
	}
}

// Test WorkerPool methods

func TestWorkerPoolSubmitTask(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()

	workerPool, err := pool.GetWorkerPool(ctx, 2)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}

	// Submit a task
	var executed bool

	var mu sync.Mutex

	task := func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	}

	err = workerPool.SubmitTask(task)
	if err != nil {
		t.Fatalf("SubmitTask failed: %v", err)
	}

	// Wait for task execution
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	wasExecuted := executed
	mu.Unlock()

	if !wasExecuted {
		t.Error("Task was not executed")
	}
}

func TestWorkerPoolGetStats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()

	workerPool, err := pool.GetWorkerPool(ctx, 3)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}

	stats := workerPool.GetStats()

	if stats["size"] != 3 {
		t.Errorf("Expected size 3, got %v", stats["size"])
	}

	if stats["active"] == nil {
		t.Error("Expected active count in stats")
	}

	if stats["processed"] == nil {
		t.Error("Expected processed count in stats")
	}

	if stats["queue_len"] == nil {
		t.Error("Expected queue length in stats")
	}
}

func TestWorkerPoolTaskPanic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()

	workerPool, err := pool.GetWorkerPool(ctx, 1)
	if err != nil {
		t.Fatalf("GetWorkerPool failed: %v", err)
	}

	// Submit a task that panics
	panicTask := func() {
		panic("test panic")
	}

	err = workerPool.SubmitTask(panicTask)
	if err != nil {
		t.Fatalf("SubmitTask failed: %v", err)
	}

	// Wait for panic handling
	time.Sleep(100 * time.Millisecond)

	// Check error channel
	select {
	case err := <-workerPool.Error:
		if err == nil {
			t.Error("Expected error from panic")
		}

		expectedMsg := "worker panic: test panic"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
		}
	case <-time.After(time.Second):
		t.Error("Expected error from panic within timeout")
	}

	// Worker pool should still be functional
	var executed bool

	var mu sync.Mutex

	normalTask := func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	}

	err = workerPool.SubmitTask(normalTask)
	if err != nil {
		t.Fatalf("SubmitTask failed after panic: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	wasExecuted := executed
	mu.Unlock()

	if !wasExecuted {
		t.Error("Normal task was not executed after panic")
	}
}

// Concurrent access tests

func TestResourcePoolConcurrentWorkerPoolAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()
	done := make(chan bool, 10)

	// Concurrent access to worker pools
	for i := 0; i < 10; i++ {
		go func(_ int) {
			defer func() { done <- true }()

			for j := 0; j < 10; j++ {
				_, err := pool.GetWorkerPool(ctx, 2)
				if err != nil {
					t.Errorf("GetWorkerPool failed: %v", err)
				}
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestResourcePoolConcurrentBufferAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			t.Errorf("Release failed: %v", err)
		}
	}()

	bufferPool := pool.GetBufferPool()
	done := make(chan bool, 10)

	// Concurrent buffer operations
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				buf := bufferPool.GetBuffer()
				buf = append(buf, byte(j))
				bufferPool.PutBuffer(buf)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Benchmark tests

func BenchmarkResourcePoolGetWorkerPool(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			b.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := pool.GetWorkerPool(ctx, 2)
		if err != nil {
			b.Fatalf("GetWorkerPool failed: %v", err)
		}
	}
}

func BenchmarkBufferPoolGetPut(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			b.Errorf("Release failed: %v", err)
		}
	}()

	bufferPool := pool.bufferPool

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bufferPool.GetBuffer()
		bufferPool.PutBuffer(buf)
	}
}

func BenchmarkChannelPoolGetPut(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()

	pool := NewResourcePool(logger, config).(*ConcreteResourcePool)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			b.Errorf("Release failed: %v", err)
		}
	}()

	channelPool := pool.channelPool

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ch := channelPool.GetEventChannel()
		channelPool.PutEventChannel(ch)
	}
}

func BenchmarkWorkerPoolSubmitTask(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := createTestConfig()

	pool := NewResourcePool(logger, config)
	defer func() {
		if err := pool.Release(context.Background()); err != nil {
			b.Errorf("Release failed: %v", err)
		}
	}()

	ctx := context.Background()

	workerPool, err := pool.GetWorkerPool(ctx, 4)
	if err != nil {
		b.Fatalf("GetWorkerPool failed: %v", err)
	}

	task := func() {
		// Simple task
		time.Sleep(time.Microsecond)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := workerPool.SubmitTask(task)
		if err != nil {
			b.Fatalf("SubmitTask failed: %v", err)
		}
	}
}
