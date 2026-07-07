# Quickstart: Predicate-First Graph Node Model

Validates spec.md's three user stories against a real local graph. Builds
on `specs/003-apply-patch/quickstart.md`'s setup — this feature changes the
shape `arc apply` writes into node front matter/body and what every other
command reads back, not how any command is invoked (no flag/command
changes, plan.md Constraints).

## Prerequisites

- `git` on `PATH`
- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout

## Setup

```sh
$ arc init ./demo-graph && cd demo-graph
```

## Scenario 1 — A node round-trips with `"@id"`/`"@type"`, named prose, list attrs, and one unified edge list (User Story 1)

```sh
$ cat > tls13.patch.md <<'EOF'
---
kind: patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
```yaml
title: "TLS 1.3: Design and Rationale"
authors: [Eric Rescorla, David Benjamin]
```
This is the abstract of the TLS 1.3 design paper.

# Entity

## tls-1.3
```yaml
category: [independent, abstract, occurrent, script]
tags: [cryptography]
```
A cryptographic protocol establishing authenticated, confidential channels.

- replaces:: [[SSL Protocol]]
- conformsTo:: [[RFC 8446]]

## Mentions
- mentionedIn:: [[rescorla-2026-tls13]]
EOF

$ arc apply tls13.patch.md
✅ Ingested rescorla-2026-tls13: source: +1 created, entity: +1 created (commit a1b2c3d)

$ cat sources/rescorla-2026-tls13.md
---
"@id": rescorla-2026-tls13
"@type": source
authors: [Eric Rescorla, David Benjamin]
title: "TLS 1.3: Design and Rationale"
---
# rescorla-2026-tls13

This is the abstract of the TLS 1.3 design paper.

$ cat entities/tls-1.3.md
---
"@id": tls-1.3
"@type": entity
category: [independent, abstract, occurrent, script]
tags: cryptography
---
# tls-1.3

A cryptographic protocol establishing authenticated, confidential channels.

- replaces:: [[SSL Protocol]]
- conformsTo:: [[RFC 8446]]
- mentionedIn:: [[rescorla-2026-tls13]]
```

Expected:
- Both files declare `"@id"` (equal to their own basename) and `"@type"`
  explicitly, quoted — no `kind`/`id` fallback fields remain (spec
  FR-001/FR-002/FR-003).
- `tags` had a single value (`cryptography`) in the patch's list syntax and
  renders back as a bare scalar on disk, while `category`'s four values
  stay a list — both are held internally as `[]Predicate`, cardinality is
  a rendering choice, not a representation one (spec FR-004, research.md
  D3).
- `tls-1.3`'s leading prose became its `abstract`/`definition`-equivalent
  text field (here, `entity`'s own paragraph is the node's sole prose —
  see Scenario 2 for a node whose `Texts` map has more than one key); the
  `## Mentions` heading's grouped link and the bare `- replaces::`/
  `- conformsTo::` bullets all land in one flat list on re-render (spec
  FR-007/FR-008) — the heading is gone, the connectivity is not.
- Re-running `arc apply` with an empty follow-up patch and diffing
  `entities/tls-1.3.md` against itself before/after shows zero byte
  changes (spec FR-015, idempotent round-trip).

## Scenario 2 — Every command operates correctly against the predicate-first graph (User Story 2)

```sh
$ arc lint
✅ 2 nodes checked, 0 issues

$ arc grep "cryptographic protocol"
entities/tls-1.3.md: A cryptographic protocol establishing authenticated, confidential channels.

$ arc subgraph tls-1.3 --depth 1 --json
{
  "patch": {
    "document": "subgraph:tls-1-3@2026-07-07T12:00:00Z",
    "published": "2026-07-07T00:00:00Z",
    "nodes": [
      {
        "id": "tls-1.3",
        "type": "entity",
        "attrs": {
          "category": [{"value": "independent"}, {"value": "abstract"}, {"value": "occurrent"}, {"value": "script"}],
          "tags": [{"value": "cryptography"}]
        },
        "texts": { "definition": "A cryptographic protocol establishing authenticated, confidential channels." },
        "edges": [
          {"predicate": "replaces", "target": "SSL Protocol"},
          {"predicate": "conformsTo", "target": "RFC 8446"},
          {"predicate": "mentionedIn", "target": "rescorla-2026-tls13"}
        ]
      }
    ]
  }
}
```

Expected: `arc lint` evaluates the real `attrs`/`edges` (not a stale
two-slot shape); `arc grep` matches content inside a named `texts` field
regardless of its key name; `arc subgraph --json` exposes `attrs` as lists,
`texts` by name, and one unified `edges` array — consistent with what
`arc subgraph` (Markdown form, no `--json`) would round-trip to (spec
FR-017, contracts/subgraph-json-contract.md).

## Scenario 3 — An old-format graph fails safely instead of being misread (User Story 3)

```sh
$ mkdir -p legacy-graph/entities
$ cat > legacy-graph/entities/old-node.md <<'EOF'
---
kind: entity
id: old-node
category: [independent]
---
# old-node

An entity written before this feature shipped.
EOF

$ cd legacy-graph && arc lint
❌ entities/old-node.md: unsupported node format — front matter declares "kind" (a pre-0.5 identity field); this graph requires "@id"/"@type" instead. Run `arc help lint` for guidance.
$ echo $?
1

$ arc apply ../tls13.patch.md
❌ entities/old-node.md: unsupported node format — front matter declares "kind" (a pre-0.5 identity field); this graph requires "@id"/"@type" instead. Run `arc help apply` for guidance.
$ echo $?
1
$ git log --oneline -1
# unchanged — no commit was made
```

Expected: every command that reads `old-node.md` exits non-zero with a
message identifying the exact file and the exact unsupported field, makes
no write, and creates no commit (spec FR-012/FR-013, US3 Acceptance
Scenarios 1 and 4). The same failure mode applies uniformly whether
`"@id"` is missing, `"@type"` is missing, or `"@id"` mismatches the file's
basename (US3 Acceptance Scenarios 2 and 3) — not exercised verbatim above
for brevity, but covered by the E2E cases enumerated in plan.md's Project
Structure (`cmd/arc/graph/apply_test.go`, `cmd/arc/lint`'s suite).
