//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package http is the stdlib net/http-backed real implementation of
// port.Fetcher.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fogfish/faults"
)

const ErrFetch = faults.Type("failed to fetch config seed")

// Fetcher wraps a stdlib *http.Client with a fixed, non-overridable
// timeout (research.md D5 revised, contracts/config-contract.md's flagged
// Principle VII gap) and no retries — one attempt is sufficient given
// config.Default's always-safe fallback.
type Fetcher struct {
	Client *http.Client
}

func New() Fetcher {
	return Fetcher{Client: &http.Client{Timeout: 3 * time.Second}}
}

func (f Fetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ErrFetch.With(err)
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, ErrFetch.With(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ErrFetch.With(fmt.Errorf("unexpected status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrFetch.With(err)
	}

	return body, nil
}
