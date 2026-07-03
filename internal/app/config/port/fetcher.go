//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package port declares secondary ports private to the config use-case.
package port

import "context"

// Fetcher retrieves the config-seed content arc init writes into a new
// graph's .arc/config.yml.
type Fetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}
