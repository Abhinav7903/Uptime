package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"uptime/internal/domain"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type Scheduler struct {
	client *asynq.Client
	repo   domain.MonitorRepository
	logger *zap.Logger
}

func NewScheduler(client *asynq.Client, repo domain.MonitorRepository, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

func (s *Scheduler) Start() error {
	s.logger.Info("starting scheduler")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.dispatch()
		}
	}
}

func (s *Scheduler) dispatch() {
	monitors, err := s.repo.ListActive(context.Background())
	if err != nil {
		s.logger.Error("failed to list active monitors", zap.Error(err))
		return
	}

	s.logger.Info("dispatching monitor checks", zap.Int("count", len(monitors)))

	for _, m := range monitors {
		payload, _ := json.Marshal(m)
		task := asynq.NewTask("monitor:check", payload)
		info, err := s.client.Enqueue(task)
		if err != nil {
			s.logger.Error("failed to enqueue task", zap.Error(err), zap.String("monitor_id", m.ID.String()))
		} else {
			s.logger.Info("enqueued task", zap.String("task_id", info.ID), zap.String("monitor_id", m.ID.String()))
		}
	}
}
