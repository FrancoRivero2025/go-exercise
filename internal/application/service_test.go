package application

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/FrancoRivero2025/go-exercise/internal/domain/mocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) SetLevel(level int)                    {}
func (m *MockLogger) SetOutput(w io.Writer)                 {}
func (m *MockLogger) SetOutputToFile(filename string) error { return nil }
func (m *MockLogger) GetLevel() int                         { return 0 }
func (m *MockLogger) Debug(message string, v ...interface{}) {
	m.Called(append([]interface{}{message}, v...)...)
}
func (m *MockLogger) Info(message string, v ...interface{}) {}
func (m *MockLogger) Warn(message string, v ...interface{}) {
	m.Called(append([]interface{}{message}, v...)...)
}
func (m *MockLogger) Error(message string, v ...interface{}) {}
func (m *MockLogger) Fatal(message string, v ...interface{}) {}

func createLTP(pair domain.Pair, amount string, timestamp time.Time) domain.LTP {
	decAmount, _ := decimal.NewFromString(amount)
	return domain.LTP{
		Pair:      pair,
		Amount:    decAmount,
		Timestamp: timestamp,
	}
}

func VerifyLTPConsistency(ltp domain.LTP) error {
	data, err := json.Marshal(ltp)
	if err != nil {
		return fmt.Errorf("Marshal error: %w", err)
	}

	var ltp2 domain.LTP
	err = json.Unmarshal(data, &ltp2)
	if err != nil {
		return fmt.Errorf("Unmarshal error: %w", err)
	}

	if ltp.Pair != ltp2.Pair {
		return fmt.Errorf("Pair mismatch: %s != %s", ltp.Pair, ltp2.Pair)
	}

	if !ltp.Amount.Equal(ltp2.Amount) {
		return fmt.Errorf("Amount mismatch: %s != %s",
			ltp.Amount.String(), ltp2.Amount.String())
	}

	if !ltp.Timestamp.Equal(ltp2.Timestamp) {
		return fmt.Errorf("Timestamp mismatch: %v != %v",
			ltp.Timestamp, ltp2.Timestamp)
	}

	return nil
}

func TestLTPService_GetLTP(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTC/USD")
	result := service.GetLTP(pair)
	assert.True(t, result.IsEmpty())

	expectedLTP := createLTP(pair, "50000.00", time.Now())
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
	pairs := []domain.Pair{"BTC/USD", "BTC/EUR"}

	results := service.GetLTPs(pairs)
	assert.Nil(t, results)

	mockLogger.AssertExpectations(t)

	mockLogger.ExpectedCalls = nil

	btcLTP := createLTP("BTC/USD", "50000.00", time.Now())
	btcEurLTP := createLTP("BTC/EUR", "45000.00", time.Now())

	mockProvider.SetResponse("BTC/USD", btcLTP)
	mockProvider.SetResponse("BTC/EUR", btcEurLTP)

	results = service.GetLTPs(pairs)
	require.Len(t, results, 2)
	assert.Contains(t, results, btcLTP)
	assert.Contains(t, results, btcEurLTP)
}

func TestLTPService_ForceRefresh(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTC/USD")
	newLTP := createLTP(pair, "51000.00", time.Now())

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

	pair := domain.Pair("BTC/USD")
	ltp := createLTP(pair, "50000.00", time.Now())

	mockProvider.SetResponse(pair, ltp)

	result := service.GetLTP(pair)
	assert.Equal(t, ltp, result)

	result = service.GetLTP(pair)
	assert.Equal(t, ltp, result)

	time.Sleep(time.Millisecond * 20)

	newLTP := createLTP(pair, "51000.00", time.Now())
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
		Pairs: []domain.Pair{"BTC/USD", "BTC/EUR"},
	}
	config.SetInstance(testCfg)
	defer config.SetInstance(originalCfg)

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()
	result := service.GetAllLTPs()
	assert.Nil(t, result)

	btcLTP := createLTP("BTC/USD", "50000.00", time.Now())
	btcEurLTP := createLTP("BTC/EUR", "45000.00", time.Now())
	mockProvider.SetResponse("BTC/USD", btcLTP)
	mockProvider.SetResponse("BTC/EUR", btcEurLTP)

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
	pairs := []domain.Pair{"BTC/USD", "BTC/EUR"}

	mockLogger.On("Warn", "Cannot refresh and update cache", mock.Anything).Return()
	mockProvider.SetResponse("BTC/USD", domain.LTP{})
	mockProvider.SetResponse("BTC/EUR", domain.LTP{})

	service.RefreshPairs(pairs)

	mockLogger.AssertCalled(t, "Warn", "Cannot refresh and update cache", mock.Anything)

	mockLogger.ExpectedCalls = nil

	btcLTP := createLTP("BTC/USD", "50000.00", time.Now())
	btcEurLTP := createLTP("BTC/EUR", "45000.00", time.Now())
	mockProvider.SetResponse("BTC/USD", btcLTP)
	mockProvider.SetResponse("BTC/EUR", btcEurLTP)

	service.RefreshPairs(pairs)

	cachedBTC, exists := mockCache.Get("BTC/USD")
	assert.True(t, exists)
	assert.Equal(t, btcLTP, cachedBTC)

	cachedBTCEUR, exists := mockCache.Get("BTC/EUR")
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
	pair := domain.Pair("BTC/USD")

	mockProvider.SetPanic(pair, true)

	mockLogger.On("Debug", "PANIC in provider.Fetch for pair %s: %v", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", "Failed to get LTP for %s: %v", mock.Anything, mock.Anything).Return()

	result := service.GetLTP(pair)
	assert.True(t, result.IsEmpty())

	mockLogger.AssertCalled(t, "Debug", "PANIC in provider.Fetch for pair %s: %v", mock.Anything, mock.Anything)
	mockLogger.AssertCalled(t, "Warn", "Failed to get LTP for %s: %v", mock.Anything, mock.Anything)
}

