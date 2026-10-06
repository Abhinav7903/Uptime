package executor

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"uptime/internal/domain"
)

type HTTPExecutor struct {
	client *http.Client
}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, m *domain.Monitor) (*domain.MonitorResult, error) {
	start := time.Now()
	
	req, err := http.NewRequestWithContext(ctx, "GET", m.Target, nil)
	if err != nil {
		return &domain.MonitorResult{
			Time:         time.Now(),
			MonitorID:    m.ID,
			Status:       domain.StatusDown,
			ErrorMessage: err.Error(),
		}, err
	}

	resp, err := e.client.Do(req)
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
	defer resp.Body.Close()

	status := domain.StatusUp
	if resp.StatusCode >= 400 {
		status = domain.StatusDown
	}

	return &domain.MonitorResult{
		Time:      time.Now(),
		MonitorID: m.ID,
		Status:    status,
		LatencyMS: int(latency),
		Metadata: map[string]any{
			"status_code": resp.StatusCode,
		},
	}, nil
}
