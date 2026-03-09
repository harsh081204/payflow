package logger

import (
	"log/slog"
	"os"
)

// Init initializes the global logger for the entire application.
// In production, we use JSON format. In development, you might prefer text.
func Init(serviceName string, isDev bool) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // Set minimum level to Info by default
	}

	if isDev {
		opts.Level = slog.LevelDebug // More verbose in dev
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	// Pre-attach the service name so all logs are tagged automatically
	logger = logger.With(slog.String("service", serviceName))

	// Set as default logger for the whole app
	slog.SetDefault(logger)
}
