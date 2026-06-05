// Package obs (observability) provides structured logging and trace ID helpers
// for the Backend. It wraps the standard library slog package so the rest of
// the codebase never imports slog directly — if the logging backend changes
// (e.g. to OpenTelemetry) only this package needs updating.
//
// Log format: JSON to stderr, one object per line.
// Every log line includes at minimum: time, level, msg.
// Structured fields are passed as key/value pairs.
package obs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
)

// contextKey is the private type used for context keys in this package.
type contextKey int

const traceKey contextKey = 0

// Logger is the package-level structured JSON logger.
var Logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

// NewTraceID generates a random 8-byte (16 hex char) trace ID.
func NewTraceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// WithTrace returns a context with the given trace ID attached.
func WithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey, traceID)
}

// TraceFrom extracts the trace ID from ctx, or returns "" if none.
func TraceFrom(ctx context.Context) string {
	if v, ok := ctx.Value(traceKey).(string); ok {
		return v
	}
	return ""
}

// Log emits a structured INFO log line with optional trace_id from ctx.
func Log(ctx context.Context, msg string, args ...any) {
	if t := TraceFrom(ctx); t != "" {
		args = append([]any{"trace_id", t}, args...)
	}
	Logger.InfoContext(ctx, msg, args...)
}

// LogError emits a structured ERROR log line with optional trace_id from ctx.
func LogError(ctx context.Context, msg string, args ...any) {
	if t := TraceFrom(ctx); t != "" {
		args = append([]any{"trace_id", t}, args...)
	}
	Logger.ErrorContext(ctx, msg, args...)
}

// LogDebug emits a structured DEBUG log line with optional trace_id from ctx.
func LogDebug(ctx context.Context, msg string, args ...any) {
	if t := TraceFrom(ctx); t != "" {
		args = append([]any{"trace_id", t}, args...)
	}
	Logger.DebugContext(ctx, msg, args...)
}
