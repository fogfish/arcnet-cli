
# Implementation Plan: Predicate-First Graph Node Model

**Branch**: `010-predicate-node-model` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/010-predicate-node-model/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Rewrite `internal/core.Node`'s field shape to match ARCNET-AST v0.6 §4-7 and re-derive `internal/core`'s codec (`markdown.go`) and merge algebra (`merge.go`) around it: `Kind Kind` → `Type string` (from `"@type"`, no fallback); `ID` stays but is strictly `"@id"`, required equal to the file basename, no `id`/`title`/`period` fallback; `Attrs map[string]any` → `Attrs map[string][]Predicate` (`Predicate{Value any; Target string; Alias string}`, exactly one of `Value`/`Target` set, every entry a non-empty ordered list even for a single value); `Text string` + `Notes string` → `Texts map[string]string` keyed by a small, explicitly-flagged-as-a-stopgap `@type`→text-predicate lookup table (`source`→`abstract`, `entity`→`definition`, `resource`→`relevance`, `hypothesis`→`claim`, `aporia`→`tension`, `thought`→`claim`, generic fallback `"text"`/`"notes"`), pending spec 011's Schema Index; `Edges []Link` + `Links map[string]LinkBlock` → one flat `Edges []Link` (grouping-title storage dropped entirely — AST §3 invariant 4, grouping is derived not stored); `HRefs []Link` unchanged. `RenderNode`/`RenderPatch` render `Edges` as one flat bulleted list for now (role-driven grouped-heading rendering deferred to spec 013). Old-format detection (`kind` field, missing `"@id"`/`"@type"`, `"@id"` ≠ basename) fails loudly via the existing `ErrManifestInvalid` fault, never a silent reinterpretation — no old-format read support is implemented. This is entirely an `internal/core` change; every other touched package (`internal/app/schema`, `internal/app/graph`, `internal/app/lint`, and their `cmd/arc/...` callers) only needs mechanical field-name/type updates to keep compiling — no new business logic — landed in the same PR because Go will not build otherwise.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: No new dependency. Reuses `github.com/yuin/goldmark`/`goldmark-meta` (Markdown parsing) and `gopkg.in/yaml.v3` (front-matter codec) — `internal/core`'s existing codec stack, unchanged — plus `github.com/fogfish/faults` for the old-format rejection error path and `github.com/fogfish/it/v2` for tests.

