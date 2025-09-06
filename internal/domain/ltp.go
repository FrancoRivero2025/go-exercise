package domain

import "time"

type Pair string

type LTP struct {
	Pair      Pair      `json:"pair"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}
