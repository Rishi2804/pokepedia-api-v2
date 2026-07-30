package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
)

type SpeciesHandler struct {
	service *service.SpeciesService
}

func NewSpeciesHandler(s *service.SpeciesService) *SpeciesHandler {
	return &SpeciesHandler{service: s}
}

func (h *SpeciesHandler) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	detail, err := h.service.GetSpeciesDetail(r.Context(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no species exists of id "+idParam, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get species", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *SpeciesHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	detail, err := h.service.GetSpeciesDetailByName(r.Context(), name)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no species exists of name "+name, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get species", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
