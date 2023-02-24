// Copyright 2023 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package rest

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

// GetURLParam retrieves the url parameter with your given key from the request,
// unescapes the value and returns it. If the url parameter does not exist an
// empty string will be returned. If the unescaping fails the unescaped string
// will be returned.
func GetURLParam(r *http.Request, key string) string {
	val := chi.URLParam(r, key)
	if val == "" {
		return ""
	}

	unescapedVal, err := url.PathUnescape(val)
	if err != nil {
		return val
	}
	return unescapedVal
}
