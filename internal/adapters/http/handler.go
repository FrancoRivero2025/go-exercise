package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/application"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	service *application.LTPService
}

func NewHandler(s *application.LTPService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(metricsMiddleware)

	r.Get("/api/v1/ltp", h.getLTP)
	r.Get("/health", h.health)
	r.Get("/ready", h.ready)
	r.Get("/healthz", h.healthz)
	r.Get("/metrics", h.metrics) 

	return r
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type successResponse struct {
	Data interface{}            `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

type statusResponse struct {
	Status string `json:"status"`
}

func parsePairsParam(q string) []domain.Pair {
	q = strings.TrimSpace(q)
	if q == "" {
		return []domain.Pair{}
	}

	parts := strings.Split(q, ",")
	res := make([]domain.Pair, 0, len(parts))
	seen := make(map[string]bool)

	for _, p := range parts {
		pair := strings.TrimSpace(p)
		if pair != "" && !seen[pair] {
			res = append(res, domain.Pair(pair))
			seen[pair] = true
		}
	}
	return res
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (h *Handler) getLTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("pairs")
	pairs := parsePairsParam(q)

	var ltps []domain.LTP
	if len(pairs) == 0 {
		ltps = h.service.GetAllLTPs()
	} else {
		ltps = h.service.GetLTPs(pairs)
	}

	if len(ltps) == 0 {
		respondJSON(w, http.StatusNotFound, errorResponse{
			Error: "Requested pairs not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	respondJSON(w, http.StatusOK, successResponse{
		Data: ltps,
		Meta: map[string]interface{}{
			"count": len(ltps),
		},
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		respondJSON(w, http.StatusServiceUnavailable,
			statusResponse{Status: "not ready"})
		return
	}
	respondJSON(w, http.StatusOK, statusResponse{Status: "ready"})
}

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: []float64{0.1, 0.5, 1, 2, 5},
	}, []string{"method", "path"})

	cacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of cache hits",
	})

	cacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of cache misses",
	})
)

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			http.StatusText(rw.statusCode),
		).Inc()

		httpRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
		).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type healthzResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services,omitempty"`
}

func IncrementCacheHit() {
	cacheHitsTotal.Inc()
}

func IncrementCacheMiss() {
	cacheMissesTotal.Inc()
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	servicesStatus := make(map[string]string)

	redisReachable := h.service.CheckRedisConnectivity()
	if redisReachable {
		servicesStatus["redis"] = "reachable"
	} else {
		servicesStatus["redis"] = "unreachable"
	}

	krakenReachable := h.service.CheckKrakenConnectivity()
	if krakenReachable {
		servicesStatus["kraken"] = "reachable"
	} else {
		servicesStatus["kraken"] = "unreachable"
	}

	status := "ok"
	if !redisReachable || !krakenReachable {
		status = "degraded"
		respondJSON(w, http.StatusServiceUnavailable, healthzResponse{
			Status:   status,
			Services: servicesStatus,
		})
		return
	}

	respondJSON(w, http.StatusOK, healthzResponse{
		Status:   status,
		Services: servicesStatus,
	})
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
