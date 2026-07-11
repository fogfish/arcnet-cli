# Research: Full ARCNET-CORE §16 Conformance Checks for `arc lint`

**Feature**: [spec.md](spec.md) | **Branch**: `014-lint-spec-conformance`

All unknowns raised by the user-provided technical approach are resolved below by reading the actual
current state of `internal/core`, `internal/app/schema`, and `internal/app/lint` rather than assumed —
several resolve differently than the technical approach's own framing anticipated.

## D1: Is `"@id"`/`"@type"` quoting enforceable after a YAML parse, or must scope be dropped?

**Decision**: Enforceable — via raw-text/line-based regex detection, exactly mirroring the existing
`locateFrontMatterField`/`locateFrontMatterDelimiter` pattern in `internal/app/lint/service/locate.go`.
Kept in scope (not dropped).

**Rationale**: `internal/core/markdown.go` decodes front matter with `gopkg.in/yaml.v3`'s
`yaml.Unmarshal` into a generic map (line ~502). yaml.v3 parses a bare `@id: x` mapping key
identically to a quoted `"@id": x` — both decode to the Go string key `"@id"`. Confirmed by reading the
decode path: nothing in `core.ParseNode` or its callers distinguishes the two forms once decoded, so
`Node.ID`/`Node.Type` are populated the same way regardless of quoting. This means the quoting
violation is real (invisible post-parse) exactly as the technical approach suspected, and the only way
to detect it is inspecting `raw []byte` directly — the same technique `locateFrontMatterField` already
uses for other line-location needs. A new pair of small regexes (`^@id\s*:` / `^@type\s*:`, matching the
*unquoted* bare form) added to `locate.go` is sufficient; no YAML-library replacement or AST-level
change is needed.

**Alternatives considered**: Parsing front matter twice (once permissive, once strict-quote-only) and
diffing results — rejected as unnecessarily indirect compared to a direct raw-text regex check, and it
would not gracefully cover the "key present but written in an unexpected style" case (e.g. single- vs
double-quoted, both of which CORE §10.1 accepts — the check only needs to reject the *bare* form).

## D2: Does `checkUnrecognizedKind` still need to be rewritten to use the Schema Index?

**Decision**: No rewrite needed — already done.

