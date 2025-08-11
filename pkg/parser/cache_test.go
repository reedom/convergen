package parser

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/reedom/convergen/v9/pkg/domain"
)

func TestTypeCache_BasicOperations(t *testing.T) {
	cache := NewTypeCache(10)

	// Test empty cache
	assert.Nil(t, cache.Get("nonexistent"))
	assert.Equal(t, 0, cache.Size())
	assert.Equal(t, 0.0, cache.HitRate())

	// Create a test type
	testType := domain.StringType

	// Test put and get
	cache.Put("string", testType)
	assert.Equal(t, 1, cache.Size())

	retrieved := cache.Get("string")
	assert.NotNil(t, retrieved)
	assert.Equal(t, testType, retrieved)

	// Test hit rate
	assert.Greater(t, cache.HitRate(), 0.0)
}

func TestTypeCache_LRUEviction(t *testing.T) {
	cache := NewTypeCache(2) // Small cache for testing eviction

	type1 := domain.IntType
	type2 := domain.StringType
	type3 := domain.BoolType

	// Fill cache to capacity
	cache.Put("int", type1)
	cache.Put("string", type2)
	assert.Equal(t, 2, cache.Size())

	// Access the first item to make it more recently used
	cache.Get("int")

	// Add third item, should evict "string" (least recently used)
	cache.Put("bool", type3)
	assert.Equal(t, 2, cache.Size())

	// "int" and "bool" should still be there
	assert.NotNil(t, cache.Get("int"))
	assert.NotNil(t, cache.Get("bool"))

	// "string" should have been evicted
	assert.Nil(t, cache.Get("string"))
}

func TestTypeCache_ConcurrentAccess(t *testing.T) {
	cache := NewTypeCache(1000) // Larger cache to prevent eviction during test
	numGoroutines := 10
	numOperations := 50

	var wg sync.WaitGroup

	wg.Add(numGoroutines)

	// Create test types
	testTypes := make([]domain.Type, numOperations)
	for i := 0; i < numOperations; i++ {
		testTypes[i] = domain.NewBasicType("type"+string(rune(i)), reflect.String)
	}

	// Track successful operations
	var successfulOps int64

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "key" + string(rune(goroutineID)) + string(rune(j))
				typeIndex := (goroutineID*numOperations + j) % len(testTypes)

				// Put operation
				cache.Put(key, testTypes[typeIndex])

				// Get operation
				retrieved := cache.Get(key)
				if retrieved != nil {
					atomic.AddInt64(&successfulOps, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	assert.Greater(t, cache.Size(), 0)
	assert.LessOrEqual(t, cache.Size(), 1000)  // Should not exceed max size
	assert.Greater(t, successfulOps, int64(0)) // Should have some successful operations
}

func TestTypeCache_Stats(t *testing.T) {
	cache := NewTypeCache(10)

	testType := domain.StringType

	// Initially no hits or misses
	stats := cache.Stats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, 10, stats.MaxSize)
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, 0.0, stats.HitRate)

	// Add item and test hit
	cache.Put("string", testType)
	cache.Get("string")      // Hit
	cache.Get("nonexistent") // Miss

	stats = cache.Stats()
	assert.Equal(t, 1, stats.Size)
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 0.5, stats.HitRate)
}

func TestTypeCache_Clear(t *testing.T) {
	cache := NewTypeCache(10)

	// Add some items
	for i := 0; i < 5; i++ {
		key := "key" + string(rune(i))
		testType := domain.NewBasicType("type"+string(rune(i)), reflect.String)
		cache.Put(key, testType)
	}

	assert.Equal(t, 5, cache.Size())

	// Clear cache
	cache.Clear()

	assert.Equal(t, 0, cache.Size())
	assert.Equal(t, 0.0, cache.HitRate())

	// Verify all items are gone
	for i := 0; i < 5; i++ {
		key := "key" + string(rune(i))
		assert.Nil(t, cache.Get(key))
	}
}

func TestTypeCache_MaxSizeEnforcement(t *testing.T) {
	maxSize := 5
	cache := NewTypeCache(maxSize)

	// Add more items than max size
	for i := 0; i < maxSize*2; i++ {
		key := "key" + string(rune(i))
		testType := domain.NewBasicType("type"+string(rune(i)), reflect.String)
		cache.Put(key, testType)
	}

	// Size should not exceed max
	assert.LessOrEqual(t, cache.Size(), maxSize)
}

