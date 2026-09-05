# Implementation Plan: ARCNET-CORE v0.11 — Schema Vocabulary Conformance

**Branch**: `023-core-vocabulary-conformance` | **Date**: 2026-08-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/023-core-vocabulary-conformance/spec.md`

**Upstream authority**: [ARCNET-CORE.md](https://github.com/fogfish/arcnet-spec/blob/main/ARCNET-CORE.md)
— fetched and validated entry-by-entry on 2026-08-23. **Status: Draft · Version: 0.11.**

**Bugfix**: 2026-08-23 — [BUG-001](bugs/BUG-001.md). Upstream moved again, 0.11 → 0.12, retiring
or renaming six of the seven predicates this plan's F4 table and the Assumptions section (spec.md)
settled in the tool's favour. See new "F7" below and the added Open Item for `/speckit-clarify`.

**Bugfix**: 2026-09-05 — [BUG-002](bugs/BUG-002.md). F7's own `relevance` row above claimed the
0.11→0.12 revision note only dropped `relevance` "from *prose*, never from an actual
Requires/Optional list" — wrong; the same revision note separately retires it outright, and
`Reference`'s current worked example's Optional list confirms it (no `relevance`, no `notes`). Row
corrected below; companion fix to spec 010's own [BUG-007](../010-predicate-node-model/bugs/BUG-007.md),
which retires the `textPredicateFor` trailing-slot design this vocabulary retirement would
otherwise leave half-fixed.

## Summary

Correct the built-in predicate and type vocabulary `arc init` seeds so a new graph is conformant
with ARCNET-CORE v0.11 from its first commit.

Two tables change — `CorePredicateDefs` (15 corrections, 3 additions) and `CoreTypeDefs` (all 8
lose the retired type-level `merge`; 5 change their predicate contract) — plus the deletion of a
seventh merge operation from `core.MergeOp`. Every type contract change is a strict relaxation,
so no graph that lints clean today can start failing.

> **Post-implementation note (2026-08-23).** This plan originally scoped a fifth user story and a
> new `arc upgrade` command giving an existing graph seeded by a previous release an explicit
> migration path onto the corrected vocabulary (see "the migration is ordered so it can run on a
> graph whose schema no longer validates" below, research D10–D12, and contract C3 in
> `contracts/upgrade-command-contract.md`). That command was implemented, then **removed** by
> explicit decision: `arc` is pre-1.0/experimental, and the compatibility/migration machinery it
> required — tolerating a retired merge value long enough to remediate it, ordering writes before
> reads to escape the resulting deadlock, reporting prose drift — was judged unnecessary tech
> debt at this stage. The sections below that describe `arc upgrade`, US5, Phase 7/8 sequencing,
> and the migration deadlock are retained as a historical record of the reasoning that produced
> and then reversed that design; they no longer describe shipped behaviour. See `spec.md`'s
> User Story 5 (marked REMOVED) and `ARCHITECTURE.md`'s Compatibility Policy for the current,
> load-bearing statement: the merge-vocabulary closure is a plain breaking change, with no
> remedy path, accepted because the tool has no compatibility guarantee yet.

## Validation Findings — What Changed Against the Spec

The user's plan input directed validation against the live upstream document. That validation
**materially changed the feature**. Six findings, in descending order of consequence:

### F1. Upstream is v0.11, not v0.10 — and it fixed three of our workarounds

`spec.md`, `CORE-FIX.md`, and the branch name all say v0.10. The published document is **v0.11,
dated 2026-08-23**. Feature 022 already targeted v0.11, so the repo was already straddling both.

Three `CORE-FIX.md` §8 "upstream defects requiring a decision" are now settled upstream and
their workarounds must be withdrawn: §10.7 exists (§8.1); `author` *and* `authors` are both
registered, so "register both" is no longer a hedge but a requirement (§8.2); and `Reference`'s
three-way contradiction is resolved by a rewritten §11.6 (§8.3 — see F3). Section-number
remapping for every citation carried forward is in [research.md D0](research.md).

### F2. Two of `spec.md`'s premises are factually wrong

- **US1** claims re-applying a patch "concatenates the summary onto itself" and "a third apply
  triples it". It does not. `mergeText` (BUG-004) already drops any incoming paragraph whose
  Jaccard similarity over 3-word shingles exceeds 0.8, so a byte-identical re-apply is **already**
  a no-op. The real defect is that **reworded** prose accumulates — a slow drift, not a doubling.
  US1's acceptance scenarios 1 and 2 pass against `main` today and would have certified a fix
  that was never made (research D4).
- **US4** claims first use of `author`/`about`/`genre` "silently creates a definition with a role
  and a merge behaviour the tool guessed". It does not. `distinctPredicates`
  (`internal/app/graph/service/apply.go:113`) walks `node.Edges` and `node.Texts` only, never
  `node.Attrs` — and all three are `role: meta`. Nothing is registered, nothing is guessed; the
  merge falls through to the `union` default, which is coincidentally correct for all three. The
  real symptom is two lint violations per occurrence (research D5).

Both stories have been rewritten in `spec.md` against the real behaviour. The corrections do not
remove any requirement — §10.2/§10.8 declare the merge operations and the registrations
regardless — they correct *why* and *how it is tested*.

### F3. `Reference` — v0.11 reverses spec 022's recorded Clarification

Spec 022 clarified, on the strength of v0.10's 0.9→0.10 revision note, that `Reference` requires
`title`, `ref`, `relevance`. **v0.11 §11.6's normative `Class` block requires `title` alone** and
the revision note is gone.

This plan adopts v0.11: `Requires: title`, with `ref`, `relevance`, `status`, `notes` retained as
**Optional** so §11.6's own worked example (which carries `ref` and `status`) conforms to the type
it illustrates, and so spec 022's body-prose keying (`Reference` leading prose → `relevance`)
survives untouched. The change is a pure relaxation.

**This supersedes a recorded Clarification in another feature's spec.** It is called out here
rather than applied quietly. 022's decision was correct for the document that existed when it was
made. Full reasoning: [research.md D3](research.md).

### F4. Nine defects the spec did not know about

Validation entry-by-entry against §10.1–§10.8 and §11.1–§11.6 found, beyond the five in `spec.md`:

| # | Defect | Authority |
| --- | --- | --- |
| 1 | `year` declares `fillIfEmpty`; spec says `immutable` | §10.7 |
| 2 | `definition` declares `append`; should be `firstWriteWin` | §10.2 rationale (research D8) |
| 3 | `relevance` declares `append`; should be `firstWriteWin` | §10.2 rationale (research D8) |
| 4 | `title`, `abstract`, `url`, `doi`, `aliases`, `authors` carry no `aligned` term | §10.1, §10.2, §10.7 |
| 5 | `Source` omits `tags` from Optional | §11.2 |
| 6 | `Entity` omits `tags` from Optional | §11.3 |
| 7 | `Reference` requires two predicates v0.11 does not | §11.6 (F3) |
| 8 | `Class` does not require `description` | §9.2 |
| 9 | `Class` does not declare `subClassOf`, which the tool writes onto five seeded types | §9.2 |

Defects 2 and 3 matter most: `definition` and `relevance` are the leading-prose predicates for
`Entity` and `Reference` (spec 022 keying). Without them, this feature would fix prose drift for
`Source` and leave the identical defect in two other core types. Defect 9 is a violation the
seeded tree inflicts on itself.

### F5. `validatedOverwrite` does not do what it claims

`mergeScalar` groups it with `immutable` in the **freeze** class, so `scoreZ`/`scoreC` are today
effectively permanent — directly contradicting their own registered descriptions ("recomputed by
a validation/ingest pass"). Moving to `lastWriteWin` is both the conformance fix and a bug fix.
This confirms `spec.md`'s assumption with a reason the assumption did not have (research D7).

### F7. Upstream moved again — 0.11 → 0.12 ([BUG-001](bugs/BUG-001.md), added 2026-08-23)

F1 above settled seven predicates' registration status in the tool's favour ("more conformant than
the specification"). CORE 0.12 answers the same ambiguity from the spec side, in the opposite
direction for six of them:

| Predicate | 0.11 status (this plan) | 0.12 fix |
| --- | --- | --- |
| `definition` (`Entity` Requires) | Registered, kept | Renamed to `text` |
| `authors` (plural) | Registered *alongside* `author` (F1) | Retired; `author` alone survives |
| `year` | Registered, `MergeImmutable` (T042) | Retired; `Reference` points at the already-registered `published` instead |
| `notes` (`Entity`/`Resource` Optional) | Registered, kept | Retired |
| `ref` (`Reference` Optional) | Registered, kept (T035) | Retired |
| `status` (`Reference` Optional) | Registered, kept (T035) | Retired |
| `granularity` (`Timeline` Optional) | Registered, kept (T034) | Retired |
| `relevance` (`Reference` leading prose) | Registered, `firstWriteWin` (T019) | ~~Unaffected — 0.12 drops it from *prose* only, never from an actual Requires/Optional list; no code change~~ **Corrected 2026-09-05 ([BUG-002](bugs/BUG-002.md))**: wrong — the *same* revision note quoted above separately retires `relevance` outright ("mentioned only in prose, never actually registered or required"), and `Reference`'s current (0.12) worked example's own Optional list (`url`/`author`/`published`/`doi`/`isCitedBy`) confirms it: no `relevance`, no `notes`. This row's "never from an actual Requires/Optional list" claim held for 0.10/0.11 (where `Reference` did carry both) but not 0.12. `relevance` retires; `Reference`'s leading prose becomes the shared `text` predicate. |

`href`-as-a-predicate (the ninth item in BUG-001, from CORE's own `.nt` worked example) does not
apply to this codebase — no RDF/N-Triples exporter exists here today.

One touch point this plan's Project Structure never listed: `internal/core/markdown.go`'s
`textPredicateFor`, which hardcodes `"definition"` and a type-independent `"notes"` as literal
round-trip keys, outside `CorePredicateDefs`/`CoreTypeDefs` entirely. Any 0.12 fix must update it
in lockstep or parsing/rendering silently diverges (data-model.md's own round-trip warning from
spec 022 applies here verbatim).

**`definition`→`text` merge collision — resolved 2026-08-23 (round 2).** `textPredicateFor` already
returns the literal `"text"` for `Resource`'s leading prose, and that predicate is seeded
`MergeAppend` (T018/T019 deliberately gave `definition` `MergeFirstWriteWin` instead). Renaming
`Entity`'s key to the same `"text"` predicate regresses `Entity`'s prose back to accumulating — the
exact defect US1 of this feature fixed for `Entity` specifically. **Decision**: accept the
regression. `text` stays `MergeAppend` uniformly; no per-type or role-qualified merge override is
introduced — the merge algebra has no such scoping mechanism today, and inventing one to save one
predicate's first-fixed behaviour was judged not worth the complexity. `Reference`'s `relevance`
is unaffected and keeps `MergeFirstWriteWin`. Formerly Open Item 3 below; now closed.

### F6. Spec amendments applied

`spec.md` has been amended in place: title and citations retargeted to v0.11; US1 and US4
rewritten against real behaviour (F2); FR-013 extended to `definition`/`relevance`; new FR-026
(aligned terms), FR-027 (`year`), FR-028 (`Reference`/`Class`/Optional corrections); Assumptions
for `author`/`authors` and `Reference` replaced with upstream citations. A plan must not carry
requirements its spec lacks.

## Technical Context

**Language/Version**: Go 1.26.6 (`go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra` v1.10.2, `github.com/charmbracelet/lipgloss`
v1.1.0, `github.com/yuin/goldmark` v1.8.2 + `goldmark-meta`, `gopkg.in/yaml.v3`,
`github.com/fogfish/faults` v0.3.2. **No new dependency.**

**Storage**: local git working tree; all I/O through `internal/adapter/fsys` (Principle VII)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` v2.2.4 (Principles VI, VIII), plus a
golden-file snapshot of the seeded `_schema/` tree (research D13, contract C2.1)

