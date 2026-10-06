package postgres

import (
	"context"
	"encoding/json"
	"uptime/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.NotificationChannel, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, user_id, name, type, config FROM notification_channels WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []*domain.NotificationChannel{}
	for rows.Next() {
		var c domain.NotificationChannel
		var configJSON []byte
		err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &configJSON)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(configJSON, &c.Config); err != nil {
			return nil, err
		}
		channels = append(channels, &c)
	}
	return channels, nil
}

func (r *NotificationRepository) Create(ctx context.Context, channel *domain.NotificationChannel) error {
	configJSON, err := json.Marshal(channel.Config)
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx,
		"INSERT INTO notification_channels (user_id, name, type, config) VALUES ($1, $2, $3, $4) RETURNING id",
		channel.UserID, channel.Name, channel.Type, configJSON,
	).Scan(&channel.ID)
}

func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM notification_channels WHERE id = $1", id)
	return err
}
