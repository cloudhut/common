package rest

import "go.uber.org/zap/zapcore"

// Error must be created to issue a REST compliant error
type Error struct {
	Err          error           `json:"-"`
	Status       int             `json:"statusCode"`
	Message      string          `json:"message"`
	InternalLogs []zapcore.Field `json:"-"`
	IsSilent     bool            `json:"-"`
}
