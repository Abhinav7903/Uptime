package postgres

import (
	"context"
	"encoding/json"
	"time"

	"uptime/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MonitorRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorRepository(pool *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

func (r *MonitorRepository) Create(ctx context.Context, m *domain.Monitor) error {
	query := `
		INSERT INTO monitors (id, user_id, name, type, target, interval_seconds, timeout_seconds, retry_count, status, is_active, metadata)
		VALUES ($1, $2, $3, $4::monitor_type, $5, $6, $7, $8, $9::monitor_status, $10, $11)`
	
	metadata, _ := json.Marshal(m.Metadata)
	_, err := r.pool.Exec(ctx, query, m.ID, m.UserID, m.Name, string(m.Type), m.Target, m.IntervalSeconds, m.TimeoutSeconds, m.RetryCount, string(m.Status), m.IsActive, metadata)
	return err
}

func (r *MonitorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	query := `SELECT id, user_id, name, type, target, interval_seconds, timeout_seconds, retry_count, status, is_active, metadata, created_at, updated_at FROM monitors WHERE id = $1`
	
	var m domain.Monitor
	var metadata []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.UserID, &m.Name, &m.Type, &m.Target, &m.IntervalSeconds, &m.TimeoutSeconds, &m.RetryCount, &m.Status, &m.IsActive, &metadata, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metadata, &m.Metadata)
	return &m, nil
}

func (r *MonitorRepository) ListActive(ctx context.Context) ([]*domain.Monitor, error) {
	query := `SELECT id, user_id, name, type, target, interval_seconds, timeout_seconds, retry_count, status, is_active, metadata, created_at, updated_at FROM monitors WHERE is_active = TRUE`
	
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monitors := []*domain.Monitor{}
	for rows.Next() {
		var m domain.Monitor
		var metadata []byte
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.Type, &m.Target, &m.IntervalSeconds, &m.TimeoutSeconds, &m.RetryCount, &m.Status, &m.IsActive, &metadata, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(metadata, &m.Metadata)
		monitors = append(monitors, &m)
	}
	return monitors, nil
}

func (r *MonitorRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) (domain.MonitorStatus, error) {
	// We use a transaction to get the old status and update to the new one
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var oldStatus domain.MonitorStatus
	err = tx.QueryRow(ctx, "SELECT status FROM monitors WHERE id = $1 FOR UPDATE", id).Scan(&oldStatus)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, "UPDATE monitors SET status = $1::monitor_status, updated_at = NOW() WHERE id = $2", string(status), id)
	if err != nil {
		return "", err
	}

	return oldStatus, tx.Commit(ctx)
}

func (r *MonitorRepository) SaveResult(ctx context.Context, res *domain.MonitorResult) error {
	query := `INSERT INTO monitor_results (time, monitor_id, status, latency_ms, error_message, metadata) VALUES ($1, $2, $3::monitor_status, $4, $5, $6)`
	
	metadata, _ := json.Marshal(res.Metadata)
	_, err := r.pool.Exec(ctx, query, res.Time, res.MonitorID, string(res.Status), res.LatencyMS, res.ErrorMessage, metadata)
	return err
}

func (r *MonitorRepository) GetStats(ctx context.Context, id uuid.UUID, duration time.Duration) (map[string]any, error) {
	query := `
		SELECT 
			avg(latency_ms) as avg_latency,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_latency,
			count(*) FILTER (WHERE status = 'up')::float / count(*)::float * 100 as uptime_pct
		FROM monitor_results 
		WHERE monitor_id = $1 AND time > NOW() - $2::interval`

	var avg, p95, uptime *float64
	err := r.pool.QueryRow(ctx, query, id, duration.String()).Scan(&avg, &p95, &uptime)
	if err != nil {
		return nil, err
	}

	res := map[string]any{
		"avg_latency": 0.0,
		"p95_latency": 0.0,
		"uptime_pct":  100.0,
	}
	if avg != nil {
		res["avg_latency"] = *avg
	}
	if p95 != nil {
		res["p95_latency"] = *p95
	}
	if uptime != nil {
		res["uptime_pct"] = *uptime
	}
	return res, nil
}

func (r *MonitorRepository) GetStatusPage(ctx context.Context, slug string) (map[string]any, error) {
	query := `SELECT id, title, description, logo_url, theme_config FROM status_pages WHERE slug = $1 AND is_public = TRUE`
	
	var page struct {
		ID          uuid.UUID
		Title       string
		Description string
		LogoURL     *string
		ThemeConfig []byte
	}
	err := r.pool.QueryRow(ctx, query, slug).Scan(&page.ID, &page.Title, &page.Description, &page.LogoURL, &page.ThemeConfig)
	if err != nil {
		return nil, err
	}

	// Fetch monitors for this status page
	monitorQuery := `
		SELECT m.name, m.status, 
			(SELECT avg(latency_ms) FROM monitor_results WHERE monitor_id = m.id AND time > NOW() - INTERVAL '24 hours') as avg_latency
		FROM monitors m
		JOIN status_page_monitors spm ON m.id = spm.monitor_id
		WHERE spm.status_page_id = $1`
	
	rows, err := r.pool.Query(ctx, monitorQuery, page.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []map[string]any
	for rows.Next() {
		var name, status string
		var avgLatency *float64
		if err := rows.Scan(&name, &status, &avgLatency); err != nil {
			return nil, err
		}
		monitors = append(monitors, map[string]any{
			"name":        name,
			"status":      status,
			"avg_latency": avgLatency,
		})
	}

	return map[string]any{
		"title":       page.Title,
		"description": page.Description,
		"monitors":    monitors,
	}, nil
}
