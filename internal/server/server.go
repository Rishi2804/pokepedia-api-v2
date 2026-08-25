package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rishi2804/pokepedia-api-v2/internal/cache"
	appmiddleware "github.com/Rishi2804/pokepedia-api-v2/internal/middleware"
	"github.com/Rishi2804/pokepedia-api-v2/internal/search"
)

type Server struct {
	pool  *pgxpool.Pool
	cache *cache.Cache
	es    *search.Client
}

func New(pool *pgxpool.Pool, c *cache.Cache, es *search.Client) *Server {
	return &Server{pool: pool, cache: c, es: es}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(appmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(appmiddleware.CORS())

	s.registerRoutes(r)

	return r
}
