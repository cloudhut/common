package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Decode decodes the request body into v and calls its OK() function to validate the object.
func Decode(r *http.Request, v interface{}) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		return fmt.Errorf("decoding json failed: %w", err)
	}
	if valid, ok := v.(interface {
		OK() error
	}); ok {
		err = valid.OK()
		if err != nil {
			return fmt.Errorf("validating the decoded object failed: %w", err)
		}
	}
	return nil
}
