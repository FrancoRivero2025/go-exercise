//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/kraken"
	"github.com/FrancoRivero2025/go-exercise/internal/application"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/FrancoRivero2025/go-exercise/internal/domain/mocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_LTP_Endpoint(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)

	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedPairs  int
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name:           "Single pair request",
			url:            "/api/v1/ltp?pairs=BTC/USD",
			expectedStatus: http.StatusOK,
			expectedPairs:  1,
			checkData:      func(t *testing.T, data interface{}) {},
		},
		{
			name:           "Multiple pairs request",
			url:            "/api/v1/ltp?pairs=BTC/USD,BTC/EUR",
			expectedStatus: http.StatusOK,
			expectedPairs:  2,
			checkData:      func(t *testing.T, data interface{}) {},
		},
		{
			name:           "No pairs parameter - get all",
			url:            "/api/v1/ltp",
			expectedStatus: http.StatusOK,
			expectedPairs:  3,
			checkData:      func(t *testing.T, data interface{}) {},
		},
		{
			name:           "Non-existent pair",
			url:            "/api/v1/ltp?pairs=INVALID/USD",
			expectedStatus: http.StatusOK,
			expectedPairs:  1,
			checkData:      func(t *testing.T, data interface{}) {},
		},
		{
			name:           "Mixed valid and invalid pairs",
			url:            "/api/v1/ltp?pairs=BTC/USD,INVALID/USD,BTC/EUR",
			expectedStatus: http.StatusOK,
			expectedPairs:  3,
			checkData:      func(t *testing.T, data interface{}) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", server.URL+tt.url, nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				var response successResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, float64(tt.expectedPairs), response.Meta["count"])
				tt.checkData(t, response.Data)
			} else if resp.StatusCode == http.StatusNotFound {
				var response errorResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "Requested pairs not found", response.Error)
				assert.Equal(t, "NOT_FOUND", response.Code)
			}
		})
	}
}

func TestIntegration_Health_Endpoints(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)
	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("Health endpoint", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Status string `json:"status"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "ok", response.Status)
	})

	t.Run("Ready endpoint with service", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Status string `json:"status"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "ready", response.Status)
	})
}

func TestIntegration_Edge_Cases(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)

	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "Empty pairs parameter",
			url:            "/api/v1/ltp?pairs=",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Duplicate pairs",
			url:            "/api/v1/ltp?pairs=BTC/USD,BTC/USD,ETH/USD",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Pairs with spaces",
			url:            "/api/v1/ltp?pairs=%20BTC/USD%20,%20ETH/USD%20",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid query parameter",
			url:            "/api/v1/ltp?invalid=param",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(server.URL + tt.url)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestIntegration_LTP_ExactDecimal_Precision(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)
	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(server.URL + "/api/v1/ltp?pairs=BTC/USD")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response struct {
		Data []struct {
			Pair   string `json:"pair"`
			Amount string `json:"amount"`
		} `json:"data"`
		Meta map[string]interface{} `json:"meta"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	require.NotEmpty(t, response.Data)
	ltp := response.Data[0]

	assert.IsType(t, "", ltp.Amount)

	d, err := decimal.NewFromString(ltp.Amount)
	require.NoError(t, err)

	str := d.String()
	assert.Equal(t, ltp.Amount, str, "amount should remain exact after decimal parse")

	assert.NotContains(t, ltp.Amount, "e")
	assert.NotContains(t, ltp.Amount, "E")
}

func TestIntegration_LTP_MultiplePairs_ExactDecimal(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)
	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(server.URL + "/api/v1/ltp?pairs=BTC/USD,BTC/EUR")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response struct {
		Data []struct {
			Pair   string `json:"pair"`
			Amount string `json:"amount"`
		} `json:"data"`
		Meta map[string]interface{} `json:"meta"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	require.Len(t, response.Data, 2)

	for _, ltp := range response.Data {
		_, err := decimal.NewFromString(ltp.Amount)
		require.NoError(t, err)
		assert.NotContains(t, ltp.Amount, "e")
	}
}

func TestIntegration_LTP_JSONFormatting(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)
	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(server.URL + "/api/v1/ltp?pairs=BTC/USD")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	rawBody := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&rawBody)
	require.NoError(t, err)

	data, ok := rawBody["data"].([]interface{})
	require.True(t, ok, "data should be an array")
	require.NotEmpty(t, data)

	first, ok := data[0].(map[string]interface{})
	require.True(t, ok, "first element should be an object")

	_, hasPair := first["pair"]
	_, hasAmount := first["amount"]
	assert.True(t, hasPair)
	assert.True(t, hasAmount)

	amountVal, ok := first["amount"].(string)
	require.True(t, ok, "amount must be a string in JSON")

	assert.NotContains(t, amountVal, "e")
	assert.NotContains(t, amountVal, "E")

	if strings.Contains(amountVal, ".") {
		parts := strings.Split(amountVal, ".")
		assert.True(t, len(parts[1]) > 0, "amount must have decimals after the dot")
	}
}

func TestIntegration_LTP_JSONFormatting_MultiplePairs(t *testing.T) {
	cfg := config.Initialize("")

	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)
	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(server.URL + "/api/v1/ltp?pairs=BTC/USD,BTC/EUR")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	rawBody := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&rawBody)
	require.NoError(t, err)

	data, ok := rawBody["data"].([]interface{})
	require.True(t, ok, "data should be array")
	require.Len(t, data, 2)

	for _, entry := range data {
		obj, ok := entry.(map[string]interface{})
		require.True(t, ok)

		amountVal, ok := obj["amount"].(string)
		require.True(t, ok, "amount must be string")
		assert.NotContains(t, amountVal, "e")
		assert.NotContains(t, amountVal, "E")
	}
}

func TestIntegration_Ready_NotReady(t *testing.T) {
	handler := &Handler{service: nil}
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(server.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var response statusResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "not ready", response.Status)
}

func TestIntegration_Metrics_Endpoint(t *testing.T) {
	cfg := config.Initialize("")
	cache := mocks.NewMockCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)
	service := application.NewLTPService(cache, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)

	handler := NewHandler(service)
	server := httptest.NewServer(handler.Router())
	defer server.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "http_requests_total")
	assert.Contains(t, string(body), "cache_hits_total")
}

func Test_parsePairsParam(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []domain.Pair
	}{
		{"empty string", "", []domain.Pair{}},
		{"single pair", "BTC/USD", []domain.Pair{"BTC/USD"}},
		{"multiple with spaces", " BTC/USD , ETH/USD ", []domain.Pair{"BTC/USD", "ETH/USD"}},
		{"duplicates", "BTC/USD,BTC/USD", []domain.Pair{"BTC/USD"}},
		{"extra commas", ",BTC/USD,,ETH/USD,", []domain.Pair{"BTC/USD", "ETH/USD"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePairsParam(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

type badWriter struct{}

func (bw badWriter) Header() http.Header        { return http.Header{} }
func (bw badWriter) Write([]byte) (int, error)  { return 0, fmt.Errorf("write error") }
func (bw badWriter) WriteHeader(statusCode int) {}

func Test_respondJSON_Error(t *testing.T) {
	bw := badWriter{}
	respondJSON(bw, http.StatusOK, map[string]interface{}{"x": make(chan int)})
}

func Test_CacheMetricsHelpers(t *testing.T) {
	IncrementCacheHit()
	IncrementCacheMiss()
}

func Test_metricsMiddleware(t *testing.T) {
	handler := metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTeapot, rec.Code)
}
