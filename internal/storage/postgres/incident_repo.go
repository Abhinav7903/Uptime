package postgres

import (
	"context"
	"uptime/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{pool: pool}
}

func (r *IncidentRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Incident, error) {
	query := `
		SELECT i.id, i.monitor_id, i.start_time, i.end_time, i.status, i.summary 
		FROM incidents i
		JOIN monitors m ON i.monitor_id = m.id
		WHERE m.user_id = $1
		ORDER BY i.start_time DESC`
	
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := []*domain.Incident{}
	for rows.Next() {
		var i domain.Incident
		if err := rows.Scan(&i.ID, &i.MonitorID, &i.StartTime, &i.EndTime, &i.Status, &i.Summary); err != nil {
			return nil, err
		}
		incidents = append(incidents, &i)
	}
	return incidents, nil
}

func (r *IncidentRepository) Create(ctx context.Context, i *domain.Incident) error {
	query := `INSERT INTO incidents (monitor_id, status, summary) VALUES ($1, $2, $3) RETURNING id, start_time`
	return r.pool.QueryRow(ctx, query, i.MonitorID, i.Status, i.Summary).Scan(&i.ID, &i.StartTime)
}

func (r *IncidentRepository) Update(ctx context.Context, i *domain.Incident) error {
	query := `UPDATE incidents SET end_time = $1, status = $2, summary = $3 WHERE id = $4`
	_, err := r.pool.Exec(ctx, query, i.EndTime, i.Status, i.Summary, i.ID)
	return err
}
