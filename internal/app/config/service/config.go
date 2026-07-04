//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package service implements the config use-case's business logic.
package service

import (
	"errors"
	"io"
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/config/kernel"
)

// Load reads .arc/config.yml. An absent file is not an error — it returns
// the zero Config.
func Load(store fsys.Store) (kernel.Config, error) {
	f, err := store.Open(kernel.ConfigPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return kernel.Config{}, nil
		}
		return kernel.Config{}, err
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return kernel.Config{}, err
	}

	var cfg kernel.Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return kernel.Config{}, ErrConfigMalformed.With(err, kernel.ConfigPath)
	}
	return cfg, nil
}

// Save writes cfg back to .arc/config.yml as YAML.
func Save(store fsys.Store, cfg kernel.Config) error {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	f, err := store.Create(kernel.ConfigPath)
	if err != nil {
		return err
	}

	if _, err := f.Write(raw); err != nil {
		_ = f.Discard()
		return err
	}

	return f.Close()
}
