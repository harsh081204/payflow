package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestLogger generates a slog log for every incoming HTTP request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate a unique request ID (helpful for distributed tracing)
		reqID := uuid.New().String()
		r.Header.Set("X-Request-Id", reqID)

		// Create a custom ResponseWriter to intercept the status code
		ww := &wrappedWriter{ResponseWriter: w, status: http.StatusOK}

		// Pass down the request
		next.ServeHTTP(ww, r)

		// Log what happened
		slog.Info(
			"HTTP request",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// wrappedWriter helps us grab the HTTP status code (since http.ResponseWriter doesn't expose it)
type wrappedWriter struct {
	http.ResponseWriter
	status int
}

func (rw *wrappedWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
