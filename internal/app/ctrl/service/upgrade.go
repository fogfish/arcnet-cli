//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/port"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// migrateCommitSubject is the CORE §13.3 subject line identifying the one
// commit arc upgrade produces (contract C3.6).
const migrateCommitSubject = "graph(migrate): adopt ARCNET-CORE v0.11 built-in schema"

// schemaDir is the namespace prefix every built-in vocabulary document
// lives under. Named once here because it is also the boundary the
// prose-drift scan must NOT cross: schema documents are vocabulary, not
// content.
const schemaDir = "_schema"

// upgradePlan is planUpgrade's pure result: which built-in paths differ
// from what this release seeds, and the bytes to write. Every slice is
// sorted; writes is keyed by the same paths replaced/added name.
type upgradePlan struct {
	replaced []string
	added    []string
	removed  []string

	// writes holds the new content for every replaced/added path.
	writes map[string][]byte
	// originals holds the on-disk bytes of every path the plan will
	// overwrite or delete, so a failure part-way through can put the graph
	// back exactly as it was (contract C3.8).
	originals map[string][]byte
}

func (p upgradePlan) empty() bool {
	return len(p.replaced) == 0 && len(p.added) == 0 && len(p.removed) == 0
}

// planUpgrade byte-compares each seeded built-in document against its
// on-disk counterpart, classifying it as replaced, added, or unchanged, and
// separately collects every retired built-in still present.
//
// It NEVER decodes a schema document (contract C3.2 step 3). That is what
// lets arc upgrade run on a graph whose own scoreZ.md declares the retired
// seventh merge value: decodePredicateDef would reject it before anything
// else ran, and the command that exists to remove that declaration would be
// blocked by it (research.md D10, FR-022).
func planUpgrade(store fsys.Store, seed map[string][]byte, retired []string) (upgradePlan, error) {
	plan := upgradePlan{
		writes:    map[string][]byte{},
		originals: map[string][]byte{},
	}

	for path, want := range seed {
		got, found, err := readIfPresent(store, path)
		if err != nil {
			return upgradePlan{}, err
		}

		switch {
		case !found:
			plan.added = append(plan.added, path)
		case bytes.Equal(got, want):
			continue
		default:
			plan.replaced = append(plan.replaced, path)
			plan.originals[path] = got
		}
		plan.writes[path] = want
	}

	for _, path := range retired {
		if _, ok := seed[path]; ok {
			continue
		}
		got, found, err := readIfPresent(store, path)
		if err != nil {
			return upgradePlan{}, err
		}
		if !found {
			continue
		}
		plan.removed = append(plan.removed, path)
		plan.originals[path] = got
	}

	sort.Strings(plan.replaced)
	sort.Strings(plan.added)
	sort.Strings(plan.removed)
	return plan, nil
}

// readIfPresent reads path's bytes, reporting found=false (with no error)
// when it does not exist.
func readIfPresent(store fsys.Store, path string) (content []byte, found bool, err error) {
	f, err := store.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, ErrUpgradeRead.With(err, path)
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, false, ErrUpgradeRead.With(err, path)
	}
	return raw, true, nil
}

// applyUpgrade writes every replacement and addition and deletes every
// retired built-in. It touches nothing else: a file under _schema/ that is
// not part of the built-in set, and every file outside _schema/, is left
// exactly as it was (FR-019, FR-020, contract C3.3).
func applyUpgrade(store fsys.Store, plan upgradePlan) error {
	for _, path := range append(append([]string(nil), plan.replaced...), plan.added...) {
		if err := writeFile(store, path, string(plan.writes[path])); err != nil {
			return err
		}
	}
	for _, path := range plan.removed {
		if err := store.Remove(path); err != nil {
			return ErrUpgradeWrite.With(err, path)
		}
	}
	return nil
}

