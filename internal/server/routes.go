package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/handlers"
	appmiddleware "github.com/Rishi2804/pokepedia-api-v2/internal/middleware"
	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
)

func (s *Server) registerRoutes(r chi.Router) {
	queries := store.New(s.pool)
	pokemonService := service.NewPokemonService(queries)
	speciesService := service.NewSpeciesService(queries, pokemonService)
	pokedexService := service.NewPokedexService(queries)
	movesService := service.NewMovesService(queries)
	abilitiesService := service.NewAbilitiesService(queries)

	healthHandler := handlers.NewHealthHandler(s.pool, s.cache)
	pokemonHandler := handlers.NewPokemonHandler(pokemonService)
	speciesHandler := handlers.NewSpeciesHandler(speciesService)
	pokedexHandler := handlers.NewPokedexHandler(pokedexService)
	movesHandler := handlers.NewMovesHandler(movesService)
	abilitiesHandler := handlers.NewAbilitiesHandler(abilitiesService)

	r.Get("/healthz", healthHandler.Check)

	// Every route below is a path-only GET over static reference data, so a
	// response cache keyed on the URL path is correct for all of them.
	// /healthz is registered above, outside this group, so it always stays live.
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.ResponseCache(s.cache))

		r.Route("/api/v1/pokemon", func(r chi.Router) {
			r.Get("/{id:[0-9]+}", pokemonHandler.Get)
			r.Get("/{name}", pokemonHandler.GetByName)
		})

		r.Route("/api/v1/species", func(r chi.Router) {
			r.Get("/{id:[0-9]+}", speciesHandler.Get)
			r.Get("/{name}", speciesHandler.GetByName)
		})

		r.Get("/api/v1/pokedex/{name}", pokedexHandler.GetDex)
		r.Route("/api/v1/team-building", func(r chi.Router) {
			r.Get("/{version}", pokedexHandler.GetTeamCandidates)
			r.Get("/{version}/{id:[0-9]+}", pokedexHandler.GetTeamCandidate)
		})

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
	})
}
