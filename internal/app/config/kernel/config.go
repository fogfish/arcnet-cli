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

// Config is the on-disk shape of .arc/config.yml. Empty until a future,
// unrelated configuration need arrives — this feature's own schema data
// lives under _schema/, not here (research.md D8).
type Config struct{}
