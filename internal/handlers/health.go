package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rishi2804/pokepedia-api-v2/internal/cache"
	"github.com/Rishi2804/pokepedia-api-v2/internal/search"
)

type HealthHandler struct {
	pool  *pgxpool.Pool
	cache *cache.Cache
	es    *search.Client
}

func NewHealthHandler(pool *pgxpool.Pool, c *cache.Cache, es *search.Client) *HealthHandler {
	return &HealthHandler{pool: pool, cache: c, es: es}
}

// Check reports DB, cache, and search status. The response code reflects the
// DB only — cache/search health must never turn a restart-triggering health
// check into an outage caused by an optional dependency alone.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK

	if err := h.pool.Ping(r.Context()); err != nil {
		status = "db unreachable"
		code = http.StatusServiceUnavailable
	}

	cacheStatus := "disabled"
	if h.cache.Enabled() {
		if h.cache.Stats().Healthy {
			cacheStatus = "ok"
		} else {
			cacheStatus = "degraded"
		}
	}

	searchStatus := "disabled"
	if h.es.Enabled() {
		if h.es.Stats().Healthy {
			searchStatus = "ok"
		} else {
			searchStatus = "degraded"
		}
	}

	writeJSON(w, code, map[string]string{"status": status, "cache": cacheStatus, "search": searchStatus})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
