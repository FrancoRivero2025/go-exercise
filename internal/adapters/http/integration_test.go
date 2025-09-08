//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/kraken"
	"github.com/FrancoRivero2025/go-exercise/internal/application"
	"github.com/FrancoRivero2025/go-exercise/internal/domain/mocks"
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

	client := &http.Client{Timeout: 5 * time.Second}

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

	client := &http.Client{Timeout: 5 * time.Second}

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

	client := &http.Client{Timeout: 5 * time.Second}

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