// rollbackUpgrade restores the graph to its pre-upgrade state: every
// overwritten or deleted document is written back verbatim, and every
// document the run newly added is removed. Mirrors service.Init's discipline
// (contract C3.8) — a failure leaves no partial state — but restores rather
// than only deleting, because unlike init, upgrade overwrites files that
// already held content.
func rollbackUpgrade(store fsys.Store, plan upgradePlan) {
	for path, original := range plan.originals {
		_ = writeFile(store, path, string(original))
	}
	for _, path := range plan.added {
		_ = store.Remove(path)
	}
}

// scanProseDrift reports every content node holding more than one paragraph
// in a text predicate the corrected vocabulary declares firstWriteWin — the
// shape only the previous append behaviour could have produced (research.md
// D12).
//
// Advisory and best-effort by design (FR-023, contract C3.7): the boundary
// between original text and accumulated text is unrecoverable, and
// mergeText's near-duplicate guard means the paragraphs are not necessarily
// even similar. Nodes are reported, never repaired, and the result never
// affects the exit code.
func scanProseDrift(store fsys.Store, index core.Index) ([]string, error) {
	paths, err := walkContentNodes(store)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, path := range paths {
		raw, found, err := readIfPresent(store, path)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		node, perr := core.ParseNode(bytes.NewReader(raw), index)
		if perr != nil {
			// A node this release cannot parse is a lint concern, not an
			// upgrade failure — the migration is about vocabulary, and
			// refusing to finish over unrelated malformed content would
			// leave the graph unloadable for no gain.
			continue
		}

		for _, predicate := range sortedTextKeys(node.Texts) {
			def, ok := index.Predicates[predicate]
			if !ok || def.Merge != core.MergeFirstWriteWin {
				continue
			}
			if len(core.SplitParagraphs(node.Texts[predicate])) > 1 {
				out = append(out, path)
				break
			}
		}
	}

	sort.Strings(out)
	return out, nil
}

func sortedTextKeys(texts map[string]string) []string {
	keys := make([]string, 0, len(texts))
	for k := range texts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// walkContentNodes lists every .md file that is graph content — excluding
// .arc/ (tool state) and _schema/ (vocabulary, not content). Mirrors
// lint/service.walkNodeFiles' own exclusions.
func walkContentNodes(store fsys.Store) ([]string, error) {
	var out []string

	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := store.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			full := e.Name()
			if dir != "." {
				full = dir + "/" + e.Name()
			}
			if e.IsDir() {
				if full == arcDir || full == schemaDir {
					continue
				}
				if err := walk(full); err != nil {
					return err
				}
				continue
			}
			if strings.HasSuffix(full, ".md") {
				out = append(out, full)
			}
		}
		return nil
	}

	if err := walk("."); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// Upgrade replaces a graph's built-in vocabulary documents with the ones
