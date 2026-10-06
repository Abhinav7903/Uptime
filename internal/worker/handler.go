package worker

import (
	"context"
	"encoding/json"
	"uptime/internal/domain"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type MonitorWorker struct {
	ExecutorFactory interface {
		Get(t domain.MonitorType) (domain.CheckExecutor, error)
	}
	Repo   domain.MonitorRepository
	Alerts interface {
		ProcessResult(ctx context.Context, prevStatus domain.MonitorStatus, result *domain.MonitorResult) error
	}
	Logger *zap.Logger
}

func (w *MonitorWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var m domain.Monitor
	if err := json.Unmarshal(t.Payload(), &m); err != nil {
		return err
	}

	current, err := w.Repo.GetByID(ctx, m.ID)
	if err != nil {
		return err
	}

	executor, err := w.ExecutorFactory.Get(current.Type)
	if err != nil {
		return err
	}

	result, err := executor.Execute(ctx, current)
	if err != nil {
		w.Logger.Error("execution failed", zap.Error(err))
	}

	if err := w.Repo.SaveResult(ctx, result); err != nil {
		return err
	}

	if err := w.Repo.UpdateStatus(ctx, m.ID, result.Status); err != nil {
		return err
	}

	return w.Alerts.ProcessResult(ctx, current.Status, result)
}
