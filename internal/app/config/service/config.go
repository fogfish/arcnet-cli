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
	"context"
	"errors"
	"io"
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/config/port"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// DefaultSourceURL is the canonical, public, unauthenticated location
// Default fetches a graph's config seed from (research.md D5 revised).
const DefaultSourceURL = "https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/config.yml"

// Load reads .arc/config.yml. An absent file is not an error — it returns
// the zero Config (spec User Story 3 Acceptance Scenario 2, "no domain
// kinds registered").
func Load(store fsys.Store) (kernel.Config, error) {
	f, err := store.Open(core.ConfigPath)
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
		return kernel.Config{}, ErrConfigMalformed.With(err, core.ConfigPath)
	}
	return cfg, nil
}

// Save writes cfg back to .arc/config.yml as YAML.
func Save(store fsys.Store, cfg kernel.Config) error {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	f, err := store.Create(core.ConfigPath)
	if err != nil {
		return err
	}

	if _, err := f.Write(raw); err != nil {
		_ = f.Discard()
		return err
	}

	return f.Close()
}

// Resolve returns the graph's effective merge-rule set: the format's fixed
// kinds (core.CoreMergeRules) unioned with whatever .arc/config.yml
// additionally registers. An absent file resolves to core.CoreMergeRules
// alone.
func Resolve(store fsys.Store) (core.MergeRuleSet, error) {
	cfg, err := Load(store)
	if err != nil {
		return nil, err
	}
	if cfg.MergeRules == nil {
		return core.CoreMergeRules, nil
	}
	return core.CoreMergeRules.Union(cfg.MergeRules), nil
}

// Default resolves the seed content arc init writes into a new graph's
// .arc/config.yml: one fetch attempt against fetcher, falling back to
// core.CoreMergeRules on any failure whatsoever (network error, non-2xx,
// timeout, malformed payload) — never an error return, by construction,
// guaranteeing specs/002-arc-init/spec.md FR-017's "initialization MUST
// NOT fail... on this basis alone" (research.md D5 revised).
func Default(ctx context.Context, fetcher port.Fetcher) (cfg kernel.Config, usedFallback bool) {
	fallback := kernel.Config{MergeRules: core.CoreMergeRules}

	raw, err := fetcher.Fetch(ctx, DefaultSourceURL)
	if err != nil {
		return fallback, true
	}

	var fetched kernel.Config
	if err := yaml.Unmarshal(raw, &fetched); err != nil || fetched.MergeRules == nil {
		return fallback, true
	}

	return fetched, false
}
