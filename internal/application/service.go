package application

import (
	"errors"
	"time"

	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/domain"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("not found")

type Cache interface {
	Get(pair domain.Pair) (domain.LTP, bool)
	Set(pair domain.Pair, ltp domain.LTP)
}

type MarketDataProvider interface {
	Fetch(pair domain.Pair) (domain.LTP, error)
}

type LTPService struct {
	cache    Cache
	provider MarketDataProvider
	ttl      time.Duration
	sf       singleflight.Group
}

func NewLTPService(c Cache, p MarketDataProvider, ttl time.Duration) *LTPService {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &LTPService{
		cache:    c,
		provider: p,
		ttl:      ttl,
	}
}

// GetLTP returns the LTP for a single pair
func (s *LTPService) GetLTP(pair domain.Pair) (domain.LTP, error) {
	// cache check
	if ltp, ok := s.cache.Get(pair); ok && time.Since(ltp.Timestamp) < s.ttl {
		return ltp, nil
	}

	// singleflight to avoid thundering herd
	val, err := s.sf.Do(string(pair), func() (interface{}, error) {
		ltp, err := s.provider.Fetch(pair)
		if err != nil {
			return nil, err
		}
		s.cache.Set(pair, ltp)
		return ltp, nil
	})
	if err != nil {
		return domain.LTP{}, err
	}
	return val.(domain.LTP), nil
}

// GetLTPs returns LTPs for multiple pairs (best-effort: returns successful ones)
func (s *LTPService) GetLTPs(pairs []domain.Pair) ([]domain.LTP, error) {
	var out []domain.LTP
	for _, p := range pairs {
		ltp, err := s.GetLTP(p)
		if err == nil {
			out = append(out, ltp)
		}
		// if error, we skip — caller can decide
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}

// RefreshPairs forces fetch for list of pairs and updates cache
func (s *LTPService) RefreshPairs(pairs []domain.Pair) {
	for _, p := range pairs {
		ltp, err := s.provider.Fetch(p)
		if err != nil {
			// TODO: add logging / metrics
			continue
		}
		s.cache.Set(p, ltp)
	}
}
