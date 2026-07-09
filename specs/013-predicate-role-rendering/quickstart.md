# Quickstart: Validating Schema-Driven Link Rendering

## Prerequisites

- A built `arc` binary (`go build ./cmd/arc`) from this feature's branch.
- A scratch directory outside this repo, with git available on `PATH` (`arc init`/`arc apply` shell out to
  git commit).

## Scenario A — same predicate, same shape, everywhere it appears (spec User Story 1)

```sh
cd /tmp && mkdir arc-quickstart && cd arc-quickstart
arc init
```

Confirm `_schema/predicates/mentions.md` declares `role: link` and `_schema/predicates/replaces.md` declares
`role: edge`.

Write and apply a patch contributing an `entity` node `TLS` with a `replaces` edge to `SSL Protocol` and a
`mentions` link to `A`, then a second patch contributing a different `entity` node `X` with only a `mentions`
link to `B`:

```sh
arc apply patch-1.md
arc apply patch-2.md
cat entities/TLS.md entities/X.md
```

**Expected**: `TLS.md`'s `replaces` occurrence is a flat `- replaces:: [[SSL Protocol]]` bullet with no
heading; `TLS.md`'s `mentions` occurrence renders under a `## Mentions` heading, since `replaces` is also
present on that node. `X.md`'s `mentions` occurrence is `X`'s *only* predicate occurrence, so User Story 2's
single-link-role-predicate-body omission (spec Edge Cases) applies and its heading is omitted — the same
predicate still renders identically wherever else it appears alongside other content (spec's own resolved
Edge Case: a node whose only content happens to be one link-role predicate's occurrences is indistinguishable,
at render time, from a type that only ever allows that one predicate).

## Scenario B — single link-role predicate body omits its heading (spec User Story 2)

```sh
cat timeline/yearly/2026.md
```

**Expected**: the year's entries list renders as a bare bulleted list directly under the node's leading
prose — no `## Entries` heading. In today's codebase this holds for two independent reasons, not just the
omission rule: `arc apply`'s timeline writer (`applyTimeline`/`upsertTimelinePeriod`,
`internal/app/graph/service/apply.go`) uses its own specialized bullet format (CORE §9.4, a bare
`- [[id]] — *title* (authors) — date` line with no `entries::` predicate prefix), bypassing
`core.RenderNode`/`renderNodeBody` entirely — so this file is never schema-rendered in the first place. The
general single-link-role-predicate-body omission rule this feature implements is instead verified directly
against `core.RenderNode` in `internal/core/markdown_test.go`
(`TestRenderNodeSingleLinkRolePredicateBodyOmitsHeading`/
`TestRenderNodeSingleLinkRolePredicateHeadingReappearsWithOtherContent`) and end-to-end for `arc apply`'s own
timeline output in `cmd/arc/graph/apply_test.go`
(`TestApplyCreatesTimelineEntriesChronologically`'s no-`"## "`-anywhere assertion) — there is no live
hand-edit-and-re-render CLI demo for this scenario, since `arc subgraph`'s own traversal does not include a
timeline node's body regardless of predicate shape (a separate, pre-existing behavior this feature does not
change).

## Scenario C — normalization overrides a hand-written, non-canonical shape (spec User Story 3)

`X.md`'s own `mentions` occurrence is already flat, but only because it is `X`'s *sole* predicate occurrence
(Scenario A's omission rule) — re-rendering it unchanged would stay flat either way, so it cannot demonstrate
correction on its own. Hand-edit `entities/X.md` to add a second, unrelated bullet (any `edge`-role predicate,
e.g. `- replaces:: [[Old X]]`) alongside the existing flat `mentions` bullet — this takes `mentions` out of
the single-predicate-body omission case, so its `link`-declared role is what now decides its shape — then
force a re-render:

```sh
arc subgraph X --depth 0 > /tmp/x-subgraph.md
cat /tmp/x-subgraph.md
```

**Expected**: the re-rendered `mentions` occurrence is grouped under `## Mentions`, while `replaces` stays a
flat bullet — the hand-written flat shape for `mentions` is corrected, not preserved.

## Scenario D — byte-stable round-trip on already-canonical content

```sh
cp entities/TLS.md /tmp/TLS.before.md
arc subgraph TLS --depth 0 > /tmp/TLS.subgraph.md
diff <(sed -n '/^# TLS$/,$p' /tmp/TLS.before.md | tail -n +2) \
     <(awk 'found && /^```$/{start=1; next} start{print} /^## TLS$/{found=1}' /tmp/TLS.subgraph.md)
```

**Expected**: no diff — a node already in canonical schema-driven shape (its `mentions` occurrence already
grouped under `## Mentions`, its `replaces` occurrence already a flat bullet) re-renders body content
byte-identically; the two files' surrounding front matter/heading shape necessarily differs (a standalone
node file's own `---`-delimited front matter plus `# TLS` H1 vs. a patch section's `## TLS` H2 plus fenced
yaml block), which is why the comparison extracts each file's body content first, rather than diffing whole
files.

## Scenario E — full command-level regression

```sh
go build ./... && go test ./... -cover
```

**Expected**: all packages pass, including `internal/core` (rewritten flat/grouped-shape and round-trip
tests per research.md D8), `internal/app/schema/service`, `internal/app/graph/service`, and `cmd/arc/graph`
(E2E, 1:1 with spec.md's acceptance scenarios per constitution Principle VIII).
