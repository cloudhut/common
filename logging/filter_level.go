package logging

import (
	"context"
	"log/slog"
)

// FilterLevel wraps an existing slog.Handler but logs at a different level.
// This function can be used to create child loggers which should print
// only log levels at a different log level than the parent logger.
func FilterLevel(level slog.Level) func(slog.Handler) slog.Handler {
	return func(h slog.Handler) slog.Handler {
		return newLevelFilterHandler(h, level)
	}
}

// levelFilterHandler allows to change the log level on the fly.
type levelFilterHandler struct {
	handler slog.Handler
	level   slog.Level
}

func newLevelFilterHandler(handler slog.Handler, level slog.Level) slog.Handler {
	return &levelFilterHandler{handler, level}
}

// Enabled checks if the level is to be printed
func (h *levelFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle determines whether the supplied Record should be logged
func (h *levelFilterHandler) Handle(ctx context.Context, record slog.Record) error {
	if !h.Enabled(ctx, record.Level) {
		return nil
	}

	return h.handler.Handle(ctx, record)
}

func (h *levelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelFilterHandler{
		handler: h.handler.WithAttrs(attrs),
		level:   h.level,
	}
}

func (h *levelFilterHandler) WithGroup(name string) slog.Handler {
	return &levelFilterHandler{
		handler: h.handler.WithGroup(name),
		level:   h.level,
	}
}
