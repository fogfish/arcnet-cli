# Quickstart: Validate the New §16 Conformance Checks

**Feature**: [spec.md](spec.md) | **Contract**: [contracts/lint-rules-contract.md](contracts/lint-rules-contract.md)

Manual/automated validation that the five new checks work end-to-end, against a real graph. Mirrors
`specs/004-arc-lint`'s existing E2E pattern (`cmd/arc/lint/lint_test.go`'s `buildConformantGraph`) — no
new tooling is introduced; this is a run-guide, not new test code (the actual tests live in
`internal/app/lint/service/*_test.go` and `cmd/arc/lint/lint_test.go` per tasks.md).

## Prerequisites

- Go toolchain matching `go.mod`.
- A real git binary on `PATH` (lint's `RuleIngestCommit` check shells out to `git log`).

## Setup: a real graph on disk

```sh
cd $(mktemp -d)
git init -q
arc init            # seeds .arc/, _schema/predicates/, _schema/types/ with CorePredicateDefs/CoreTypeDefs
```

## Scenario 1 — a genuinely conformant `source` node passes every new check (User Stories 1-5 negative case)

```sh
cat > sources/quickstart-2026-x.md <<'EOF'
---
"@id": "quickstart-2026-x"
"@type": source
title: "A Quickstart Document"
published: "2026-07-09"
---
# quickstart-2026-x

A test document used to validate the new arc lint conformance checks.

## Mentions
- mentions:: [[Widget]]
EOF
git add -A && git commit -q -m "graph(ingest): quickstart-2026-x — A Quickstart Document

Source-Id: quickstart-2026-x"

arc lint
```

**Expected**: clean pass — `abstract` (required, `text`-role) is present as the node's body prose, `title`/
`published` (required, `meta`-role) are present, `mentions` (required, `link`-role, rendered as an
`Edges` occurrence) is present; `"@id"`/`"@type"` are both present and quoted. No `typeRequires`,
`typeOptional`, `identityQuoting`, or `predicateRole` violation is reported.

## Scenario 2 — missing required predicate (User Story 1)

Remove the `## Mentions` block from `sources/quickstart-2026-x.md`, leaving `mentions` absent, then
re-run `arc lint`.

**Expected**: exactly one new violation:
```text
❌ sources/quickstart-2026-x.md — [typeRequires] type "source" requires predicate "mentions", but this node does not carry it
```

## Scenario 3 — predicate not permitted by type (User Story 2)

Restore the `## Mentions` block, then add an attribute the `source` type's schema does not list under
either `## Requires` or `## Optional` — e.g. `status: read` (a `resource`-type-only predicate) to the
front matter, then re-run `arc lint`.

**Expected**: exactly one new violation:
```text
❌ sources/quickstart-2026-x.md:N — [typeOptional] predicate "status" is not permitted by type "source" (not listed under its Requires or Optional)
```

## Scenario 4 — malformed identity key quoting (User Story 3)

Edit the front matter to remove quotes from `"@id"`, i.e. change `"@id": "quickstart-2026-x"` to
`@id: quickstart-2026-x`, then re-run `arc lint`.

**Expected**: an `identityQuoting` violation naming the key, in place of a raw YAML error:
```text
❌ sources/quickstart-2026-x.md:2 — [identityQuoting] "@id" must be a quoted YAML string key, found it unquoted
```
**Verified correction (research.md D1 is wrong on this point)**: a leading `@` is a reserved YAML
plain-scalar indicator character, so a bare `@id`/`@type` key is not "invisible post-parse" as D1
assumed — it makes the *entire document* invalid YAML, verified directly against `gopkg.in/yaml.v3`.
`core.ParseNode` therefore fails outright for this node (as it would for any malformed front matter);
`arc lint` intercepts that specific failure shape (a bare identity key, detected via raw-text regex,
independent of the parser's own generic error) and reports the friendly `identityQuoting` message in
place of the raw YAML lexer error, rather than letting a cryptic `frontMatter` violation surface. The
node does **not** otherwise parse — every other check that needs a successfully parsed node (including
`entities/Widget.md`'s own `derivedProvenance` check, which can no longer see `quickstart-2026-x` as a
recognized `source`-kind node) is affected too, so this scenario's real output also includes a
secondary `derivedProvenance` violation on `entities/Widget.md`. This is the correct, verified behavior,
not a regression — CORE §10.1's quoting requirement exists precisely because an unquoted identity key
is not valid YAML at all, not because of some other, more forgiving parser behavior.

## Scenario 5 — domain-registered `cito:`-aligned citation predicate (User Story 4)

```sh
cat > _schema/predicates/citesAsExample.md <<'EOF'
---
"@id": "citesAsExample"
"@type": Property
role: edge
merge: union
aligned: "cito:citesAsExample"
---
# citesAsExample

A domain-specific citation relationship, not built into arc itself.
EOF
git add -A && git commit -q -m "seed: register citesAsExample predicate"
```

Add an inline citation using it inside `sources/quickstart-2026-x.md`'s prose (e.g.
`[citesAsExample:: [[Widget]]]`), then re-run `arc lint`.

**Expected**: no `citationPredicate` violation for this usage — `arc` never needed a code change to
recognize `citesAsExample` as valid, since its `aligned` field is `cito:`-prefixed in the graph's own
schema. Using an unregistered or non-`cito:`-aligned predicate the same way still reports
`[citationPredicate]`, unchanged from today.

**Verified**: the run also reports `sources/quickstart-2026-x.md:N — [typeOptional] predicate
"citesAsExample" is not permitted by type "source" (not listed under its Requires or Optional)` — this
is correct, independent behavior (User Story 2), not a defect: `citesAsExample` is a citation-tagged
inline occurrence, which still counts as "present on the node" for the Requires/Optional contract
(spec Assumptions), and `source`'s type schema was never updated to list it under `## Optional`. The
acceptance criterion this scenario targets is specifically the *absence* of a `citationPredicate`
violation, not a zero-violation run.

## Scenario 6 — predicate used outside its declared structural role (User Story 5)

Add `- broader:: [[SomeType]]` as prose text inside `abstract`'s paragraph rather than as a body edge
bullet is not directly authorable this way (parsing rules already route bullets to `Edges`); instead,
demonstrate the reverse direction: add `category: [independent, abstract, occurrent, script]` (a `meta`-role
predicate on a `source` node, which doesn't register `category` under Required/Optional so `typeOptional`
fires) — for a role-specific violation, register a *new* predicate as `role: text` in
`_schema/predicates/`, add it to `entity`'s `## Optional`, then use it as a body edge bullet
(`- newPredicate:: [[Target]]`) on an `entity` node instead of prose.

**Expected**:
```text
❌ entities/Widget.md:N — [predicateRole] predicate "newPredicate" is registered with role "text", but appears as a edge occurrence
```

## Full-suite validation

```sh
go test ./internal/app/lint/... ./cmd/arc/lint/... -run . -v
go test ./... -cover
```

**Expected**: all tests pass, including every pre-existing lint test (per research.md D6, the
`conformantSource`/`conformantEntity`/`coreIndexFixtureLint` fixtures were brought into genuine
conformance with `CoreTypeDefs`/`CorePredicateDefs` as part of this feature's implementation — no
pre-existing test should newly fail).
