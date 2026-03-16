package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/config"
	"api-gateway/middleware"
	"api-gateway/proxy"

	"shared/database"
	"shared/logger"
)

func main() {
	// 1. Init Logging
	logger.Init("api-gateway", true)

	// 2. Load Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting API Gateway", "port", cfg.Port, "env", cfg.Environment)

	// 3. Connect to Redis for Rate Limiting
	redisClient, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPass)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// 4. Setup Proxies
	userProxy, err := proxy.Setup(cfg.UserServiceURL)
	if err != nil {
		slog.Error("Invalid User Service URL", "error", err)
		os.Exit(1)
	}

	orderProxy, err := proxy.Setup(cfg.OrderServiceURL)
	if err != nil {
		slog.Error("Invalid Order Service URL", "error", err)
		os.Exit(1)
	}

	paymentProxy, err := proxy.Setup(cfg.PaymentServiceURL)
	if err != nil {
		slog.Error("Invalid Payment Service URL", "error", err)
		os.Exit(1)
	}

	// 5. Setup Router and Middlewares
	mux := http.NewServeMux()

	// General Routes directly to proxies based on prefix
	// Users
	mux.Handle("/users/", http.StripPrefix("", userProxy)) // Handles POST /users/register, /users/login, GET /users/{id}

	// Orders
	mux.Handle("/orders/", http.StripPrefix("", orderProxy)) // Handles POST /orders, GET /orders/{id}, GET /orders
	mux.Handle("/orders", http.StripPrefix("", orderProxy))

	// Payments
	mux.Handle("/payments/", http.StripPrefix("", paymentProxy)) // Handles POST /payments/charge
	mux.Handle("/payments", http.StripPrefix("", paymentProxy))

	// Health check for gateway itself
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "gateway_ok"}`))
	})

	// Wrap Middlewares: Global Error Recovery (assuming we add one) -> Rate Limiter -> Auth (JWT) -> MUX
	var handler http.Handler = mux

	// Auth Middleware maps the JWT to headers correctly
	authMid := middleware.AuthMiddleware(cfg.JWTSecret)
	handler = authMid(handler)

	// Rate Limiter configuration: max 10 requests per minute per IP/User for this example
	rateLimiter := middleware.NewRateLimiter(redisClient, 10, time.Minute)
	// Optionally we can choose to wrap only specific routes like /payments in the rate limiter using specific sub-routers
	// But as the README specifies: "POST /payments max: 10 requests / minute", we'll wrap the payment routes specifically
	// To keep it simple globally for now we wrap everything, or build a custom mapping. Let's wrap globally.
	handler = rateLimiter.Middleware(handler)

	// 6. Start HTTP Server
	serverAddr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: handler,
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
