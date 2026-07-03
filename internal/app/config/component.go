//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package config is the .arc/config.yml load/save/resolve/default domain
// use-case.
package config

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/config/port"
	"github.com/fogfish/arcnet-cli/internal/app/config/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Resolve returns the graph's effective merge-rule set. Thin delegator
// into service.Resolve.
func Resolve(store fsys.Store) (core.MergeRuleSet, error) {
	return service.Resolve(store)
}

// Save writes cfg to .arc/config.yml. Thin delegator into service.Save.
func Save(store fsys.Store, cfg kernel.Config) error {
	return service.Save(store, cfg)
}

// Default resolves the config-seed content arc init writes into a new
// graph. Thin delegator into service.Default.
func Default(ctx context.Context, fetcher port.Fetcher) (kernel.Config, bool) {
	return service.Default(ctx, fetcher)
}
