//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package graph is the graph-mutation (graph I/O) domain use-case: it
// applies a document patch into an already-initialized knowledge graph.
package graph

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Apply ingests the patch at patchPath into the graph rooted at dir. It is
// a thin delegator into service.Apply.
func Apply(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, rules core.MergeRuleSet, predicates map[string]bool, schema port.SchemaRegistry, dir, patchPath string) (kernel.ApplyResult, error) {
	return service.Apply(ctx, mounter, vcs, reporter, rules, predicates, schema, dir, patchPath)
}

// Grep searches node file content across the graph rooted at dir for lines
// matching pattern, narrowed by filter. It is a thin delegator into
// service.Grep.
func Grep(ctx context.Context, mounter fsys.Mounter, filter core.Filter, pattern string, cfg configkernel.GrepConfig, dir string) (kernel.GrepResult, error) {
	return service.Grep(ctx, mounter, filter, pattern, cfg, dir)
}
