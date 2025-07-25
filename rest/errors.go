package rest

import "log/slog"

// Error must be created to issue a REST compliant error
type Error struct {
	Err          error      `json:"-"`
	Status       int        `json:"statusCode"`
	Message      string     `json:"message"`
	InternalLogs []slog.Attr `json:"-"`
	IsSilent     bool       `json:"-"`
}
