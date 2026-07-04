//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package lint is the graph conformance validation domain use-case: it
// walks every node file in an already-initialized knowledge graph and
// checks it against the full CORE §14 conformance checklist, without ever
// writing to the graph.
package lint

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/lint/port"
	"github.com/fogfish/arcnet-cli/internal/app/lint/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Lint validates the graph rooted at dir against the CORE §14 conformance
// checklist. It is a thin delegator into service.Lint.
func Lint(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, rules core.MergeRuleSet, dir string) (kernel.LintResult, error) {
	return service.Lint(ctx, mounter, vcs, reporter, rules, dir)
}
