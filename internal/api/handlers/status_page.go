package handlers

import (
	"encoding/json"
	"net/http"
	"uptime/internal/auth"
	"uptime/internal/domain"

	"github.com/google/uuid"
)

type StatusPageHandler struct {
	repo domain.StatusPageRepository
}

func NewStatusPageHandler(repo domain.StatusPageRepository) *StatusPageHandler {
	return &StatusPageHandler{repo: repo}
}

func (h *StatusPageHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "missing slug", http.StatusBadRequest)
		return
	}

	page, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "status page not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(page)
}

func (h *StatusPageHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	pages, err := h.repo.ListForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(pages)
}

func (h *StatusPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var p domain.StatusPage
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.UserID = userID
	if err := h.repo.Create(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *StatusPageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/status-pages/"):]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
