package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MonitorType string

const (
	MonitorTypeHTTP MonitorType = "http"
	MonitorTypeTCP  MonitorType = "tcp"
	MonitorTypeICMP MonitorType = "icmp"
	MonitorTypeDNS  MonitorType = "dns"
	MonitorTypeSSL  MonitorType = "ssl"
)

type MonitorStatus string

const (
	StatusUp          MonitorStatus = "up"
	StatusDown        MonitorStatus = "down"
	StatusPending     MonitorStatus = "pending"
	StatusMaintenance MonitorStatus = "maintenance"
)

type NotificationType string

const (
	NotificationTypeEmail      NotificationType = "email"
	NotificationTypeGoogleChat NotificationType = "google_chat"
)

type NotificationPayload struct {
	MonitorName   string
	MonitorTarget string
	Status        MonitorStatus
	Timestamp     time.Time
	ErrorMessage  string
}

type NotificationChannel struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	Name      string         `json:"name"`
	Type      NotificationType `json:"type"`
	Config    map[string]any `json:"config"`
	IsDefault bool           `json:"is_default"`
}

type Monitor struct {
	ID              uuid.UUID      `json:"id"`
	UserID          uuid.UUID      `json:"user_id"`
	Name            string         `json:"name"`
	Type            MonitorType    `json:"type"`
	Target          string         `json:"target"`
	IntervalSeconds int            `json:"interval_seconds"`
	TimeoutSeconds  int            `json:"timeout_seconds"`
	RetryCount      int            `json:"retry_count"`
	Status          MonitorStatus  `json:"status"`
	IsActive        bool           `json:"is_active"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type MonitorResult struct {
	Time         time.Time      `json:"time"`
	MonitorID    uuid.UUID      `json:"monitor_id"`
	Status       MonitorStatus  `json:"status"`
	LatencyMS    int            `json:"latency_ms"`
	ErrorMessage string         `json:"error_message"`
	Metadata     map[string]any `json:"metadata"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type MonitorRepository interface {
	Create(ctx context.Context, monitor *Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (*Monitor, error)
	ListActive(ctx context.Context) ([]*Monitor, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status MonitorStatus) (MonitorStatus, error)
	SaveResult(ctx context.Context, result *MonitorResult) error
	GetStats(ctx context.Context, id uuid.UUID, duration time.Duration) (map[string]any, error)
}

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type NotificationRepository interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*NotificationChannel, error)
	Create(ctx context.Context, channel *NotificationChannel) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Incident struct {
	ID        uuid.UUID  `json:"id"`
	MonitorID uuid.UUID  `json:"monitor_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	Status    string     `json:"status"`
	Summary   string     `json:"summary"`
}

type IncidentRepository interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*Incident, error)
	Create(ctx context.Context, incident *Incident) error
	Update(ctx context.Context, incident *Incident) error
}

type StatusPage struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	LogoURL     string         `json:"logo_url"`
	ThemeConfig map[string]any `json:"theme_config"`
	IsPublic    bool           `json:"is_public"`
	Monitors    []*Monitor     `json:"monitors,omitempty"`
}

type StatusPageRepository interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*StatusPage, error)
	GetBySlug(ctx context.Context, slug string) (*StatusPage, error)
	Create(ctx context.Context, page *StatusPage) error
	Update(ctx context.Context, page *StatusPage) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type NotificationProvider interface {
	Type() NotificationType
	Send(ctx context.Context, config map[string]any, payload NotificationPayload) error
}

type CheckExecutor interface {
	Execute(ctx context.Context, monitor *Monitor) (*MonitorResult, error)
}
