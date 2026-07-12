# Research: `arc revert`

## D1 — Locating the ingest commit (FR-001/FR-002)

**Decision**: Reuse `CommitsMatching(ctx, dir, needle string) ([]string, error)` — already implemented by the shared `internal/adapter/git.VCS` type (`internal/adapter/git/git.go:154`, `git log --all --fixed-strings --grep=<needle> --format=%H`) and already exposed on `internal/app/lint/port.VCS` for exactly this purpose (`internal/app/lint/service/rules_history.go`'s `checkIngestCommit`, spec 003 FR-012's "exactly one ingest commit per `Source-Id`" invariant). `arc revert` adds the same method signature to `internal/app/graph/port.VCS` — no adapter code changes, since `git.VCS` already satisfies it structurally (ADR 001 port isolation rule 1). Called with `needle = "Source-Id: " + sourceID`.

**Rationale**: A second, divergent implementation of the same git primitive would violate constitution Principle VII's "before adding a new adapter, verify whether one exists." Widening a port's method set to expose an adapter capability a sibling port already uses is exactly the isolation rule ADR 001 exists for.

**Alternatives considered**: Parsing `sources/<id>.md`'s own git history directly (`git log --follow`) and taking its earliest commit — rejected because a source node's earliest touching commit is provably its own ingest commit only if the file was never independently created another way; `--grep` on the mandatory trailer is the format's own documented identity marker (spec 003 FR-012) and needs no such inference.

**Edge case**: zero matches → FR-002 refuse, "no ingest commit found for `<source-id>`." More than one match is an integrity anomaly the format guarantees cannot happen (spec 003 FR-011: exactly one commit per successful apply) — treated as a hard refuse with a message pointing at `arc lint`'s `RuleIngestCommit` check, not a guess at which commit is "the" one.

**Trailer collision guard**: this invariant only holds if `arc revert`'s own commit never contains the literal substring `Source-Id: ` in its message — `CommitsMatching`'s `--grep` is a plain substring test, not a trailer-key-exact match, so a same-prefixed trailer (e.g. a hypothetical `Reverted-Source-Id:`) would still satisfy it and reintroduce the "more than one match" anomaly the moment a retracted document is later re-applied. See data-model.md's commit-message section: the per-node revert commit uses `Reverted-Document:` instead, chosen specifically to share no substring with `Source-Id:`.

## D2 — "Already retracted" detection (Clarifications, Session 2026-07-12)

**Decision**: `vcs.IsTracked(ctx, dir, "sources/"+sourceID+".md")` — the exact primitive `service.Apply` already calls for its own idempotency check (`internal/app/graph/service/apply.go:180`) — inverted: `false` means already retracted (FR-003). No new adapter method.