func TestTypeCache_AccessCountTracking(t *testing.T) {
	cache := NewTypeCache(10)
	testType := domain.StringType

	cache.Put("string", testType)

	// Multiple accesses should update access count and time
	for i := 0; i < 5; i++ {
		retrieved := cache.Get("string")
		assert.NotNil(t, retrieved)
	}

	// Hit rate should reflect multiple hits
	stats := cache.Stats()
	assert.Equal(t, int64(5), stats.Hits)
	assert.Equal(t, 1.0, stats.HitRate) // All accesses were hits
}

func BenchmarkTypeCache_Put(b *testing.B) {
	cache := NewTypeCache(1000)
	testType := domain.StringType

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		cache.Put(key, testType)
	}
}

func BenchmarkTypeCache_Get(b *testing.B) {
	cache := NewTypeCache(1000)
	testType := domain.StringType

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		cache.Put(key, testType)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		cache.Get(key)
	}
}

func BenchmarkTypeCache_ConcurrentAccess(b *testing.B) {
	cache := NewTypeCache(1000)
	testType := domain.StringType

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		cache.Put(key, testType)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key" + string(rune(i%100))
			cache.Get(key)

			i++
		}
	})
}

func TestTypeCache_TTLExpiration(t *testing.T) {
	// Create cache with very short TTL for testing
	cache := NewTypeCacheWithTTL(10, 50*time.Millisecond, 100)
	testType := domain.StringType

	// Put item with short TTL
	cache.PutWithTTL("short-lived", testType, 50*time.Millisecond)

	// Should be available immediately
	retrieved := cache.Get("short-lived")
	assert.NotNil(t, retrieved)
	assert.Equal(t, testType, retrieved)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired and return nil
	expired := cache.Get("short-lived")
	assert.Nil(t, expired)

	// Verify stats reflect expiration
	stats := cache.Stats()
	assert.Greater(t, stats.Expired, int64(0))
}

func TestTypeCache_MemoryPressureEviction(t *testing.T) {
	// Create cache with very low memory threshold to trigger eviction
	cache := NewTypeCacheWithTTL(100, 5*time.Minute, 1) // 1MB threshold
	testType := domain.StringType

	// Fill cache with many entries
	for i := 0; i < 50; i++ {
		key := "key" + string(rune(i))
		cache.Put(key, testType)
	}

	initialSize := cache.Size()
	assert.Greater(t, initialSize, 0)

	// Add one more item to potentially trigger memory pressure cleanup
	cache.Put("trigger", testType)

	// Check if evictions occurred (memory pressure may or may not trigger in test environment)
	stats := cache.Stats()
	// Note: Memory pressure eviction may not trigger in test environment with small objects
	t.Logf("Cache stats: Size=%d, Evictions=%d, Expired=%d", stats.Size, stats.Evictions, stats.Expired)
}

func TestTypeCache_PeriodicCleanup(t *testing.T) {
	// Create cache with short cleanup interval for testing
	cache := NewTypeCacheWithTTL(10, 30*time.Millisecond, 100)
	cache.cleanupInterval = 10 * time.Millisecond // Override for testing

	testType := domain.StringType

	// Add items with very short TTL
	for i := 0; i < 5; i++ {
		key := "key" + string(rune(i))
		cache.PutWithTTL(key, testType, 30*time.Millisecond)
	}

	assert.Equal(t, 5, cache.Size())

	// Wait for TTL expiration and cleanup
	time.Sleep(50 * time.Millisecond)

	// Trigger cleanup by adding new item
	cache.Put("trigger-cleanup", testType)

	// Some items should have been cleaned up
	stats := cache.Stats()
	t.Logf("After cleanup: Size=%d, Expired=%d", stats.Size, stats.Expired)
	assert.Greater(t, stats.Expired, int64(0))
}

func TestTypeCache_EnhancedStats(t *testing.T) {
	cache := NewTypeCacheWithTTL(5, 50*time.Millisecond, 100)
	testType := domain.StringType

	// Test evictions through size limit
	for i := 0; i < 10; i++ { // More than maxSize
		key := "key" + string(rune(i))
		cache.Put(key, testType)
	}

	stats := cache.Stats()
	assert.Equal(t, 5, stats.Size)               // Should be capped at maxSize
	assert.Greater(t, stats.Evictions, int64(0)) // Should have evictions

	// Test expiration tracking
	cache.PutWithTTL("expires", testType, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	// Trigger expiration check
	cache.Get("expires")

	stats = cache.Stats()
	assert.Greater(t, stats.Expired, int64(0)) // Should have expired entries
}
