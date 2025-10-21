package cache

import (
	"container/list"
	"sync"
	"time"
)

// CacheDriver type
type CacheDriver string

const (
	Memory  CacheDriver = "memory"  // Default, in-process, auto-sizing
	NoCache CacheDriver = "none"    // Disable caching
	Valkey  CacheDriver = "valkey"  // Redis-compatible
	Redis   CacheDriver = "redis"   // Native Redis
)

// CacheLayer provides caching functionality
type CacheLayer struct {
	driver     CacheDriver
	memory     *MemoryCache
	mu         sync.RWMutex
	enabled    bool
	maxSize    int64 // in bytes
	currentSize int64
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key       string
	Value     []byte
	Size      int64
	ExpiresAt time.Time
	element   *list.Element
}

// MemoryCache implements in-memory LRU cache
type MemoryCache struct {
	entries map[string]*CacheEntry
	lru     *list.List
	mu      sync.RWMutex
	maxSize int64
	size    int64
}

// NewCacheLayer creates a new cache layer
func NewCacheLayer(driver CacheDriver, maxSizeMB int64) *CacheLayer {
	if maxSizeMB == 0 {
		// Auto-size: 10% of available memory, up to 1GB
		maxSizeMB = 1024 // Default 1GB
	}

	c := &CacheLayer{
		driver:  driver,
		enabled: driver != NoCache,
		maxSize: maxSizeMB * 1024 * 1024, // Convert MB to bytes
	}

	if driver == Memory {
		c.memory = &MemoryCache{
			entries: make(map[string]*CacheEntry),
			lru:     list.New(),
			maxSize: c.maxSize,
		}
	}

	return c
}

// Get retrieves a value from cache
func (c *CacheLayer) Get(key string) ([]byte, bool) {
	if !c.enabled || c.driver != Memory {
		return nil, false
	}

	return c.memory.Get(key)
}

// Set stores a value in cache
func (c *CacheLayer) Set(key string, value []byte, ttl time.Duration) {
	if !c.enabled || c.driver != Memory {
		return
	}

	c.memory.Set(key, value, ttl)
}

// Delete removes a value from cache
func (c *CacheLayer) Delete(key string) {
	if !c.enabled || c.driver != Memory {
		return
	}

	c.memory.Delete(key)
}

// Clear removes all entries from cache
func (c *CacheLayer) Clear() {
	if !c.enabled || c.driver != Memory {
		return
	}

	c.memory.Clear()
}

// Stats returns cache statistics
func (c *CacheLayer) Stats() CacheStats {
	if !c.enabled || c.driver != Memory {
		return CacheStats{}
	}

	return c.memory.Stats()
}

// CacheStats holds cache statistics
type CacheStats struct {
	Entries     int
	SizeBytes   int64
	MaxSize     int64
	HitRate     float64
	Evictions   int64
	Hits        int64
	Misses      int64
}

// MemoryCache methods

func (m *MemoryCache) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	// Move to front (most recently used)
	m.lru.MoveToFront(entry.element)

	return entry.Value, true
}

func (m *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	size := int64(len(value))

	// Check if entry already exists
	if existing, exists := m.entries[key]; exists {
		// Update existing entry
		m.size -= existing.Size
		m.lru.Remove(existing.element)
	}

	// Evict entries if necessary
	for m.size+size > m.maxSize && m.lru.Len() > 0 {
		m.evictOldest()
	}

	// Create new entry
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		Size:      size,
		ExpiresAt: expiresAt,
	}

	entry.element = m.lru.PushFront(entry)
	m.entries[key] = entry
	m.size += size
}

func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.entries[key]; exists {
		m.lru.Remove(entry.element)
		delete(m.entries, key)
		m.size -= entry.Size
	}
}

func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]*CacheEntry)
	m.lru = list.New()
	m.size = 0
}

func (m *MemoryCache) Stats() CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return CacheStats{
		Entries:   len(m.entries),
		SizeBytes: m.size,
		MaxSize:   m.maxSize,
	}
}

func (m *MemoryCache) evictOldest() {
	element := m.lru.Back()
	if element == nil {
		return
	}

	entry := element.Value.(*CacheEntry)
	m.lru.Remove(element)
	delete(m.entries, entry.Key)
	m.size -= entry.Size
}

// CleanExpired removes expired entries
func (m *MemoryCache) CleanExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range m.entries {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			m.lru.Remove(entry.element)
			delete(m.entries, key)
			m.size -= entry.Size
			removed++
		}
	}

	return removed
}
