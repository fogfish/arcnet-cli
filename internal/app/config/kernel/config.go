//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the config (.arc/config.yml) domain's value types.
package kernel

// ConfigPath is the path, relative to a graph root, where a graph's
// configuration lives.
const ConfigPath = ".arc/config.yml"

// Config is the on-disk shape of .arc/config.yml.
type Config struct {
	// Grep holds arc grep's performance/presentation knobs
	// (specs/006-arc-grep-content-search, research.md D10).
	Grep GrepConfig `yaml:"grep,omitempty"`
	// Subgraph holds arc subgraph's traversal-cap knobs
	// (specs/007-arc-subgraph, research.md D6).
	Subgraph SubgraphConfig `yaml:"subgraph,omitempty"`
}

// GrepConfig holds arc grep's configurable knobs. A zero/absent value
// (including an absent .arc/config.yml entirely) resolves to the built-in
// default at the cmd/ wiring layer, not inside this package — Load/Save
// stay a pure YAML round-trip.
type GrepConfig struct {
	// Workers bounds the concurrent scan pool size; <= 0 (including absent)
	// resolves to the built-in default 8.
	Workers int `yaml:"workers,omitempty"`
	// MaxLineWidth bounds the display width a matched line is ellipsis-fit
	// to on a color terminal; <= 0 (including absent) resolves to the
	// built-in default 80.
	MaxLineWidth int `yaml:"maxLineWidth,omitempty"`
}

// SubgraphConfig holds arc subgraph's configurable traversal caps. A
// zero/absent value (including an absent .arc/config.yml entirely)
// resolves to the built-in default at the cmd/ wiring layer, not inside
// this package — Load/Save stay a pure YAML round-trip (research.md D6).
type SubgraphConfig struct {
	// DirectCap bounds the "direct" (outgoing-reachable) pool; <= 0
	// (including absent) resolves to the built-in default 4096.
	DirectCap int `yaml:"directCap,omitempty"`
	// BacklinkCap bounds the "backlink" (incoming-reachable) pool; <= 0
	// (including absent) resolves to the built-in default 1024.
	BacklinkCap int `yaml:"backlinkCap,omitempty"`
}