**Target Platform**: linux/darwin/windows, amd64 + arm64 (`.goreleaser.yaml`; windows/arm64 ignored)

**Project Type**: Single Cobra CLI binary (Principle III)

**Performance Goals**: N/A — `Seed()` is pure and in-memory; `arc upgrade` writes ~65 small files
and walks the graph once. No measurable budget applies.

**Constraints**:
- `MergeValidatedOverwrite`'s deletion is a **compile-time break**. Let the compiler enumerate
  the sites; do not grep-and-replace (`CORE-FIX.md` §5.7).
- `validMergeOps` is the only validation gate on a hand-written schema document. Tightening it
  turns previously-loadable graphs into hard load failures — correct per FR-002, but it means
  `arc upgrade` MUST NOT resolve the schema before replacing it (contract C3.2).
- FR-005 (tolerate a legacy `Class` merge attribute) and FR-004 (stop emitting it) pull in
  opposite directions. Tolerance lives in `decodeTypeDef`, prohibition in `typeNode`. **No lint
  rule** for the stale attribute (contract C1.3).
- Lint expectation tests MUST use a hand-built v0.11-shaped fixture graph, not the tool's own
  seeded output, or they pass vacuously.

**Scale/Scope**: 57 predicate + 8 type definitions; ~10 Go files; 1 new command.

