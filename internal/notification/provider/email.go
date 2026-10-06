package provider

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"uptime/internal/domain"
)

type EmailProvider struct{}

func NewEmailProvider() *EmailProvider {
	return &EmailProvider{}
}

func (p *EmailProvider) Type() domain.NotificationType {
	return domain.NotificationTypeEmail
}

func (p *EmailProvider) Send(ctx context.Context, config map[string]any, payload domain.NotificationPayload) error {
	host, _ := config["host"].(string)
	port, _ := config["port"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	from, _ := config["from"].(string)
	to, _ := config["to"].(string)

	subject := fmt.Sprintf("Subject: Uptime Alert: %s is %s\n", payload.MonitorName, payload.Status)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<h2>Monitor Status Changed</h2><p><b>%s</b> is now <b>%s</b></p><p>Target: %s</p>",
		payload.MonitorName, payload.Status, payload.MonitorTarget)

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", username, password, host)

	return smtp.SendMail(host+":"+port, auth, from, strings.Split(to, ","), msg)
}