**Rationale**: Reading `internal/app/lint/service/rules_identity.go` (the file that actually contains
`checkUnrecognizedKind`, not `rules_frontmatter.go` as the technical approach's file mapping assumed)
shows it already checks `index.Types[node.Type]`, where `index core.Index` is `core.Index` populated by
`internal/app/schema/service.Resolve` (spec 011) — not a `core.MergeRuleSet` map. That legacy type/map
does not exist anywhere in the current codebase (confirmed by repo-wide grep — zero matches). This part
of the technical approach describes a prior state of the code that spec 011's own implementation already
superseded. No task is generated for it; this is recorded here so the discrepancy is not silently lost
and no one re-derives the same (already-resolved) question later.

## D3: How does the citation-predicate check become schema-driven instead of a hardcoded Go list?

**Decision**: `checkCitationPredicate` gains a `registry map[string]core.PredicateDef` parameter
(mirroring `checkPredicateRegistered`'s existing signature) and treats predicate `p` as a valid citation
predicate iff `registry[p].Aligned` has prefix `"cito:"`. The hardcoded `citoPredicates` map in
`rules_predicates.go` is deleted.

**Rationale**: `internal/app/schema/kernel/schema.go`'s `CorePredicateDefs` (spec 011's seed data,
written to `_schema/predicates/` by every `arc init`) already registers exactly the same ten predicates
the hardcoded map lists (`cites`, `citesAsEvidence`, `citesAsAuthority`, `supports`, `confirms`,
`extends`, `critiques`, `disputes`, `refutes`, `isCitedBy`), each with an `Aligned` field prefixed
`"cito:"`. Every other seeded predicate has either no `Aligned` value or a non-`cito:` one (`schema:`,
`skos:`, `dcterms:`). So switching the source of truth to the schema index reproduces today's exact
accepted set for any graph created by `arc init` unchanged (zero behavior change for the common case),
while a domain profile can register a new `_schema/predicates/<name>.md` with `aligned: "cito:someTerm"`
and have it recognized with no `arc` code change (spec FR-006/FR-007, User Story 4).

**Alternatives considered**: Keeping the hardcoded list as a fallback when the registry lookup misses —
rejected; the spec (User Story 4, Acceptance Scenario 3) explicitly requires no built-in fallback vocabulary,
since a fallback would silently mask a graph's own schema being wrong or incomplete.

## D4: Does the predicate-role check apply to `HRefs` entries carrying a `Predicate`?

**Decision**: No — an `HRefs` occurrence with a non-empty `Predicate` is exempt from the new
role-conformance check (User Story 5 / FR-008), regardless of that predicate's declared schema role.

**Rationale**: `internal/core/markdown.go`'s `extractInlineLinks` (used by `walkNodeBody` when parsing a
node's prose) implements a real, already-shipped inline citation-tagging syntax —
`[predicate:: [[Target]]]` embedded directly in a text paragraph — that legitimately produces an
`HRefs` entry with `Predicate` set, for predicates whose own registered role is `edge` or `link` (e.g.
`citesAsEvidence` is role `edge`; `cites` is role `link`), not `href`. This is not test-only synthetic
data: `TestCheckCitationPredicateValid`/`Invalid` in `rules_predicates_test.go` exercise exactly this
real parser output. Applying a literal role check to these occurrences (role `edge`/`link` predicate
found in the `HRefs`/inline position) would misclassify this pre-existing, intentional convention as a
role mismatch on every graph that uses any citation predicate inline — a guaranteed false positive
against real usage the spec's own "no false positives against existing fixtures" constraint forbids.
The citation-predicate check (User Story 4/FR-007) already validates these occurrences' predicate
identity; the role check's job is the *other* four predicate-occurrence categories (`Attrs` → `meta`,
`Texts` → `text`, `Edges` → `edge`/`link`, and untyped/no-predicate `HRefs` → `href`), where no
comparable pre-existing convention creates an intentional mismatch.

**Alternatives considered**: Adding a sixth "inline-citation" role value to the schema's `role` field
vocabulary to make this explicit — rejected as out of scope; it would require changing
`internal/app/schema/service.go`'s `validRoles` set and re-seeding every graph's `_schema/predicates/`
documents, which is a schema-vocabulary change, not a lint-validation change, and the spec's own Out of
Scope section excludes changes beyond `arc lint` itself.

## D5: What does "predicate occurrence position" map to concretely, for the role check?

**Decision**: Four of `core.Node`'s five fields correspond directly to four of the five schema roles:

| Node field | Populated from | Corresponding schema `Role` |
|---|---|---|
| `Attrs` (map key) | front-matter scalar/array | `meta` |
| `Texts` (map key) | prose paragraph | `text` |
| `Edges` (`Link.Predicate`) | typed edge bullet or grouped link block | `edge` **or** `link` |
| `HRefs` (`Link.Predicate == ""`) | inline untyped wiki-link | `href` |
| `HRefs` (`Link.Predicate != ""`) | inline citation-tagged wiki-link | *(exempt — D4)* |

