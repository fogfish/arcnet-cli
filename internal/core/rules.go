//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

// MergeRuleSet associates a node Type with the MergeOp arc apply uses to
// reconcile an incoming contribution of that type with an existing node.
type MergeRuleSet map[string]MergeOp

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
// to decide "apply with the safe union default and warn".
func (m MergeRuleSet) Lookup(kind string) (op MergeOp, ok bool) {
	op, ok = m[kind]
	return op, ok
}
