package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/handlers"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
)

func (s *Server) registerRoutes(r chi.Router) {
	healthHandler := handlers.NewHealthHandler(s.pool)
	pokemonQueries := store.New(s.pool)
	pokemonHandler := handlers.NewPokemonHandler(pokemonQueries)

	r.Get("/healthz", healthHandler.Check)

	r.Route("/api/v1/pokemon", func(r chi.Router) {
		r.Get("/", pokemonHandler.List)
		r.Get("/{id}", pokemonHandler.Get)
	})
}
