//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package service implements the schema use-case's business logic.
package service

import (
	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// propertyType/classType are CORE §9.1/§9.2's literal "@type" values for a
// predicate/type schema node.
const (
	propertyType = "Property"
	classType    = "Class"
)

// descriptionKey is the Texts key a Property/Class node's mandatory
// description prose decodes into. internal/core.ParseNode/RenderNode's
// textPredicateFor has no Property/Class case of its own (research.md D3 —
// this feature adds zero parser/renderer code to internal/core), so a
// Property/Class node's leading prose falls through to the default text
// key, "text" — this constant names that fact once rather than repeating
// the literal at every call site.
const descriptionKey = "text"

const (
	autoRegisteredPredicateDescription = "Auto-registered by arc apply; describe this predicate's meaning here."
	autoRegisteredTypeDescription      = "Auto-registered by arc apply; describe this type's meaning here."
)

var validRoles = map[string]bool{"meta": true, "text": true, "href": true, "edge": true, "link": true}

var validMergeOps = map[core.MergeOp]bool{
	core.MergeImmutable:          true,
	core.MergeUnion:              true,
	core.MergeFirstWriteWin:      true,
	core.MergeFillIfEmpty:        true,
	core.MergeLastWriteWin:       true,
	core.MergeAppend:             true,
	core.MergeValidatedOverwrite: true,
}

// Seed renders every CorePredicateDefs/CoreTypeDefs entry as a conformant
// Property/Class schema document, keyed by on-disk path. Pure: no I/O, no
// context.Context, no network call. Rendering a fixed, always-valid
// built-in value never fails in practice — a failure here would be a
// programming error, not a runtime condition callers need to handle.
func Seed() map[string][]byte {
	out := make(map[string][]byte, len(kernel.CorePredicateDefs)+len(kernel.CoreTypeDefs))
	index := core.Index{Predicates: kernel.CorePredicateDefs, Types: kernel.CoreTypeDefs}

	for name, def := range kernel.CorePredicateDefs {
		raw, err := core.RenderNode(predicateNode(name, def), index)
		if err != nil {
			panic(err)
		}
		out[kernel.PredicatesDir+"/"+name+".md"] = raw
	}

	for name, def := range kernel.CoreTypeDefs {
		raw, err := core.RenderNode(typeNode(name, def), index)
		if err != nil {
			panic(err)
		}
		out[kernel.TypesDir+"/"+name+".md"] = raw
	}

	return out
}

func predicateNode(name string, def core.PredicateDef) core.Node {
	attrs := map[string][]core.Predicate{
		"role":  {{Value: def.Role}},
		"merge": {{Value: string(def.Merge)}},
	}
	if def.Label != "" {
		attrs["label"] = []core.Predicate{{Value: def.Label}}
	}
	if def.Aligned != "" {
		attrs["aligned"] = []core.Predicate{{Value: def.Aligned}}
	}
	return core.Node{
		ID:    name,
		Type:  propertyType,
		Attrs: attrs,
		Texts: map[string]string{descriptionKey: def.Description},
	}
}

func typeNode(name string, def core.TypeDef) core.Node {
	bases := kernel.CoreTypeBases[name]
	edges := make([]core.Link, 0, len(def.Required)+len(def.Optional)+len(bases))
	for _, predicate := range def.Required {
		edges = append(edges, core.Link{Predicate: "required", Target: predicate})
	}
	for _, predicate := range def.Optional {
		edges = append(edges, core.Link{Predicate: "optional", Target: predicate})
	}
	for _, base := range bases {
		edges = append(edges, core.Link{Predicate: "subClassOf", Target: base})
	}
	return core.Node{
		ID:   name,
		Type: classType,
		Attrs: map[string][]core.Predicate{
			"merge": {{Value: string(def.Merge)}},
		},
		Texts: map[string]string{descriptionKey: def.Description},
		Edges: edges,
	}
}

// Resolve checks .arc/ presence first (research.md D2), returning
// ErrNotAGraph if absent; then walks _schema/predicates/ and
// _schema/types/, decoding each document into a PredicateDef/TypeDef. A
// missing schema folder, or any document missing/invalid role/merge/
// description, fails the entire load — never skipped (spec FR-014). Never
// returns a partially-populated Index.
func Resolve(store fsys.Store) (core.Index, error) {
	if _, err := store.Stat(".arc"); err != nil {
		return core.Index{}, ErrNotAGraph.With(err)
	}

	predicates, err := resolvePredicates(store)
	if err != nil {
		return core.Index{}, err
	}

	types, err := resolveTypes(store)
	if err != nil {
		return core.Index{}, err
	}

	return core.Index{Predicates: predicates, Types: types}, nil
}

func resolvePredicates(store fsys.Store) (map[string]core.PredicateDef, error) {
	names, err := readSchemaDir(store, kernel.PredicatesDir)
	if err != nil {
		return nil, err
	}

	defs := make(map[string]core.PredicateDef, len(names))
	for _, name := range names {
		path := kernel.PredicatesDir + "/" + name
		node, perr := parseNode(store, path)
		if perr != nil {
			return nil, ErrSchemaInvalid.With(perr, path, "document")
		}

		def, invalid := decodePredicateDef(node)
		if invalid != "" {
			return nil, ErrSchemaInvalid.With(errNoCause, path, invalid)
		}
		defs[node.ID] = def
	}
	return defs, nil
}

func resolveTypes(store fsys.Store) (map[string]core.TypeDef, error) {
	names, err := readSchemaDir(store, kernel.TypesDir)
	if err != nil {
		return nil, err
	}

	raw := make(map[string]rawType, len(names))
	for _, name := range names {
		path := kernel.TypesDir + "/" + name
		node, perr := parseNode(store, path)
		if perr != nil {
			return nil, ErrSchemaInvalid.With(perr, path, "document")
		}

		def, invalid := decodeTypeDef(node)
		if invalid != "" {
			return nil, ErrSchemaInvalid.With(errNoCause, path, invalid)
		}
		raw[node.ID] = def
	}
	return resolveEffectiveTypes(raw)
}

// rawType is the package-private, pre-inheritance-resolution decoding of a
// type schema document — never exported through core.TypeDef/core.Index
// (data-model.md "Raw type record").
type rawType struct {
	merge       core.MergeOp
	required    []string
	optional    []string
	subClassOf  []string
	description string
}

// implicitBaseExempt is excluded from Node's implicit universal-base rule
// (research.md D5): Node itself (nothing to inherit from itself), and the
// Property/Class schema meta-types, which describe predicate/type schema
// documents rather than content.
var implicitBaseExempt = map[string]bool{"Node": true, propertyType: true, classType: true}

// resolveEffectiveTypes flattens every rawType's own plus every
// (transitively resolved) rdfs:subClassOf base's Required/Optional into the
// effective core.TypeDef every consumer of core.Index.Types sees
// (data-model.md "Effective (inherited) contract"). Memoized recursive
// descent; an active-recursion-stack check detects a cycle of any length
// (including direct self-reference) the moment a type is revisited before
// its own resolution completes; a base name absent from raw is reported as
// an unresolved reference — including the implicit Node base, when a
// graph's _schema/types/ carries no Node.md of its own.
func resolveEffectiveTypes(raw map[string]rawType) (map[string]core.TypeDef, error) {
	resolved := make(map[string]core.TypeDef, len(raw))
	onStack := make(map[string]bool, len(raw))

	var resolve func(name string) (core.TypeDef, error)
	resolve = func(name string) (core.TypeDef, error) {
		if def, ok := resolved[name]; ok {
			return def, nil
		}
		if onStack[name] {
			return core.TypeDef{}, ErrSchemaCycle.With(errNoCause, name)
		}
		onStack[name] = true
		defer delete(onStack, name)

		rt := raw[name]
		bases := rt.subClassOf
		if !implicitBaseExempt[name] {
			bases = append(append([]string(nil), bases...), "Node")
		}

		required := appendMissing(nil, rt.required...)
		optional := appendMissing(nil, rt.optional...)

		seenBase := make(map[string]bool, len(bases))
		for _, base := range bases {
			if seenBase[base] {
				continue
			}
			seenBase[base] = true

			if _, ok := raw[base]; !ok {
				return core.TypeDef{}, ErrSchemaUnresolvedBase.With(errNoCause, name, base)
			}
			baseDef, err := resolve(base)
			if err != nil {
				return core.TypeDef{}, err
			}
			required = appendMissing(required, baseDef.Required...)
			optional = appendMissing(optional, baseDef.Optional...)
		}
		optional = removeAny(optional, required)

		def := core.TypeDef{Merge: rt.merge, Required: required, Optional: optional, Description: rt.description}
		resolved[name] = def
		return def, nil
	}

	for name := range raw {
		if _, err := resolve(name); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

// appendMissing appends each of values not already present in list,
// preserving list's existing order and values' own relative order.
func appendMissing(list []string, values ...string) []string {
	for _, v := range values {
		present := false
		for _, existing := range list {
			if existing == v {
				present = true
				break
			}
		}
		if !present {
			list = append(list, v)
		}
	}
	return list
}

// removeAny returns list with every element also present in exclude
// dropped, preserving list's own order.
func removeAny(list, exclude []string) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		excluded := false
		for _, e := range exclude {
			if v == e {
				excluded = true
				break
			}
		}
		if !excluded {
			out = append(out, v)
		}
	}
	return out
}

func decodePredicateDef(node core.Node) (core.PredicateDef, string) {
	role, ok := attrString(node, "role")
	if !ok || !validRoles[role] {
		return core.PredicateDef{}, "role"
	}
	merge, ok := attrString(node, "merge")
	if !ok || !validMergeOps[core.MergeOp(merge)] {
		return core.PredicateDef{}, "merge"
	}
	description := node.Texts[descriptionKey]
	if description == "" {
		return core.PredicateDef{}, "description"
	}

	label, _ := attrString(node, "label")
	aligned, _ := attrString(node, "aligned")
	return core.PredicateDef{
		Role:        role,
		Merge:       core.MergeOp(merge),
		Label:       label,
		Aligned:     aligned,
		Description: description,
	}, ""
}

// decodeTypeDef validates a Class node's shape. Unlike decodePredicateDef,
// it does not require "merge" to be present or valid (spec 012 FR-020,
// Bugfix 018/BUG-001): the whole-node merge field is no longer consulted by
// reconciliation (FR-015), so an absent or unrecognized value resolves to
// the zero-value MergeOp ("no whole-node merge declared") rather than
// failing validation — a real, CORE-conformant Class definition has no
// reason to carry a functionally inert field.
func decodeTypeDef(node core.Node) (rawType, string) {
	var merge core.MergeOp
	if raw, ok := attrString(node, "merge"); ok && validMergeOps[core.MergeOp(raw)] {
		merge = core.MergeOp(raw)
	}
	description := node.Texts[descriptionKey]
	if description == "" {
		return rawType{}, "description"
	}

	var required, optional, subClassOf []string
	for _, edge := range node.Edges {
		switch edge.Predicate {
		case "required":
			required = append(required, edge.Target)
		case "optional":
			optional = append(optional, edge.Target)
		case "subClassOf":
			subClassOf = append(subClassOf, edge.Target)
		}
	}

	return rawType{
		merge:       merge,
		required:    required,
		optional:    optional,
		subClassOf:  subClassOf,
		description: description,
	}, ""
}

func attrString(node core.Node, key string) (string, bool) {
	preds := node.Attrs[key]
	if len(preds) == 0 {
		return "", false
	}
	s, ok := preds[0].Value.(string)
	return s, ok
}

// RegisterType creates typ's type schema document — merge: union, empty
// Required/Optional, a placeholder description (research.md D5) — if one
// is not already present. created is false and no write happens when the
// file already exists (spec FR-011 — never overwrite).
func RegisterType(store fsys.Store, typ string) (created bool, err error) {
	path := kernel.TypesDir + "/" + typ + ".md"
	return registerIfAbsent(store, path, core.Node{
		ID:   typ,
		Type: classType,
		Attrs: map[string][]core.Predicate{
			"merge": {{Value: string(core.MergeUnion)}},
		},
		Texts: map[string]string{descriptionKey: autoRegisteredTypeDescription},
	})
}

// RegisterPredicate creates predicate's predicate schema document — role and
// merge default to edge/union, a placeholder description (research.md D5) —
// if one is not already present. observedRole (BUG-002, spec 010 FR-019)
// is the role the predicate was actually seen in when first discovered: a
// predicate observed as "text" (non-wikilink body content) defaults instead
// to role: text, merge: append, closing spec 011 research.md's own flagged
// gap ("today's auto-discovery path only ever observes edge-position
// predicates"); a predicate observed as "link" (an edge occurrence carried
// with its own "**Label**" block, BUG-003 spec 010 FR-022) registers
// role: link, merge: union, so it renders grouped under its own heading
// instead of collapsing into the flat edge list. Any other observedRole
// (including "edge") keeps the original edge/union default. label, when
// non-empty (BUG-003, spec 010 FR-021), is stored as the document's own
// `label` attribute, so the predicate's exact original heading/bold-label
// text is recoverable on a later render instead of a derived-id
// approximation.
func RegisterPredicate(store fsys.Store, predicate, observedRole, label string) (created bool, err error) {
	role, merge := "edge", core.MergeUnion
	switch observedRole {
	case "text":
		role, merge = "text", core.MergeAppend
	case "link":
		role = "link"
	}

	attrs := map[string][]core.Predicate{
		"role":  {{Value: role}},
		"merge": {{Value: string(merge)}},
	}
	if label != "" {
		attrs["label"] = []core.Predicate{{Value: label}}
	}

	path := kernel.PredicatesDir + "/" + predicate + ".md"
	return registerIfAbsent(store, path, core.Node{
		ID:    predicate,
		Type:  propertyType,
		Attrs: attrs,
		Texts: map[string]string{descriptionKey: autoRegisteredPredicateDescription},
	})
}

func registerIfAbsent(store fsys.Store, path string, node core.Node) (bool, error) {
	if _, err := store.Stat(path); err == nil {
		return false, nil
	}

	// core.Index{} is safe here: node's Edges is always nil at both call
	// sites (RegisterType/RegisterPredicate), so the role-partitioning path
	// is never reached (research.md D6).
	raw, err := core.RenderNode(node, core.Index{})
	if err != nil {
		return false, ErrSchemaWrite.With(err, path)
	}

	f, err := store.Create(path)
	if err != nil {
		return false, ErrSchemaWrite.With(err, path)
	}
	if _, err := f.Write(raw); err != nil {
		_ = f.Discard()
		return false, ErrSchemaWrite.With(err, path)
	}
	if err := f.Close(); err != nil {
		return false, ErrSchemaWrite.With(err, path)
	}

	return true, nil
}

func readSchemaDir(store fsys.Store, dir string) ([]string, error) {
	entries, err := store.ReadDir(dir)
	if err != nil {
		return nil, ErrSchemaMissing.With(err, dir)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

func parseNode(store fsys.Store, path string) (core.Node, error) {
	f, err := store.Open(path)
	if err != nil {
		return core.Node{}, err
	}
	defer f.Close()

	return core.ParseNode(f, core.Index{})
}
