//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package port

import (
	"context"
	"io"
)

// Fetcher retrieves a patch document from a URL — invoked only after the
// caller has classified the source as an http/https URL (research.md D1).
// Satisfied by the shared internal/adapter/http.Client.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}
