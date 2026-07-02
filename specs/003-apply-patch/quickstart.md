# Quickstart: `arc apply`

Validates spec.md's four user stories end-to-end against a real local graph and real `git`. `arc apply` itself needs no network access. `arc init` (Setup, below) makes one best-effort network call to seed `.arc/config.yml`'s defaults and always succeeds locally even if that call fails — see `specs/002-arc-init/spec.md` FR-017 and this feature's research.md D5 (revised).

## Prerequisites

- `git` on `PATH`
- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout

## Setup

```sh
$ arc init ./demo-graph
✅ Initialized empty knowledge graph at .../demo-graph (commit a1b2c3d)

$ cd demo-graph
$ cat .arc/config.yml
mergeRules:
  source: none
  entity: union
  resource: union-first-writer
```

## Scenario 1 — Ingest a brand-new document (spec.md User Story 1)

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
authors: [Eric Rescorla]
published: 2026-04-12
url: https://example.org/tls13-design
```
A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]

# Entity

## Transport Layer Security
```yaml
category: [independent, abstract, occurrent, script]
```
A cryptographic protocol that establishes an authenticated, confidential channel.
EOF

$ arc apply tls13.patch.md
✅ Applied rescorla-2026-tls13: +1 source, +1 entity (commit b2c3d4e)
```

**Expected outcome**: `sources/rescorla-2026-tls13.md` and `entities/Transport Layer Security.md` exist; `timeline/yearly/2026.md` and `timeline/monthly/2026-04.md` exist and reference the new source; exactly one new git commit, subject `graph(ingest): rescorla-2026-tls13 — TLS 1.3: Design and Rationale`, with a `Source-Id: rescorla-2026-tls13` trailer.

## Scenario 2 — Merge into overlapping content (spec.md User Story 2)

```sh
$ cat > pqkex.patch.md <<'EOF'
---
kind: patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
```yaml
title: "Post-Quantum Key Exchange in Practice"
authors: [Lin Chen]
published: 2026-04-28
```
Surveys post-quantum key exchange deployment.

# Entity

## Transport Layer Security
```yaml
category: [independent, abstract, occurrent, script]
```
A cryptographic protocol.
- requires:: [[Forward Secrecy]]
EOF

$ arc apply pqkex.patch.md
✅ Applied chen-2026-pqkex: +1 source, +0 entities (1 merged) (commit c3d4e5f)
```

**Expected outcome**: `entities/Transport Layer Security.md` is unchanged as a file (no duplicate created) but now also carries `requires:: [[Forward Secrecy]]`; `timeline/monthly/2026-04.md` now lists both sources, in date order.

## Scenario 3 — Domain-specific kind, registered (spec.md User Story 3)

```sh
$ cat >> .arc/config.yml <<'EOF'
  hypothesis: validated-overwrite
EOF

$ cat > note.patch.md <<'EOF'
---
kind: patch
document: kolesnikov-2026-note
published: 2026-05-01
title: "A Working Note"
---
# Source

## kolesnikov-2026-note
```yaml
title: "A Working Note"
published: 2026-05-01
```
A short note.

# Hypothesis

## Forward Secrecy Requires Ephemeral Keys
```yaml
```
A conclusion distilled from sources.
EOF

$ arc apply note.patch.md
✅ Applied kolesnikov-2026-note: +1 source, +1 hypothesis (commit d4e5f6a)
```

Without the `.arc/config.yml` edit, the same patch still applies — using the safe `union` default instead of `hypothesis`'s intended `validated-overwrite` behavior — and warns instead of refusing:

```sh
$ arc apply note.patch.md
✅ Applied kolesnikov-2026-note: +1 source, +1 hypothesis (commit d4e5f6a)
🟧 hypothesis is not a recognized node kind for this graph — applied using the default "union" merge behavior
```

## Scenario 4 — Re-applying is a safe no-op (spec.md User Story 3, now User Story 4)

```sh
$ arc apply tls13.patch.md
✅ rescorla-2026-tls13 is already tracked — nothing to do

$ git log --oneline | wc -l
# unchanged from before this command
```

## Verifying the one-commit invariant

```sh
$ git log --oneline -1
d4e5f6a graph(ingest): kolesnikov-2026-note — A Working Note

$ git status --short
# empty — nothing uncommitted
```