**Rationale**: A document's own source node is always exclusively created by its own patch (no other patch ever writes to another document's source record — enforced by `nodePath`/`patch.Document` identity), so its continued absence is a reliable, already-proven signal, reused rather than reinvented (spec's own Assumptions section, Clarifications).

## D3 — Whole-commit-reversal eligibility (FR-005/FR-006, collapsed)

**Decision**: One new port primitive, `ChangedPaths(ctx, dir, hash string) ([]string, error)` (`git diff-tree --no-commit-id --name-only -r <hash>`), lists every path the ingest commit touched. A second, `CommitsTouching(ctx, dir, path string) ([]string, error)` (`git log --follow --format=%H -- <path>`, newest-first), lists every commit that ever changed a given path. The whole operation is eligible for a single whole-commit undo **iff, for every path in `ChangedPaths(ingestHash)`, `CommitsTouching(path)[0] == ingestHash`** — i.e. the ingest commit is still the most recent commit to touch every file it touched.

**Rationale**: This single per-path predicate subsumes both of the spec's stated cases (FR-005 "is literally HEAD," FR-006 "nothing since has touched its files") without special-casing "is this HEAD" separately, and generalizes correctly to a patch that merged into a pre-existing file — that file's own earlier history doesn't disqualify eligibility, only a *later* touch does.

**Alternatives considered**: Comparing `ingestHash` to `git rev-parse HEAD` directly for the FR-005 case, then a separate `git log <hash>..HEAD -- <paths>` emptiness check for FR-006 — rejected as two primitives and two code paths doing what one `CommitsTouching` lookup already answers per path, and the two-primitive version doesn't naturally extend to D5's per-node granularity below (one extra primitive, not reused).

## D4 — Whole-commit reversal mechanism (FR-005/FR-006 satisfied path)

**Decision**: New primitive `RevertCommit(ctx, dir, hash string) (newHash string, err error)`, mapping to `git revert --no-edit <hash>`.

**Rationale**: D3's eligibility test guarantees nothing has touched the affected files since `ingestHash`, which is exactly the precondition under which git's own three-way revert is correct by construction — reusing git's native primitive is strictly less code than reimplementing inverse-diff application, and produces exactly one commit (FR-015) with no risk of arc's own reconciliation logic disagreeing with git's.

**Consequence**: this path's commit message is git's own generated `Revert "<original subject>"` / `This reverts commit <hash>.` — the spec does not require a bespoke subject format for this path (contrast `apply`'s mandatory `buildCommitMessage`), so none is imposed.

## D5 — Per-node exclusivity test (FR-008/FR-009)

**Decision**: Reuse D3's `CommitsTouching(path)` at per-node granularity: a node is exclusively owned by the reverted patch iff `len(CommitsTouching(path)) == 1` (its one entry is necessarily `ingestHash`, since `ChangedPaths(ingestHash)` already established the ingest commit touched it).

**Rationale**: No new primitive — D3's primitive answers both the whole-operation eligibility question and this finer-grained one, avoiding a second git-log code path for what is structurally the same question asked at a different scope.

## D6 — Exclusive-node removal and backlink sweep (FR-009/FR-010)

**Decision**: Reuse `internal/app/graph/service/subgraph.go`'s existing unexported `enumerateNodes`/`nodeIndex`/`buildReverseIndex` helpers (already walk every `*.md` node file via the shared `walkNodeFiles`, spec 007's own backlink machinery) rather than a second graph-walking implementation. Node removal is `store.Remove(path)` (mirrors `apply.go`'s own `rollback` helper, `internal/app/graph/service/apply.go:467`). For every id in `reverseIndex[removedID]`, the referrer is re-read and its `Edges` filtered to drop the link whose `Target == removedID`.

**Load-bearing discovery**: this reverse index already covers timeline-period backlinks with **no extra code**. `core.TimelineEntry` (`internal/core/timeline.go:30`) renders every timeline bullet as a `cites`-predicate edge (`- cites:: [[id]] — ...`), and `core.ParseNode`'s general `listItemPattern` (`internal/core/markdown.go:665`) recognizes that predicate-tagged wikilink bullet the same as any other node's `Edges` — so a timeline period file parses back into an ordinary node carrying one `cites` edge per listed source. `buildReverseIndex` therefore finds a removed source's timeline backlinks the same way it finds an ordinary node's, with zero format-specific detection code in `arc revert`.

**Rewrite path diverges by referrer kind**: an ordinary node is rewritten via `core.RenderNode` (its existing, canonical writer). A `@type: timeline` referrer is **not** — `apply.go`'s own per-node loop explicitly diverts `node.Type == "timeline"` away from the generic `writeNode`/`core.RenderNode` path into its own hand-rolled `upsertTimelinePeriod` writer (`internal/app/graph/service/apply.go:635`), and that boundary is deliberate (BUG-007's fix: `core.RenderNode`'s generic Attrs encoding doesn't preserve `period`'s forced string-quoting the way `upsertTimelinePeriod` does by hand). `arc revert` adds a structural sibling, `removeTimelineEntry`, reusing `parseTimelineEntries` (`apply.go:522`) minus the retracted entry, re-serialized with the exact same front-matter/heading shape `upsertTimelinePeriod` already produces — preserving the existing boundary rather than introducing a second, divergent writer for the same file shape.

**Alternatives considered**: A raw full-text grep for `[[<id>` across the graph (the mechanism the planning notes originally floated, `service.Grep`) — rejected once the `cites`-edge discovery above showed the existing Edges-based reverse index already answers this correctly and more precisely (a raw substring match could false-positive inside an unrelated code block or a differently-shaped mention; the parsed-Edges index cannot).

## D7 — Shared-node text-block identification (FR-012)

**Decision**: New primitive `Blame(ctx, dir, path string) ([]port.BlameLine, error)` (`git blame --line-porcelain HEAD -- <path>`), returning one `{Number int; Commit string}` per current line — content itself is not needed, only attribution. `renderNodeBody`'s output order is deterministic and already load-bearing for round-tripping (`internal/core/markdown.go:754-767`'s documented physical-layout contract: leading Texts key, other Texts keys alphabetically sorted, Edges, trailing Texts key). `arc revert` walks the same sequence over the already-parsed `core.Node` to build a `line number → (Texts key, paragraph index)` map covering exactly the byte range `renderNodeBody` produces for that node (offset by the front-matter block and the `# <ID>` heading line `RenderNode` prepends), then intersects it with `Blame`'s `ingestHash`-attributed lines.

**Scope guard (FR-011)**: only lines falling inside a Texts-key's paragraph range participate. Lines attributed to `ingestHash` inside the front-matter block or the Edges section are never touched, even when blame implicates them — the spec's explicit "leave metadata attributes, edges and links unchanged" instruction for a shared node.

**Removal**: a matched paragraph is dropped from that Texts key's value via `strings.Split` on `"\n\n"` (mirroring `internal/core/merge.go`'s existing `splitParagraphs`/`mergeParagraphs` paragraph model, `internal/core/merge.go:220`) and the node rewritten via `core.RenderNode`.

