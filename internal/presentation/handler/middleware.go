package handler

import (
	"fmt"
	"net/http"

	"oidc-tutorial/internal/logger"
)

// statusRecorder wraps http.ResponseWriter to capture the written status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// NewTraceMiddleware returns middleware that injects a new SpanContext (trace_id +
// span_id) into every request context and logs the request/response pair.
func NewTraceMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceId, err := logger.NewTraceId()
			if err != nil {
				traceId = "unknown"
			}
			spanId, err := logger.NewSpanId()
			if err != nil {
				spanId = "unknown"
			}

			ctx := logger.WithSpanContext(r.Context(), logger.SpanContext{
				TraceId: traceId,
				SpanId:  spanId,
			})

			log.Info(ctx, fmt.Sprintf("request received: %s %s", r.Method, r.URL.Path))

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r.WithContext(ctx))

			log.Info(ctx, fmt.Sprintf("response sent: %s %s %d", r.Method, r.URL.Path, rec.status))
		})
	}
}
