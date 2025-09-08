package application

import (
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/FrancoRivero2025/go-exercise/internal/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) SetLevel(level int)                     {}
func (m *MockLogger) SetOutput(w io.Writer)                  {}
func (m *MockLogger) SetOutputToFile(filename string) error  { return nil }
func (m *MockLogger) GetLevel() int                          { return 0 }
func (m *MockLogger) Debug(message string, v ...interface{}) {
	m.Called(append([]interface{}{message}, v...)...)
}
func (m *MockLogger) Info(message string, v ...interface{})  {}
func (m *MockLogger) Warn(message string, v ...interface{}) {
	m.Called(message, v)
}
func (m *MockLogger) Error(message string, v ...interface{}) {}
func (m *MockLogger) Fatal(message string, v ...interface{}) {}

func TestLTPService_GetLTP(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTCUSD")
	result := service.GetLTP(pair)
	assert.Equal(t, domain.LTP{}, result)

	expectedLTP := domain.LTP{
		Pair:      pair,
		Amount:    50000.00,
		Timestamp: time.Now(),
	}
	mockProvider.SetResponse(pair, expectedLTP)

	result = service.GetLTP(pair)
	assert.Equal(t, expectedLTP, result)

	cachedResult, exists := mockCache.Get(pair)
	assert.True(t, exists)
	assert.Equal(t, expectedLTP, cachedResult)
}

func TestLTPService_GetLTPs(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	mockLogger := new(MockLogger)

	mockLogger.On("Warn", "Failed to get LTP for fetch for pair %s", mock.Anything).Return()
	mockLogger.On("Warn", "Cannot found a LTP for pair %v", mock.Anything).Return()

	originalLogger := log.GetInstance()
	log.SetInstance(mockLogger)
	defer log.SetInstance(originalLogger)

	service := NewLTPService(mockCache, mockProvider, time.Minute)
	pairs := []domain.Pair{"BTCUSD", "BTCEUR"}

	results := service.GetLTPs(pairs)
	assert.Nil(t, results)

	mockLogger.AssertExpectations(t)

	mockLogger.ExpectedCalls = nil

	btcLTP := domain.LTP{Pair: "BTCUSD", Amount: 50000.00, Timestamp: time.Now()}
	btcEurLTP := domain.LTP{Pair: "BTCEUR", Amount: 3000.00, Timestamp: time.Now()}

	mockProvider.SetResponse("BTCUSD", btcLTP)
	mockProvider.SetResponse("BTCEUR", btcEurLTP)

	results = service.GetLTPs(pairs)
	require.Len(t, results, 2)
	assert.Contains(t, results, btcLTP)
	assert.Contains(t, results, btcEurLTP)
}

func TestLTPService_ForceRefresh(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTCUSD")
	newLTP := domain.LTP{
		Pair:      pair,
		Amount:    51000.00,
		Timestamp: time.Now(),
	}

	mockProvider.SetResponse(pair, newLTP)

	result := service.ForceRefresh(pair)
	assert.Equal(t, newLTP, result)

	cachedResult, exists := mockCache.Get(pair)
	assert.True(t, exists)
	assert.Equal(t, newLTP, cachedResult)
}

func TestLTPService_CacheTTL(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()

	service := NewLTPService(mockCache, mockProvider, time.Millisecond*10)

	pair := domain.Pair("BTCUSD")
	ltp := domain.LTP{
		Pair:      pair,
		Amount:    50000.00,
		Timestamp: time.Now(),
	}

	mockProvider.SetResponse(pair, ltp)

	result := service.GetLTP(pair)
	assert.Equal(t, ltp, result)

	result = service.GetLTP(pair)
	assert.Equal(t, ltp, result)

	time.Sleep(time.Millisecond * 20)

	newLTP := domain.LTP{
		Pair:      pair,
		Amount:    51000.00,
		Timestamp: time.Now(),
	}
	mockProvider.SetResponse(pair, newLTP)

	result = service.GetLTP(pair)
	assert.Equal(t, newLTP, result)
}

func TestNewTestLTPService(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	httpClient := &http.Client{}

	service := NewTestLTPService(mockCache, mockProvider, time.Minute, httpClient)

	assert.NotNil(t, service)
	assert.Equal(t, httpClient, service.httpClient)
}

func TestSetHTTPClientAndBaseURL(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	httpClient := &http.Client{}
	service.SetHTTPClient(httpClient)
	assert.Equal(t, httpClient, service.httpClient)

	baseURL := "http://test.com"
	service.SetBaseURL(baseURL)
}

func TestGetCache(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	assert.Equal(t, mockCache, service.GetCache())
}