## Constitution Check

*GATE: evaluated before Phase 0 and re-evaluated after Phase 1 design. Constitution v1.5.0.*

| Principle | Status | Evidence |
| --- | --- | --- |
| **I. ADRs binding** | ✅ | ADR 001 (system architecture) governs `ctrl` domain layout; `arc upgrade` follows `arc init`'s existing `cmd → component → service → kernel` path exactly. No ADR contradicted; none needed. `ARCHITECTURE.md` glossary rows for *Predicate Schema Node* / *Type Schema Node* MUST be updated in the same PR. |
| **II. DDD & glossary** | ✅ | No new domain concept beyond *built-in vocabulary* and *upgrade*, both added to the glossary. Ubiquitous language preserved: the command verb `upgrade`, `UpgradeResult`, and the help text all name the same concept. |
| **III. Hexagonal boundaries** | ✅ | `internal/core` gains no import. `cmd/arc/ctrl/upgrade.go` does flag parsing and rendering only; logic in `internal/app/ctrl/service/upgrade.go`. `fsys.Store` and `port.VCS` are the only I/O. |
| **IV. Functional style** | ✅ | `Seed()` stays pure. `upgrade` decomposes into `planUpgrade` (pure diff over `Seed()` output vs. read bytes) and `applyUpgrade` (writes). No function over 25 lines. GoDoc only. |
| **V. SOLID / YAGNI** | ✅ | Reuses `schema.Seed()`, `service.Resolve`, `splitParagraphs`. No new abstraction: `arc upgrade` takes the existing `fsys.Mounter`/`port.VCS`. No `--force` flag (research D11). |
| **VI. TDD** | ✅ | Red-first. Ordering in Phase 2 puts every test before its implementation. |
| **VII. Adapters** | ✅ | No new external system. No new adapter. All I/O through existing `internal/adapter/fsys` and `internal/adapter/git`. |
| **VIII. E2E traceability** | ✅ | Every acceptance scenario in `spec.md` maps 1:1 to a test in `cmd/arc/ctrl/upgrade_test.go`, `cmd/arc/graph/apply_test.go`, or `cmd/arc/lint/lint_test.go`, via the existing `sut()` helper. See Traceability below. |
| **IX. CLIG** | ✅ | `arc upgrade` is a verb; `--dry-run` is conventional; `--json`/`--verbose`/`--quiet` inherited from `bios`. Non-interactive by default — no prompt. |
| **X. Terminal output** | ✅ | `bios.Registry[UpgradeResult]` with human + JSON renderers, lipgloss via `bios`; no raw ANSI. |
| **XI. Config/env** | ✅ | No new configuration or environment variable. |
| **XII. Documentation** | ✅ | `README.md` (`arc init` paragraph + new `arc upgrade`), `ARCHITECTURE.md` glossary, `specs/VISION.md`. Cobra `Short`/`Long`/`Example` on the new command. |
| **XIII. Release** | ✅ | No `.goreleaser.yaml` change. |
| **XIV. Versioning & compatibility** | ⚠️ **Justified** | Deleting `MergeValidatedOverwrite` is a breaking change to graphs in the field: every previously-seeded graph fails to load until `arc upgrade` runs. See Complexity Tracking. |

