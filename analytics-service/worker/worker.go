package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"analytics-service/repository"

	"github.com/segmentio/kafka-go"
)

type AnalyticsWorker struct {
	brokers []string
	groupID string
	repo    repository.AnalyticsRepository
	readers []*kafka.Reader
}

func NewAnalyticsWorker(brokers []string, groupID string, repo repository.AnalyticsRepository) *AnalyticsWorker {
	return &AnalyticsWorker{
		brokers: brokers,
		groupID: groupID,
		repo:    repo,
	}
}

func (w *AnalyticsWorker) Start(ctx context.Context, topics []string) {
	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  w.brokers,
			GroupID:  w.groupID,
			Topic:    topic,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		})
		w.readers = append(w.readers, reader)

		go w.consumeWithBuffer(ctx, reader, topic)
	}

	slog.Info("Analytics worker pool started", "topics", strings.Join(topics, ","))
}

func (w *AnalyticsWorker) consumeWithBuffer(ctx context.Context, reader *kafka.Reader, topic string) {
	// A more advanced version would use bounded memory buffering (Flushing every N ms),
	// but for simplicity, we insert straight into repo because our SQL handles UPSERT ON CONFLICT natively
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to fetch message", "topic", topic, "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		w.processMessage(ctx, msg, topic)

		// Commit offset synchronously to ensure at-least-once processing properly recorded
		if err := reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("Failed to commit messages", "topic", topic, "error", err)
		}
	}
}

func (w *AnalyticsWorker) processMessage(ctx context.Context, msg kafka.Message, topic string) {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		slog.Error("Failed to decode message", "topic", topic, "error", err)
		return
	}

	now := time.Now()

	switch topic {
	case "order.created":
		if err := w.repo.RecordOrderVolume(ctx, now); err != nil {
			slog.Error("Failed to record order volume", "error", err)
		}
	case "payment.succeeded":
		amountF, ok := payload["amount"].(float64)
		if ok {
			if err := w.repo.RecordDailyRevenue(ctx, now, int64(amountF)); err != nil {
				slog.Error("Failed to record daily revenue", "error", err)
			}
		} else {
			slog.Warn("Payment succeeded missing amount field")
		}
	case "payment.failed":
		if err := w.repo.RecordFailedPayment(ctx, now); err != nil {
			slog.Error("Failed to record failed payment", "error", err)
		}
	default:
		slog.Debug("Unhandled topic for analytics", "topic", topic)
	}
}

func (w *AnalyticsWorker) Stop() {
	for _, reader := range w.readers {
		reader.Close()
	}
	slog.Info("Analytics worker gracefully stopped")
}
