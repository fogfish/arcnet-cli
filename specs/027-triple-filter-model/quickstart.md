# Quickstart: Triple Filter for Node Attributes and Graph Edges

Validates this feature end-to-end against a real graph: predicate-scoped traversal (User Story 1), unified attribute-or-edge matching (User Story 2), and unchanged CLI filter flags (User Story 3).

## Prerequisites

- `go build ./...` succeeds at the feature branch tip.
- A scratch graph initialized via `arc init` (or reuse any existing test fixture graph under `cmd/arc/graph/testdata/`).

## Setup: a graph with more than one relation

```sh
arc init /tmp/triple-filter-demo
cd /tmp/triple-filter-demo
```

Author two `Source` nodes where one cites the other and also mentions a third node, so the seed carries connections of two distinct predicates. Structural connections (`Edges`) are authored in the body via `predicate:: [[target]]` bullets — never as a front-matter list, which always parses as an attribute fact (AST §4-6), not a relation fact:

```sh
cat > Source/paper-a.md <<'EOF'
---
"@id": paper-a
"@type": Source
title: Paper A
---
Paper A discusses foundational work.

- cites:: [[paper-b]]
- mentions:: [[paper-c]]
EOF

cat > Source/paper-b.md <<'EOF'
---
"@id": paper-b
"@type": Source
title: Paper B
---
Paper B is the cited work.
EOF

cat > Source/paper-c.md <<'EOF'
---
"@id": paper-c
"@type": Source
title: Paper C
---
Paper C is merely mentioned.
EOF

git add -A && git commit -m "seed demo graph" -q
```

## Scenario 1 — CLI: predicate-scoped expansion (User Story 1)

```sh
arc subgraph paper-a --predicate cites
```

**Expected**: output contains `paper-a` (seed) and `paper-b` (reached via `cites`) but **not** `paper-c` (reached only via `mentions`).

```sh
arc subgraph paper-a
```

**Expected**: output contains `paper-a`, `paper-b`, **and** `paper-c` — unscoped default, unchanged from before this feature (contracts/cli-contract.md).

## Scenario 2 — MCP: predicate-scoped `subgraph_get` (User Story 1)

Start the server (stdio) and, using any MCP-capable client or the test harness's `mcp.NewInMemoryTransports()` pattern already established in `cmd/arc/graph/serve_test.go`, call:

```json
{ "name": "subgraph_get", "arguments": { "id": "paper-a", "filter": { "statements": [ { "predicate": "cites" } ] } } }
```

**Expected**: reply's patch document contains `paper-a` and `paper-b`, excludes `paper-c` — matching Scenario 1's CLI result exactly (contracts/mcp-contract.md).

## Scenario 3 — Matching a node by its relation, not its attributes (User Story 2)

```json
{ "name": "node_grep", "arguments": { "pattern": ".", "filter": { "statements": [ { "predicate": "cites", "target": "paper-b" } ] } } }
```

**Expected**: only `paper-a` is scanned/matched — it is the only node carrying a `cites` fact whose target is `paper-b`; `paper-b`/`paper-c` are excluded even though the pattern `.` would otherwise match every line of every node.

## Scenario 4 — Existing CLI flags unchanged (User Story 3)

```sh
arc grep --type Source "Paper"
arc subgraph paper-a --type Source --tag nonexistent
```

**Expected**: identical output to the pre-feature binary for the same graph and flags — the first returns every `Source` node's matching lines; the second returns only `paper-a` (the seed; `--tag nonexistent` matches no reached node, exactly as an unmatched filter behaved before this feature).

## Validation checklist

- [X] Scenario 1's `--predicate cites` output excludes `paper-c`.
- [X] Scenario 1's unscoped output includes `paper-c`.
- [X] Scenario 2's `subgraph_get` result matches Scenario 1's CLI result exactly.
- [X] Scenario 3's `node_grep` result includes only `paper-a`.
- [X] Scenario 4 reproduces pre-feature output byte-for-byte.
- [X] `go test ./...` passes, including the regression suite proving `--type`/`--tag`/`--attr`/`--attr~=` are byte-identical to before this feature (plan.md Testing section).
