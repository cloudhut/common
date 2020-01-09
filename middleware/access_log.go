package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AccessLog implements the middleware interface
type AccessLog struct {
	logger                *zap.Logger
	extraHeader           string
	blockWhenExtraMissing bool
}

// NewAccessLog creates a new middleware which prints access logs
func NewAccessLog(logger *zap.Logger, extraHeader string, blockWhenExtraMissing bool) *AccessLog {
	return &AccessLog{logger, extraHeader, blockWhenExtraMissing}
}

// Wrap implements the middleware interface
func (a *AccessLog) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		fields := []zapcore.Field{
			zap.String("log_type", "access"),
			zap.String("remote_address", r.RemoteAddr),
			//zap.Int64("response_time", durationMs),  // Added below
			zap.String("protocol", r.Proto),
			zap.String("request_method", r.Method),
			zap.String("query_string", r.URL.RawQuery),
			//zap.String("status", strconv.Itoa(ww.Status())),  // Added below
			zap.String("uri", r.URL.Path),
			zap.String("server_name", r.URL.Host),
			zap.Int64("bytes_received", r.ContentLength),
			//zap.Int("bytes_sent", ww.BytesWritten()),  // Added below
			zap.String("remote_client_id", r.Header.Get("remoteClientId")),
		}

		if len(a.extraHeader) > 0 {
			v := r.Header.Get(a.extraHeader)
			if len(v) > 0 {
				// We found the wanted extra header
				fields = append(fields, zap.String(a.extraHeader, v))
			} else {
				// Extra header is missing, maybe we should block the request
				if a.blockWhenExtraMissing {
					a.logger.Info("request blocked, required extra header missing", fields...)
					return // dont even execute the request!
				}
			}
		}

		// Call next handler
		next.ServeHTTP(w, r)

		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			panic("could not convert to middleware.WrapResponseWriter")
		}

		duration := time.Since(start)
		durationMs := duration.Nanoseconds() / (1000 * 1000)

		// Append fields that are only available after the request has completed
		fields = append(fields,
			zap.Int64("response_time", durationMs),
			zap.String("status", strconv.Itoa(ww.Status())),
			zap.Int("bytes_sent", ww.BytesWritten()),
		)

		a.logger.Info("", fields...)
	})
}
