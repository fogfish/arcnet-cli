//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package schema is the graph schema domain use-case: it isolates
// ARCNET-CORE's declared vocabulary of predicates and types as
// _schema/predicates/*.md and _schema/types/*.md — one versioned,
// human-readable document per predicate and type. No port/adapter
// subdirectory: its only I/O is the already-shared internal/adapter/fsys,
// consumed directly.
package schema

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/schema/port"
	"github.com/fogfish/arcnet-cli/internal/app/schema/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Seed renders every core predicate/type's built-in schema document, keyed
// by on-disk path. Thin delegator into service.Seed.
func Seed() map[string][]byte {
	return service.Seed()
}

// Resolve returns the graph's effective schema index, read back from
// _schema/. Thin delegator into service.Resolve.
func Resolve(store fsys.Store) (core.Index, error) {
	return service.Resolve(store)
}

// RegisterType creates typ's type schema document if one is not already
// present. Thin delegator into service.RegisterType.
func RegisterType(store fsys.Store, typ string) (created bool, err error) {
	return service.RegisterType(store, typ)
}

// RegisterPredicate creates predicate's predicate schema document if one is
// not already present. Thin delegator into service.RegisterPredicate.
func RegisterPredicate(store fsys.Store, predicate, observedRole, label string) (created bool, err error) {
	return service.RegisterPredicate(store, predicate, observedRole, label)
}

// ApplyPatch reads the patch document at source (a local path, URL, or
// arcnet: catalog reference) and creates/merges every Property/Class node
// it carries into _schema/. Thin delegator into service.ApplyPatch.
func ApplyPatch(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, fetcher port.Fetcher, reporter bios.Reporter, dir, source string) (kernel.ApplySchemaResult, error) {
	return service.ApplyPatch(ctx, mounter, vcs, fetcher, reporter, dir, source)
}

// Component is schema's concrete, zero-value primary port, satisfying
// internal/app/graph/port.SchemaRegistry structurally (ADR 001 port
// isolation rule 1 — no explicit "implements" needed) — the same technique
// internal/adapter/git.Git already uses to satisfy three separate
// port.VCS interfaces.
type Component struct{}

func (Component) RegisterType(store fsys.Store, typ string) (created bool, err error) {
	return RegisterType(store, typ)
}

func (Component) RegisterPredicate(store fsys.Store, predicate, observedRole, label string) (created bool, err error) {
	return RegisterPredicate(store, predicate, observedRole, label)
}
