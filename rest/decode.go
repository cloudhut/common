package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudhut/common/header"
	"io"
	"net/http"
	"strings"
)

// Decode tries to decode the request body into dst and calls its OK() function to validate the object.
// It returns an error if:
// - the content-type does not contain "application/json"
// - body is smaller than 1MB
// - any unknown fields were set
// - deserialization fails
// - the OK() method returns an error
func Decode(w http.ResponseWriter, r *http.Request, dst interface{}) *Error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			err := fmt.Errorf("wrong or missing Content-Type header value")
			return &Error{Err: err, Status: http.StatusUnsupportedMediaType, Message: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("request body contains badly-formed JSON")
			return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &Error{Err: err, Status: http.StatusRequestEntityTooLarge, Message: msg}

		default:
			msg := fmt.Sprintf("Unknown error while decoding the request: %v", err.Error())
			return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &Error{Err: err, Status: http.StatusBadRequest, Message: msg}
	}

	if valid, ok := dst.(interface {
		OK() error
	}); ok {
		err = valid.OK()
		if err != nil {
			return &Error{Err: err, Status: http.StatusBadRequest, Message: fmt.Sprintf("validating the decoded object failed: %v", err.Error())}
		}
	}

	return nil
}
