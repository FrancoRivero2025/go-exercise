package mocks

import (
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/domain"
)

type MockMarketDataProvider struct {
	baseURL    string
	responses  map[domain.Pair]domain.LTP
	panicPairs map[domain.Pair]bool
	delays     map[domain.Pair]time.Duration
	callCount  map[domain.Pair]int
}

func NewMockMarketDataProvider() *MockMarketDataProvider {
	return &MockMarketDataProvider{
		responses:  make(map[domain.Pair]domain.LTP),
		panicPairs: make(map[domain.Pair]bool),
		delays:     make(map[domain.Pair]time.Duration),
		callCount:  make(map[domain.Pair]int),
	}
}

func (m *MockMarketDataProvider) SetBaseURL(url string) {
	m.baseURL = url
}

func (m *MockMarketDataProvider) SetResponse(pair domain.Pair, ltp domain.LTP) {
	m.responses[pair] = ltp
}

func (m *MockMarketDataProvider) ClearResponses() {
	m.responses = make(map[domain.Pair]domain.LTP)
}

func (m *MockMarketDataProvider) SetPanic(pair domain.Pair, shouldPanic bool) {
	if m.panicPairs == nil {
		m.panicPairs = make(map[domain.Pair]bool)
	}
	m.panicPairs[pair] = shouldPanic
}

func (m *MockMarketDataProvider) SetDelay(pair domain.Pair, delay time.Duration) {
	if m.delays == nil {
		m.delays = make(map[domain.Pair]time.Duration)
	}
	m.delays[pair] = delay
}

func (m *MockMarketDataProvider) GetCallCount(pair domain.Pair) int {
	if m.callCount == nil {
		return 0
	}
	return m.callCount[pair]
}

func (m *MockMarketDataProvider) Fetch(pair domain.Pair) domain.LTP {
	if m.callCount == nil {
		m.callCount = make(map[domain.Pair]int)
	}
	m.callCount[pair]++

	if m.panicPairs != nil && m.panicPairs[pair] {
		panic("mock panic")
	}
	if m.delays != nil {
		if delay, exists := m.delays[pair]; exists {
			time.Sleep(delay)
		}
	}
	if m.responses != nil {
		if response, exists := m.responses[pair]; exists {
			return response
		}
	}

	return domain.LTP{}
}