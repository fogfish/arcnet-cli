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
	// Kinds is empty-matches-every-kind, otherwise OR'd: a node matches if
	// its Kind is any listed value.
	Kinds []Kind
	// Tags is empty-matches-every-node, otherwise AND'd: every listed tag
	// must be present in node.Attrs["tags"].
	Tags []string
	// Attrs is empty-matches-every-node, otherwise AND'd: name=value,
	// case-insensitive equality for a scalar attribute, membership test for
	// an array attribute.
	Attrs map[string]string
	// AttrPatterns is empty-matches-every-node, otherwise AND'd:
	// name~=pattern, regexp match against a scalar, or against any element
	// of an array attribute.
	AttrPatterns map[string]*regexp.Regexp
}

// Match reports whether node satisfies every condition in f. Match mutates
// neither f nor node.
func (f Filter) Match(node Node) bool {
	return f.matchKinds(node) && f.matchTags(node) && f.matchAttrs(node) && f.matchAttrPatterns(node)
}

func (f Filter) matchKinds(node Node) bool {
	if len(f.Kinds) == 0 {
		return true
	}
	for _, k := range f.Kinds {
		if node.Kind == k {
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

// attrStrings normalizes a front-matter value into a string slice: a scalar
// becomes a single-element slice, an array becomes its stringified
// elements, anything else (including nil) becomes empty.
func attrStrings(v any) []string {
	switch val := v.(type) {
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			out = append(out, toString(item))
		}
		return out
	case nil:
		return nil
	default:
		return []string{toString(val)}
	}
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

func matchAttrValue(v any, want string) bool {
	switch val := v.(type) {
	case []any:
		for _, item := range val {
			if strings.EqualFold(toString(item), want) {
				return true
			}
		}
		return false
	case nil:
		return false
	default:
		return strings.EqualFold(toString(val), want)
	}
}

func matchAttrPattern(v any, want *regexp.Regexp) bool {
	switch val := v.(type) {
	case []any:
		for _, item := range val {
			if want.MatchString(toString(item)) {
				return true
			}
		}
		return false
	case nil:
		return false
	default:
		return want.MatchString(toString(val))
	}
}
