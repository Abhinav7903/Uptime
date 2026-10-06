package alert

import (
	"context"
	"uptime/internal/domain"
	"go.uber.org/zap"
)

type AlertEngine struct {
	logger *zap.Logger
}

func NewAlertEngine(logger *zap.Logger) *AlertEngine {
	return &AlertEngine{logger: logger}
}

func (e *AlertEngine) ProcessResult(ctx context.Context, prevStatus domain.MonitorStatus, result *domain.MonitorResult) error {
	if prevStatus == result.Status {
		return nil
	}

	e.logger.Info("monitor state transition",
		zap.String("monitor_id", result.MonitorID.String()),
		zap.String("from", string(prevStatus)),
		zap.String("to", string(result.Status)),
	)

	return nil
}
