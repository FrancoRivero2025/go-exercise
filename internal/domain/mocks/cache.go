package mocks

import (
	"sync"

	"github.com/FrancoRivero2025/go-exercise/internal/domain"
)

type MockCache struct {
	data  map[domain.Pair]domain.LTP
	mutex sync.RWMutex
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[domain.Pair]domain.LTP),
	}
}

func (m *MockCache) Get(pair domain.Pair) (domain.LTP, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	ltp, exists := m.data[pair]
	return ltp, exists
}

func (m *MockCache) Set(pair domain.Pair, ltp domain.LTP) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[pair] = ltp
}

func (m *MockCache) CheckConnectivity() bool {
	return true
}

func (m *MockCache) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data = make(map[domain.Pair]domain.LTP)
}

func (m *MockCache) GetAll() map[domain.Pair]domain.LTP {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.data
}