**Post-Phase-1 re-evaluation**: unchanged. The design added no dependency, no adapter, no port,
and no domain package; `arc upgrade` reuses `ctrl`'s existing structure verbatim.

## Project Structure

### Documentation (this feature)

```text
specs/023-core-vocabulary-conformance/
├── plan.md              # This file
├── spec.md              # Amended during planning (F6)
├── research.md          # Phase 0 — D0–D13, upstream validation
├── data-model.md        # Phase 1 — target vocabulary, entry by entry
├── quickstart.md        # Phase 1 — runnable validation per user story
├── contracts/
│   ├── merge-vocabulary-contract.md    # C1 — closed set of six, rejection, tolerance
│   ├── seeded-schema-contract.md       # C2 — golden snapshot + tree invariants
│   └── upgrade-command-contract.md     # C3 — arc upgrade, normative ordering
├── checklists/requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
cmd/arc/
├── root.go                          # + cmd.AddCommand(ctrl.NewUpgradeCmd())
└── ctrl/
    ├── upgrade.go                   # NEW — Cobra wiring, --dry-run, renderers
    └── upgrade_test.go              # NEW — E2E, 1:1 with US5 scenarios (Principle VIII)

internal/
├── core/
│   ├── ast.go                       # ▲ delete MergeValidatedOverwrite; menu doc 7→6
│   ├── rules.go                     # ▲ delete TypeDef.Merge
│   ├── merge.go                     # ▲ freeze class loses validatedOverwrite
│   ├── ast_test.go                  # ▲ assert exactly six (C1.1)
│   ├── merge_test.go                # ▲ + idempotency property test (SC-001)
│   ├── markdown.go                  # ▲ NEW (BUG-001/F7) — textPredicateFor: Entity leading
│   │                                 #   key definition→text, MergeAppend (Open Item 3 resolved);
│   │                                 #   notes branch becomes type-aware, not universal
│   └── markdown_test.go             # ▲ NEW (BUG-001/F7) — round-trip fixtures per renamed key
└── app/
    ├── schema/
    │   ├── kernel/schema.go         # ▲ CorePredicateDefs (15▲ 3+), CoreTypeDefs (8▲)
    │   └── service/
    │       ├── schema.go            # ▲ validMergeOps→6; typeNode drops merge;
    │       │                        #   decodeTypeDef keeps tolerance; ErrSchemaInvalid
    │       │                        #   names `arc upgrade` (C1.2)
    │       └── seed_golden_test.go  # NEW — C2.1 snapshot + C2.2 invariants
    ├── ctrl/
    │   ├── component.go             # + Upgrade delegator
    │   ├── kernel/graph.go          # + UpgradeResult
    │   └── service/upgrade.go       # NEW — planUpgrade / applyUpgrade, C3.2 ordering
    └── lint/service/
        └── rules_type_conformance_test.go  # ▲ v0.11-shaped fixture, not seeded output

testdata/golden/schema/              # NEW — byte-exact seeded tree (C2.1)
```

