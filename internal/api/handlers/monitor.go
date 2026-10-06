package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/google/uuid"
	"uptime/internal/auth"
	"uptime/internal/domain"
)

func (h *MonitorHandler) Stats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.GetStats(r.Context(), id, 24*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

type MonitorHandler struct {
	repo domain.MonitorRepository
}

func NewMonitorHandler(repo domain.MonitorRepository) *MonitorHandler {
	return &MonitorHandler{repo: repo}
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.repo.ListActive(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(monitors)
}

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m domain.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Set initial status and defaults
	m.ID = uuid.New()
	m.Status = domain.StatusPending
	m.IsActive = true
	
	if m.IntervalSeconds == 0 {
		m.IntervalSeconds = 60
	}
	if m.TimeoutSeconds == 0 {
		m.TimeoutSeconds = 10
	}
	if m.RetryCount == 0 {
		m.RetryCount = 3
	}
	
	// Set user ID from context (added by JWT middleware)
	if userID, ok := auth.GetUserID(r.Context()); ok {
		m.UserID = userID
	} else {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.repo.Create(r.Context(), &m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}
