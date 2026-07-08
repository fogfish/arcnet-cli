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
heading; both `TLS.md`'s and `X.md`'s `mentions` occurrence(s) render under a `## Mentions` heading — the
same shape for `mentions` on both files, regardless of what else each node carries.

## Scenario B — single link-role predicate body omits its heading (spec User Story 2)

```sh
cat timeline/yearly/2026.md
```

**Expected**: the year's `entries` list (role `link`, and the only edge-bearing predicate `timeline` ever
carries) renders as a bare bulleted list directly under the node's leading prose — no `## Entries` heading.
Then hand-edit that same file to add an unrelated `edge`-role bullet under the same body (e.g. a `related::`
line to any existing node) and re-run:

```sh
arc apply patch-1.md   # any no-op-safe re-apply that triggers a re-render, or use arc subgraph to force a render
```

**Expected**: once a second predicate's occurrence is present in the body, `## Entries` reappears.

## Scenario C — normalization overrides a hand-written, non-canonical shape (spec User Story 3)

Hand-edit `entities/X.md` to write its `mentions` occurrence as a flat bullet (against its `link`-declared
role) instead of grouped, then force a re-render:

```sh
arc subgraph X --depth 0 > /tmp/x-subgraph.md
cat /tmp/x-subgraph.md
```

**Expected**: the re-rendered `mentions` occurrence is grouped under `## Mentions` — the hand-written flat
shape is corrected, not preserved.

## Scenario D — byte-stable round-trip on already-canonical content

```sh
cp entities/TLS.md /tmp/TLS.before.md
arc subgraph TLS --depth 0 > /tmp/TLS.subgraph.md
diff <(tail -n +2 /tmp/TLS.before.md) <(sed -n '/^## TLS$/,$p' /tmp/TLS.subgraph.md | tail -n +3)
```

**Expected**: no diff — a node already in canonical schema-driven shape re-renders identically.

## Scenario E — full command-level regression

```sh
go build ./... && go test ./... -cover
```

**Expected**: all packages pass, including `internal/core` (rewritten flat/grouped-shape and round-trip
tests per research.md D8), `internal/app/schema/service`, `internal/app/graph/service`, and `cmd/arc/graph`
(E2E, 1:1 with spec.md's acceptance scenarios per constitution Principle VIII).
