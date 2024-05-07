// Internal logging interface
package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
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

// Return a copy of the context that contains logger with given attributes
func With(ctx context.Context, keyval ...any) (context.Context, Logger) {
	logger := FromContext(ctx).With(keyval...)
	return Insert(ctx, logger), logger
}

type contextKey struct{}

var loggerContextKey contextKey

// Configure top level logger
func Setup() {
	if !setup.TryLock() {
		return // setup was already called
	}

	const timestamp = true // TODO: automatically detect systemd and disable timestamps
	replace := func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.TimeKey && groups == nil {
			if !timestamp {
				return slog.Attr{} // empty Attr will be omitted during output
			}
			t, ok := attr.Value.Any().(time.Time)
			if !ok {
				return attr
			}
			return slog.String(slog.TimeKey, t.UTC().Format(time.RFC3339))
		}
		return attr
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: replace,
	})
	slog.SetDefault(slog.New(handler))
}

var setup sync.Mutex
