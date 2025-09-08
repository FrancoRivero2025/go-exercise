package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type TickerResponse struct {
	Error  []string               `json:"error"`
	Result map[string]TickerEntry `json:"result"`
}

type TickerEntry struct {
	C []string `json:"c"`
}

func main() {
	url := "https://api.kraken.com/0/public/Ticker?pair=BTCUSD"

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var tickerResp TickerResponse
	if err := json.Unmarshal(body, &tickerResp); err != nil {
		panic(err)
	}

	for pair, data := range tickerResp.Result {
		fmt.Printf("Último precio de %s: %s USDT\n", pair, data.C[0])
	}
}
