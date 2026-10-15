package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type ItemsHandler struct {
	service *service.ItemsService
}

func NewItemsHandler(s *service.ItemsService) *ItemsHandler {
	return &ItemsHandler{service: s}
}

func (h *ItemsHandler) Get(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.GetItemList(r.Context())
	if err != nil {
		http.Error(w, "failed to get items", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *ItemsHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	item, err := h.service.GetItemDetail(r.Context(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no item exists of id "+idParam, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get item", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *ItemsHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	item, err := h.service.GetItemByName(r.Context(), name)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "no item exists of name "+name, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get item", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// GetHoldable serves the team builder's item picker for one battle format.
func (h *ItemsHandler) GetHoldable(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	items, err := h.service.GetHoldableItems(r.Context(), format)
	if errors.Is(err, service.ErrInvalidItemFormat) {
		http.Error(w, "invalid format "+format, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "failed to get items", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, items)
}
