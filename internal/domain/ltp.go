package domain

import "time"

type Pair string

type LTP struct {
	Pair      Pair      `json:"pair"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type Cache interface {
	Get(pair Pair) (LTP, bool)
	Set(pair Pair, ltp LTP)
}