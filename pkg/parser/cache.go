package parser

import (
	"runtime"
	"sync"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// TypeCache provides thread-safe caching for type resolution with TTL and memory pressure awareness
type TypeCache struct {
	cache             map[string]*cacheEntry
	mutex             sync.RWMutex
	maxSize           int
	defaultTTL        time.Duration
	memoryThresholdMB int64
	hits              int64
	misses            int64
	evictions         int64
	expired           int64
	statsMux          sync.RWMutex
	lastCleanup       time.Time
	cleanupInterval   time.Duration
}

// cacheEntry represents a cached type with metadata
type cacheEntry struct {
	domainType  domain.Type
	lastAccess  time.Time
	createdAt   time.Time
	accessCount int64
	ttl         time.Duration
}

// NewTypeCache creates a new type cache with the specified maximum size
func NewTypeCache(maxSize int) *TypeCache {
	return NewTypeCacheWithTTL(maxSize, 5*time.Minute, 100) // 5 minute default TTL, 100MB memory threshold
}

// NewTypeCacheWithTTL creates a new type cache with TTL and memory pressure settings
func NewTypeCacheWithTTL(maxSize int, defaultTTL time.Duration, memoryThresholdMB int64) *TypeCache {
	return &TypeCache{
		cache:             make(map[string]*cacheEntry),
		maxSize:           maxSize,
		defaultTTL:        defaultTTL,
		memoryThresholdMB: memoryThresholdMB,
		lastCleanup:       time.Now(),
		cleanupInterval:   1 * time.Minute,
	}
}

// Get retrieves a type from the cache
func (tc *TypeCache) Get(key string) domain.Type {
	tc.mutex.RLock()
	entry, exists := tc.cache[key]
	tc.mutex.RUnlock()

	tc.statsMux.Lock()
	if exists {
		// Check if entry has expired
		if tc.isExpired(entry) {
			tc.expired++
			tc.statsMux.Unlock()
			// Remove expired entry
			tc.mutex.Lock()
			delete(tc.cache, key)
			tc.mutex.Unlock()
			return nil
		}

		tc.hits++
		entry.lastAccess = time.Now()
		entry.accessCount++
		tc.statsMux.Unlock()
		return entry.domainType
	} else {
		tc.misses++
		tc.statsMux.Unlock()
		return nil
	}
}

// Put stores a type in the cache
func (tc *TypeCache) Put(key string, domainType domain.Type) {
	tc.PutWithTTL(key, domainType, tc.defaultTTL)
}

// PutWithTTL stores a type in the cache with custom TTL
func (tc *TypeCache) PutWithTTL(key string, domainType domain.Type, ttl time.Duration) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Perform periodic cleanup
	tc.maybeCleanup()

	// Check memory pressure and perform aggressive cleanup if needed
	tc.checkMemoryPressure()

	// Check if we need to evict entries
	if len(tc.cache) >= tc.maxSize {
		tc.evictLRU()
	}

	now := time.Now()
	tc.cache[key] = &cacheEntry{
		domainType:  domainType,
		lastAccess:  now,
		createdAt:   now,
		accessCount: 1,
		ttl:         ttl,
	}
}

// evictLRU removes the least recently used entry
func (tc *TypeCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range tc.cache {
		if oldestKey == "" || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
		}
	}

	if oldestKey != "" {
		delete(tc.cache, oldestKey)
		tc.statsMux.Lock()
		tc.evictions++
		tc.statsMux.Unlock()
	}
}

// Size returns the current number of cached entries
func (tc *TypeCache) Size() int {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	return len(tc.cache)
}

// HitRate returns the cache hit rate
func (tc *TypeCache) HitRate() float64 {
	tc.statsMux.RLock()
	defer tc.statsMux.RUnlock()

	total := tc.hits + tc.misses
	if total == 0 {
		return 0.0
	}
	return float64(tc.hits) / float64(total)
}

// Stats returns cache statistics
func (tc *TypeCache) Stats() CacheStats {
	tc.statsMux.RLock()
	defer tc.statsMux.RUnlock()

	return CacheStats{
		Size:      tc.Size(),
		MaxSize:   tc.maxSize,
		Hits:      tc.hits,
		Misses:    tc.misses,
		HitRate:   tc.HitRate(),
		Evictions: tc.evictions,
		Expired:   tc.expired,
	}
}

// Clear removes all entries from the cache
func (tc *TypeCache) Clear() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.cache = make(map[string]*cacheEntry)

	tc.statsMux.Lock()
	tc.hits = 0
	tc.misses = 0
	tc.evictions = 0
	tc.expired = 0
	tc.statsMux.Unlock()
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Evictions int64   `json:"evictions"`
	Expired   int64   `json:"expired"`
}

// isExpired checks if a cache entry has expired
func (tc *TypeCache) isExpired(entry *cacheEntry) bool {
	if entry.ttl <= 0 {
		return false // No TTL set
	}
	return time.Since(entry.createdAt) > entry.ttl
}

// maybeCleanup performs periodic cleanup of expired entries
func (tc *TypeCache) maybeCleanup() {
	if time.Since(tc.lastCleanup) < tc.cleanupInterval {
		return
	}

	tc.lastCleanup = time.Now()
	tc.cleanupExpired()
}

// cleanupExpired removes all expired entries from the cache
func (tc *TypeCache) cleanupExpired() {
	var keysToDelete []string

	for key, entry := range tc.cache {
		if tc.isExpired(entry) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(tc.cache, key)
	}

	if len(keysToDelete) > 0 {
		tc.statsMux.Lock()
		tc.expired += int64(len(keysToDelete))
		tc.statsMux.Unlock()
	}
}

// checkMemoryPressure performs aggressive cleanup if memory usage is high
func (tc *TypeCache) checkMemoryPressure() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Convert to MB
	allocMB := int64(memStats.Alloc / 1024 / 1024)

	if allocMB > tc.memoryThresholdMB {
		// Under memory pressure - remove 25% of entries (oldest first)
		tc.evictOldest(len(tc.cache) / 4)
	}
}

// evictOldest removes the specified number of oldest entries
func (tc *TypeCache) evictOldest(count int) {
	if count <= 0 || len(tc.cache) == 0 {
		return
	}

	// Create a slice of entries with their keys sorted by creation time
	type keyEntry struct {
		key     string
		created time.Time
	}

	entries := make([]keyEntry, 0, len(tc.cache))
	for key, entry := range tc.cache {
		entries = append(entries, keyEntry{key: key, created: entry.createdAt})
	}

	// Sort by creation time (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].created.After(entries[j].created) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove the oldest entries
	evicted := 0
	for i := 0; i < len(entries) && evicted < count; i++ {
		delete(tc.cache, entries[i].key)
		evicted++
	}

	if evicted > 0 {
		tc.statsMux.Lock()
		tc.evictions += int64(evicted)
		tc.statsMux.Unlock()
	}
}
