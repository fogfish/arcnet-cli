# Quickstart: Node Provenance Timestamps

Validates spec.md's three user stories against a real local graph. Builds directly on `specs/003-apply-patch/quickstart.md`'s setup — this feature changes what `arc apply` writes into node front matter, not how the command is invoked.

## Prerequisites

- `git` on `PATH`
- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout

## Setup

```sh
$ arc init ./demo-graph && cd demo-graph
```

## Scenario 1 — Creation stamps `published` + `indexed` (User Story 1)

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
```

# Entity

## tls-1.3
```yaml
score: 0.9
```
EOF

$ arc apply tls13.patch.md
✅ Ingested rescorla-2026-tls13: source: +1 created, entity: +1 created (commit a1b2c3d)

$ cat sources/rescorla-2026-tls13.md
---
authors: [Eric Rescorla]
id: rescorla-2026-tls13
indexed: "2026-07-05T14:22:31Z"
published: "2026-04-12"
title: "TLS 1.3: Design and Rationale"
---
# rescorla-2026-tls13

$ cat entities/tls-1.3.md
---
id: tls-1.3
indexed: "2026-07-05T14:22:31Z"
published: "2026-04-12"
score: 0.9
---
# tls-1.3
```

Expected: both files carry identical `indexed` values (one Application Timestamp per invocation, spec FR-005) and matching `published` values (the patch's own declared date, spec FR-001).

## Scenario 2 — A real merge stamps `updated`; a no-op merge does not (User Story 2)

```sh
$ cat > tls13-followup.patch.md <<'EOF'
---
kind: patch
document: rescorla-2026-tls13-followup
published: 2026-05-01
---
# Entity

## tls-1.3
```yaml
```
- related:: [[record-layer]]
EOF

$ arc apply tls13-followup.patch.md
✅ Ingested rescorla-2026-tls13-followup: entity: +0 created, +1 merged (commit b2c3d4e)

$ cat entities/tls-1.3.md
---
id: tls-1.3
indexed: "2026-07-05T14:22:31Z"
published: "2026-04-12"
score: 0.9
updated: "2026-07-05T14:31:09Z"
---
# tls-1.3

- related:: [[record-layer]]
```

Expected: `published`/`indexed` are unchanged from Scenario 1 (never overwritten, spec FR-006/FR-010); `updated` is new, at the second application's own timestamp.

```sh
$ arc apply tls13-followup.patch.md
✅ Ingested rescorla-2026-tls13-followup: entity: +0 created, +1 merged (commit c3d4e5f)

$ cat entities/tls-1.3.md   # byte-identical to the previous cat — no new `updated` value
```

Expected: re-applying the exact same follow-up patch a second time contributes nothing new (`related:: [[record-layer]]` is already present) — `updated`'s existing value is left exactly as it was; it is not re-stamped with the third application's newer timestamp (spec FR-007/FR-008, research.md D6).

## Scenario 3 — `published` survives export; `indexed`/`updated` are apply-only (User Story 3)

```sh
$ arc subgraph tls-1.3 --depth 0
---
kind: patch
document: subgraph:tls-1-3@2026-07-05T14:40:00Z
published: 2026-07-05
---
# Entity

## tls-1.3
```yaml
published: "2026-04-12"
score: 0.9
```
- related:: [[record-layer]]
```

Expected: the extracted node's own `published: "2026-04-12"` is preserved verbatim (spec FR-011/SC-006) even though the subgraph patch's own manifest-level `published` is today's date (the extraction time, not a real publication date, research.md D11).
