package httpapi

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/FrancoRivero2025/go-exercise/internal/application"
    "github.com/FrancoRivero2025/go-exercise/internal/domain"
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
    r.Get("/health", h.health)
    r.Get("/ready", h.ready)
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

    if ltps == nil {
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
