package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uptime/internal/alert"
	"uptime/internal/domain"
	"uptime/internal/monitor/executor"
	"uptime/internal/monitor/scheduler"
	"uptime/internal/notification/provider"
	"uptime/internal/realtime"
	"uptime/internal/storage/postgres"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
		},
	)

	monitorRepo := postgres.NewMonitorRepository(pool)
	notifRepo := postgres.NewNotificationRepository(pool)
	execFactory := executor.NewFactory()

	emailProv := provider.NewEmailProvider()
	gchatProv := provider.NewGoogleChatProvider()
	alertSvc := alert.NewAlertService(logger, notifRepo, monitorRepo, []domain.NotificationProvider{emailProv, gchatProv})
	
	sched := scheduler.NewScheduler(asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}), monitorRepo, logger)
	go func() {
		if err := sched.Start(); err != nil {
			logger.Fatal("scheduler failed", zap.Error(err))
		}
	}()

	mux := asynq.NewServeMux()
	mux.HandleFunc("monitor:check", func(ctx context.Context, t *asynq.Task) error {
		var m domain.Monitor
		if err := json.Unmarshal(t.Payload(), &m); err != nil {
			logger.Error("failed to unmarshal task payload", zap.Error(err))
			return err
		}

		logger.Info("processing monitor check", zap.String("monitor_id", m.ID.String()), zap.String("name", m.Name))

		ex, err := execFactory.Get(m.Type)
		if err != nil {
			logger.Error("failed to get executor", zap.Error(err))
			return err
		}

		result, err := ex.Execute(ctx, &m)
		if err != nil {
			logger.Error("check execution failed", zap.Error(err))
			if result == nil {
				result = &domain.MonitorResult{
					Time:         time.Now(),
					MonitorID:    m.ID,
					Status:       domain.StatusDown,
					ErrorMessage: err.Error(),
				}
			}
		}

		if err := monitorRepo.SaveResult(ctx, result); err != nil {
			logger.Error("failed to save monitor result", zap.Error(err))
			return err
		}

		// Update status and trigger alerts via AlertService
		if err := alertSvc.ProcessResult(ctx, &m, result); err != nil {
			logger.Error("failed to process result for alerting", zap.Error(err))
		}

		// Publish to Redis for real-time updates
		updateMsg := realtime.Message{
			Type:    "monitor_update",
			Payload: result,
		}
		payload, _ := json.Marshal(updateMsg)
		rdb.Publish(ctx, "updates", payload)

		logger.Info("monitor check completed", zap.String("monitor_id", m.ID.String()), zap.String("status", string(result.Status)))
		return nil
	})

	if err := srv.Run(mux); err != nil {
		logger.Fatal("worker failed", zap.Error(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
