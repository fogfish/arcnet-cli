//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package mock is a fake implementing port.Fetcher with configurable
// return values/errors, for internal/app/config/service's Default unit
// tests — no real network access in go test (constitution Principle VI).
package mock

import "context"

type Fetcher struct {
	Body []byte
	Err  error
}

func (f Fetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Body, nil
}
