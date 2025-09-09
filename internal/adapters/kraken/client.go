package kraken

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/cenkalti/backoff/v4"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout uint) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

type krakenTickerResp struct {
	Error  []string                     `json:"error"`
	Result map[string]krakenTickerEntry `json:"result"`
}

type krakenTickerEntry struct {
	A []string `json:"a"`
	B []string `json:"b"`
	C []string `json:"c"`
	V []string `json:"v"`
}

func (c *Client) Fetch(pair domain.Pair) domain.LTP {
	symbolPair, err := convertCurrencyPairToKrakenSymbol(string(pair))
	if err != nil {
		return domain.LTP{
			Pair:      pair,
			Error:     fmt.Sprintf("unsupported pair: %s", err.Error()),
			Timestamp: time.Now().UTC(),
		}
	}

	var parsed krakenTickerResp
	op := func() error {
		url := fmt.Sprintf("%s/0/public/Ticker?pair=%s", c.baseURL, symbolPair)
		resp, err := c.http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return err
		}
		if len(parsed.Error) > 0 {
			return fmt.Errorf("kraken error: %v", parsed.Error)
		}
		return nil
	}

	// Retry with exponential backoff (max 3 attempts)
	expBackoff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
	if err := backoff.Retry(op, expBackoff); err != nil {
		log.GetInstance().Warn("Failed to fetch pair %s: %v", pair, err)
		return domain.LTP{
			Pair:      pair,
			Error:     err.Error(),
			Timestamp: time.Now().UTC(),
		}
	}

	entry, exists := parsed.Result[symbolPair]
	if !exists {
		return domain.LTP{
			Pair:      pair,
			Error:     fmt.Sprintf("Pair %s not found in response", symbolPair),
			Timestamp: time.Now().UTC(),
		}
	}

	if len(entry.C) == 0 {
		return domain.LTP{
			Pair:      pair,
			Error:     "No last trade price data available",
			Timestamp: time.Now().UTC(),
		}
	}

	price, err := decimal.NewFromString(entry.C[0])
	if err != nil {
		return domain.LTP{
			Pair:      pair,
			Error:     fmt.Sprintf("Invalid price format: %v", err),
			Timestamp: time.Now().UTC(),
		}
	}

	return domain.LTP{
		Pair:      pair,
		Amount:    price,
		Timestamp: time.Now().UTC(),
	}
}

func convertCurrencyPairToKrakenSymbol(pair string) (string, error) {
	krakenSymbols := map[string]string{
		"BTC/USD": "XXBTZUSD",
		"BTC/CHF": "XXBTZCHF",
		"BTC/EUR": "XXBTZEUR",
	}

	symbol, exists := krakenSymbols[pair]
	if !exists {
		return "", fmt.Errorf("pair not supported: %s", pair)
	}

	return symbol, nil
}
