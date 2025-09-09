package application

import (
	"fmt"
	"net/http"
	"time"

	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"
)

type MarketDataProvider interface {
	Fetch(pair domain.Pair) domain.LTP
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type LTPService struct {
	cache      domain.Cache
	provider   MarketDataProvider
	ttl        time.Duration
	sf         singleflight.Group
	httpClient HTTPClient
	baseURL    string
}

func NewLTPService(c domain.Cache, p MarketDataProvider, ttl time.Duration) *LTPService {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &LTPService{
		cache:    c,
		provider: p,
		ttl:      ttl,
		sf:       singleflight.Group{},
	}
}

func NewTestLTPService(c domain.Cache, p MarketDataProvider, ttl time.Duration, httpClient HTTPClient) *LTPService {
	service := NewLTPService(c, p, ttl)
	service.httpClient = httpClient
	return service
}

func (s *LTPService) SetHTTPClient(client HTTPClient) {
	s.httpClient = client
}

func (s *LTPService) SetBaseURL(url string) {
	s.baseURL = url
	if setter, ok := s.provider.(interface{ SetBaseURL(string) }); ok {
		setter.SetBaseURL(url)
	}
}

func (s *LTPService) GetCache() domain.Cache {
	return s.cache
}

func (s *LTPService) GetLTP(pair domain.Pair) domain.LTP {
	if ltp, ok := s.cache.Get(pair); ok && time.Since(ltp.Timestamp) < s.ttl {
		return ltp
	}

	val, err, _ := s.sf.Do(string(pair), func() (result interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.GetInstance().Debug("PANIC in provider.Fetch for pair %s: %v", string(pair), r)
				err = fmt.Errorf("service temporarily unavailable")
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
	for _, p := range pairs {
		ltp := s.GetLTP(p)
		if ltp != (domain.LTP{}) {
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
	for _, p := range pairs {
		ltp := s.provider.Fetch(p)
		if ltp == (domain.LTP{}) {
			log.GetInstance().Warn("Cannot refresh and update cache", pairs)
			continue
		}
		s.cache.Set(p, ltp)
	}
}

func (s *LTPService) ForceRefresh(pair domain.Pair) domain.LTP {
	ltp := s.provider.Fetch(pair)
	s.cache.Set(pair, ltp)
	return ltp
}

func (s *LTPService) CheckRedisConnectivity() bool {
	if s.cache == nil {
		return false
	}
	return s.cache.CheckConnectivity()
}

func (s *LTPService) CheckKrakenConnectivity() bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get("https://api.kraken.com/0/public/Time")
	if err != nil {
		return false
	}
	defer func() {
			if err := resp.Body.Close(); err != nil {
				log.GetInstance().Error("Error close HTTP request: %v", err)
			}
	}()

	return resp.StatusCode == http.StatusOK
}

func NewLTP(pair domain.Pair, amount string, timestamp time.Time) (domain.LTP, error) {
	decAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return domain.LTP{}, fmt.Errorf("invalid decimal amount: %w", err)
	}
	return domain.LTP{
		Pair:      pair,
		Amount:    decAmount,
		Timestamp: timestamp,
	}, nil
}
