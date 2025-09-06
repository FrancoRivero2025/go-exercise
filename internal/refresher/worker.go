package refresher

import (
	"time"

	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/application"
	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/domain"
)

type Refresher struct {
	service  *application.LTPService
	pairs    []domain.Pair
	interval time.Duration
	quit     chan struct{}
}

func NewRefresher(s *application.LTPService, pairs []string, interval time.Duration) *Refresher {
	dpairs := make([]domain.Pair, 0, len(pairs))
	for _, p := range pairs {
		dpairs = append(dpairs, domain.Pair(p))
	}
	return &Refresher{
		service:  s,
		pairs:    dpairs,
		interval: interval,
		quit:     make(chan struct{}),
	}
}

func (r *Refresher) Start() {
	go func() {
		t := time.NewTicker(r.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				r.service.RefreshPairs(r.pairs)
			case <-r.quit:
				return
			}
		}
	}()
}

func (r *Refresher) Stop() {
	close(r.quit)
}
