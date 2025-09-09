package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
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
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer func() {
			stop()
		}()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/tmp/local.yaml"
	}

	cfg := config.Initialize(configPath)
	logger := log.GetInstance()

	useRedis := os.Getenv("USE_REDIS") == "true"

	var c domain.Cache
	if useRedis {
		redisTTL := os.Getenv("REDIS_TTL")
		if redisTTL == "" {
			logger.Fatal("REDIS_TTL is required when using Redis")
		}

		ttl, err := time.ParseDuration(redisTTL)
		if err != nil {
			logger.Fatal("invalid REDIS_TTL: %v", err)
		}

		redisDB := 0
		if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
			if db, err := strconv.Atoi(dbStr); err == nil {
				redisDB = db
			}
		}

		redisCache := cache.NewRedisCache(
			os.Getenv("REDIS_ADDR"),
			os.Getenv("REDIS_PASSWORD"),
			redisDB,
			ttl,
		)
		if err != nil {
			logger.Fatal("failed to create Redis cache: %v", err)
		}
		c = redisCache
		logger.Info("Using Redis cache")
	} else {
		c = cache.NewInMemoryCache(time.Duration(cfg.Cache.TTL) * time.Second)
		logger.Info("Using in-memory cache")
	}

	krakenClient := kraken.NewClient(cfg.Kraken.URL, 15)

	service := application.NewLTPService(c, krakenClient, time.Duration(cfg.Cache.TTL)*time.Second)

	httpHandler := httpapi.NewHandler(service)

	refresherInterval := 30 * time.Second
	ref := refresher.NewRefresher(service, cfg.Pairs, refresherInterval)
	ref.Start()
	defer ref.Stop()

	r := chi.NewRouter()
	r.Mount("/", httpHandler.Router())

	addr := ":" + strconv.Itoa(cfg.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("starting server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, initiating graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error: %v", err)
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	ref.Stop()
	logger.Info("Refresher stopped")

	if closer, ok := c.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Cache close error: %v", err)
		} else {
			logger.Info("Cache connections closed")
		}
	}

	wg.Wait()
	logger.Info("All components stopped successfully")
}