**Storage**: The mounted graph root, accessed exclusively through the existing, unmodified `internal/adapter/fsys` `Store`/`Mounter` — no new I/O path; only the `core.Node`/`core.Patch` values every caller constructs and consumes change shape.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII). New table-driven round-trip cases in `internal/core/ast_test.go`/`internal/core/markdown_test.go` covering every ARCNET-CORE §11 worked example (`source`, `entity`, `resource`, `timeline`) plus at least one DOMAIN-ARTICLE `hypothesis` example (`derivedFrom`/`assumes`/`addresses`), written first per Principle VI. Existing fixtures across `internal/app/schema`, `internal/app/graph`, `internal/app/lint`, and every `cmd/arc/...` E2E test MUST be rewritten from the old `kind`/two-slot shape to the new `"@id"`/`"@type"`/`Texts`/single-`Edges` shape to keep compiling and to keep asserting real behavior (an old-format fixture no longer parses at all) — this is necessarily a fixture-wide edit, not a no-op recompile, even though no new *business logic* is added outside `internal/core`. `cmd/arc/lint` and `cmd/arc/graph`'s existing E2E suites gain new cases for spec.md User Story 3's old-format-rejection scenarios (Acceptance Scenarios 1-4).

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`; no platform-specific code introduced.

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — no new command, no new `internal/app/<domain>` use-case, no new port/adapter; this feature reshapes the existing `internal/core` domain type every use-case already depends on.

**Performance Goals**: No measurable change — parsing/rendering remain single-pass over the goldmark AST and the front-matter map; representing every attribute as a list and merging two prose slots into a map trades a handful of extra allocations per node for no algorithmic complexity change.

**Constraints**: No new third-party dependency; `"@id"` MUST equal the file's basename with no fallback (spec FR-002); a node missing `"@id"` or `"@type"`, or whose `"@id"` mismatches its basename, MUST fail with a clear, non-zero-status error and zero writes (spec FR-012/FR-013) — old-format (`kind`/fallback-`id`) read support MUST NOT be implemented, not even partially; round-trip and idempotent-round-trip fidelity MUST hold except for the explicitly-permitted cosmetic edges-grouping normalization (spec FR-014/FR-015) — **note (Bugfix BUG-003, 2026-07-21)**: a `**Label**`-derived predicate's occurrences collapsing into an *undifferentiated* flat list together with a different labeled predicate's occurrences is not this permitted normalization (which only reorders/corrects an already-known predicate's shape to match its own declared role) — it is a loss of distinctness between two different predicates, now closed by spec FR-022; per-predicate merge behavior itself (union/first-writer/append/etc. policy) is unchanged — only the shapes `merge.go` operates over change; **this feature is a breaking change to the `arc subgraph --json` `Node` schema** (`kind`→`type`, `attrs` values become arrays of `{value|target, alias}`, `text`/`notes` become a `texts` map, `edges`+`links` collapse into one `edges` array) with no prior deprecation warning — acceptable pre-1.0 (current release train is `0.0.x`) but called out explicitly per Principle XIV rather than hidden, see Complexity Tracking; ~~**spec FR-005/Acceptance Scenario 2's "several independently named prose sections per node" is only partially satisfied by this increment** — `walkNodeBody`'s structural parser still recognizes exactly two prose positions (leading paragraphs, trailing paragraphs), now labelled via the `@type`→predicate lookup table above rather than fixed `text`/`notes` names, so `Texts` genuinely supports open keys as a *representation* (unblocking spec 011+) but a single node cannot yet declare a third distinct named prose section (e.g. both `## Abstract` and `## Relevance` bodies) purely through this feature — flagged explicitly, not silently narrowed, see Complexity Tracking.~~ Closed (Bugfix BUG-002, 2026-07-20): FR-019 now has `walkNodeBody` resolve a `**Label**`-headed block's predicate identity against the schema index (spec 011's Schema Index, finally consulted at parse time) and dispatch by that predicate's declared role, so a third+ named prose section is captured under its own name instead of being dropped — see Complexity Tracking. **Partially reopened (Bugfix BUG-003, 2026-07-21)**: closing the data-loss gap did not close the formatting/round-trip gap — a captured block's list-item markup, recoverable label, and per-block grouping were not preserved on write; see spec FR-020/FR-021/FR-022 and the new Complexity Tracking row below. **BUG-001**: the "@id"/"@type" MUST-be-present constraint above applies to a *standalone node file's own front matter*, which has no other place to declare identity/type; it does NOT mean `parsePatchBody` must require the same two keys duplicated inside every patch node's own yaml fence — a patch section's `"## <ID>"`/`"# <Type>"` headings satisfy the requirement by themselves (spec FR-011/FR-018), consistent with `internal/core/markdown.go`'s pre-spec-010 `parsePatchBody`, which already derived `currentKind`/`id` from exactly these headings.

**Bugfix**: 2026-07-07 — BUG-001 Updated from bugfix patch: clarified that this feature's "@id"/"@type" mandatory-declaration constraint does not require `parsePatchBody` to abandon deriving identity/type from patch section headings; see spec.md FR-011/FR-018 and data-model.md's `Node.Type`/`Patch.Nodes` rows.

**Bugfix**: 2026-07-20 — BUG-002 Updated from bugfix patch: closed the Constraints/Complexity Tracking entries that deferred multi-named-prose-section parsing to spec 011. `internal/core.ParsePatch`/`ParseNode`/`parsePatchBody`/`walkNodeBody` gain a `core.Index` parameter, mirroring `RenderNode`/`RenderPatch`'s existing signature shape — no new layering violation, since `core.Index` is already domain-native (`internal/core/rules.go`) and rendering already depends on it.