func TestGetLTP_SingleFlight(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTC/USD")
	expectedLTP := createLTP(pair, "50000.00", time.Now())

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

	result = service.GetLTPs([]domain.Pair{"BTC/USD", "BTC/EUR"})
	assert.Nil(t, result)
	mockLogger.AssertExpectations(t)
}

func TestForceRefresh_EmptyLTP(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	pair := domain.Pair("BTC/USD")
	mockProvider.SetResponse(pair, domain.LTP{})

	result := service.ForceRefresh(pair)
	assert.True(t, result.IsEmpty())

	cached, exists := mockCache.Get(pair)
	assert.True(t, exists)
	assert.True(t, cached.IsEmpty())
}

func TestLTP_DecimalPrecision(t *testing.T) {
	testCases := []struct {
		name     string
		amount   string
		expected string
	}{
		{"test 1 0.1", "0.1", "0.1"},
		{"test 2 0.2", "0.2", "0.2"},
		{"test 3", "0.1", "0.1"},
		{"test 4", "1000000.00000001", "1000000.00000001"},
		{"test 5", "0.0000000000000001", "0.0000000000000001"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ltp, err := NewLTP(domain.Pair("BTC/USD"), tc.amount, time.Now())
			require.NoError(t, err)

			assert.Equal(t, tc.expected, ltp.Amount.String(),
				"La precisión decimal no se mantuvo")
		})
	}
}

func TestLTP_JSONPrecision(t *testing.T) {
	testCases := []struct {
		name   string
		amount string
	}{
		{"test 1", "123.4567890123456789"},
		{"test 2", "0.1"},
		{"test 3 0.2", "0.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amount, _ := decimal.NewFromString(tc.amount)
			ltp := domain.LTP{
				Pair:      domain.Pair("BTC/USD"),
				Amount:    amount,
				Timestamp: time.Now(),
			}

			data, err := json.Marshal(ltp)
			require.NoError(t, err)

			var jsonMap map[string]interface{}
			err = json.Unmarshal(data, &jsonMap)
			require.NoError(t, err)

			amountStr, ok := jsonMap["amount"].(string)
			require.True(t, ok, "Amount debería ser string en JSON")
			assert.Equal(t, tc.amount, amountStr, "Formato JSON incorrecto")

			var ltp2 domain.LTP
			err = json.Unmarshal(data, &ltp2)
			require.NoError(t, err)

			assert.True(t, ltp.Amount.Equal(ltp2.Amount),
				"Precisión perdida en serialización JSON")
			assert.Equal(t, ltp.Pair, ltp2.Pair)
			assert.WithinDuration(t, ltp.Timestamp, ltp2.Timestamp, time.Millisecond)
		})
	}
}

func TestLTP_ArithmeticPrecision(t *testing.T) {
	a, _ := decimal.NewFromString("0.1")
	b, _ := decimal.NewFromString("0.2")

	result := a.Add(b)
	expected, _ := decimal.NewFromString("0.3")

	assert.True(t, result.Equal(expected),
		"0.1 + 0.2 should be exactly 0.3, got %s", result.String())
}

