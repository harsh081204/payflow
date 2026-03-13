package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"order-service/config"
	"order-service/handler"
	"order-service/repository"
	"order-service/service"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shared/database"
	"shared/logger"
)

func main() {
	// 1. Initialize Logger
	logger.Init("order-service", true)

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting order-service", "port", cfg.Port, "env", cfg.Environment)

	// 3. Connect to Database (Postgres)
	dbPool, err := database.ConnectPostgres(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 4. Initialize Dependency Injection
	orderRepo := repository.NewOrderRepository(dbPool)
	orderSvc := service.NewOrderService(orderRepo)
	orderHandler := handler.NewOrderHandler(orderSvc)

	// 5. Setup Router
	mux := http.NewServeMux()

	// Register Routes
	orderHandler.RegisterRoutes(mux)

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

	// 6. Start HTTP Server
	serverAddr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: mux, // In production we should wrap with logging & recovery middlewares
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}
	slog.Info("Server exited gracefully")
}
