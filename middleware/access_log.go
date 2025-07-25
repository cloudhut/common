package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// AccessLog implements the middleware interface
type AccessLog struct {
	logger *slog.Logger
}

// NewAccessLog creates a new middleware which prints access logs
func NewAccessLog(logger *slog.Logger, extraHeader string) *AccessLog {
	return &AccessLog{logger}
}

// Wrap implements the middleware interface
func (a *AccessLog) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call next handler
		next.ServeHTTP(w, r)

		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			panic("could not convert to middleware.WrapResponseWriter")
		}

		duration := time.Since(start)
		durationMs := duration.Nanoseconds() / (1000 * 1000)

		a.logger.InfoContext(r.Context(), "http request",
			slog.String("log_type", "access"),
			slog.String("remote_address", r.RemoteAddr),
			slog.Int64("response_time", durationMs),
			slog.String("protocol", r.Proto),
			slog.String("request_method", r.Method),
			slog.String("query_string", r.URL.RawQuery),
			slog.String("status", strconv.Itoa(ww.Status())),
			slog.String("uri", r.URL.Path),
			slog.String("server_name", r.URL.Host),
			slog.Int64("bytes_received", r.ContentLength),
			slog.Int("bytes_sent", ww.BytesWritten()),
			slog.String("remote_client_id", r.Header.Get("remoteClientId")))
	})
}
