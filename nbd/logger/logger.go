// Internal logging interface
package logger

import (
	"context"
	"log/slog"
)

type Logger = *slog.Logger

// Retrieve logger from context
func FromContext(ctx context.Context) Logger {
	value := ctx.Value(loggerContextKey)
	logger, ok := value.(Logger)
	if !ok || value == nil {
		return slog.Default()
	}
	return logger
}

// Return a copy of the context with logger inserted
func Insert(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

type contextKey struct{}

var loggerContextKey contextKey
