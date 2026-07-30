package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
)

type PokemonHandler struct {
	service *service.PokemonService
}

func NewPokemonHandler(s *service.PokemonService) *PokemonHandler {
	return &PokemonHandler{service: s}
}

// Get mirrors PokemonController.getPokemon(Integer id) — the {id:\d+} route.
func (h *PokemonHandler) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	detail, err := h.service.GetPokemonDetail(r.Context(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no pokemon exists of id "+idParam, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get pokemon", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// GetByName mirrors PokemonController.getPokemon(String name) — the {name} route.
func (h *PokemonHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	detail, err := h.service.GetPokemonDetailByName(r.Context(), name)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no pokemon exists of name "+name, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get pokemon", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
