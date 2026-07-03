//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import "gopkg.in/yaml.v3"

// ConfigPath is the path, relative to a graph root, where a graph's
// domain-specific merge-rule registrations live. Shared by
// internal/app/ctrl (seeding) and internal/app/config (load/save).
const ConfigPath = ".arc/config.yml"

// MergeRuleSet associates a node Kind with the MergeOp arc apply uses to
// reconcile an incoming contribution of that kind with an existing node.
type MergeRuleSet map[Kind]MergeOp

// CoreMergeRules is the graph format's own fixed kinds; always recognized,
// never requires registration (spec FR-018).
var CoreMergeRules = MergeRuleSet{
	"source":   MergeNone,
	"entity":   MergeUnion,
	"resource": MergeUnionFirstWriter,
	"timeline": MergeAppend,
}

// KnownProfileMergeRules are the two example domain profiles documented in
// github.com/fogfish/arcnet-spec — ready-made values a user copies into
// .arc/config.yml to opt in, never auto-registered.
var KnownProfileMergeRules = MergeRuleSet{
	"hypothesis": MergeValidatedOverwrite,
	"aporia":     MergeValidatedOverwrite,
	"thought":    MergeUnion,
}

func (m MergeRuleSet) MarshalYAML() (any, error) {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[string(k)] = string(v)
	}
	return out, nil
}

func (m *MergeRuleSet) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]string
	if err := value.Decode(&raw); err != nil {
		return err
	}

	out := make(MergeRuleSet, len(raw))
	for k, v := range raw {
		out[Kind(k)] = MergeOp(v)
	}
	*m = out
	return nil
}

// Union is a pure, non-mutating merge of two rule sets, m authoritative on
// conflict.
func (m MergeRuleSet) Union(other MergeRuleSet) MergeRuleSet {
	out := make(MergeRuleSet, len(m)+len(other))
	for k, v := range other {
		out[k] = v
	}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Lookup reports the MergeOp registered for kind. ok is false when kind is
// absent from the set — the condition internal/app/graph/service.Apply uses
// to decide "apply with the safe union default and warn" (research.md D5-revised).
func (m MergeRuleSet) Lookup(kind Kind) (op MergeOp, ok bool) {
	op, ok = m[kind]
	return op, ok
}
