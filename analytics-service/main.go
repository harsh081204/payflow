package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"analytics-service/config"
	"analytics-service/handler"
	"analytics-service/repository"
	"analytics-service/worker"

	"shared/database"
	"shared/logger"
)

func main() {
	// 1. Initialize Logger
	logger.Init("analytics-service", true)

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting analytics-service", "port", cfg.Port, "env", cfg.Environment)

	// 3. Connect to Database (Postgres)
	dbPool, err := database.ConnectPostgres(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 4. Initialize Dependency Injection
	analyticsRepo := repository.NewAnalyticsRepository(dbPool)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsRepo)

	// 5. Setup Kafka Worker
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	kafkaWorker := worker.NewAnalyticsWorker(brokers, cfg.KafkaGroupID, analyticsRepo)

	topics := []string{
		"order.created",
		"payment.succeeded",
		"payment.failed",
	}

	ctx, cancelKafka := context.WithCancel(context.Background())
	// Start consuming
	go kafkaWorker.Start(ctx, topics)

	// 6. Setup Router
	mux := http.NewServeMux()

	// Register Routes
	analyticsHandler.RegisterRoutes(mux)

	// Add Health and Ready routes
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		err := dbPool.Ping(r.Context())
		if err != nil {
			http.Error(w, `{"status": "error"}`, http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready"}`))
	})

	// 7. Start HTTP Server
	serverAddr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Server listening", "addr", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
		}
	}()

	<-done
	slog.Info("Server stopped, starting graceful shutdown")

	// Shutdown Kafka Consumers
	cancelKafka()
	kafkaWorker.Stop()

	// Shutdown HTTP Server
	shutdownCtx, cancelHTTP := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelHTTP()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}
	slog.Info("Server exited gracefully")
}
