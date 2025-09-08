package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/FrancoRivero2025/go-exercise/config"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/cache"
	httpapi "github.com/FrancoRivero2025/go-exercise/internal/adapters/http"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/kraken"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/refresher"
	"github.com/FrancoRivero2025/go-exercise/internal/application"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"

	"github.com/go-chi/chi/v5"
)

func main() {

	cfg := config.Initialize("/tmp/local.yaml")
	logger := log.GetInstance()

	useRedis := os.Getenv("USE_REDIS") == "true"

	var c domain.Cache
	if useRedis {
		ttl, err := time.ParseDuration(os.Getenv("REDIS_TTL"))
		if err != nil {
			log.GetInstance().Fatal("invalid REDIS_TTL: %v", err)
		}

		c = cache.NewRedisCache(
			os.Getenv("REDIS_ADDR"),
			os.Getenv("REDIS_PASSWORD"),
			0,
			ttl,
		)
		logger.Info("Using Redis cache")
	} else {
		c = cache.NewInMemoryCache(time.Duration(cfg.Cache.TTL) * time.Second)
		logger.Info("Using in-memory cache")
	}

	krakenClient := kraken.NewClient(cfg.Kraken.URL, 5)

	service := application.NewLTPService(c, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)


	httpHandler := httpapi.NewHandler(service)

	ref := refresher.NewRefresher(service, cfg.Pairs, 30*time.Second)
	ref.Start()
	defer ref.Stop()

	r := chi.NewRouter()
	r.Mount("/", httpHandler.Router())

	addr := ":" + strconv.Itoa(cfg.Server.Port)
	log.GetInstance().Info("starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.GetInstance().Fatal(fmt.Sprintf("%w", err))
	}
}
