package parser

import (
	"reflect"
	"sync"
	"testing"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/stretchr/testify/assert"
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
	cache := NewTypeCache(100)
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Create test types
	testTypes := make([]domain.Type, numOperations)
	for i := 0; i < numOperations; i++ {
		testTypes[i] = domain.NewBasicType("type"+string(rune(i)), reflect.String)
	}

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
				assert.NotNil(t, retrieved)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	assert.Greater(t, cache.Size(), 0)
	assert.LessOrEqual(t, cache.Size(), 100) // Should not exceed max size
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
	cache.Get("string") // Hit
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