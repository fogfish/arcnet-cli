# Phase 0 Research: Lint Conformance Gaps

**Feature**: `024-lint-conformance-gaps` · **Date**: 2026-08-23

No `[NEEDS CLARIFICATION]` markers remain in spec.md, so this phase resolves *design* unknowns
against the existing codebase rather than requirement ambiguity — the six decisions below are
what the Technical Context in plan.md relies on.

---

## D1 — Sowa tuple validation replaces four independent word-set checks

**Decision**: Replace `sowaPosition1`/`sowaPosition2`/`sowaPosition3`/`sowaLeaf`
(`internal/app/lint/kernel/lint.go:121-130`) with one literal `[12][4]string` table. `ValidSowaCategory`
keeps its existing signature (`func([]string) (ok bool, reason string)`, `internal/app/lint/kernel/lint.go:135`,
consumed by exactly one call site, `rules_identity.go:61`) — length check unchanged, then an exact
row match against the table instead of four independent membership tests.

**Rationale**: `ValidSowaCategory` has one call site and one exported symbol; the table is a pure
value with no other consumer, so it stays in `lint/kernel` (existing package boundary), not a new
package.

**Alternatives considered**: A `map[[4]string]bool` keyed by array — rejected, because FR-002 needs
to iterate the twelve rows in order to compute the "closest" suggestion (D6), so a slice/array the
suggestion logic can range over is more useful than a set.

## D2 — "Closest legal combination" is longest-common-prefix, first match wins ties

**Decision**: Given a rejected 4-word tuple, scan the twelve rows in table order, count how many
leading words match exactly, keep the row with the highest count; the first row reaching a given
count wins on a tie.

**Rationale**: Assumption already recorded in spec.md — the simplest rule that satisfies "obvious
fix." The twelve rows are literally ARCNET-CORE §10.2's own table order (Appendix A.4 of
`CORE-FIX.md`), so "first match wins" is deterministic and reproducible without a documented
tie-break rule of its own.

**Alternatives considered**: Edit distance over all four words — explicitly rejected by the
feature's own plan input as unnecessary complexity for a 12-row table.

## D3 — Identity charset check reuses `checkSchemaTypeCase`'s graph-spanning pattern for schema nodes

**Decision**: Confirmed via code research: `core.Index` (`internal/core/rules.go:47-50`) carries only
decoded `Predicates`/`Types` maps — no path, no raw bytes, no line numbers — populated by
`schema/service.Resolve`, which discards raw bytes after parsing. `checkSchemaTypeCase`
(`internal/app/lint/service/rules_types_case.go:42-62`) already reports a **graph-spanning**
violation per `index.Types` key with a **synthesized** path (`TypesDir + "/" + name + ".md"`) and
`Line: 0` — no real file is opened.

The new identity-charset check follows the identical shape: `checkSchemaIdentityCharset(index
core.Index) []kernel.Violation`, iterating sorted `index.Types` keys then sorted `index.Predicates`
keys, synthesizing `Path` from `schemakernel.TypesDir`/`PredicatesDir`, `Line: 0`.

**Rationale**: This closes the "may be a larger change than it appears" concern raised in the
feature's own plan input. It is not a larger change: `checkSchemaTypeCase` already established
that a graph-spanning schema check needs no raw bytes, because FR-005's "1-indexed position" is
computed over the identity *string itself*, not a file offset — a synthesized path and `Line: 0`
already satisfies every other graph-spanning rule (`RuleUniqueBasename`, `RuleTypeCase`) today,
per `graphSpanningViolations`' own doc comment in `cmd/arc/lint/lint.go:41-58`.

**Correction to the feature's own plan input**: the schema folders are `_schema/Class` and
`_schema/Property` (`schemakernel.TypesDir`/`PredicatesDir`, `internal/app/schema/kernel/schema.go:30-31`),
not `_schema/types`/`_schema/predicates`.

**Note for a future feature, not this one**: schema resolution never verifies a `Class`/`Property`
document's `@id` matches its own filename — unlike content nodes (`lint.go:105-111`). The new
graph-spanning check inherits this same trust (it reports the identity string, not a verified file
location), consistent with `checkSchemaTypeCase`'s existing behavior. Fixing that separate,
pre-existing gap is out of scope here.

