package rest

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// SendResponse tries to send your data as JSON. If this fails it will print REST compliant errors
func SendResponse(w http.ResponseWriter, r *http.Request, logger *slog.Logger, status int, data interface{}) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		serverError(w, r, logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonBytes)
}

// SendRESTError accepts a REST error which can be send to the user
func SendRESTError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, restErr *Error) {
	if !restErr.IsSilent {
		logAttrs := []slog.Attr{
			slog.String("route", r.RequestURI),
			slog.String("method", r.Method),
			slog.Int("status_code", restErr.Status),
			slog.String("remote_address", r.RemoteAddr),
			slog.String("public_error", restErr.Message),
			slog.Any("error", restErr.Err),
		}
		logAttrs = append(logAttrs, restErr.InternalLogs...)
		logger.LogAttrs(r.Context(), slog.LevelError, "Sending REST error", logAttrs...)
	}

	SendResponse(w, r, logger, restErr.Status, restErr)
}

// ServerError prints a plain JSON error message
func serverError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	// Log the detailed error
	logger.ErrorContext(r.Context(), "internal server error",
		slog.String("route", r.RequestURI),
		slog.String("method", r.Method),
		slog.Int("status_code", http.StatusInternalServerError),
		slog.String("remote_address", r.RemoteAddr),
		slog.Any("error", err),
	)

	// Send a generic response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	jsonErrorString := fmt.Sprintf(
		`{"statusCode":%d,"message":"Internal Server Error"}`,
		http.StatusInternalServerError,
	)
	w.Write([]byte(jsonErrorString))
}

// HandleNotFound returns a handler func to respond to non existent routes with a REST compliant
// error message
func HandleNotFound(logger *slog.Logger) http.HandlerFunc {
	restErr := &Error{
		Err:      fmt.Errorf("the requested resource does not exist"),
		Status:   http.StatusNotFound,
		Message:  "Resource was not found.",
		IsSilent: true,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		SendRESTError(w, r, logger, restErr)
	}
}

// HandleMethodNotAllowed returns a handler func to respond to routes requested with the wrong verb a
// REST compliant error message
func HandleMethodNotAllowed(logger *slog.Logger) http.HandlerFunc {
	restErr := &Error{
		Err:      fmt.Errorf("the method used in the request is not allowed"),
		Status:   http.StatusMethodNotAllowed,
		Message:  "Method is not allowed.",
		IsSilent: true,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		SendRESTError(w, r, logger, restErr)
	}
}
