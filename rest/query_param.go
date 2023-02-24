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
)

// GetQueryParam retrieves the query parameter with your given key from the request,
// unescapes the value and returns it. If the query parameter does not exist an
// empty string will be returned. If the unescaping fails the unescaped string
// will be returned.
func GetQueryParam(r *http.Request, key string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return ""
	}

	unescapedVal, err := url.QueryUnescape(val)
	if err != nil {
		return val
	}
	return unescapedVal
}
