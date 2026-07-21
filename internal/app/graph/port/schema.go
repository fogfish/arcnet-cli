//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package port

import (
	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
)

// SchemaRegistry is graph's own narrow, private port onto the schema
// use-case's write side, satisfied structurally by
// internal/app/schema's concrete component (ADR 001 port isolation rule 1)
// — Apply's auto-discovery hook registers a previously-unseen type or
// predicate into _schema/ mid-transaction, without importing
// internal/app/schema directly.
type SchemaRegistry interface {
	RegisterType(store fsys.Store, typ string) (created bool, err error)
	// RegisterPredicate auto-registers predicate on first observation.
	// observedRole ("edge", "link", or "text", BUG-002/BUG-003) is the role
	// the predicate was actually seen in — a text-observed predicate
	// defaults to role: text, merge: append instead of always role: edge,
	// merge: union, so newly discovered non-wikilink content (spec 010
	// FR-019) merges correctly on a later re-apply instead of being
	// silently coerced into edge shape; an edge occurrence carried with its
	// own "**Label**" block observes role: link instead of role: edge, so
	// its per-block grouping survives a write (spec 010 FR-022). label is
	// the block's own literal label text, when the predicate was carried
	// with one (BUG-003, spec 010 FR-021) — set as the registered
	// document's `label` attribute so its exact heading text (not a
	// derived-id approximation) is recoverable on a later render; empty
	// when the predicate has no carried label.
	RegisterPredicate(store fsys.Store, predicate, observedRole, label string) (created bool, err error)
}
