//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import (
	"fmt"
	"regexp"
	"strings"
)

// Filter is the optional, composable node-selection criteria shared by
// every VISION.md Filtering-section command (research.md D8 in
// specs/006-arc-grep-content-search). A zero-value Filter{} matches every
// node.
type Filter struct {
	// Types is empty-matches-every-type, otherwise OR'd: a node matches if
	// its Type is any listed value.
	Types []string
	// Tags is empty-matches-every-node, otherwise AND'd: every listed tag
	// must be present among the values of node.Attrs["tags"].
	Tags []string
	// Attrs is empty-matches-every-node, otherwise AND'd: name=value,
	// case-insensitive equality against any Predicate value in
	// node.Attrs[name].
	Attrs map[string]string
	// AttrPatterns is empty-matches-every-node, otherwise AND'd:
	// name~=pattern, regexp match against any Predicate value in
	// node.Attrs[name].
	AttrPatterns map[string]*regexp.Regexp
}

// Match reports whether node satisfies every condition in f. Match mutates
// neither f nor node.
func (f Filter) Match(node Node) bool {
	return f.matchTypes(node) && f.matchTags(node) && f.matchAttrs(node) && f.matchAttrPatterns(node)
}

func (f Filter) matchTypes(node Node) bool {
	if len(f.Types) == 0 {
		return true
	}
	for _, k := range f.Types {
		if node.Type == k {
			return true
		}
	}
	return false
}

func (f Filter) matchTags(node Node) bool {
	if len(f.Tags) == 0 {
		return true
	}
	tags := attrStrings(node.Attrs["tags"])
	for _, want := range f.Tags {
		if !containsFold(tags, want) {
			return false
		}
	}
	return true
}

func (f Filter) matchAttrs(node Node) bool {
	for name, want := range f.Attrs {
		if !matchAttrValue(node.Attrs[name], want) {
			return false
		}
	}
	return true
}

func (f Filter) matchAttrPatterns(node Node) bool {
	for name, want := range f.AttrPatterns {
		if !matchAttrPattern(node.Attrs[name], want) {
			return false
		}
	}
	return true
}

// attrStrings converts a Predicate slice into its stringified values,
// skipping any reference-valued Predicate (nil Value, non-empty Target).
func attrStrings(preds []Predicate) []string {
	out := make([]string, 0, len(preds))
	for _, p := range preds {
		if p.Value == nil {
			continue
		}
		out = append(out, toString(p.Value))
	}
	return out
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func containsFold(items []string, want string) bool {
	for _, item := range items {
		if strings.EqualFold(item, want) {
			return true
		}
	}
	return false
}

func matchAttrValue(preds []Predicate, want string) bool {
	for _, p := range preds {
		if p.Value == nil {
			continue
		}
		if strings.EqualFold(toString(p.Value), want) {
			return true
		}
	}
	return false
}

func matchAttrPattern(preds []Predicate, want *regexp.Regexp) bool {
	for _, p := range preds {
		if p.Value == nil {
			continue
		}
		if want.MatchString(toString(p.Value)) {
			return true
		}
	}
	return false
}
