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
// — Apply's auto-discovery hook registers a previously-unseen kind or
// predicate into _schema/ mid-transaction, without importing
// internal/app/schema directly.
type SchemaRegistry interface {
	RegisterKind(store fsys.Store, kind string) (created bool, err error)
	RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
}