func TestGetAllLTPs(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	mockLogger := new(MockLogger)

	originalLogger := log.GetInstance()
	log.SetInstance(mockLogger)
	defer log.SetInstance(originalLogger)

	config.Initialize("")
	originalCfg := config.GetInstance()
	testCfg := &config.Config{
		Pairs: []domain.Pair{"BTCUSD", "BTCEUR"},
	}
	config.SetInstance(testCfg)
	defer config.SetInstance(originalCfg)

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()
	result := service.GetAllLTPs()
	assert.Nil(t, result)

	btcLTP := domain.LTP{Pair: "BTCUSD", Amount: 50000.00, Timestamp: time.Now()}
	btcEurLTP := domain.LTP{Pair: "BTCEUR", Amount: 3000.00, Timestamp: time.Now()}
	mockProvider.SetResponse("BTCUSD", btcLTP)
	mockProvider.SetResponse("BTCEUR", btcEurLTP)

	result = service.GetAllLTPs()
	require.Len(t, result, 2)
	assert.Contains(t, result, btcLTP)
	assert.Contains(t, result, btcEurLTP)
}

func TestRefreshPairs(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	mockLogger := new(MockLogger)
	
	originalLogger := log.GetInstance()
	log.SetInstance(mockLogger)
	defer log.SetInstance(originalLogger)

	service := NewLTPService(mockCache, mockProvider, time.Minute)
	pairs := []domain.Pair{"BTCUSD", "BTCEUR"}

	mockLogger.On("Warn", "Cannot refresh and update cache", mock.Anything).Return()
	mockProvider.SetResponse("BTCUSD", domain.LTP{})
	mockProvider.SetResponse("BTCEUR", domain.LTP{})
	
	service.RefreshPairs(pairs)
	
	mockLogger.AssertCalled(t, "Warn", "Cannot refresh and update cache", mock.Anything)

	mockLogger.ExpectedCalls = nil

	btcLTP := domain.LTP{Pair: "BTCUSD", Amount: 50000.00, Timestamp: time.Now()}
	btcEurLTP := domain.LTP{Pair: "BTCEUR", Amount: 3000.00, Timestamp: time.Now()}
	mockProvider.SetResponse("BTCUSD", btcLTP)
	mockProvider.SetResponse("BTCEUR", btcEurLTP)
	
	service.RefreshPairs(pairs)
	
	cachedBTC, exists := mockCache.Get("BTCUSD")
	assert.True(t, exists)
	assert.Equal(t, btcLTP, cachedBTC)
	
	cachedBTCEUR, exists := mockCache.Get("BTCEUR")
	assert.True(t, exists)
	assert.Equal(t, btcEurLTP, cachedBTCEUR)
}

func TestGetLTP_WithPanic(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	mockLogger := new(MockLogger)

	originalLogger := log.GetInstance()
	log.SetInstance(mockLogger)
	defer log.SetInstance(originalLogger)

	service := NewLTPService(mockCache, mockProvider, time.Minute)
	pair := domain.Pair("BTCUSD")

	mockProvider.SetPanic(pair, true)
	
	mockLogger.On("Debug", "PANIC in provider.Fetch for pair %s: %v", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", "Failed to get LTP for %s: %v", mock.Anything, mock.Anything).Return()

	result := service.GetLTP(pair)
	assert.Equal(t, domain.LTP{}, result)
	
	mockLogger.AssertCalled(t, "Debug", "PANIC in provider.Fetch for pair %s: %v", mock.Anything, mock.Anything)
	mockLogger.AssertCalled(t, "Warn", "Failed to get LTP for %s: %v", mock.Anything, mock.Anything)
}

func TestGetLTP_SingleFlight(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTCUSD")
	expectedLTP := domain.LTP{
		Pair:      pair,
		Amount:    50000.00,
		Timestamp: time.Now(),
	}

	mockProvider.SetDelay(pair, 100*time.Millisecond)
	mockProvider.SetResponse(pair, expectedLTP)

	var wg sync.WaitGroup
	results := make([]domain.LTP, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = service.GetLTP(pair)
		}(i)
	}
	wg.Wait()

	for _, result := range results {
		assert.Equal(t, expectedLTP, result)
	}

	assert.Equal(t, 1, mockProvider.GetCallCount(pair))
}

func TestGetLTPs_EdgeCases(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	mockLogger := new(MockLogger)

	originalLogger := log.GetInstance()
	log.SetInstance(mockLogger)
	defer log.SetInstance(originalLogger)

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	mockLogger.On("Warn", "Cannot found a LTP for pair %v", mock.Anything).Return()
	
	result := service.GetLTPs([]domain.Pair{})
	assert.Nil(t, result)
	mockLogger.AssertCalled(t, "Warn", "Cannot found a LTP for pair %v", mock.Anything)

	mockLogger.ExpectedCalls = nil

	mockLogger.On("Warn", "Failed to get LTP for fetch for pair %s", mock.Anything).Return()
	mockLogger.On("Warn", "Cannot found a LTP for pair %v", mock.Anything).Return()

	result = service.GetLTPs([]domain.Pair{"BTCUSD", "BTCEUR"})
	assert.Nil(t, result)
	mockLogger.AssertExpectations(t)
}

func TestForceRefresh_EmptyLTP(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTCUSD")
	mockProvider.SetResponse(pair, domain.LTP{})

	result := service.ForceRefresh(pair)
	assert.Equal(t, domain.LTP{}, result)

	cached, exists := mockCache.Get(pair)
	assert.True(t, exists)
	assert.Equal(t, domain.LTP{}, cached)
}
