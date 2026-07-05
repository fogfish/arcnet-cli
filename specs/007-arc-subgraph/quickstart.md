# Quickstart: `arc subgraph`

Validates spec.md's three user stories end-to-end against a real local graph. `arc subgraph` needs no network access and makes no filesystem or git-history changes (spec FR-009, SC-006) — every scenario below can be re-run repeatedly with identical results.

## Prerequisites

- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- A graph created by `arc init` with at least one document ingested via `arc apply` (see `specs/003-apply-patch/quickstart.md` Scenario 1 for a ready-made setup), containing a seed node (`Transport Layer Security`) linked to a source (`rescorla-2026-tls13`)

## Scenario 1 — Pull a node and its immediate context into a portable document (spec.md User Story 1)

```sh
$ arc subgraph "Transport Layer Security"
---
kind: patch
document: subgraph:transport-layer-security@2026-07-04T12:00:00Z
published: 2026-07-04
title: "Subgraph: Transport Layer Security"
stats: {nodes: 2, directReachable: 0, directIncluded: 0, directTruncated: false, backlinkReachable: 1, backlinkIncluded: 1, backlinkTruncated: false}
---
# Entity

## Transport Layer Security
```yaml
id: Transport Layer Security
category: form structure attribute process
```

TLS is the successor to SSL.

# Source

## rescorla-2026-tls13
```yaml
id: rescorla-2026-tls13
title: TLS 1.3
```

TLS 1.3 is the latest version of the Transport Layer Security protocol.
$ echo $?
0
```

A seed node with no connections at all:

```sh
$ arc subgraph "Isolated Note"
---
kind: patch
document: subgraph:isolated-note@2026-07-04T12:00:05Z
published: 2026-07-04
title: "Subgraph: Isolated Note"
stats: {nodes: 1, directReachable: 0, directIncluded: 0, directTruncated: false, backlinkReachable: 0, backlinkIncluded: 0, backlinkTruncated: false}
---
# Entity

## Isolated Note
```yaml
id: Isolated Note
```
```

**Expected outcome**: the seed plus everything directly connected to it (in either direction) is included, grouped by kind, front-matter and body preserved verbatim (contracts/cli-contract.md); a seed with no connections still produces a valid one-node document, never an error.

Feeding the output back into the graph:

```sh
$ arc subgraph "Transport Layer Security" > /tmp/subgraph.md
$ arc apply /tmp/subgraph.md
```

**Expected outcome**: the extracted document is accepted as a structurally valid patch (spec FR-008, SC-005) — the round-trip `RenderPatch`/`ParsePatch` property research.md D2 documents.

A basename that does not exist:

```sh
$ arc subgraph "No Such Node"
❌ no node found with basename "No Such Node". Run `arc help subgraph` for guidance.
$ echo $?
1
```

## Scenario 2 — Widen or narrow the reach of the extraction (spec.md User Story 2)

```sh
$ arc subgraph "Transport Layer Security" --depth 2
```

**Expected outcome**: includes every node reachable within 2 hops (both directions) of the seed — a broader neighborhood than Scenario 1's default depth-1 extraction.

```sh
$ arc subgraph "Transport Layer Security" --depth 0
---
kind: patch
document: subgraph:transport-layer-security@2026-07-04T12:01:00Z
published: 2026-07-04
title: "Subgraph: Transport Layer Security"
stats: {nodes: 1, directReachable: 0, directIncluded: 0, directTruncated: false, backlinkReachable: 0, backlinkIncluded: 0, backlinkTruncated: false}
---
# Entity

## Transport Layer Security
```yaml
id: Transport Layer Security
category: form structure attribute process
```

TLS is the successor to SSL.
```

**Expected outcome**: `--depth 0` produces the seed alone, regardless of how connected it is; omitting `--depth` entirely behaves exactly like `--depth 1`.

## Scenario 3 — Keep the extraction focused with a filter (spec.md User Story 3)

```sh
$ arc subgraph "Transport Layer Security" --kind source
```

**Expected outcome**: the seed (an `entity`) is still included even though it does not match `--kind source`; only reachable `source` nodes are added alongside it — reusing the exact same `--kind`/`--tag`/`--attr` flags and composition rules `arc grep` already established (research.md D6).

```sh
$ arc subgraph "Transport Layer Security" --kind resource
---
kind: patch
document: subgraph:transport-layer-security@2026-07-04T12:02:00Z
published: 2026-07-04
title: "Subgraph: Transport Layer Security"
stats: {nodes: 1, directReachable: 0, directIncluded: 0, directTruncated: false, backlinkReachable: 1, backlinkIncluded: 0, backlinkTruncated: false}
---
# Entity

## Transport Layer Security
```yaml
id: Transport Layer Security
category: form structure attribute process
```

TLS is the successor to SSL.
```

**Expected outcome**: a filter matching none of the reachable nodes still produces a valid, non-empty (seed-only) document — never an error (spec Edge Cases).

## Configuring the traversal caps (research.md D6)

```sh
$ cat >> .arc/config.yml <<'EOF'
subgraph:
  directCap: 200
  backlinkCap: 50
EOF

$ arc subgraph "Transport Layer Security" --depth 3
```

**Expected outcome**: on a graph small enough that neither pool exceeds these lowered caps, output is unchanged; on a graph where a highly-referenced node's backlink pool exceeds `backlinkCap`, the retained nodes are exactly the highest-degree candidates (SC-007), the document's `stats` block reports `backlinkTruncated: true`, and a plain diagnostic line is printed to stderr (research.md D10) — stdout itself stays a clean, valid patch document either way.

## Verifying read-only behavior (spec SC-006)

```sh
$ git status --short > /tmp/before.txt
$ arc subgraph "Transport Layer Security" --depth 2 > /dev/null
$ git status --short > /tmp/after.txt
$ diff /tmp/before.txt /tmp/after.txt
# empty — arc subgraph changed nothing
```