**Bugfix**: 2026-07-21 — BUG-003 Updated from bugfix patch: annotated the BUG-002 closure note and the Constraints line's cosmetic-normalization carve-out (both above), and added a new Complexity Tracking row — BUG-002 closed the data-loss half of the deferred multi-named-prose-section gap but left a narrower formatting/round-trip gap (list-item markup, label recoverability, per-block grouping) that spec FR-020/FR-021/FR-022 now cover.

**Scale/Scope**: Primary change in `internal/core` (`ast.go`, `markdown.go`, `merge.go`, `filter.go`, `rules.go` + their tests). Mechanical compile-fix ripple (rename `core.Kind`→`string`, `node.Kind`→`node.Type`, `node.Text`/`node.Notes`→`node.Texts[...]`, `node.Edges`+`node.Links`→`node.Edges`, `node.Attrs[k].(T)`→list-aware access) across `internal/app/schema/{component.go,kernel/schema.go,service/schema.go}`, `internal/app/graph/{port/schema.go,kernel/apply.go,kernel/grep.go,service/apply.go,service/grep.go,service/subgraph.go}`, `internal/app/lint/service/{rules_frontmatter.go,rules_history.go,rules_identity.go,rules_links.go,rules_predicates.go,locate.go}`, and `cmd/arc/graph/{apply.go,grep.go,serve.go}` plus every corresponding `_test.go` and `testdata/` fixture in those packages.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary MUST update the "Node" entry (identity/type/attrs/texts/edges shape) and add "Predicate" in the same PR; no ADR is superseded — this follows ADR 001's existing `internal/core` (shared domain) split unchanged, only the domain type's internal shape changes. `tasks.md` MUST include this glossary-update task. |
| II — DDD & Glossary | Yes | PASS, with the same obligation — `"@id"`/`"@type"`, `Attrs`/`Predicate`, `Texts`, and the unified `Edges` are the ubiquitous language for every node file a graph maintainer writes by hand; the Glossary MUST reflect the renamed/reshaped concepts consistently with code (`Kind`→`Type`, `Text`/`Notes`→`Texts`, `Links`+`Edges`→`Edges`). |
| III — Hexagonal Architecture | Yes | PASS — all changes live in `internal/core` (shared domain) plus mechanical updates inside existing `internal/app/<use-case>` service/kernel packages; no `cmd/` package gains business logic, no new port/adapter, import direction unchanged. |
| IV — Functional Programming Style | Yes | PASS — new/changed functions (`decodeAttrs`, `textPredicateFor`, `mergeTexts`, `mergeAttrLists`) stay small and single-purpose; no inline comments beyond existing GoDoc conventions; parsing/rendering remain pure transformations of `[]byte`/`Node`, no new side effects introduced. |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS, with a documented deviation — the `@type`→text-predicate lookup table is a deliberate, temporary hardcoded stopgap (not a speculative abstraction) explicitly superseded by spec 011's Schema Index; kept as a single small map + lookup function, not a pluggable strategy interface nobody else needs yet. |
| VI — TDD | Yes | PASS — new table-driven cases in `ast_test.go`/`markdown_test.go`/`merge_test.go`/`filter_test.go` are written first per Principle VI, using `github.com/fogfish/it/v2` exclusively; each covers a CORE §11 worked example and must compile and fail semantically (wrong shape / lost content) before implementation. |
| VII — External Integration & Adapter Consistency | Yes | PASS — no new external integration; the only I/O this feature touches (`fsys.Store` reads/writes) is unchanged, still exclusively through `internal/adapter/fsys`. |
| VIII — E2E Acceptance Testing | Yes | PASS, with obligation — every acceptance scenario across spec.md's 3 user stories (14 scenarios total) needs a colocated E2E case: US1 via `internal/core` round-trip tests (not Cobra-level, since US1 has no CLI surface of its own — covered at the domain layer per Principle VIII's "pragmatic deviation for small tools" allowance is not invoked here; instead US1 is proven through `cmd/arc/graph/apply_test.go`'s existing round-trip-via-apply path), US2 via updated `cmd/arc/graph/{apply,grep,subgraph,serve}_test.go` and `cmd/arc/lint`'s E2E suite against predicate-first fixtures, US3 via new old-format-fixture cases added to those same suites asserting non-zero exit and zero writes. Existing E2E tests change substantially (fixture rewrite), which is expected here since the on-disk contract itself is the thing changing, not a symptom of poorly-derived tests. |
| IX — CLIG/Cobra (ADR 002) | No | N/A — no command, flag, or help text changes; every command's CLI surface (flags, arguments, exit-code meanings) is byte-for-byte identical before and after this feature. |
| X — Terminal Output, Color & Interactivity | No | N/A — no Reporter phase added or changed; existing per-node reporting is reused unmodified. |
| XI — Configuration, Environment Variables & Secrets | No | N/A — no configuration surface touched. |
| XII — Documentation & Help System | Yes | PASS — the new old-format-rejection error MUST be a `faults.Type`/`faults.SafeN` constant (reusing or extending the existing `ErrManifestInvalid`) with human-readable guidance naming the offending file and the missing/mismatched field, not a raw parse error; no help text changes since no flag/command changed. |
| XIII — Distribution & Release Engineering | No | N/A — no release pipeline change. |
| XIV — Versioning/Security | Yes, flagged | PARTIAL, explicitly justified — this feature breaks the `arc subgraph --json` `Node` schema (Constraints above) with no prior deprecation warning, which Principle XIV's letter asks to precede with one. Accepted here because the project is pre-1.0 (`0.0.x` release train per recent tags) and `--json` has not yet been declared a stable contract in any release notes; recorded in Complexity Tracking for visibility rather than silently absorbed. No other Principle XIV rule (SemVer tagging, `govulncheck`, telemetry) is affected. |

