package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Pair string

type LTP struct {
	Pair      Pair            `json:"pair"`
	Amount    decimal.Decimal `json:"amount,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

type Cache interface {
	Get(pair Pair) (LTP, bool)
	Set(pair Pair, ltp LTP)
	CheckConnectivity() bool
}

func (l LTP) IsEmpty() bool {
	return l.Pair == "" && l.Amount.IsZero() && l.Timestamp.IsZero()
}
