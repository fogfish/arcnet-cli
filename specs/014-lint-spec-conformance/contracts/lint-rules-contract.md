# Rule Contract: New/Changed `arc lint` Checks (CORE §16)

This feature adds no new CLI flag, exit code, or top-level output shape — the existing
`arc lint` CLI contract ([specs/004-arc-lint/contracts/cli-contract.md](../../004-arc-lint/contracts/cli-contract.md))
is unchanged: same usage line, same exit codes (`0`/`1`), same human/`--verbose`/`--json` renderers, same
`kernel.Violation`/`kernel.LintResult` JSON shape. This document is the addendum contract for the five
new rule identities that now appear inside that unchanged shape's `rule`/`message` fields.

## New/changed `Rule` values

| Rule string | Status | Reported when |
|---|---|---|
| `typeRequires` | New | A predicate the node's type's `## Requires` lists is absent from the node |
| `typeOptional` | New | A predicate the node carries is in neither its type's `## Requires` nor `## Optional` |
| `identityQuoting` | New | `"@id"`/`"@type"` is missing, or present but written as a bare (unquoted) key |
| `predicateRole` | New | A registered predicate's occurrence position doesn't match its schema-declared role |
| `citationPredicate` | Changed (message/source only, rule string unchanged) | A citation predicate is unregistered or not `cito:`-aligned — now sourced from the graph's schema, not a hardcoded list |

## Message templates

Each row is a `fmt.Sprintf`-style template; exact wording is illustrative and MAY be refined during
implementation, but MUST name the predicate/key involved (per spec FR-010) and MUST stay a single
human-readable sentence, consistent with every existing lint message's style (see
`internal/app/lint/service/rules_predicates.go`'s current messages for tone).

| Rule | Template | Example |
|---|---|---|
| `typeRequires` | `type %q requires predicate %q, but this node does not carry it` | `type "source" requires predicate "abstract", but this node does not carry it` |
| `typeOptional` | `predicate %q is not permitted by type %q (not listed under its Requires or Optional)` | `predicate "mentions" is not permitted by type "entity" (not listed under its Requires or Optional)` |
| `identityQuoting` (missing) | `front matter is missing the mandatory %q key` | `front matter is missing the mandatory "@type" key` |
| `identityQuoting` (unquoted) | `%q must be a quoted YAML string key, found it unquoted` | `"@id" must be a quoted YAML string key, found it unquoted` |
| `predicateRole` | `predicate %q is registered with role %q, but appears as a %s occurrence` | `predicate "abstract" is registered with role "text", but appears as a meta occurrence` |
| `citationPredicate` (unchanged wording, changed determination) | `citation predicate %q is not a recognized cito-aligned predicate` | `citation predicate "randomPredicate" is not a recognized cito-aligned predicate` |

## Location (`path`/`line`) contract

Every new rule follows the existing `Violation.Path`/`Violation.Line` convention
(`internal/app/lint/kernel/lint.go`):

| Rule | `Path` | `Line` |
|---|---|---|
| `typeRequires` | the node's file | The node's front-matter delimiter line (no single line "is" a missing predicate; falls back like other node-level-only violations, e.g. `unrecognizedKind`) |
| `typeOptional` | the node's file | The line the offending predicate occurrence is located at (via the existing `locatePredicateToken`/front-matter-field locate helpers, per occurrence category) |
| `identityQuoting` | the node's file | The line the `"@id"`/`"@type"` key appears on (or the front-matter delimiter line, if the key is missing entirely) |
| `predicateRole` | the node's file | The line the mismatched occurrence is located at |
| `citationPredicate` | the node's file | Unchanged — the line of the citation occurrence (already implemented) |

## Applicability preconditions (no violation fires; check is skipped, not "always passes")

| Rule | Skipped entirely when |
|---|---|
| `typeRequires` / `typeOptional` | The node's `@type` is not itself a key in `index.Types` (already reported once via the pre-existing `unrecognizedKind` rule) |
| `predicateRole` | The occurrence's predicate is not a key in `index.Predicates` (already reported once via the pre-existing `predicateRegistered` rule); OR the occurrence is an `HRefs` entry with a non-empty `Predicate` (the inline citation-tagging convention — governed by `citationPredicate` instead, see research.md D4); OR the registered `PredicateDef.Role` is empty/not one of `meta`/`text`/`href`/`edge`/`link` (research.md D7) |
| `identityQuoting` | Never skipped — runs for every node whose front matter was parseable enough to reach this check (a node that fails to parse at all is already reported via the pre-existing `frontMatter` rule and excluded from further checks, matching today's `continue`-on-parse-failure control flow in `service.Lint`) |
| `citationPredicate` | Unchanged — only evaluated for occurrences carrying a non-empty `Predicate` |

## Non-goals restated (explicit, per spec Out of Scope)

- No new `--fix`/auto-correct flag or behavior.
- No change to `arc apply`'s merge/write behavior.
- No change to `_schema/` document format, `## Requires`/`## Optional` authoring syntax, or the
  `role`/`aligned` predicate-schema fields themselves — this feature only adds *readers* of data
  `internal/app/schema` already resolves.
