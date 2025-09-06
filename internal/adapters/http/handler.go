package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/application"
	"github.com/FrancoRivero2025/go-exercise/ltp-service/internal/domain"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *application.LTPService
}

func NewHandler(s *application.LTPService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/api/v1/ltp", h.getLTP)
	r.Get("/healthz", h.health)
	r.Get("/readyz", h.ready)
	return r
}

func parsePairsParam(q string) []domain.Pair {
	q = strings.TrimSpace(q)
	if q == "" {
		return []domain.Pair{}
	}
	parts := strings.Split(q, ",")
	res := make([]domain.Pair, 0, len(parts))
	for _, p := range parts {
		res = append(res, domain.Pair(strings.TrimSpace(p)))
	}
	return res
}

func (h *Handler) getLTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("pairs")
	pairs := parsePairsParam(q)
	// if no pairs provided, default to configured ones is option,
	// for this skeleton we return bad request
	if len(pairs) == 0 {
		http.Error(w, "missing 'pairs' query param", http.StatusBadRequest)
		return
	}

	ltps, err := h.service.GetLTPs(pairs)
	if err != nil {
		// if ErrNotFound, return 204 or empty list
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ltp": []interface{}{}})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ltp": ltps})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}
