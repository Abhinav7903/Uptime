package alert

import (
	"context"
	"fmt"
	"uptime/internal/domain"

	"go.uber.org/zap"
)

type AlertService struct {
	logger        *zap.Logger
	notifRepo     domain.NotificationRepository
	monitorRepo   domain.MonitorRepository
	providers     map[domain.NotificationType]domain.NotificationProvider
}

func NewAlertService(
	logger *zap.Logger,
	notifRepo domain.NotificationRepository,
	monitorRepo domain.MonitorRepository,
	providers []domain.NotificationProvider,
) *AlertService {
	providerMap := make(map[domain.NotificationType]domain.NotificationProvider)
	for _, p := range providers {
		providerMap[p.Type()] = p
	}

	return &AlertService{
		logger:      logger,
		notifRepo:   notifRepo,
		monitorRepo: monitorRepo,
		providers:   providerMap,
	}
}

func (s *AlertService) ProcessResult(ctx context.Context, monitor *domain.Monitor, result *domain.MonitorResult) error {
	oldStatus, err := s.monitorRepo.UpdateStatus(ctx, monitor.ID, result.Status)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Trigger alert if status changed
	if oldStatus != result.Status && oldStatus != domain.StatusPending {
		s.logger.Info("status changed, triggering alerts",
			zap.String("monitor", monitor.Name),
			zap.String("old_status", string(oldStatus)),
			zap.String("new_status", string(result.Status)),
		)

		return s.Notify(ctx, monitor, result)
	}

	return nil
}

func (s *AlertService) Notify(ctx context.Context, monitor *domain.Monitor, result *domain.MonitorResult) error {
	channels, err := s.notifRepo.ListForUser(ctx, monitor.UserID)
	if err != nil {
		return fmt.Errorf("failed to list notification channels: %w", err)
	}

	payload := domain.NotificationPayload{
		MonitorName:   monitor.Name,
		MonitorTarget: monitor.Target,
		Status:        result.Status,
		Timestamp:     result.Time,
		ErrorMessage:  result.ErrorMessage,
	}

	for _, ch := range channels {
		provider, ok := s.providers[ch.Type]
		if !ok {
			s.logger.Warn("no provider found for channel type", zap.String("type", string(ch.Type)))
			continue
		}

		go func(ch *domain.NotificationChannel) {
			err := provider.Send(context.Background(), ch.Config, payload)
			if err != nil {
				s.logger.Error("failed to send notification",
					zap.String("channel", ch.Name),
					zap.Error(err),
				)
			}
		}(ch)
	}

	return nil
}
