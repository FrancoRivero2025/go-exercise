package cache

import (
	"sync"
	"time"

	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/domain"
)

type InMemoryCache struct {
	data map[domain.Pair]domain.LTP
	mu   sync.RWMutex
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: make(map[domain.Pair]domain.LTP),
	}
}

func (c *InMemoryCache) Get(pair domain.Pair) (domain.LTP, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[pair]
	// optional: avoid returning expired values here; application handles ttl
	return val, ok
}

func (c *InMemoryCache) Set(pair domain.Pair, ltp domain.LTP) {
	// ensure timestamp exists
	if ltp.Timestamp.IsZero() {
		ltp.Timestamp = time.Now()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[pair] = ltp
}
