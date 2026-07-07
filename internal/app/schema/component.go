//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package schema is the graph schema domain use-case: it isolates
// ARCNET-CORE's declared vocabulary of node kinds, merge behaviors, and
// predicates as _schema/nodes/*.md and _schema/predicates/*.md — one
// versioned, human-readable document per node kind and predicate. No
// port/adapter subdirectory: its only I/O is the already-shared
// internal/adapter/fsys, consumed directly.
package schema

import (
	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/schema/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Seed renders every core kind/predicate's built-in schema document,
// keyed by on-disk path. Thin delegator into service.Seed.
func Seed() map[string][]byte {
	return service.Seed()
}

// Resolve returns the graph's effective merge-rule set and registered
// predicate set, read back from _schema/. Thin delegator into
// service.Resolve.
func Resolve(store fsys.Store) (core.MergeRuleSet, map[string]bool, error) {
	return service.Resolve(store)
}

// RegisterKind creates kind's node-kind schema document if one is not
// already present. Thin delegator into service.RegisterKind.
func RegisterKind(store fsys.Store, kind string) (created bool, err error) {
	return service.RegisterKind(store, kind)
}

// RegisterPredicate creates predicate's predicate schema document if one is
// not already present. Thin delegator into service.RegisterPredicate.
func RegisterPredicate(store fsys.Store, predicate string) (created bool, err error) {
	return service.RegisterPredicate(store, predicate)
}

// Component is schema's concrete, zero-value primary port, satisfying
// internal/app/graph/port.SchemaRegistry structurally (ADR 001 port
// isolation rule 1 — no explicit "implements" needed) — the same technique
// internal/adapter/git.Git already uses to satisfy three separate
// port.VCS interfaces.
type Component struct{}

func (Component) RegisterKind(store fsys.Store, kind string) (created bool, err error) {
	return RegisterKind(store, kind)
}

func (Component) RegisterPredicate(store fsys.Store, predicate string) (created bool, err error) {
	return RegisterPredicate(store, predicate)
}