func TestLTP_VerifyConsistency(t *testing.T) {
	testCases := []struct {
		name   string
		amount string
	}{
		{"test 1", "50000.00"},
		{"test 2", "50000.1234567890123456"},
		{"test 3", "0.1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ltp, err := NewLTP(domain.Pair("BTC/USD"), tc.amount, time.Now())
			require.NoError(t, err)

			err = VerifyLTPConsistency(ltp)
			assert.NoError(t, err, "The consistency of the LTP was not maintained.")
		})
	}
}

func TestLTP_PrecisionInServiceOperations(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	precisionLTP := createLTP("BTC/USD", "0.1", time.Now())
	mockProvider.SetResponse("BTC/USD", precisionLTP)

	result := service.GetLTP("BTC/USD")
	assert.True(t, precisionLTP.Amount.Equal(result.Amount),
		"Accuracy lost during service operation: expected %s, got %s",
		precisionLTP.Amount.String(), result.Amount.String())
}

func TestLTP_EdgeCasePrecision(t *testing.T) {
	edgeCases := []struct {
		name     string
		amount   string
		expected string
	}{
		{"test 1", "1.12345678901234567890", "1.1234567890123456789"},
		{"test 2", "0.0", "0"},
		{"test 3", "-1.5", "-1.5"},
		{"test 4", "1e-10", "0.0000000001"},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			ltp, err := NewLTP(domain.Pair("TEST/USD"), tc.amount, time.Now())
			if err != nil {
				t.Logf("Case %s generated an error (could be expected): %v", tc.name, err)
				return
			}

			assert.Equal(t, tc.expected, ltp.Amount.String())
		})
	}
}

func TestLTP_ServicePrecisionWithMultipleOperations(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Minute)

	initialAmount := "100.0000000000000001"
	ltp := createLTP("BTC/USD", initialAmount, time.Now())
	mockProvider.SetResponse("BTC/USD", ltp)

	for i := 0; i < 10; i++ {
		result := service.GetLTP("BTC/USD")
		assert.Equal(t, initialAmount, result.Amount.String(),
			"Loss of accuracy during operation %d", i+1)
	}

	refreshed := service.ForceRefresh("BTC/USD")
	assert.Equal(t, initialAmount, refreshed.Amount.String(),
		"Loss of accuracy during force refresh")
}

func TestLTP_PrecisionInCache(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()
	service := NewLTPService(mockCache, mockProvider, time.Hour)

	precisionAmount := "123.4567890123456789"
	ltp := createLTP("BTC/USD", precisionAmount, time.Now())
	mockProvider.SetResponse("BTC/USD", ltp)

	result1 := service.GetLTP("BTC/USD")
	assert.Equal(t, precisionAmount, result1.Amount.String())

	result2 := service.GetLTP("BTC/USD")
	assert.Equal(t, precisionAmount, result2.Amount.String())

	cached, exists := mockCache.Get("BTC/USD")
	assert.True(t, exists)
	assert.Equal(t, precisionAmount, cached.Amount.String())
}

func TestGetLTPs_PerPairErrorObjects(t *testing.T) {
	mockCache := mocks.NewMockCache()
	mockProvider := mocks.NewMockMarketDataProvider()

	service := NewLTPService(mockCache, mockProvider, time.Minute)

	errLTP := domain.LTP{
		Pair:      domain.Pair("BTC/USD"),
		Amount:    decimal.Zero,
		Error:     "timeout contacting external provider",
		Timestamp: time.Now().UTC(),
	}
	validLTP := createLTP("BTC/EUR", "45000.00", time.Now())

	mockProvider.SetResponse("BTC/USD", errLTP)
	mockProvider.SetResponse("BTC/EUR", validLTP)

	pairs := []domain.Pair{"BTC/USD", "BTC/EUR"}
	results := service.GetLTPs(pairs)

	require.NotNil(t, results)
	require.Len(t, results, 2)

	var gotErr bool
	var gotValid bool
	for _, r := range results {
		if string(r.Pair) == "BTC/USD" {
			require.NotEmpty(t, r.Error, "Expected error for BTC/USD to be propagated")
			assert.Equal(t, errLTP.Error, r.Error)
			gotErr = true
		}
		if string(r.Pair) == "BTC/EUR" {
			require.True(t, r.Amount.Equal(validLTP.Amount), "Expected BTC/EUR amount to match")
			require.Empty(t, r.Error)
			gotValid = true
		}
	}
	assert.True(t, gotErr, "Did not find BTC/USD in results")
	assert.True(t, gotValid, "Did not find BTC/EUR in results")
}
