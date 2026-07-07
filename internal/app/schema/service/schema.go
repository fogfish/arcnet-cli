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
	"errors"
	"io/fs"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Seed renders every core kind's node-kind schema document and every core
// predicate's predicate schema document, keyed by their on-disk path. Pure:
// no I/O, no context.Context, no network call (research.md D5). Rendering a
// fixed, always-valid built-in value never fails in practice — a failure
// here would be a programming error, not a runtime condition callers need
// to handle.
func Seed() map[string][]byte {
	out := make(map[string][]byte, len(kernel.CoreMergeRules)+len(kernel.CorePredicates))

	for kind, op := range kernel.CoreMergeRules {
		node := core.Node{
			ID:   kind,
			Type: kernel.SchemaKind,
			Attrs: map[string][]core.Predicate{
				"merge": {{Value: string(op)}},
			},
		}
		if description := kernel.KindDescription(kind); description != "" {
			node.Texts = map[string]string{"text": description}
		}
		raw, err := core.RenderNode(node)
		if err != nil {
			panic(err)
		}
		out[kernel.NodesDir+"/"+kind+".md"] = raw
	}

	for predicate, description := range kernel.CorePredicates {
		node := core.Node{
			ID:   predicate,
			Type: kernel.SchemaKind,
		}
		if description != "" {
			node.Texts = map[string]string{"text": description}
		}
		raw, err := core.RenderNode(node)
		if err != nil {
			panic(err)
		}
		out[kernel.PredicatesDir+"/"+predicate+".md"] = raw
	}

	return out
}

// Resolve walks _schema/nodes/ and _schema/predicates/, parsing each file
// back into the graph's effective merge-rule set and registered predicate
// set. A file that fails to parse is skipped, not an error — that
// kind/predicate is simply absent from the returned set (spec.md Edge
// Cases). An absent _schema/ folder resolves to two empty results, not an
// error.
func Resolve(store fsys.Store) (core.MergeRuleSet, map[string]bool, error) {
	rules := core.MergeRuleSet{}
	predicates := map[string]bool{}

	entries, err := readDir(store, kernel.NodesDir)
	if err != nil {
		return nil, nil, err
	}
	for _, name := range entries {
		node, ok := parseNode(store, kernel.NodesDir+"/"+name)
		if !ok {
			continue
		}
		preds := node.Attrs["merge"]
		if len(preds) == 0 {
			continue
		}
		op, ok := preds[0].Value.(string)
		if !ok {
			continue
		}
		rules[node.ID] = core.MergeOp(op)
	}

	entries, err = readDir(store, kernel.PredicatesDir)
	if err != nil {
		return nil, nil, err
	}
	for _, name := range entries {
		node, ok := parseNode(store, kernel.PredicatesDir+"/"+name)
		if !ok {
			continue
		}
		predicates[node.ID] = true
	}

	return rules, predicates, nil
}

// RegisterKind creates kind's node-kind schema document, always with
// merge: union (spec FR-010, clarified), if one is not already present.
// created is false and no write happens when the file already exists (spec
// FR-011 — never overwrite).
func RegisterKind(store fsys.Store, kind string) (created bool, err error) {
	path := kernel.NodesDir + "/" + kind + ".md"
	return registerIfAbsent(store, path, core.Node{
		ID:   kind,
		Type: kernel.SchemaKind,
		Attrs: map[string][]core.Predicate{
			"merge": {{Value: string(core.MergeUnion)}},
		},
	})
}

// RegisterPredicate creates predicate's predicate schema document if one is
// not already present.
func RegisterPredicate(store fsys.Store, predicate string) (created bool, err error) {
	path := kernel.PredicatesDir + "/" + predicate + ".md"
	return registerIfAbsent(store, path, core.Node{
		ID:   predicate,
		Type: kernel.SchemaKind,
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

func readDir(store fsys.Store, dir string) ([]string, error) {
	entries, err := store.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
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

func parseNode(store fsys.Store, path string) (core.Node, bool) {
	f, err := store.Open(path)
	if err != nil {
		return core.Node{}, false
	}
	defer f.Close()

	node, err := core.ParseNode(f)
	if err != nil {
		return core.Node{}, false
	}
	return node, true
}
