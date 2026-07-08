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

	for name, def := range kernel.CorePredicateDefs {
		raw, err := core.RenderNode(predicateNode(name, def))
		if err != nil {
			panic(err)
		}
		out[kernel.PredicatesDir+"/"+name+".md"] = raw
	}

	for name, def := range kernel.CoreTypeDefs {
		raw, err := core.RenderNode(typeNode(name, def))
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
	edges := make([]core.Link, 0, len(def.Required)+len(def.Optional))
	for _, predicate := range def.Required {
		edges = append(edges, core.Link{Predicate: "required", Target: predicate})
	}
	for _, predicate := range def.Optional {
		edges = append(edges, core.Link{Predicate: "optional", Target: predicate})
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

	defs := make(map[string]core.TypeDef, len(names))
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
		defs[node.ID] = def
	}
	return defs, nil
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

func decodeTypeDef(node core.Node) (core.TypeDef, string) {
	merge, ok := attrString(node, "merge")
	if !ok || !validMergeOps[core.MergeOp(merge)] {
		return core.TypeDef{}, "merge"
	}
	description := node.Texts[descriptionKey]
	if description == "" {
		return core.TypeDef{}, "description"
	}

	var required, optional []string
	for _, edge := range node.Edges {
		switch edge.Predicate {
		case "required":
			required = append(required, edge.Target)
		case "optional":
			optional = append(optional, edge.Target)
		}
	}

	return core.TypeDef{
		Merge:       core.MergeOp(merge),
		Required:    required,
		Optional:    optional,
		Description: description,
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

// RegisterPredicate creates predicate's predicate schema document — role:
// edge, merge: union, a placeholder description (research.md D5) — if one
// is not already present.
func RegisterPredicate(store fsys.Store, predicate string) (created bool, err error) {
	path := kernel.PredicatesDir + "/" + predicate + ".md"
	return registerIfAbsent(store, path, core.Node{
		ID:   predicate,
		Type: propertyType,
		Attrs: map[string][]core.Predicate{
			"role":  {{Value: "edge"}},
			"merge": {{Value: string(core.MergeUnion)}},
		},
		Texts: map[string]string{descriptionKey: autoRegisteredPredicateDescription},
	})
}

func registerIfAbsent(store fsys.Store, path string, node core.Node) (bool, error) {
	if _, err := store.Stat(path); err == nil {
		return false, nil
	}

	raw, err := core.RenderNode(node)
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

	return core.ParseNode(f)
}
