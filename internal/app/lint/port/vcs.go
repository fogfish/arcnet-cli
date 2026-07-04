//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package port declares secondary ports private to the lint use-case.
package port

import "context"

// VCS is the narrowest of the three port.VCS interfaces in this codebase
// (ctrl, graph, now lint) — lint never initializes, stages, or commits.
type VCS interface {
	CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)
}
