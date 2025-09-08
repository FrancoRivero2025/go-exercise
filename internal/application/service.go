package application

import (
	"fmt"
	"time"

	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"golang.org/x/sync/singleflight"
)

type MarketDataProvider interface {
	Fetch(pair domain.Pair) domain.LTP
}

type LTPService struct {
	cache    domain.Cache
	provider MarketDataProvider
	ttl      time.Duration
	sf       singleflight.Group
}

func NewLTPService(c domain.Cache, p MarketDataProvider, ttl time.Duration) *LTPService {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &LTPService{
		cache:    c,
		provider: p,
		ttl:      ttl,
	}
}

func (s *LTPService) GetLTP(pair domain.Pair) domain.LTP {
	if ltp, ok := s.cache.Get(pair); ok && time.Since(ltp.Timestamp) < s.ttl {
		return ltp
	}

	val, err, _ := s.sf.Do(string(pair), func() (result interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.GetInstance().Debug("PANIC in provider.Fetch for pair %s: %v", string(pair), r)
				err = fmt.Errorf("Service temporarily unavailable")
			}
		}()

		ltp := s.provider.Fetch(pair)
		s.cache.Set(pair, ltp)
		return ltp, nil
	})

	if err != nil {
		log.GetInstance().Warn("Failed to get LTP for %s: %v", string(pair), err)
		return domain.LTP{}
	}

	return val.(domain.LTP)
}

func (s *LTPService) GetLTPs(pairs []domain.Pair) []domain.LTP {
	var out []domain.LTP
	empty_response := domain.LTP{}
	for _, p := range pairs {
		ltp := s.GetLTP(p)
		if ltp != empty_response {
			out = append(out, ltp)
		} else {
			log.GetInstance().Warn("Failed to get LTP for fetch for pair %s", string(p))
		}
	}
	if len(out) == 0 {
		log.GetInstance().Warn("Cannot found a LTP for pair %v", pairs)
		return nil
	}
	return out
}

func (s *LTPService) GetAllLTPs() []domain.LTP {
	cfg := config.GetInstance()
	return s.GetLTPs([]domain.Pair(cfg.Pairs))
}

func (s *LTPService) RefreshPairs(pairs []domain.Pair) {
	empty_response := domain.LTP{}
	for _, p := range pairs {
		ltp := s.provider.Fetch(p)
		if ltp == empty_response {
			log.GetInstance().Warn("Cannot refresh and update cache", pairs)
			continue
		}
		s.cache.Set(p, ltp)
	}
}
