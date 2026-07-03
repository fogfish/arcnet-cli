//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the config (.arc/config.yml) domain's value types.
package kernel

import "github.com/fogfish/arcnet-cli/internal/core"

// Config is the on-disk shape of .arc/config.yml.
type Config struct {
	MergeRules core.MergeRuleSet `yaml:"mergeRules"`
}
