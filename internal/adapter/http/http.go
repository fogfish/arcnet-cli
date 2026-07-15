//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package http is the shared, cross-use-case HTTP-fetch adapter (ADR 001's
// application-level adapter tier), alongside internal/adapter/fsys/git. Its
// one capability — a context-respecting GET with a default, overridable
// timeout — is placed here rather than nested under a single use-case's
// own adapter directory since URL fetching is a capability any future
// command could also need (research.md D2).
package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/fogfish/faults"
)

// errNoCause is passed to ErrFetchStatus.With for a non-2xx response — not
// caused by an underlying Go error — so the rendered message has no
// trailing "%!s(<nil>)" artifact (mirrors internal/core's own precedent).
var errNoCause = errors.New("")

const (
	// ErrFetch is returned when the HTTP request fails outright (network
	// error, timeout); names the URL.
	ErrFetch = faults.Safe1[string]("failed to fetch %s")

	// ErrFetchStatus is returned when the server responds with a non-2xx
	// status; names the URL and status code.
	ErrFetchStatus = faults.Safe2[string, int]("failed to fetch %s: server responded with status %d")
)

// Client is a small port.Fetcher-satisfying HTTP client, backed by
// net/http.Client with a default timeout.
type Client struct {
	HTTPClient *http.Client
}

// New constructs a Client with Timeout: timeout (defaulting to 30s at the
// cmd/ wiring layer when the --timeout flag is unset).
func New(timeout time.Duration) Client {
	return Client{HTTPClient: &http.Client{Timeout: timeout}}
}

// Fetch issues a single GET against url, returning a readable, caller-
// closed body on a 2xx response. ctx carries the effective deadline so
// cancellation/timeout is honored mid-fetch, not just at connection time.
func (c Client) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ErrFetch.With(err, url)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, ErrFetch.With(err, url)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, ErrFetchStatus.With(errNoCause, url, resp.StatusCode)
	}

	return resp.Body, nil
}