## D8 — Conflict-marker provenance (FR-013)

**Decision**: before D7's blame-based stripping is attempted on a Texts value, check whether it matches `conflictMarker`'s exact format (`internal/core/merge.go:256`: `<<<<<<< existing\n...\n=======\n...\n>>>>>>> <sourceID>`). Blame's line attribution is unreliable for a marker — the whole block is one commit's diff (the later contributor's), regardless of which side's text is chronologically older (the planning notes' own finding). Two sub-cases, resolved without a blame call:

- **(a) reverted patch is the marker's own incoming side**: the trailing `>>>>>>> <sourceID>` token is compared, as plain text already on disk, against the reverted patch's own `sourceID` — no git call needed, since `conflictMarker` already self-documents this side's author. On match, the marker is replaced by its own "existing" (left) side's plain text.
- **(b) reverted patch is the marker's frozen existing side**: resolved by walking `CommitsTouching(path)` oldest-first, reading each historical revision via a new primitive `ShowFile(ctx, dir, hash, path string) ([]byte, error)` (`git show <hash>:<path>`), parsing it with the existing `core.ParseNode`, and finding the first commit at which this predicate held a non-empty value. Every scalar behavior a conflict marker can appear under (`firstWriteWin`/`fillIfEmpty`, per spec 012's `mergeScalar`) freezes a predicate's value at its first writer forever, so that first commit is unambiguously the "existing" side's true author. If it is `ingestHash`, the marker is replaced by its own "incoming" (right) side's plain text — promoting the later contribution now that the frozen original is being retracted.
- Neither (a) nor (b): the reverted patch made no contribution to this specific predicate; it is left untouched.

**Rationale**: this is exactly the case the planning notes flagged as blame's blind spot ("its original value is only preserved as literal text embedded inside a block that blame credits to someone else... that provenance has to be reconstructed from the marker text itself") — reconstructed here via (a) the marker's own self-documented `sourceID` suffix, or (b) a bounded historical walk, never by blame alone.

## D9 — No-further-attribution case (FR-014)

**Decision**: no new mechanism. If D7/D8 find no paragraph and no marker-side attributable to `ingestHash` on a shared node, the node is left unmodified — not an error.

**Rationale**: mirrors `core.Merge`'s own silent-absorption precedent for a Jaccard-deduplicated paragraph (`internal/core/merge.go`'s `paragraphAlreadyPresent`) — the same dedup that made a contribution non-attributable in the first place means there is nothing on disk to remove.

## D10 — Confirmation gate (new UX pattern; ADR 002 / Constitution Principle IX)

