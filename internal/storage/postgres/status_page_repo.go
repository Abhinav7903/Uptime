package postgres

import (
	"context"
	"encoding/json"
	"uptime/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatusPageRepository struct {
	pool *pgxpool.Pool
}

func NewStatusPageRepository(pool *pgxpool.Pool) *StatusPageRepository {
	return &StatusPageRepository{pool: pool}
}

func (r *StatusPageRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.StatusPage, error) {
	query := `SELECT id, user_id, slug, title, description, logo_url, theme_config, is_public FROM status_pages WHERE user_id = $1`
	
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pages := []*domain.StatusPage{}
	for rows.Next() {
		var p domain.StatusPage
		var themeJSON []byte
		if err := rows.Scan(&p.ID, &p.UserID, &p.Slug, &p.Title, &p.Description, &p.LogoURL, &themeJSON, &p.IsPublic); err != nil {
			return nil, err
		}
		json.Unmarshal(themeJSON, &p.ThemeConfig)
		pages = append(pages, &p)
	}
	return pages, nil
}

func (r *StatusPageRepository) GetBySlug(ctx context.Context, slug string) (*domain.StatusPage, error) {
	query := `SELECT id, user_id, slug, title, description, logo_url, theme_config, is_public FROM status_pages WHERE slug = $1`
	
	var p domain.StatusPage
	var themeJSON []byte
	err := r.pool.QueryRow(ctx, query, slug).Scan(&p.ID, &p.UserID, &p.Slug, &p.Title, &p.Description, &p.LogoURL, &themeJSON, &p.IsPublic)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(themeJSON, &p.ThemeConfig)

	// Fetch monitors
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.name, m.status, m.target, m.type
		FROM monitors m
		JOIN status_page_monitors spm ON m.id = spm.monitor_id
		WHERE spm.status_page_id = $1`, p.ID)
	if err != nil {
		return &p, nil // Return page without monitors if query fails
	}
	defer rows.Close()

	p.Monitors = []*domain.Monitor{}
	for rows.Next() {
		var m domain.Monitor
		if err := rows.Scan(&m.ID, &m.Name, &m.Status, &m.Target, &m.Type); err != nil {
			continue
		}
		p.Monitors = append(p.Monitors, &m)
	}

	return &p, nil
}

func (r *StatusPageRepository) Create(ctx context.Context, p *domain.StatusPage) error {
	themeJSON, _ := json.Marshal(p.ThemeConfig)
	query := `INSERT INTO status_pages (user_id, slug, title, description, logo_url, theme_config, is_public) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	return r.pool.QueryRow(ctx, query, p.UserID, p.Slug, p.Title, p.Description, p.LogoURL, themeJSON, p.IsPublic).Scan(&p.ID)
}

func (r *StatusPageRepository) Update(ctx context.Context, p *domain.StatusPage) error {
	themeJSON, _ := json.Marshal(p.ThemeConfig)
	query := `UPDATE status_pages SET slug = $1, title = $2, description = $3, logo_url = $4, theme_config = $5, is_public = $6 WHERE id = $7`
	_, err := r.pool.Exec(ctx, query, p.Slug, p.Title, p.Description, p.LogoURL, themeJSON, p.IsPublic, p.ID)
	return err
}

func (r *StatusPageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM status_pages WHERE id = $1", id)
	return err
}
