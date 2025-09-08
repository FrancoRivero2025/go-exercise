package kraken

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
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

func (c *Client) Fetch(pair domain.Pair) (result domain.LTP) {
	defer func() {
		if r := recover(); r != nil {
			log.GetInstance().Warn("Recovered from panic: %v", r)
			result = domain.LTP{
				Pair:      pair,
				Amount:    -1,
				Timestamp: time.Now().UTC(),
			}
		}
	}()

	symbol, err := convertCurrencyPair(string(pair))
	if err != nil {
		panic(fmt.Sprintf("Unsupported pair: %s", pair))
	}

	url := fmt.Sprintf("%s/0/public/Ticker?pair=%s", c.baseURL, symbol)
	resp, err := c.http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var parsed krakenTickerResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		panic(err)
	}
	if len(parsed.Error) > 0 {
		panic(fmt.Sprintf("Kraken error: %v", parsed.Error))
	}
	for _, entry := range parsed.Result {
		if len(entry.C) == 0 {
			continue
		}
		priceStr := entry.C[0]
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			panic(err)
		}
		return domain.LTP{
			Pair:      pair,
			Amount:    price,
			Timestamp: time.Now().UTC(),
		}
	}
	panic(fmt.Sprintf("No price data for %s", pair))
}

func convertCurrencyPair(pair string) (string, error) {
	parts := strings.Split(pair, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("Invalid pair")
	}

	base := parts[0]
	quote := parts[1]

	conversionMap := map[string]string{
		"BTC": "XXBT",
		"ETH": "XETH",
		"LTC": "XLTC",
		"XRP": "XXRP",
		"ADA": "ADA",
		"CHF": "ZCHF",
		"USD": "ZUSD",
		"EUR": "ZEUR",
		"GBP": "ZGBP",
		"JPY": "ZJPY",
	}

	convertedBase, ok := conversionMap[base]
	if !ok {
		return "", fmt.Errorf("Crypto currency not supported: %s", base)
	}

	convertedQuote, ok := conversionMap[quote]
	if !ok {
		return "", fmt.Errorf("Fiat currency not supported: %s", quote)
	}

	return convertedBase + convertedQuote, nil
}