**Decision**: `arc revert` is the first command in this codebase whose default behavior deletes a tracked file — no existing confirmation helper exists to reuse (`grep -rn "Confirm\|--yes\|--force" cmd/ internal/bios/` returns nothing). A small `bios.Confirm(prompt string) (bool, error)` is added to `internal/bios` (the existing shared UX package, already home to `Reporter`/`Registry`/theme per DS-04/DS-06): TTY-gated — prompts and reads `y`/`N` from stdin when `os.Stdout` is a terminal, otherwise refuses with a clear error unless `--force` was passed. Per the constitution's own reserved-shorthand table (`-f` already named "file/force"), the bypass flag is `--force`/`-f`, not `--yes`/`-y`.

**Rationale**: Constitution Principle IX ("Destructive or irreversible operations MUST require explicit confirmation, or an explicit `--yes`/`--force` flag for non-interactive use") and ADR 002's CLIG checklist item are both binding and, until this feature, unimplemented anywhere — this is a gap being closed, not a deviation from an established pattern. Placing the helper in `internal/bios` rather than inline in `cmd/arc/graph/revert.go` means the next destructive command (a hypothetical `arc prune`, etc.) inherits it for free (Principle V, no duplicated logic).

**Blast radius**: a single, non-repeated `y/N` prompt — Principle IX's "confirmation rigor MUST scale with the blast radius" places a single-patch revert at the "single-resource delete" end of that scale, not the heavier "bulk/recursive" end.

## D11 — Test doubles

**Decision**: `internal/app/graph/adapter/mock/mock.go`'s `VCS` fake gains the five new methods (`CommitsMatching`, `ChangedPaths`, `CommitsTouching`, `RevertCommit`, `Blame`, `ShowFile`), each with a configurable return value/error and appended to the existing `Calls` log, mirroring the shape of every existing method there. `internal/adapter/git/git.go`'s real `VCS` type gains the concrete `os/exec` implementations, each gated behind a new `faults.Type` sentinel (`ErrGitDiffTree`, `ErrGitRevert`, `ErrGitBlame`, `ErrGitShow`), mirroring the existing `ErrGitLog`/`ErrGitLsFiles` pattern exactly.

## D12 — Command and package placement

**Decision** (per plan input): `arc revert` lives inside the existing `graph` domain, not a new domain package — `internal/app/graph/component.go` gains a thin `Revert` delegator (mirroring `Apply`), `internal/app/graph/service/revert.go` holds the business logic, `internal/app/graph/kernel/revert.go` holds `RevertResult`, `internal/app/graph/port/vcs.go` gains the new methods (D1/D3/D4/D7/D8), and `cmd/arc/graph/revert.go` provides the Cobra wiring — matching `apply.go`'s existing four-file shape in both trees exactly.

## Summary of blast radius

| File | Change |
|---|---|
| `internal/app/graph/port/vcs.go` | +6 method signatures (`CommitsMatching`, `ChangedPaths`, `CommitsTouching`, `RevertCommit`, `Blame`, `ShowFile`) + `BlameLine` type |
| `internal/adapter/git/git.go` | +5 new methods (`CommitsMatching` already exists structurally via the shared type — only the port needs it added), +4 error sentinels |
| `internal/app/graph/adapter/mock/mock.go` | +5 configurable fields/methods on the fake `VCS` |
| `internal/app/graph/service/revert.go` (new) | reconciliation algorithm (D1–D9) |
| `internal/app/graph/service/apply.go` | `removeTimelineEntry` sibling to `upsertTimelinePeriod`; `enumerateNodes`/`buildReverseIndex` reused as-is from `subgraph.go`, no change needed there |
| `internal/app/graph/kernel/revert.go` (new) | `RevertResult` |
| `internal/app/graph/component.go` | +`Revert` delegator |
| `internal/bios/confirm.go` (new) | `Confirm` helper (D10) |
| `cmd/arc/graph/revert.go` (new) | Cobra wiring, `--force`/`-f` |
| `ARCHITECTURE.md` | Glossary: "Ingest Commit", "Exclusively-Owned Node", "Shared Node" entries added (constitution Principle II) |
