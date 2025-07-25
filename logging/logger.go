package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NewLogger creates a preconfigured slog logger with Prometheus metrics hook
func NewLogger(cfg *Config, metricsNamespace string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}
	
	handler := slog.NewJSONHandler(os.Stdout, opts)
	
	// Wrap with Prometheus metrics hook
	wrappedHandler := newPrometheusHandler(handler, metricsNamespace)
	
	logger := slog.New(wrappedHandler)
	slog.SetDefault(logger)

	return logger
}

// prometheusHandler wraps an slog.Handler to expose Prometheus counters for various log levels
type prometheusHandler struct {
	handler           slog.Handler
	messageCounterVec *prometheus.CounterVec
}

func newPrometheusHandler(handler slog.Handler, metricsNamespace string) slog.Handler {
	messageCounterVec := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "log_messages_total",
		Help:      "Total number of log messages.",
	}, []string{"level"})

	// Preinitialize counters for all supported log levels so that they expose 0 for each level on startup
	supportedLevels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}
	for _, level := range supportedLevels {
		messageCounterVec.WithLabelValues(level.String())
	}

	return &prometheusHandler{
		handler:           handler,
		messageCounterVec: messageCounterVec,
	}
}

func (h *prometheusHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *prometheusHandler) Handle(ctx context.Context, record slog.Record) error {
	h.messageCounterVec.WithLabelValues(record.Level.String()).Inc()
	return h.handler.Handle(ctx, record)
}

func (h *prometheusHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &prometheusHandler{
		handler:           h.handler.WithAttrs(attrs),
		messageCounterVec: h.messageCounterVec,
	}
}

func (h *prometheusHandler) WithGroup(name string) slog.Handler {
	return &prometheusHandler{
		handler:           h.handler.WithGroup(name),
		messageCounterVec: h.messageCounterVec,
	}
}