**Structure Decision**: `arc upgrade` lands in the existing **`ctrl`** (control-plane) domain as a
top-level sibling of `init` — the same operation over a graph's built-in vocabulary, one for a new
graph and one for an existing one. Rejected: `arc init --upgrade` (overloads a command whose whole
contract is "target must be empty") and `arc apply schema --replace` (a destructive flag on a
command documented as additive merge, under a noun that means *patches*, not *releases*).
Full reasoning: [research.md D11](research.md).

## Traceability (Principle VIII)

| Story | Scenarios | E2E test file |
| --- | --- | --- |
| US1 prose | 5 | `cmd/arc/graph/apply_test.go` |
| US2 seed | 6 | `cmd/arc/ctrl/init_test.go` + `internal/app/schema/service/seed_golden_test.go` |
| US3 lint | 5 | `cmd/arc/lint/lint_test.go` (hand-built v0.11 fixture) |
| US4 predicates | 4 | `cmd/arc/graph/apply_test.go`, `cmd/arc/lint/lint_test.go` |
| US5 upgrade | 7 | `cmd/arc/ctrl/upgrade_test.go` |

## Implementation Sequencing (input to `/speckit-tasks`)

The order is load-bearing — `CORE-FIX.md` §5.7 asks for it to be explicit.

1. **Golden snapshot first**, against *current* `Seed()` output. It is the regression net; built
   after the change it certifies nothing.
2. **Type/predicate table corrections** + `typeNode` dropping `merge`. Regenerate golden; the
   diff is the reviewable artifact of this whole feature.
3. **Lint expectation tests** on a hand-built v0.11 fixture. Green after step 2 — proves US3
   without any merge-vocabulary change.
4. **`arc upgrade`**, while `validatedOverwrite` is still accepted. It must exist and be tested
   *before* the gate tightens, or there is no remedy for the graphs step 5 breaks.
5. **Delete `MergeValidatedOverwrite`** and tighten `validMergeOps`; update `ErrSchemaInvalid` to
   name `arc upgrade`. Compile-time breaks surface here, by design.
6. **Idempotency property test** (SC-001) and docs (`README.md`, `ARCHITECTURE.md`, `VISION.md`).

