// Package logger provides structured JSON logging with distributed tracing support.
package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

type spanContextKey struct{}

// SpanContext holds distributed tracing identifiers for a request.
type SpanContext struct {
	TraceId      string
	SpanId       string
	ParentSpanId string
}

// WithSpanContext stores sc in ctx and returns the updated context.
func WithSpanContext(ctx context.Context, sc SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey{}, sc)
}

// spanContextFrom retrieves the SpanContext from ctx, returning zero-value if absent.
func spanContextFrom(ctx context.Context) SpanContext {
	sc, _ := ctx.Value(spanContextKey{}).(SpanContext)
	return sc
}

const traceIdByteLength = 16 // 16 bytes = 32 hex chars

// NewTraceId generates a cryptographically random 32-char hex string (16 bytes).
func NewTraceId() string {
	b := make([]byte, traceIdByteLength)
	rand.Read(b) //nolint:errcheck // crypto/rand.Read never returns an error since Go 1.20
	return hex.EncodeToString(b)
}

const spanIdByteLength = 8 // 8 bytes = 16 hex chars

// NewSpanId generates a cryptographically random 16-char hex string (8 bytes).
func NewSpanId() string {
	b := make([]byte, spanIdByteLength)
	rand.Read(b) //nolint:errcheck // crypto/rand.Read never returns an error since Go 1.20
	return hex.EncodeToString(b)
}

// Logger writes structured JSON log entries using slog.
type Logger struct {
	slog *slog.Logger
}

// New creates a Logger that writes JSON to stdout.
// Call this after setting Service and Version.
func New(service, version string) *Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})
	return &Logger{
		slog: slog.New(h).With("service", service, "version", version),
	}
}

func (l *Logger) withTrace(ctx context.Context) *slog.Logger {
	sc := spanContextFrom(ctx)
	sl := l.slog
	if sc.TraceId != "" {
		sl = sl.With("trace_id", sc.TraceId)
	}
	if sc.SpanId != "" {
		sl = sl.With("span_id", sc.SpanId)
	}
	if sc.ParentSpanId != "" {
		sl = sl.With("parent_span_id", sc.ParentSpanId)
	}
	return sl
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(ctx context.Context, message string) {
	l.withTrace(ctx).Debug(message)
}

// Info logs a message at INFO level.
func (l *Logger) Info(ctx context.Context, message string) {
	l.withTrace(ctx).Info(message)
}

// InfoWithError logs a message and error at INFO level.
func (l *Logger) InfoWithError(ctx context.Context, message string, err error) {
	l.withTrace(ctx).Info(message, slog.String("error", err.Error()))
}

// Warn logs a message and error at WARN level.
func (l *Logger) Warn(ctx context.Context, message string, err error) {
	l.withTrace(ctx).Warn(message, slog.String("error", err.Error()))
}

// Error logs a message, error code, and error at ERROR level (includes stack trace).
func (l *Logger) Error(ctx context.Context, message, errorCode string, err error) {
	l.withTrace(ctx).Error(message,
		slog.String("error_code", errorCode),
		slog.String("error", err.Error()),
		slog.String("stack_trace", captureStack()),
	)
}

const stackBufSize = 4096

func captureStack() string {
	buf := make([]byte, stackBufSize)
	n := runtime.Stack(buf, false)
	return strings.TrimSpace(string(buf[:n]))
}
