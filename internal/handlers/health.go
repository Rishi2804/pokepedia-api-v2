package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rishi2804/pokepedia-api-v2/internal/cache"
)

type HealthHandler struct {
	pool  *pgxpool.Pool
	cache *cache.Cache
}

func NewHealthHandler(pool *pgxpool.Pool, c *cache.Cache) *HealthHandler {
	return &HealthHandler{pool: pool, cache: c}
}

// Check reports DB and cache status. The response code reflects the DB only —
// cache health must never turn a restart-triggering health check into an
// outage caused by Redis alone.
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

	writeJSON(w, code, map[string]string{"status": status, "cache": cacheStatus})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
