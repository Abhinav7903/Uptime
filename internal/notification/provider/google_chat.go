package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"uptime/internal/domain"
)

type GoogleChatProvider struct {
	client *http.Client
}

func NewGoogleChatProvider() *GoogleChatProvider {
	return &GoogleChatProvider{client: &http.Client{}}
}

func (p *GoogleChatProvider) Type() domain.NotificationType {
	return domain.NotificationTypeGoogleChat
}

func (p *GoogleChatProvider) Send(ctx context.Context, config map[string]any, payload domain.NotificationPayload) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("missing webhook_url in config")
	}

	text := fmt.Sprintf("*Monitor %s is %s*\nTarget: %s\nTime: %s",
		payload.MonitorName, payload.Status, payload.MonitorTarget, payload.Timestamp)
	if payload.ErrorMessage != "" {
		text += fmt.Sprintf("\nError: `%s`", payload.ErrorMessage)
	}

	body, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google chat responded with status: %d", resp.StatusCode)
	}
	return nil
}
