package main

import (
	"log"
	"net/http"
	"time"

	"github.com/FrancoRivero2025/go-exercise/ltp-service/config"
	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/adapters/cache"
	httpapi "github.com/FrancoRivero2025/go-exercise/ltp-service/internal/adapters/http"
	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/adapters/kraken"
	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/refresher"
	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/application"

	"github.com/go-chi/chi/v5"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load("config/local.yaml")
	if err != nil {
		log.Printf("warning: no config file found, using defaults: %v", err)
		cfg = config.Default()
	}

	cacheAdapter := cache.NewInMemoryCache()
	krakenClient := kraken.NewClient(cfg.Kraken.URL)

	service := application.NewLTPService(cacheAdapter, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)
	handler := httpapi.NewHandler(service)

	pairs := cfg.PairsAsDomain()
	ref := refresher.NewRefresher(service, pairs, 30*time.Second)
	ref.Start()
	defer ref.Stop()

	r := chi.NewRouter()
	r.Mount("/", handler.Router())

	// Metrics endpoint
	// r.Handle("/metrics", promhttp.Handler())

	addr := ":" + cfg.Server.PortString()
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
