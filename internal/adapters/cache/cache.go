package cache

import (
	"sync"
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
)

type cacheEntry struct {
	ltp       domain.LTP
	expiresAt time.Time
}

type InMemoryCache struct {
	data map[domain.Pair]cacheEntry
	mu   sync.RWMutex
	ttl  time.Duration
	lastValues map[domain.Pair]domain.LTP
}

func NewInMemoryCache(ttl time.Duration) *InMemoryCache {
	return &InMemoryCache{
		data:       make(map[domain.Pair]cacheEntry),
		ttl:        ttl,
		lastValues: make(map[domain.Pair]domain.LTP),
	}
}

func (c *InMemoryCache) Get(pair domain.Pair) (ltp domain.LTP, found bool) {
	// Mecanismo de recuperación para Get
	defer func() {
		if rec := recover(); rec != nil {
			log.GetInstance().Debug("Recovered from panic in InMemoryCache Get: %v", rec)
			ltp = domain.LTP{}
			found = false
		}
	}()

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[pair]
	if !ok {
		return domain.LTP{}, false
	}

	if c.ttl > 0 && time.Now().After(entry.expiresAt) {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.data, pair)
		c.mu.Unlock()
		c.mu.RLock()

		return domain.LTP{}, false
	}

	return entry.ltp, true
}

func (c *InMemoryCache) Set(pair domain.Pair, ltp domain.LTP) {
	defer func() {
		if rec := recover(); rec != nil {
			log.GetInstance().Debug("Recovered from panic in InMemoryCache Set: %v", rec)
			c.mu.Lock()
			defer c.mu.Unlock()
			c.lastValues[pair] = ltp
			log.GetInstance().Debug("Stored last value for %s as fallback in InMemoryCache", pair)
		}
	}()

	if ltp.Timestamp.IsZero() {
		ltp.Timestamp = time.Now()
	}

	exp := time.Time{}
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[pair] = cacheEntry{
		ltp:       ltp,
		expiresAt: exp,
	}
	c.lastValues[pair] = ltp
}
