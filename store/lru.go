package store

import (
	"net/http"
	"sync"
	"time"
)

// LRU implements a fixed-size thread safe LRU cache.
// It is based on the LRU cache in Groupcache.
type LRU struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted OnEvicted

	// TTL To expire a value in cache.
	// 0 TTL means no expiry policy specified.
	TTL time.Duration

	MU *sync.Mutex

	cache *cache
}

// New creates a new LRU Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func New(maxEntries int) *LRU {
	return &LRU{
		MaxEntries: maxEntries,
		MU:         new(sync.Mutex),
	}
}

// Store sets the value for a key.
func (l *LRU) Store(key string, value interface{}, _ *http.Request) error {
	l.MU.Lock()
	defer l.MU.Unlock()

	e := l.cache.store(key, value)
	l.cache.list.MoveToFront(e)

	if l.MaxEntries != 0 && l.cache.len() > l.MaxEntries {
		l.removeOldest()
	}

	return nil
}

// Update the value for a key without updating the "recently used".
func (l *LRU) Update(key string, value interface{}, _ *http.Request) error {
	l.MU.Lock()
	defer l.MU.Unlock()
	l.cache.update(key, value)
	return nil
}

// Load returns the value stored in the Cache for a key, or nil if no value is present.
// The ok result indicates whether value was found in the Cache.
func (l *LRU) Load(key string, _ *http.Request) (interface{}, bool, error) {
	l.MU.Lock()
	defer l.MU.Unlock()

	e, ok, err := l.cache.load(key)

	if ok && err == nil {
		l.cache.list.MoveToFront(e)
		return e.Value.(*record).Value, ok, err
	}

	return nil, ok, err
}

// Peek returns the value stored in the Cache for a key
// without updating the "recently used", or nil if no value is present.
// The ok result indicates whether value was found in the Cache.
func (l *LRU) Peek(key string, _ *http.Request) (interface{}, bool, error) {
	l.MU.Lock()
	defer l.MU.Unlock()

	e, ok, err := l.cache.load(key)

	if ok && err == nil {
		return e.Value.(*record).Value, ok, err
	}

	return nil, ok, err
}

// Delete the value for a key.
func (l *LRU) Delete(key string, _ *http.Request) error {
	l.MU.Lock()
	defer l.MU.Unlock()
	l.cache.delete(key)
	return nil
}

// RemoveOldest removes the oldest item from the cache.
func (l *LRU) RemoveOldest() {
	l.MU.Lock()
	defer l.MU.Unlock()
	l.removeOldest()
}

func (l *LRU) removeOldest() {
	if e := l.cache.list.Back(); e != nil {
		l.cache.evict(e)
	}
}

// Len returns the number of items in the cache.
func (l *LRU) Len() int {
	l.MU.Lock()
	defer l.MU.Unlock()
	return l.cache.len()
}

// Clear purges all stored items from the cache.
func (l *LRU) Clear() {
	l.MU.Lock()
	defer l.MU.Unlock()
	l.cache.clear()
}

// Keys return cache records keys.
func (l *LRU) Keys() []string {
	l.MU.Lock()
	defer l.MU.Unlock()
	return l.cache.keys()
}
