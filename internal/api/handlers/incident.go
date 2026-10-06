package handlers

import (
	"encoding/json"
	"net/http"
	"uptime/internal/auth"
	"uptime/internal/domain"
)

type IncidentHandler struct {
	repo domain.IncidentRepository
}

func NewIncidentHandler(repo domain.IncidentRepository) *IncidentHandler {
	return &IncidentHandler{repo: repo}
}

func (h *IncidentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	incidents, err := h.repo.ListForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(incidents)
}
