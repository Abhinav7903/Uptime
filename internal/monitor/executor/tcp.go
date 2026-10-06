package executor

import (
	"context"
	"net"
	"time"
	"uptime/internal/domain"
)

type TCPExecutor struct{}

func NewTCPExecutor() *TCPExecutor {
	return &TCPExecutor{}
}

func (e *TCPExecutor) Execute(ctx context.Context, m *domain.Monitor) (*domain.MonitorResult, error) {
	start := time.Now()
	dialer := net.Dialer{Timeout: time.Duration(m.TimeoutSeconds) * time.Second}
	
	conn, err := dialer.DialContext(ctx, "tcp", m.Target)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &domain.MonitorResult{
			Time:         time.Now(),
			MonitorID:    m.ID,
			Status:       domain.StatusDown,
			LatencyMS:    int(latency),
			ErrorMessage: err.Error(),
		}, nil
	}
	defer conn.Close()

	return &domain.MonitorResult{
		Time:      time.Now(),
		MonitorID: m.ID,
		Status:    domain.StatusUp,
		LatencyMS: int(latency),
	}, nil
}