Steps 1–3 are shippable without 4–6 and deliver US1–US4 for new graphs.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --- | --- | --- |
| **XIV** — breaking change to graphs in the field: every previously-seeded graph fails to load until `arc upgrade` runs | §9.3's menu is closed at six and §14 requires profile predicates to draw from it. Keeping a seventh operation is *the* violation, not a workaround for it. `arc` is pre-1.0 and CORE is Draft. | *Lenient reader for one release* — contradicts FR-001/FR-004 and leaves the non-conformant value live indefinitely, since nothing would force the upgrade. *Silent rewrite on read* — mutates a user's graph without consent. The hard failure is mitigated to an inconvenience by C1.2's error naming `arc upgrade`, and by sequencing step 4 before step 5. |
| **Superseding spec 022's Clarification** on `Reference` | v0.11's normative `Class` block replaced the revision note 022 relied on. | *Keep 022's decision* — leaves the tool requiring two predicates the current spec does not, which is the exact false-positive class this feature exists to remove. Raised explicitly per Principle I rather than diverged from quietly. |
| **XIV** — breaking change (round 2, [BUG-001](bugs/BUG-001.md)): a graph carrying `definition`/`authors`/`year`/`notes`/`ref`/`status`/`granularity` front-matter is not specially supported once these are retired | Same reasoning as the row above, applied consistently: `arc` is pre-1.0 with no compatibility guarantee (`ARCHITECTURE.md` Compatibility Policy); introducing leniency for this retirement but not the merge-vocabulary one would be an arbitrary inconsistency. | *Reader leniency for one release* — FR-037 originally proposed this and was corrected during bugfix-verify precisely because it contradicts the row above's own rejection of the identical alternative for the same feature. |
| **Accepting `Entity`'s prose-drift regression** ([BUG-001](bugs/BUG-001.md), round 2) | `definition`→`text` collides with `Resource`'s already-`MergeAppend` `text` predicate; the merge algebra has no per-type/role-qualified override. | *Invent a merge-scoping mechanism* (per-type `text`, or a role+type-qualified override) — rejected as new complexity to preserve one predicate's first-fixed behaviour, when the existing six-value, node-agnostic merge model is itself a load-bearing simplicity constraint (FR-001/FR-004). |
| **XIV** — breaking change ([BUG-002](bugs/BUG-002.md), 2026-09-05): a `Reference` node carrying `relevance` front-matter/prose, or any node carrying `notes`, is not specially supported once both are retired everywhere; `Reference`'s own leading prose migrates from `relevance` (`MergeFirstWriteWin`) to the shared `text` predicate (`MergeAppend`) — a second, narrower instance of the same `Entity`-style prose-drift regression accepted immediately above | Same reasoning as the two rows above, applied consistently: `arc` is pre-1.0, and `relevance`/`notes` are no more entitled to reader leniency than the seven predicates BUG-001 already retired without it. `Reference`'s `relevance`→`text` migration has no other registered predicate to land on — `text` is the only shared generic prose predicate CORE registers (§10.2). | *Keep `relevance` registered, `Reference`-only* — this is F7's own original (now-corrected) row's mistake: the upstream spec has already retired it, so keeping it would silently diverge from CORE 0.12 again, the exact drift this feature and BUG-001 both exist to close. *Invent a `Reference`-specific merge override to preserve `relevance`'s first-fixed behaviour under the `text` name* — rejected for the same reason the row above already rejected it for `Entity`. |

## Open Items for `/speckit-clarify`

None blocking. Three items below; #3 is resolved, #1 and #2 remain worth confirming:

1. **`Reference` retraction (F3)** supersedes another feature's recorded Clarification. The
   reasoning is sound and the change is a pure relaxation, but it is the user's call whether to
   fold it into this feature or split it out.
2. **`relevance` → `firstWriteWin` (F4 defect 3)** rests on §10.2's rationale sentence, not an
   explicit `merge:` declaration — §10.7 states none. Worth filing upstream alongside the `cites`
   role contradiction (§10.6) and the seven-vs-`granularity` registration gap. **Narrowed 2026-08-23
   (round 2, [BUG-001](bugs/BUG-001.md)): this item no longer includes `definition` — CORE 0.12
   retires that predicate outright (F7), so there is no upstream merge-value ambiguity left to
   file for it, only for the surviving `relevance`.**
3. ~~**`definition`→`text` merge collision (F7, [BUG-001](bugs/BUG-001.md)), added 2026-08-23.**~~
   **Resolved 2026-08-23 (round 2)**: `text` adopts `MergeAppend` uniformly; see F7 above. No
   longer blocking — T065 is unblocked.