**Alternatives considered**: Adding a new recursive `_schema/` file walker (mirroring
`walkNodeFiles`) to get real paths and line numbers for schema nodes — rejected as unnecessary
weight; no other graph-spanning rule does this, and FR-005 does not require a line number, only a
character and a position within the identity value.

## D4 — Line-locating for the content-node check uses the quoted `"@id"` key, not `rules_identity.go`'s bare `"id"`

**Decision**: The new `checkIdentityCharset` (content-node variant) locates its violation's line
via `locateFrontMatterField(raw, `"@id"`)` (quoted key) — the same call `lint.go:107` already uses
for the `@id`-basename-mismatch check — not `rules_identity.go:28`'s `locateFrontMatterField(raw,
"id")` (bare, unquoted).

**Rationale**: Every node's front matter stores the key quoted (`"@id": ...`, spec 018/019's
`@`-prefix quoting rule). `locateFrontMatterField`'s regex anchors on the literal key text
(`internal/app/lint/service/locate.go:47-56`), so a bare `"id"` pattern never matches a line
starting with `"@id":` and silently falls back to the front-matter delimiter line — an existing,
pre-dating imprecision in `checkSourceCitekey`'s own line-locating, not a bug this feature
introduces. The new check must not copy it. Fixing `rules_identity.go:28` itself is a one-line,
unrelated cleanup — out of scope, noted here only so it is not repeated.

## D5 — Position is a 1-indexed rune index, not a byte offset

**Decision**: Forbidden-character position (FR-002, FR-005) is computed by iterating
`[]rune(identity)`, not `[]byte(identity)`.

**Rationale**: All ten forbidden characters are single-byte ASCII, but an identity containing an
earlier multi-byte UTF-8 character (permitted — CORE identities are free text, e.g. `§11.6`
document titles) would make a byte offset disagree with what a user counts by eye. Rune indexing
costs nothing extra here (identities are short) and avoids a class of off-by-N reports on
non-ASCII identities, which spec.md's Edge Cases already anticipates for Unicode look-alikes
(out of scope for *detecting* those, but the position of a genuine ASCII forbidden character must
still be reported correctly regardless of what non-ASCII text precedes it).

## D6 — The forbidden-character scan is one shared primitive, consumed by both `arc lint` and `arc apply`

**Decision**: The character set and the scan-for-violations logic live once, in `internal/core`
(alongside the existing `isCamelCase`/`ErrTypeCasing` precedent at `internal/core/markdown.go:296-297,359`
and `internal/core/errors.go:60`), not duplicated between `internal/app/lint/service` and
`internal/app/graph/service`.

**Rationale**: FR-010 requires the forbidden set to be a single closed list; a shared primitive in
`internal/core` — the one package both `internal/app/lint` and `internal/app/graph/service`
already depend on — is the only way to guarantee `arc lint` and `arc apply` can never drift onto
two different character sets. It mirrors the precedent set by spec 019: CamelCase type-name
validation already lives in `internal/core` and is consumed by both the parser (hard rejection)
and, indirectly, by lint's own reporting. `internal/app/lint` and `internal/app/graph/service` have
no dependency on each other today and none should be introduced merely to share a ten-character
set.

**Consumers**:
- `internal/app/lint/service/rules_identity.go` — reports a `RuleIdentityCharset` violation per
  offending content node (does not stop the lint walk).
- `internal/app/lint/service/rules_types_case.go`-adjacent (D3) — reports the same rule,
  graph-spanning, for schema node identities.
- `internal/app/graph/service/apply.go` — rejects the whole `arc apply` operation (D7).

## D7 — `arc apply` enforcement covers patch-carried node identities and auto-registered schema-node identities; `arc apply-schema` is out of scope

**Decision**: Two enforcement points inside `Apply` (`internal/app/graph/service/apply.go:215`):

1. A pre-scan over `patch.Nodes` immediately after `readPatch` succeeds (`apply.go:229`) and before
   the main per-node loop begins any write — mirrors `guardNoOldFormatNodes`'s existing shape
   (scan everything, abort before the first write) but scans the **incoming patch**, not the
   existing graph. Checks every `node.ID`.
2. Inside the main per-node loop (`apply.go:262` onward), before `schema.RegisterType(store,
   node.Type)` (`apply.go:275`) and before `schema.RegisterPredicate(store, obs.name, ...)`
   (`apply.go:317`) — because both calls synthesize and write a **new schema-node identity**
   (`node.Type` becomes a `Class` document's `@id`; `obs.name` becomes a `Property` document's
   `@id`, per `RegisterType`/`RegisterPredicate`, `internal/app/schema/service/schema.go:391,416`).
   A violation here follows the loop's existing failure shape exactly:
   `reporter.Error(...)` → `rollback(store, createdPaths)` → return err.

**Rationale (why not one single check)**: content-node identities (`node.ID`) are all known before
any write begins, so a pre-scan gives the fastest, cleanest all-or-nothing rejection (FR-003).
Schema-node identities implied by `node.Type`/predicate names are only fully known progressively,
as each node in the patch is processed — but they are still writes the existing `createdPaths`
rollback mechanism already unwinds on any later error, so checking them in-loop is both correct
and requires no new rollback plumbing.

**Scope boundary — confirmed, not assumed**: `arc apply` (this feature's Input names it
literally) has no code path that parses an explicit `# Property`/`# Class` H1 section from a
content patch; schema documents are only ever (a) auto-registered by name as above, or (b)
authored through the separate `arc apply-schema` command
(`cmd/arc/ctrl/apply_schema.go`, `internal/app/schema/service/apply.go`) — a different pipeline
this feature does not touch. Spec.md's Acceptance Scenario 1.4 ("a patch that introduces a schema
node... whose identity contains a forbidden character") is satisfied by the `node.Type`/predicate-name
check above, not by a Property/Class-section parser that does not exist in `arc apply`.

