//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package ctrl is the graph management (control plane) domain use-case: it
// creates, and will later inspect and validate, a knowledge graph's local
// state.
package ctrl

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/port"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/service"
)

// Init bootstraps a new, empty knowledge graph at dir, seeding _schema/
// with schemaSeed. It is a thin delegator into service.Init.
func Init(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, dir string, schemaSeed map[string]string) (kernel.InitResult, error) {
	return service.Init(ctx, mounter, vcs, dir, schemaSeed)
}
