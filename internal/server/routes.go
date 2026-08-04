package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/handlers"
	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
)

func (s *Server) registerRoutes(r chi.Router) {
	queries := store.New(s.pool)
	pokemonService := service.NewPokemonService(queries)
	speciesService := service.NewSpeciesService(queries)
	pokedexService := service.NewPokedexService(queries)
	movesService := service.NewMovesService(queries)
	abilitiesService := service.NewAbilitiesService(queries)

	healthHandler := handlers.NewHealthHandler(s.pool)
	pokemonHandler := handlers.NewPokemonHandler(pokemonService)
	speciesHandler := handlers.NewSpeciesHandler(speciesService)
	pokedexHandler := handlers.NewPokedexHandler(pokedexService)
	movesHandler := handlers.NewMovesHandler(movesService)
	abilitiesHandler := handlers.NewAbilitiesHandler(abilitiesService)

	r.Get("/healthz", healthHandler.Check)

	r.Route("/api/v1/pokemon", func(r chi.Router) {
		r.Get("/{id:[0-9]+}", pokemonHandler.Get)
		r.Get("/{name}", pokemonHandler.GetByName)
	})

	r.Route("/api/v1/species", func(r chi.Router) {
		r.Get("/{id:[0-9]+}", speciesHandler.Get)
		r.Get("/{name}", speciesHandler.GetByName)
	})

	r.Get("/api/v1/pokedex/{name}", pokedexHandler.GetDex)
	r.Get("/api/v1/team-building/{version}", pokedexHandler.GetTeamCandidates)

	r.Route("/api/v1/move", func(r chi.Router) {
		r.Get("/", movesHandler.Get)
		r.Get("/{id:[0-9]+}", movesHandler.GetDetail)
		r.Get("/{name}", movesHandler.GetByName)
	})

	r.Route("/api/v1/ability", func(r chi.Router) {
		r.Get("/", abilitiesHandler.Get)
		r.Get("/{id:[0-9]+}", abilitiesHandler.GetDetail)
		r.Get("/{name}", abilitiesHandler.GetByName)
	})
}
