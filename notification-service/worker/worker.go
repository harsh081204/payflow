package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"notification-service/service"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// EventType defines the topic the event came from
type EventType string

const (
	MaxRetries = 3
)

type WorkerPool struct {
	brokers   []string
	groupID   string
	svc       service.NotificationService
	readers   []*kafka.Reader
	dlqWriter *kafka.Writer
	workers   int
}

func NewWorkerPool(brokers []string, groupID string, workers int, svc service.NotificationService) *WorkerPool {
	return &WorkerPool{
		brokers: brokers,
		groupID: groupID,
		svc:     svc,
		workers: workers,
	}
}

func (wp *WorkerPool) Start(ctx context.Context, topics []string) {
	// Initialize Dead Letter Queue Writer
	wp.dlqWriter = &kafka.Writer{
		Addr:     kafka.TCP(wp.brokers...),
		Topic:    "dead-letter-queue",
		Balancer: &kafka.LeastBytes{},
	}

	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  wp.brokers,
			GroupID:  wp.groupID,
			Topic:    topic,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		})
		wp.readers = append(wp.readers, reader)

		// Start a dispatcher for each topic
		go wp.dispatch(ctx, reader, topic)
	}

	slog.Info("Worker pool started", "topics", strings.Join(topics, ","), "workers", wp.workers)
}

func (wp *WorkerPool) dispatch(ctx context.Context, reader *kafka.Reader, topic string) {
	// Worker channel
	jobs := make(chan kafka.Message, 100)
	var wg sync.WaitGroup

	// Spin up N workers
	for i := 0; i < wp.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for msg := range jobs {
				wp.processWithRetry(ctx, msg, topic)
			}
		}(i)
	}

	// Fetch messages and send to workers
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			slog.Error("Failed to fetch message", "topic", topic, "error", err)
			time.Sleep(1 * time.Second)
			continue
		}
		jobs <- msg
		reader.CommitMessages(ctx, msg)
	}

	close(jobs)
	wg.Wait()
}

func (wp *WorkerPool) processWithRetry(ctx context.Context, msg kafka.Message, topic string) {
	var err error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential Backoff: 1s, 2s, 4s...
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			slog.Info("Retrying message processing", "topic", topic, "attempt", attempt, "backoff", backoff)
			time.Sleep(backoff)
		}

		err = wp.process(ctx, msg, topic)
		if err == nil {
			return // Success
		}

		// If context canceled, stop retrying
		if ctx.Err() != nil {
			return
		}
	}

	slog.Error("Message failed after max retries, sending to DLQ", "topic", topic, "error", err)
	wp.sendToDLQ(ctx, msg, err)
}

func (wp *WorkerPool) process(ctx context.Context, msg kafka.Message, topic string) error {
	// Dummy payload decoding
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		return fmt.Errorf("failed to decode json payload: %w", err)
	}

	switch topic {
	case "order.created":
		// Example: payload{"user_id": "...", "order_id": "...""}
		// Typically, we would fetch User from User-Service to get their email via internal API
		// or User details are included in event payload. Assume payload has "user_email".
		email, _ := payload["user_email"].(string)
		if email == "" {
			email = "user@example.com" // mock default
		}
		subject := fmt.Sprintf("Order %v Created Successfully", payload["id"])
		return wp.svc.SendEmail(ctx, email, subject, "Thank you for your order.")

	case "payment.succeeded":
		email, _ := payload["user_email"].(string)
		if email == "" {
			email = "user@example.com"
		}
		subject := "Payment Received"
		return wp.svc.SendEmail(ctx, email, subject, "We received your payment securely.")

	case "payment.failed":
		email, _ := payload["user_email"].(string)
		if email == "" {
			email = "user@example.com"
		}
		subject := "Payment Failed"
		return wp.svc.SendEmail(ctx, email, subject, "Your recent payment try has failed. Please check your card.")

	case "user.created":
		email, _ := payload["email"].(string)
		if email == "" {
			email = "user@example.com"
		}
		subject := "Welcome to Payflow!"
		return wp.svc.SendEmail(ctx, email, subject, "Your account has been set up successfully.")

	default:
		slog.Warn("Received unknown topic event", "topic", topic)
		return nil // Ignore unknown topics
	}
}

func (wp *WorkerPool) sendToDLQ(ctx context.Context, originalMsg kafka.Message, err error) {
	failReason := err.Error()
	dlqMsg := kafka.Message{
		Key:   originalMsg.Key,
		Value: originalMsg.Value,
		Headers: append(originalMsg.Headers, kafka.Header{
			Key:   "Original-Topic",
			Value: []byte(originalMsg.Topic),
		}, kafka.Header{
			Key:   "Error-Reason",
			Value: []byte(failReason),
		}),
	}

	if wErr := wp.dlqWriter.WriteMessages(ctx, dlqMsg); wErr != nil {
		slog.Error("Failed to write to DLQ", "error", wErr)
	}
}

func (wp *WorkerPool) Stop() {
	for _, reader := range wp.readers {
		reader.Close()
	}
	if wp.dlqWriter != nil {
		wp.dlqWriter.Close()
	}
	slog.Info("Worker pool gracefully stopped")
}
