package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type AbilitiesHandler struct {
	service *service.AbilitiesService
}

func NewAbilitiesHandler(s *service.AbilitiesService) *AbilitiesHandler {
	return &AbilitiesHandler{service: s}
}

func (h *AbilitiesHandler) Get(w http.ResponseWriter, r *http.Request) {
	abilities, err := h.service.GetAbilityList(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, abilities)
}

func (h *AbilitiesHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ability, err := h.service.GetAbilityDetail(r.Context(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no ability exists of id "+idParam, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get ability", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, ability)
}

func (h *AbilitiesHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	ability, err := h.service.GetAbilityByName(r.Context(), name)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no ability exists of name "+name, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get ability", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, ability)
}
