// Package logger provides structured JSON logging with distributed tracing support.
package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// Level represents a log severity level.
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
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
func NewTraceId() (string, error) {
	b := make([]byte, traceIdByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const spanIdByteLength = 8 // 8 bytes = 16 hex chars

// NewSpanId generates a cryptographically random 16-char hex string (8 bytes).
func NewSpanId() (string, error) {
	b := make([]byte, spanIdByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Logger writes structured JSON log entries to an output writer.
type Logger struct {
	service    string
	appVersion string
	out        io.Writer
}

// New creates a Logger that writes to stdout.
func New(service, version string) *Logger {
	return &Logger{
		service:    service,
		appVersion: version,
		out:        os.Stdout,
	}
}

// logEntry is the JSON structure for a single log line.
type logEntry struct {
	Timestamp    string `json:"timestamp"`
	Level        Level  `json:"level"`
	Service      string `json:"service"`
	Version      string `json:"version"`
	TraceId      string `json:"trace_id,omitempty"`
	SpanId       string `json:"span_id,omitempty"`
	ParentSpanId string `json:"parent_span_id,omitempty"`
	Message      string `json:"message"`
	ErrorCode    string `json:"error_code,omitempty"`
	Error        string `json:"error,omitempty"`
	StackTrace   string `json:"stack_trace,omitempty"`
}

func (l *Logger) write(ctx context.Context, level Level, message, errorCode string, err error) {
	sc := spanContextFrom(ctx)
	entry := logEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		Level:        level,
		Service:      l.service,
		Version:      l.appVersion,
		TraceId:      sc.TraceId,
		SpanId:       sc.SpanId,
		ParentSpanId: sc.ParentSpanId,
		Message:      message,
	}
	if err != nil {
		entry.ErrorCode = errorCode
		entry.Error = err.Error()
		if level == LevelError {
			entry.StackTrace = captureStack()
		}
	}
	b, _ := json.Marshal(entry)
	b = append(b, '\n')
	_, _ = l.out.Write(b)
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(ctx context.Context, message string) {
	l.write(ctx, LevelDebug, message, "", nil)
}

// Info logs a message at INFO level.
func (l *Logger) Info(ctx context.Context, message string) {
	l.write(ctx, LevelInfo, message, "", nil)
}

// InfoWithError logs a message and error at INFO level.
func (l *Logger) InfoWithError(ctx context.Context, message string, err error) {
	l.write(ctx, LevelInfo, message, "", err)
}

// Warn logs a message and error at WARN level.
func (l *Logger) Warn(ctx context.Context, message string, err error) {
	l.write(ctx, LevelWarn, message, "", err)
}

// Error logs a message, error code, and error at ERROR level (includes stack trace).
func (l *Logger) Error(ctx context.Context, message, errorCode string, err error) {
	l.write(ctx, LevelError, message, errorCode, err)
}

const stackBufSize = 4096

func captureStack() string {
	buf := make([]byte, stackBufSize)
	n := runtime.Stack(buf, false)
	return strings.TrimSpace(string(buf[:n]))
}
