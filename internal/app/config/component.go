//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package config is the .arc/config.yml load/save domain use-case.
package config

import (
	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/config/service"
)

// Load reads .arc/config.yml. Thin delegator into service.Load.
func Load(store fsys.Store) (kernel.Config, error) {
	return service.Load(store)
}

// Save writes cfg to .arc/config.yml. Thin delegator into service.Save.
func Save(store fsys.Store, cfg kernel.Config) error {
	return service.Save(store, cfg)
}
