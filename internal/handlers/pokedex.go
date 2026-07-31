package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
)

type PokedexHandler struct {
	service *service.PokedexService
}

func NewPokedexHandler(s *service.PokedexService) *PokedexHandler {
	return &PokedexHandler{service: s}
}

// GetDex mirrors PokemonController.getDex — /pokedex/{name}.
func (h *PokedexHandler) GetDex(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	entries, err := h.service.GetDexByVersion(r.Context(), name)
	if err != nil {
		http.Error(w, "No pokedex exists of name "+name, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// GetTeamCandidates mirrors PokemonController.getCandidates — /team-building/{version}.
func (h *PokedexHandler) GetTeamCandidates(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")

	candidates, err := h.service.GetTeamCandidates(r.Context(), version)
	if err != nil {
		http.Error(w, "No Version Group exists of name "+version, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, candidates)
}
