package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type MovesHandler struct {
	service *service.MovesService
}

func NewMovesHandler(s *service.MovesService) *MovesHandler {
	return &MovesHandler{service: s}
}

func (h *MovesHandler) Get(w http.ResponseWriter, r *http.Request) {
	moves, err := h.service.GetMoveList(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, moves)
}

func (h *MovesHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	move, err := h.service.GetMoveDetail(r.Context(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no move exists of id "+idParam, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get move", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, move)
}

func (h *MovesHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	move, err := h.service.GetMoveByName(r.Context(), name)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no move exists of name "+name, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get move", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, move)
}
