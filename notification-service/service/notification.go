package service

import (
	"context"
	"log/slog"
	"time"
)

type NotificationService interface {
	SendEmail(ctx context.Context, to string, subject string, body string) error
	SendSMS(ctx context.Context, phone string, message string) error
}

type notificationService struct{}

func NewNotificationService() NotificationService {
	return &notificationService{}
}

func (s *notificationService) SendEmail(ctx context.Context, to string, subject string, body string) error {
	// Simulate sending email
	slog.Info("Simulating sending email", "to", to, "subject", subject)
	time.Sleep(100 * time.Millisecond) // Simulate network delay

	// Example specific simulation: could fail randomly to trigger retries, but we keep it deterministic for now
	slog.Info("Email sent successfully", "to", to)
	return nil
}

func (s *notificationService) SendSMS(ctx context.Context, phone string, message string) error {
	slog.Info("Simulating sending SMS", "phone", phone)
	time.Sleep(50 * time.Millisecond)
	slog.Info("SMS sent successfully", "phone", phone)
	return nil
}