**Alternatives considered**: Extending `arc apply-schema` in the same feature — rejected; the
spec's Input and every FR name `arc apply` specifically, and `arc apply-schema` is a materially
different command with its own patch format. Left as a residual gap for a future feature, not
silently folded in.

## D8 — Message wording is shared between the lint violation and the apply rejection

**Decision**: Both surfaces render the same detail string per identity (`identity %q contains
forbidden character %q at position %d` — joined for more than one offending character), produced
by the D6 shared primitive.

**Rationale**: Precedent — `ErrIdentityQuoting`'s message is already worded to match
`RuleIdentityQuoting`'s lint message verbatim (`internal/core/errors.go:55`), establishing that a
core/apply-time error and a lint violation for the same underlying rule share identical wording in
this codebase. A user who sees the `arc apply` rejection today and later runs `arc lint` on a
graph with a legacy violation of the same kind should recognize it as the same rule.

---

## Summary of resolved unknowns

| # | Question | Resolution |
|---|----------|------------|
| D1 | Where does the 12-row table live? | `lint/kernel`, replacing the four word-set maps |
| D2 | How is "closest" computed? | Longest common prefix, table order breaks ties |
| D3 | How are schema nodes reached without a new walker? | Reuse `index.Types`/`index.Predicates` keys, `checkSchemaTypeCase`'s synthesized-path/`Line:0` pattern |
| D4 | Which `locateFrontMatterField` key? | Quoted `"@id"`, matching `lint.go:107`, not `rules_identity.go:28`'s bare `"id"` |
| D5 | Byte or rune position? | Rune, 1-indexed |
| D6 | One check or two (lint + apply)? | One shared primitive in `internal/core`, two call sites |
| D7 | Does `arc apply` enforcement reach schema-node identities, and does it touch `arc apply-schema`? | Yes via `node.Type`/predicate-name checks in-loop; `arc apply-schema` is out of scope |
| D8 | Do the two surfaces word violations identically? | Yes, precedent from `ErrIdentityQuoting`/`RuleIdentityQuoting` |