// this release seeds, in exactly one commit, leaving every other file
// untouched.
//
// The step order is the contract, not an implementation detail (C3.2): the
// corrected seed is WRITTEN BEFORE anything is RESOLVED. Tightening the
// merge menu to six makes every previously-seeded graph fail to load — its
// own scoreZ.md declares the retired seventh value — so a command that
// resolved first could never run on the graphs it exists to repair
// (research.md D10, FR-022).
//
// Nothing to do means no write, no commit, and an empty CommitHash (FR-021,
// FR-024): running it twice is indistinguishable from running it once.
// dryRun reports exactly what a real run would do and writes nothing (C3.5).
func Upgrade(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, resolver port.SchemaResolver, dir string, seed map[string][]byte, retired []string, dryRun bool) (kernel.UpgradeResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.UpgradeResult{}, err
	}

	// Step 1 — no schema read.
	if err := guardIsInitialized(store, dir); err != nil {
		return kernel.UpgradeResult{}, err
	}

	// Steps 2-3 — pure diff, no decode.
	plan, err := planUpgrade(store, seed, retired)
	if err != nil {
		return kernel.UpgradeResult{}, err
	}

	result := kernel.UpgradeResult{
		Root:        kernel.GraphRoot{Root: dir},
		Replaced:    plan.replaced,
		Added:       plan.added,
		Removed:     plan.removed,
		NeedsReview: []string{},
		DryRun:      dryRun,
	}

	if dryRun {
		review, err := scanProseDrift(store, resolveOrSeed(resolver, store))
		if err == nil {
			result.NeedsReview = review
		}
		return result, nil
	}

	if plan.empty() {
		// Still worth reporting drift: the vocabulary being current says
		// nothing about prose an earlier release already accumulated.
		if review, rerr := scanProseDrift(store, resolveOrSeed(resolver, store)); rerr == nil {
			result.NeedsReview = review
		}
		return result, nil
	}

	// Step 4 — write.
	if err := applyUpgrade(store, plan); err != nil {
		rollbackUpgrade(store, plan)
		return kernel.UpgradeResult{}, err
	}

	// Step 5 — NOW resolve. Seed() output is always resolvable, so a
	// failure here is a programming error rather than a user-facing schema
	// problem, and it surfaces as such (contract C3.8).
	index, err := resolver.Resolve(store)
	if err != nil {
		rollbackUpgrade(store, plan)
		return kernel.UpgradeResult{}, ErrUpgradeUnresolvable.With(err)
	}

	// Step 6 — advisory scan.
	review, err := scanProseDrift(store, index)
	if err != nil {
		rollbackUpgrade(store, plan)
		return kernel.UpgradeResult{}, err
	}
	result.NeedsReview = review

	// Step 7 — exactly one commit.
	if err := vcs.StageAll(ctx, dir); err != nil {
		rollbackUpgrade(store, plan)
		return kernel.UpgradeResult{}, err
	}

	hash, err := vcs.Commit(ctx, dir, upgradeCommitMessage(plan))
	if err != nil {
		rollbackUpgrade(store, plan)
		return kernel.UpgradeResult{}, err
	}
	result.CommitHash = hash

	return result, nil
}

// resolveOrSeed is the read-only path's tolerant resolve, used by --dry-run
// and by the already-current case. It prefers the graph's own vocabulary,
// which includes any predicate the author registered themselves, and falls
// back to the built-in index when the graph does not resolve.
//
// The fallback is what makes --dry-run honest on an UN-UPGRADED graph:
// such a graph cannot be resolved at all (its own scoreZ.md declares the
// retired seventh merge value), so without it the scan would find no
// firstWriteWin predicates, report no drift candidates, and then the real
// run — which resolves the corrected vocabulary — would report several
// (contract C3.5).
func resolveOrSeed(resolver port.SchemaResolver, store fsys.Store) core.Index {
	index, err := resolver.Resolve(store)
	if err != nil {
		return resolver.SeedIndex()
	}
	return index
}

// upgradeCommitMessage shapes the single migration commit per CORE §13.3:
// a subject naming the migration, and a body counting what moved.
func upgradeCommitMessage(plan upgradePlan) string {
	var body strings.Builder
	body.WriteString(migrateCommitSubject)
	body.WriteString("\n\n")
	body.WriteString(pluralizeDocuments("Replaced", len(plan.replaced)))
	body.WriteString(pluralizeDocuments(", added", len(plan.added)))
	if len(plan.removed) > 0 {
		body.WriteString(pluralizeDocuments(", removed", len(plan.removed)))
	}
	body.WriteString(".\n")
	return body.String()
}

func pluralizeDocuments(label string, n int) string {
	noun := " schema documents"
	if n == 1 {
		noun = " schema document"
	}
	return label + " " + strconv.Itoa(n) + noun
}

func guardIsInitialized(store fsys.Store, dir string) error {
	if _, err := store.Stat(arcDir); err != nil {
		return ErrNotInitialized.With(errNoCause, dir)
	}
	return nil
}