No blocking violations — the two flagged items (breaking `--json` schema pre-warning, and FR-005's partial satisfaction) are pre-1.0 scope trade-offs the user's own technical approach explicitly called for, recorded below rather than hidden.

## Project Structure

### Documentation (this feature)

```text
specs/010-predicate-node-model/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — D1-D8 design decisions
├── data-model.md         # Phase 1 output — Node/Predicate/Texts/Edges shapes, Patch delta
├── quickstart.md         # Phase 1 output — 3 runnable scenarios, one per user story
├── contracts/            # Phase 1 output
│   ├── ast-contract.md    # supersedes specs/003-apply-patch/contracts/ast-contract.md's Node shape
│   └── subgraph-json-contract.md  # delta over kernel.SubgraphResult.Patch.Nodes's --json shape
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/                             # primary change
│   ├── ast.go                          # Node.Type (was Kind Kind → Type string, no Kind type);
│   │                                     #   Node.Attrs map[string][]Predicate (new Predicate struct);
│   │                                     #   Node.Texts map[string]string (was Text/Notes string);
│   │                                     #   Node.Edges []Link only (LinkBlock type removed);
│   │                                     #   Patch.Nodes unaffected in shape, Node itself changes
│   ├── ast_test.go                      # + Predicate/Attrs-list, Texts-map, unified-Edges cases
│   ├── rules.go                        # MergeRuleSet map[Kind]MergeOp → map[string]MergeOp
│   ├── markdown.go                      # deriveNodeID: delete id/title/period fallback, "@id"-only;
│   │                                     #   decode "@type" not "kind"; wrap every front-matter
│   │                                     #   scalar/array into []Predicate; textPredicateFor(type,
│   │                                     #   isLeading bool) string lookup table; walkNodeBody keeps
│   │                                     #   its structural leading/list/heading-blocks/trailing
│   │                                     #   parse but returns Texts map + single Edges slice;
│   │                                     #   RenderNode/RenderPatch render Edges as one flat list
│   ├── markdown_test.go                 # + ParseNode/RenderNode/RenderPatch round-trip cases per
│   │                                     #   CORE §11 worked example + 1 DOMAIN-ARTICLE hypothesis;
│   │                                     #   + old-format rejection cases (missing @id/@type,
│   │                                     #   @id≠basename, legacy kind field)
│   ├── merge.go                         # mergeCore: Texts merged key-by-key (union of key sets,
│   │                                     #   mergeScalarInto per key); Attrs merged as
│   │                                     #   list-of-Predicate per key (existing per-attribute
│   │                                     #   policy preserved, only the list shape is new); Edges
│   │                                     #   unioned as one list (LinkBlock merge removed)
│   ├── merge_test.go                    # + Texts per-key merge, Attrs-list merge, unified-Edges
│   │                                     #   union cases per existing MergeOp
│   ├── filter.go                        # node.Kind→node.Type; node.Attrs[name] list-aware match
│   ├── filter_test.go                   # + list-valued Attrs match cases
│   └── errors.go                        # ErrManifestInvalid gains explicit old-format guidance
│                                          #   (unrecognized "kind" field / missing "@id"/"@type" /
│                                          #   "@id" basename mismatch), still one faults.Type
│
└── app/
    ├── schema/
    │   ├── component.go                  # core.Kind → string
    │   ├── kernel/schema.go               # core.Kind → string (SchemaKind, coreKindDescriptions)
    │   ├── service/schema.go              # node.Attrs["merge"] list-aware access; core.Kind → string
    │   └── service/schema_test.go         # fixtures updated to predicate-first shape
    ├── graph/
    │   ├── port/schema.go                 # core.Kind → string
    │   ├── kernel/apply.go                # map[core.Kind]int → map[string]int
    │   ├── kernel/grep.go                 # Kind core.Kind json field → Type string
    │   ├── service/apply.go               # node.Kind→node.Type; isStub checks Texts/Attrs/Edges;
    │   │                                    #   node.Links removed; core.Kind → string throughout
    │   ├── service/apply_test.go          # fixtures updated to predicate-first shape
    │   ├── service/grep.go                # node.Kind→node.Type; match against Texts values
    │   ├── service/grep_test.go           # fixtures updated to predicate-first shape
    │   ├── service/subgraph.go            # n.Edges/n.Links collapse to n.Edges; target.Kind→Type
    │   └── service/subgraph_test.go       # fixtures + --json golden output updated
    └── lint/
        ├── service/rules_frontmatter.go   # node.Kind→node.Type
        ├── service/rules_history.go       # node.Kind→node.Type
        ├── service/rules_identity.go      # node.Kind→node.Type; node.Attrs["category"] list-aware
        ├── service/rules_links.go         # sortedLinkKeys/LinkBlock iteration removed; single
        │                                    #   node.Edges slice; node.Kind→node.Type
        ├── service/rules_predicates.go    # node.Links iteration removed; node.Edges only;
        │                                    #   node.Kind→node.Type
        ├── service/rules_links_test.go    # fixtures updated
        ├── service/rules_predicates_test.go # fixtures updated
        └── service/locate.go              # field references updated if any

cmd/
└── arc/
    ├── graph/
    │   ├── apply.go                       # no CLI-visible change; internal Node references only
    │   ├── apply_test.go                  # fixtures rewritten to predicate-first shape; + old-
    │   │                                    #   format-rejection E2E cases (spec US3, all 4 scenarios)
    │   ├── grep.go                        # no CLI-visible change
    │   ├── grep_opts_test.go              # fixtures updated
    │   ├── serve.go                       # no CLI-visible change
    │   ├── serve_test.go                  # fixtures updated
    │   └── subgraph_test.go               # fixtures + --json golden output updated (US2 scenario 4)
    └── lint/                              # E2E suite gains old-format-rejection case (US3)

ARCHITECTURE.md                          # + Glossary updates: Node (reshaped), Predicate (new),
                                            #   Text Predicate / Prose Field, Edge (unified),
                                            #   removal of "Link Block" entry (superseded)
```

**Structure Decision**: No new command, no new `internal/app/<domain>` use-case, no new port/adapter — this feature reshapes the one domain type (`internal/core.Node`, plus its `Patch` container) every existing use-case already imports, and mechanically updates every caller to keep compiling and to exercise the new shape end to end. `internal/adapter/fsys` is untouched (no I/O contract changes), and no `cmd/` package gains new flags, commands, or business logic.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Breaking `arc subgraph --json` `Node` schema (Principle XIV) with no prior deprecation warning | The old shape (`kind`, `attrs` as plain scalars, `text`/`notes`, split `edges`/`links`) cannot represent ARCNET-AST v0.6 additively — `attrs` values change type (scalar → array-of-`Predicate`) and `text`/`notes` become an open map, which is not expressible as an additive/optional field the way spec 009's `published` field was | An additive-only shim (keep old fields alongside new ones) was rejected: it would require maintaining two parallel representations of the same data indefinitely, contradicting Principle V (YAGNI) for a pre-1.0 tool with no documented `--json` stability guarantee yet to preserve |
| ~~`Texts` supports open predicate keys as a *type*, but this increment's parser only ever populates two (leading/trailing, per-`@type` labelled) — spec FR-005/US1 Acceptance Scenario 2's "several independently named prose sections" is not fully realized~~ **Closed by Bugfix BUG-002 (2026-07-20)** | ~~Recognizing an arbitrary `"## <PredicateName>"` heading immediately followed by prose (not a list) as a third+ named text block requires deciding how that heading is disambiguated from a link-block heading without schema role knowledge — exactly the Schema Index question spec 011 owns; building an interim heuristic here risks producing behavior spec 011 then has to un-teach~~ Spec 011's Schema Index now exists and is threaded into `ParsePatch`/`ParseNode` (FR-019); the disambiguation this row worried about is resolved by inverse `labelFor` lookup against `index.Predicates`, not a throwaway heuristic |
| **Bugfix BUG-003 (2026-07-21)**: BUG-002's fix captured a `**Label**`-resolved block's *content* but not its *formatting* — a text-role block's list items lost their literal wikilink brackets/list shape (routed through `extractInlineLinks`/`reconstructHRefs`, a heuristic built for free-flowing paragraph prose, not discrete list items), the block's `**Label**` never reappeared on write (no `label` attribute auto-registered; `renderNodeBody` never emits a heading for any `role: text` predicate), and distinct labeled edge blocks collapsed into one flat list (auto-discovery only distinguished `text` vs `edge` observed role, with no signal for "edge content under its own label" → `role: link`) | A predicate's captured *content* and its recoverable *shape* (list markers, literal link markup, label, grouping) are both required for spec FR-006/FR-015's pre-existing verbatim/byte-stable guarantees and spec 013 FR-001/FR-004/FR-014's label/grouping guarantees — BUG-002's own verification (T089/T090/T091) checked content presence only | Treating this as acceptable "cosmetic normalization" under FR-014's existing carve-out was considered and rejected: FR-014's carve-out is scoped to *link-grouping layout* changes that don't alter which predicate an occurrence belongs to; collapsing three distinctly-labeled predicates' occurrences into one undifferentiated list does change what a reader can tell about which predicate an occurrence belongs to, which is a real information loss, not a layout choice | Extending `walkNodeBody`'s heading/bold-label matching to also claim paragraph-followed headings as named text blocks was considered and rejected for *this* spec to keep the diff reviewable and avoid a second, throwaway heuristic living alongside spec 011's eventual real one — deferred, not dropped; the deferral is now resolved, see FR-019 |