**Rationale**: `Edges` is CORE's single unified list for both `edge`-role (flat bullet) and `link`-role
(grouped block) predicates as of spec 010/013 — the visual shape is a render-time decision, not a
parse-time distinction the AST preserves (`internal/core/ast.go`'s own doc comment: "regardless of
whether the source document wrote it as a flat bullet or grouped under a heading/bold label"). So a
predicate occurrence found in `Edges` satisfies the role check whether its registered role is `edge` or
`link`; the check only flags an `Edges` occurrence whose registered role is `meta`, `text`, or `href`
(i.e., structurally impossible categories for that position), and symmetrically flags a `meta`/`text`
occurrence whose registered role is `edge`/`link`/`href`, etc.

**Alternatives considered**: None — this mapping falls directly out of `core.Node`'s existing,
already-documented field semantics; no new parsing or AST change is introduced by this feature.

## D6: What existing fixtures does the new Requires/Optional check break, and where?

**Decision**: Two fixture surfaces need updating as part of implementation, identified by manually
walking each against the real `CoreTypeDefs`/`CorePredicateDefs` (spec 011 seed data):

1. **`cmd/arc/lint/lint_test.go`**: `conformantSource`/`conformantEntity` (used by `buildConformantGraph`,
   which most of that file's E2E tests build on). `conformantSource` (`@type: source`) is missing
   `abstract` (`CoreTypeDefs["source"].Required` includes it). `conformantEntity` (`@type: entity`) is
   missing `definition` and `mentionedIn` (both in `CoreTypeDefs["entity"].Required`), and separately
   carries `mentions` — which is not listed under `entity`'s `Required` *or* `Optional` — a second,
   independent FR-002 violation once the Optional check ships. This file already runs against real
   `appschema.Seed()` output (`initGraph`'s doc comment), so it exercises the real `CoreTypeDefs`
   contract, not a loosened stand-in.
2. **`internal/app/lint/service/lint_test.go`**: `coreIndexFixtureLint`, the package's own hand-built
   `core.Index{}}` fixture (deliberately loose — `TypeDef{Merge: ...}` values with zero-value
   `Required`/`Optional`, and `Predicates: map[string]core.PredicateDef{"mentions": {}}` — a zero-value
   `PredicateDef` with an empty `Role`). Both `conformantSourceFixture`/`conformantEntityFixture` use
   `mentions`; under FR-002 every predicate not listed in `Required`/`Optional` is a violation, and an
   empty `Required`/`Optional` list means *nothing* is listed — every node's every predicate would fail.

**Rationale**: Per the spec's own Edge Cases ("an absent `## Requires`/`## Optional` section means
'requires/permits nothing'") and the user's explicit constraint ("new rules must not report false
positives against every existing testdata fixture graph... plan should include a task to run the new
rules against testdata/ and fix any fixtures that were only conformant under the old, weaker checks"),
both fixture surfaces must be brought into genuine conformance with `CoreTypeDefs`/`CorePredicateDefs`
before the new checks ship, or every existing lint test that currently asserts "0 violations" on these
fixtures breaks. `coreIndexFixtureLint` additionally needs `mentions`'s `PredicateDef` given a real
`Role` (`"link"`, matching `CorePredicateDefs["mentions"]`) for the new role check (User Story 5) to have
anything meaningful to validate, and `"source"`/`"entity"` `TypeDef` entries need real `Required`/
`Optional` lists (the simplest correct fix: replace the package's hand-rolled `Types`/`Predicates` maps
with `kernel.CoreTypeDefs`/`kernel.CorePredicateDefs` directly via an import of
`internal/app/schema/kernel`, since that is already the graph's actual real seed data and duplicating a
second, parallel copy by hand is exactly the drift risk that produced this fixture gap in the first
place).

**Scope note**: `testdata/` (the repo-root fixture directory, `testdata/ctrl`) is unrelated to `arc
lint` — grep confirms no lint test reads from it. "Fixture graphs" in this feature's scope means the
two Go-source fixture builders above (the only places node content is built for a lint test run), not a
repo-wide fixture directory. Other commands' own test fixtures (`cmd/arc/graph/*_test.go`) never invoke
`service.Lint`/`applint.Lint`, so the new checks cannot fire against them regardless of fixture content;
they are out of this feature's blast radius.

## D7: How should an occurrence of a predicate with no recognized/blank `Role` be treated by the role check?

**Decision**: Skipped — no role-mismatch violation is reported for a predicate whose registry entry has
an empty or otherwise-unrecognized `Role` string (i.e., not one of `meta`/`text`/`href`/`edge`/`link`).

**Rationale**: Mirrors FR-009's existing "no registered schema at all → skip" rule for the same reason:
without a recognized declared role there is no expected position to compare the occurrence's actual
position against, so flagging it would be a false signal, not a real conformance failure. This case is
distinct from "unregistered predicate" (already caught by the pre-existing `predicateRegistered` check)
and is only reachable for a predicate that *is* registered but whose `role` schema field is missing or
was written as something outside the five-value vocabulary — itself arguably a schema-authoring defect,
but not one this feature's node-level checks are responsible for catching (research.md's own Assumptions
carry-over from spec.md: validating a type/predicate schema's own internal well-formedness beyond what's
needed to run node-level checks is out of scope).

## Summary of resulting technical decisions

- New file `internal/app/lint/service/rules_type_conformance.go` holds the three genuinely new rule
  functions: `checkTypeRequires`, `checkTypeOptional`, `checkPredicateRole` (User Stories 1, 2, 5).
- `internal/app/lint/service/rules_frontmatter.go` (the file containing today's front-matter/identity
  checks) gains `checkIdentityKeyQuoting` (User Story 3) plus two new small regex helpers in `locate.go`.
- `internal/app/lint/service/rules_predicates.go`'s `checkCitationPredicate` is rewritten in place (User
  Story 4); its hardcoded `citoPredicates` map is deleted.
- `internal/app/lint/service/lint.go`'s `Lint` orchestrator gains four new call sites (one per new rule
  function; `checkCitationPredicate`'s existing call site is updated in place for its new signature) in
  the existing "Checking predicates and citations" phase — no new Reporter phase label is needed, these
  checks belong to the same phase conceptually and the spec sets no distinct performance budget for them.
- `internal/app/lint/kernel/lint.go` gains four new `Rule` constants: `RuleTypeRequires`,
  `RuleTypeOptional`, `RuleIdentityQuoting`, `RulePredicateRole`.
- No new port, adapter, CLI flag, or Cobra command is introduced — this feature is additive validation
  logic inside the existing `arc lint` command and `internal/app/lint` domain package.

## Bugfix: BUG-001 — `kernel.CoreTypeDefs`/`CorePredicateDefs` seed-data gaps

**Reported**: 2026-07-09, after this feature's own implementation was complete and its new checks were run
against real graphs for the first time. Full detail in `bugs/BUG-001.md`; this section records the
concrete scope decisions spec.md's FR-014–FR-020 reference, per this feature's own convention of keeping
technology/decision-level detail out of spec.md's technology-agnostic requirements.

**D8 — Which predicates get added to which type's `## Optional`, concretely**:

| Predicate(s) | Added to `## Optional` of | Rationale |
|---|---|---|
| `tags`, `text`, `created`, `updated` | `source`, `entity`, `resource`, `timeline` (all four) | ARCNET-CORE §10.2/§10.3 give these no `Used by:` restriction, unlike §10.7's explicitly type-scoped predicates — read structurally as cross-cutting. User's original report explicitly requested "allowed to all classes/type." |
| `indexed` | `source`, `entity`, `resource`, `timeline` (all four) | New registration (D9 below); same cross-cutting treatment as the other Metadata/Control predicates, since spec 009 stamps it on every non-stub/non-schema node regardless of kind. |
| `published` | `entity`, `resource` only — **not** `timeline` | Spec 009 FR-001 enumerates exactly which kinds `arc apply` auto-stamps `published` on: "source, entity, resource, or a registered domain/extension kind" — `timeline` is conspicuously absent from that list, and ARCNET-CORE's own worked `timeline` type-schema example (§11.5) does not list `published` either. `source`'s existing `## Requires` entry is unaffected — this only adds `## Optional` elsewhere. |
| `mentions`, `mentionedIn` | `source`, `entity`, `resource`, `timeline` (all four) | User's original report explicitly requested "allowed to all core classes." This is a broader reading than §10.4's own `from → to` annotation (`source → entity` / `entity → source` specifically), which is why spec.md's FR-016 calls this out explicitly as broadening beyond the canonical direction, not silently expanding scope. |
| `notes` + all nine §10.5 semantic predicates (`broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`) | `entity` | Directly and unambiguously specified: ARCNET-CORE §11.3's own worked `entity` type-schema example ends its `## Optional` list with "any §10.5 semantic predicate, as applicable," and separately lists `notes`. `kernel.CoreTypeDefs["entity"].Optional` was `["aliases", "tags"]` only — this is the confirmed, unambiguous defect (not an interpretation) reproduced directly against a real graph (see BUG-001.md). |
| Same nine §10.5 semantic predicates | `resource` (additionally) | User's original report explicitly requested "allowed to all entities and resources." Not directly supported by §10.5's own prose ("written as `edge`-role predicates in the entity body," implying `entity` originates them and `resource` is merely a valid link *target*) — recorded here as a deliberate scope decision matching the user's explicit ask, not a spec-mandated reading. |

**D9 — Registering `indexed` (new `CorePredicateDefs` entry)**: `role: meta`, `merge: immutable`,
`aligned: "arc:indexed"`. Not part of `ARCNET-CORE.md` (confirmed absent by exhaustive grep of the fetched
spec text) — it is spec 009's own tool-native provenance timestamp (`specs/009-node-timestamp-attrs`),
already unconditionally written to every non-stub, non-schema node `arc apply` creates
(`internal/app/graph/service/apply.go`'s `setAttr(merged.Attrs, "indexed", stamp)`) but never registered.
`merge: immutable` matches spec 009 FR-006 exactly ("Once set at a node's creation, `indexed` MUST NOT be
modified by any later merge") — the same behavior already assigned to the pre-existing, but actually
unused in real code, `created` entry. `created` itself is left unchanged (not removed): it is legitimate
per ARCNET-CORE §10.3's own vocabulary even though no current `arc` command stamps it, and removing a
registered predicate is a destructive, unrequested change outside this bugfix's scope.

**D10 — Registering `scoreZ`/`scoreC` (new `CorePredicateDefs` entries)**: `role: meta`, `merge:
validatedOverwrite`, `aligned: "arc:scoreZ"` / `"arc:scoreC"` respectively. Confirmed absent from
`ARCNET-CORE.md` entirely (no heading, no inline mention anywhere in §9/§10/§11 or elsewhere) — this is a
local/tool-native extension, not an upstream-spec-backed predicate, hence the `arc:` alignment prefix
(CORE §9.1's own convention for a graph-native predicate with no external vocabulary term). `merge:
validatedOverwrite` is chosen because historical usage (`specs/003-apply-patch/bugs/BUG-003.md`,
`BUG-004.md`) describes `score-c`/`score-z` as "graph-analytics scores (e.g. centrality)... recomputed
from scratch by the ingesting pipeline on every run" — matching CORE §9.3's own description of
`validatedOverwrite` almost verbatim ("overwritten only by a designated validation pass... used by profile
predicates whose value is computed after the fact"). The hyphenated forms `score-z`/`score-c` were never
a formally registered predicate anywhere in the codebase (confirmed by repository-wide search) — they
appear only as illustrative attribute names in `cmd/arc/graph/apply_test.go`'s
`TestApplyEntityReContributionAppendsProseAndAccumulatesUnregisteredScalars`, whose entire point is to
exercise *unregistered*-predicate union-fallback dispatch. **That test's fixture is deliberately left
unchanged** — renaming its literal `score-c`/`score-z` keys to the newly-registered camelCase
`scoreZ`/`scoreC` would defeat the test's own purpose (it would stop exercising the unregistered-fallback
path entirely) and is out of this bugfix's scope.

**Not resolved by this bugfix — flagged, not silently decided**: whether `arc lint`'s existing
`predicateCase` check (camelCase enforcement) should also apply to front-matter `Attrs` keys, not just
`Edges`/`HRefs` predicate tokens (today's `predicateOccurrences` helper in `rules_predicates.go` only
walks `Edges`/`HRefs`) — this bugfix's own new predicates are all camelCase-correct as registered, so no
test currently exercises a hyphenated `Attrs` key being caught or missed by that check. Raising this here
so it isn't silently lost; a follow-up bug/feature should decide it explicitly rather than this bugfix
quietly expanding `predicateCase`'s scope as a side effect.

## Bugfix: BUG-002 — timeline period files never satisfy their own schema's Requires; the spec's own annotated-bullet format is unparseable

**Reported**: 2026-07-10. Full detail in `bugs/BUG-002.md`; this section records the concrete decisions
spec.md's FR-021–FR-024 reference.

**D11 — Registering `period`**: `role: meta`, `merge: immutable`, `aligned: "arc:period"`. Not part of
`ARCNET-CORE.md`'s own timeline worked example (§11.5) — confirmed by re-reading that section, which
shows only `granularity`/`entries` in front matter, no `period` field. `period` is purely an
`arc`-internal duplicate of a timeline node's own `@id` value, introduced by spec 003's BUG-007 fix
(`internal/app/graph/service/apply.go`'s `upsertTimelinePeriod` already writes
`period: "<value>"`, explicitly quoted, so a bare 4-digit yearly period doesn't decode as a YAML integer
the way an unquoted `@id` almost did before that fix). No code change is needed to *produce* `period` —
it is already written; only its schema registration was missing, the same category of gap as BUG-001's
`indexed`. Added to `CoreTypeDefs["timeline"].Required`, per the reporter's explicit instruction.

**D12 — `entries` → `cites` (explicit product decision, not a spec-conformance reading)**: The reporter
was offered a choice between keeping `entries` (matches `ARCNET-CORE.md` §11.5 verbatim, zero collision
risk) and reusing `cites` (the reporter's original preference, conflicting with §10.6's existing,
different meaning — a `source`'s own citation of an external `resource`). **The reporter chose to reuse
`cites`.** Consequences worked through and recorded here so the reasoning isn't rediscovered later:
- `CoreTypeDefs["timeline"].Required` changes from `["granularity", "entries"]` to
  `["granularity", "cites", "period"]` (period per D11).
- `CorePredicateDefs["entries"]` is **removed** (not merely deprecated/left registered): no real graph has
  ever contained a genuine `entries`-tagged edge (D13 explains why — `TimelineEntry` never wrote one), so
  no existing content depends on the name remaining registered; keeping an now-orphaned, never-actually-
  used predicate around is dead weight the constitution's YAGNI principle argues against.
- `CorePredicateDefs["cites"].Merge` changes from `MergeUnion` to `MergeAppend`. `entries`'s own
  (pre-removal) `Merge` was `MergeAppend` specifically because a timeline's chronological order matters
  ("ordered by date" — its own description). `cites`'s pre-existing `union` merge gives no ordering
  guarantee. Since `cites` now also serves the ordering-sensitive timeline role, its merge must become
  `append` — CORE §9.3's own description of `append` ("grow the predicate's content without discarding
  what is already there... an ordered, uniquely-keyed list gets a new entry") is a strict improvement over
  `union` for `cites`'s pre-existing `source`-node usage too (deterministic insertion order, still
  deduplicated), so this is a safe, compatible change for both usages, not a narrowing one.
- `CorePredicateDefs["cites"].Description` is broadened to name both usages (a source's own citation of an
  external resource, and a timeline's chronological reference to a source it contains), so a reader of the
  registered schema isn't misled by a description that only mentions one.
- `CorePredicateDefs["cites"].Role` (`link`) and `.Aligned` (`cito:cites`) are unchanged — both usages
  render as a grouped link-role block, and `cito:cites` remains an accurate external-vocabulary mapping
  for a timeline's own reference too (a timeline "cites" the sources it indexes, in the same loose sense
  CORE §12 already uses for reference relationships).

**D13 — Fixing the timeline-file writer to actually emit a `cites`-tagged edge**:
`internal/core.TimelineEntry` (`internal/core/timeline.go`) currently renders
`"- [[%s]] — *%s* (%s) — %s"` — a bare, untyped wikilink. Change to
`"- cites:: [[%s]] — *%s* (%s) — %s"`. Its own doc comment ("the timeline node's own Edges carry only the
bare target") predates `CoreTypeDefs["timeline"].Required` demanding this predicate (spec 011) and is now
corrected. `TestTimelineEntry` (`internal/core/timeline_test.go`) asserts the old bare-link string and
must be updated to the new `cites::`-prefixed one. Separately, `internal/app/graph/service/apply.go`'s
own `timelineEntryPattern` (`^- \[\[([^\]]+)\]\].* — (\d{4}-\d{2}-\d{2})$`) — used only to re-parse an
*existing* period file's already-written entries so `upsertTimelinePeriod` can insert a new one in
chronological order — must also accept the new `cites:: ` prefix: `^- (?:cites:: )?\[\[([^\]]+)\]\].* —
(\d{4}-\d{2}-\d{2})$`. The optional-prefix form (rather than requiring it) tolerates re-reading an
already-existing period file written before this fix lands, so an in-place upgrade doesn't lose or
duplicate historical entries on first re-application.

**D14 — Parser fix for predicate-tagged wikilinks followed by trailing annotation**:
`internal/core/markdown.go`'s `listItemPattern`
(`^(?:(\w+)::\s*)?\[\[([^\]|]+?)(?:\|([^\]]+?))?\]\]$`) anchors `$` immediately after the wikilink's
closing `]]`, rejecting any trailing text — confirmed by direct reproduction that ARCNET-CORE §11.5's own
literal worked-example line produces zero `Edges`/`HRefs` when parsed. Fix: relax the trailing anchor to
`(?:\s.*)?$` — i.e. nothing, or whitespace followed by anything — so
`predicate:: [[target]] — *annotation*` parses into `Link{Predicate: "predicate", Target: "target"}`
exactly as `predicate:: [[target]]` alone already does, with the trailing annotation discarded (it is
display-only decoration per CORE §11.5's own convention, never structured data — `internal/core.Node` has
no field to preserve it in, matching how the existing bare/aliased wikilink forms already discard
anything outside their own captured groups). Requiring a leading whitespace character before any trailing
content (rather than accepting anything immediately after `]]`) is a deliberate, minimal safety margin: a
genuinely malformed line like `[[Target]]garbage` (no space) still fails to match, rather than silently
absorbing an unintentional typo as if it were annotation. This fix is independent of D12's naming
decision — it is needed for any predicate-tagged, annotated bullet, not just a timeline's own references,
and was verified by re-parsing ARCNET-CORE's own worked example line successfully after the change.

**Not resolved by this bugfix — flagged, not silently decided**: whether the same trailing-annotation
tolerance should extend to `inlineLinkPattern` (`internal/core/markdown.go`, the inline
`[predicate:: [[Target]]]`/bare-wikilink-in-prose form) — the reported bug and its evidence concern only
list-item bullets (`listItemPattern`), and no test or real usage currently exercises trailing text after
an *inline* citation tag. Left unchanged; a follow-up bug should decide it explicitly if a real case
surfaces, rather than this bugfix silently broadening scope to a pattern with different surrounding
context (inline prose vs. a standalone list item).
