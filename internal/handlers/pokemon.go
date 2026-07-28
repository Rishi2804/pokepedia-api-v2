package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
)

type PokemonHandler struct {
	store *store.PokemonStore
}

func NewPokemonHandler(s *store.PokemonStore) *PokemonHandler {
	return &PokemonHandler{store: s}
}

func (h *PokemonHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list pokemon", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *PokemonHandler) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	p, err := h.store.Get(r.Context(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "pokemon not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get pokemon error: %v", err)
		http.Error(w, "failed to get pokemon", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
