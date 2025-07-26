package parser

import (
	"sync"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// TypeCache provides thread-safe caching for type resolution
type TypeCache struct {
	cache    map[string]*cacheEntry
	mutex    sync.RWMutex
	maxSize  int
	hits     int64
	misses   int64
	statsMux sync.RWMutex
}

// cacheEntry represents a cached type with metadata
type cacheEntry struct {
	domainType  domain.Type
	lastAccess  time.Time
	accessCount int64
}

// NewTypeCache creates a new type cache with the specified maximum size
func NewTypeCache(maxSize int) *TypeCache {
	return &TypeCache{
		cache:   make(map[string]*cacheEntry),
		maxSize: maxSize,
	}
}

// Get retrieves a type from the cache
func (tc *TypeCache) Get(key string) domain.Type {
	tc.mutex.RLock()
	entry, exists := tc.cache[key]
	tc.mutex.RUnlock()

	tc.statsMux.Lock()
	if exists {
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
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Check if we need to evict entries
	if len(tc.cache) >= tc.maxSize {
		tc.evictLRU()
	}

	tc.cache[key] = &cacheEntry{
		domainType:  domainType,
		lastAccess:  time.Now(),
		accessCount: 1,
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
		Size:    tc.Size(),
		MaxSize: tc.maxSize,
		Hits:    tc.hits,
		Misses:  tc.misses,
		HitRate: tc.HitRate(),
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
	tc.statsMux.Unlock()
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Size    int     `json:"size"`
	MaxSize int     `json:"max_size"`
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hit_rate"`
}
