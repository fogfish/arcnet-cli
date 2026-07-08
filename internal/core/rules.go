//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

// PredicateDef is the decoded, in-memory shape of one
// _schema/predicates/<name>.md document (CORE §9.1): Role is one of
// meta/text/href/edge/link, Merge is the MergeOp arc apply uses when the
// predicate's own value participates in a node merge, Label/Aligned are
// optional, and Description is the document's mandatory descriptive body.
type PredicateDef struct {
	Role        string
	Merge       MergeOp
	Label       string
	Aligned     string
	Description string
}

// TypeDef is the decoded, in-memory shape of one _schema/types/<name>.md
// document (CORE §9.2 plus the arc apply-specific merge bridge field):
// Merge is the MergeOp arc apply uses to reconcile an incoming contribution
// of this type with an existing node, Required/Optional name the
// predicates a conforming instance must/may carry, and Description is the
// document's mandatory descriptive body.
type TypeDef struct {
	Merge       MergeOp
	Required    []string
	Optional    []string
	Description string
}

// Index is the graph's effective schema, built once per command invocation
// by internal/app/schema/service.Resolve from a graph's own _schema/
// documents. Predicates/Types are keyed by the predicate/type's own name
// (equal to its file's basename); absence of a key means "not registered".
// Immutable after construction — Resolve returns a fully-built value, never
// a handle a caller mutates in place.
type Index struct {
	Predicates map[string]PredicateDef
	Types      map[string]TypeDef
}
