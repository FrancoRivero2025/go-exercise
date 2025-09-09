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
	defer stop()

	cfg := config.Initialize("/tmp/local.yaml")
	logger := log.GetInstance()

	useRedis := os.Getenv("USE_REDIS") == "true"

	var c domain.Cache
	if useRedis {
		ttl, err := time.ParseDuration(os.Getenv("REDIS_TTL"))
		if err != nil {
			logger.Fatal("invalid REDIS_TTL: %v", err)
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

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All components stopped successfully")
	case <-shutdownCtx.Done():
		logger.Warn("Graceful shutdown timed out, forcing exit")
	}

	logger.Info("Application shutdown complete")
}
