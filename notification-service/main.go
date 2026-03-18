package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"notification-service/config"
	"notification-service/service"
	"notification-service/worker"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"shared/logger"
)

func main() {
	// 1. Initialize Logger
	logger.Init("notification-service", true)

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting notification-service", "port", cfg.Port, "env", cfg.Environment)

	// Dependency Injection
	notificationSvc := service.NewNotificationService()

	// 3. Setup Worker Pool for Kafka
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	kafkaWorkerPool := worker.NewWorkerPool(brokers, cfg.KafkaGroupID, cfg.WorkerCount, notificationSvc)

	topics := []string{
		"order.created",
		"payment.succeeded",
		"payment.failed",
		"user.created",
	}

	ctx, cancelKafka := context.WithCancel(context.Background())
	// Start consuming in the background
	go kafkaWorkerPool.Start(ctx, topics)

	// 4. Setup Basic HTTP Server (Healthchecks)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		// Could potentially check Kafka connectivity here via a dummy ping or dial
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready"}`))
	})

	serverAddr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("HTTP Server listening (for healthchecks)", "addr", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
		}
	}()

	<-done
	slog.Info("Server stopped, starting graceful shutdown")

	// Shutdown Kafka Consumers
	cancelKafka()
	kafkaWorkerPool.Stop()

	// Shutdown HTTP Server
	shutdownCtx, cancelHTTP := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelHTTP()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}
	slog.Info("Server exited gracefully")
}
