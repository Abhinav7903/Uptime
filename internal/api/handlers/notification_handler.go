package handlers

import (
	"encoding/json"
	"net/http"
	"uptime/internal/auth"
	"uptime/internal/domain"

	"github.com/google/uuid"
)

type NotificationHandler struct {
	repo domain.NotificationRepository
}

func NewNotificationHandler(repo domain.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	channels, err := h.repo.ListForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(channels)
}

func (h *NotificationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var ch domain.NotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ch.UserID = userID
	if err := h.repo.Create(r.Context(), &ch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ch)
}

func (h *NotificationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/notifications/"):]
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
