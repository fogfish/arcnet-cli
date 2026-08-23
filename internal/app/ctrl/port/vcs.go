//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package port declares secondary ports private to the ctrl use-case.
package port

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// VCS is the narrow, capability-scoped interface exactly the Init use-case
// needs — nothing mirroring the full git CLI surface.
type VCS interface {
	IsAvailable(ctx context.Context) error
	Init(ctx context.Context, dir string) error
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)
}

// SchemaResolver reads a mounted graph's effective schema index back from
// its own _schema/ documents. arc upgrade needs it for exactly one step —
// resolving the corrected vocabulary it has just written, so the
// prose-drift scan can tell which text predicates are first-fixed
// (contract C3.2 step 5).
//
// Declared here rather than importing internal/app/schema directly so ctrl
// keeps depending on a capability it names, not on another use-case
// (ADR 001). internal/app/schema.Component satisfies it structurally.
type SchemaResolver interface {
	Resolve(store fsys.Store) (core.Index, error)

	// SeedIndex is the same index computed from the built-in vocabulary
	// alone, with no graph read. It is what the prose-drift scan falls back
	// to when the graph on disk does not resolve — the normal state of a
	// graph that has not been upgraded yet, and precisely the case
	// --dry-run exists to inspect.
	SeedIndex() core.Index
}
