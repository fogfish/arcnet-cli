# Quickstart: Validating Schema Vocabulary Conformance

**Feature**: `023-core-vocabulary-conformance`

Runnable validation for every user story in [spec.md](spec.md). Each section is independent —
run only the story you are verifying.

## Prerequisites

```sh
go build -o /tmp/arc ./cmd/arc
export PATH=/tmp:$PATH
git config --global user.email >/dev/null || git config --global user.email you@example.org
```

## Automated gate

```sh
go test ./...                              # full suite
go test ./internal/core/... -run Merge      # C1.4 dispatch after removal
go test ./internal/app/schema/... -run Seed # C2.1 golden snapshot
```

Refresh the golden snapshot only after reading its diff:

```sh
go test ./internal/app/schema/service -run TestSeedGolden -update
git diff testdata/golden/schema/           # review as a content change, not noise
```

---

## US2 — a fresh graph is conformant (P2)

```sh
mkdir /tmp/g && cd /tmp/g && arc init
```

**Expect** — merge vocabulary is the closed set of six ([C2.2a](contracts/seeded-schema-contract.md)):

```sh
grep -h '^merge:' _schema/Property/*.md | sort -u
# merge: append
# merge: fillIfEmpty
# merge: firstWriteWin
# merge: immutable
# merge: lastWriteWin
# merge: union
```

**Expect** — no type node declares a merge (C2.2b), and no `validatedOverwrite` anywhere:

```sh
grep -l '^merge:' _schema/Class/*.md   # exits 1, no output
grep -rl validatedOverwrite .          # exits 1, no output
```

**Expect** — the score predicates are conformant:

```sh
grep '^merge:' _schema/Property/scoreZ.md   # merge: lastWriteWin
```

**Expect** — a graph that lints clean (SC-003):

```sh
arc lint     # ✓ no violations
```

**Expect** — an out-of-menu value is rejected with an actionable message
([C1.2](contracts/merge-vocabulary-contract.md)):

```sh
sed -i.bak 's/^merge: union/merge: validatedOverwrite/' _schema/Property/tags.md
arc lint
# schema document _schema/Property/tags.md has a missing or invalid merge
mv _schema/Property/tags.md.bak _schema/Property/tags.md
```

**Expect** — a stale type-level merge is tolerated silently (C1.3, FR-006):

```sh
sed -i '' '2i\
merge: union
' _schema/Class/Entity.md
arc lint     # ✓ still clean — attribute ignored, not reported
```

---

## US1 — prose holds a single first-fixed value (P1)

> Read [research.md D4](research.md) first. A **byte-identical** re-apply is already a no-op on
> `main` via `mergeText`'s near-duplicate guard — that is not what this story fixes. The defect
> is that **reworded** prose accumulates. Test the reworded case or the test passes vacuously.

```sh
cd /tmp/g
cat > /tmp/p1.md <<'PATCH'
---
"@type": patch
document: doc-a
published: 2026-04-12
---
# doc-a

## Source/rescorla-2026-tls13

---
"@id": rescorla-2026-tls13
"@type": Source
title: "TLS 1.3"
published: 2026-04-12
---
# TLS 1.3

A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]
PATCH
arc apply /tmp/p1.md
```

Now reword the abstract materially (below the 0.8 shingle threshold) and re-apply:

```sh
sed 's/A design retrospective on the TLS 1.3 handshake./An examination of handshake design choices and residual resumption risk./' /tmp/p1.md > /tmp/p2.md
arc apply /tmp/p2.md
```

**Expect** — one paragraph, the first one, preserved; divergence flagged, not appended:

```sh
sed -n '/^# TLS 1.3/,/^## /p' Source/rescorla-2026-tls13.md
# A design retrospective on the TLS 1.3 handshake.        ← unchanged, single paragraph
arc apply /tmp/p2.md --verbose | grep abstract
# abstract  firstWriteWin  flagged
```

**Before this feature** the file would carry both paragraphs.

**Expect** — the same holds for `Entity.definition` and `Reference.relevance` (research D8), and
for a predicate/type document's own `description`:

```sh
arc apply /tmp/p1.md          # third apply
git status --short            # empty — no commit (FR-016)
```

**Expect** — citation targets are unioned, never duplicated (FR-014):

```sh
grep -c 'cites:: \[\[rescorla-2018-rfc8446\]\]' Source/rescorla-2026-tls13.md   # 1
```

---

## US3 — conformant nodes stop being reported (P3)

Build a graph by hand, shaped exactly as ARCNET-CORE describes — **not** by `arc init` output,
or the test passes vacuously (`CORE-FIX.md` §5.7).

```sh
cd /tmp/g
cat > Entity/Forward\ Secrecy.md <<'NODE'
---
"@id": Forward Secrecy
"@type": Entity
category: [independent, abstract, occurrent, script]
---
# Forward Secrecy

Compromise of a long-term key does not compromise past session keys.

## mentionedIn
- mentionedIn:: [[rescorla-2026-tls13]]
NODE

cat > timeline/monthly/2026-04.md <<'NODE'
---
"@id": 2026-04
"@type": Timeline
---
# 2026-04

- cites:: [[rescorla-2026-tls13]]
NODE

arc lint
```

**Expect** — clean. Specifically **no** `typeRequires` for:

- `Forward Secrecy` missing `published` / `created` (FR-007 — `Node` requires nothing)
- `2026-04` missing `granularity` / `period` (FR-009 — `Timeline` requires `cites` alone)

**Expect** — a `Timeline` node that *does* carry `granularity`, `period`, and `heading` reports
no `typeOptional` violation either (all three stay declared).

**Expect** — the requirement that survives:

```sh
sed -i '' '/^published:/d' Source/rescorla-2026-tls13.md
arc lint     # typeRequires: Source requires published   ← still reported (FR-008)
```

---

## US4 — core predicates are registered, not left unregistered (P4)

> Read [research.md D5](research.md). Nothing is auto-registered today — front-matter attributes
> never reach `RegisterPredicate`. The symptom is lint noise, not a placeholder file.

```sh
cd /tmp/g
cat >> Source/rescorla-2026-tls13.md.new <<'X'
X
arc lint --json | grep -E 'author|about|genre'    # exits 1 — nothing reported
ls _schema/Property/author.md _schema/Property/about.md _schema/Property/genre.md
```

**Expect** — all three documents exist in a fresh graph (FR-011), each `role: meta`,
`merge: union`; `author` additionally `aligned: schema:author`.

Apply a patch using all three:

```sh
# add `author: [Eric Rescorla]`, `about: [protocols]`, `genre: paper` to a Source node,
# then:
arc apply /tmp/p3.md
arc lint
```

**Expect** — clean. No `predicateRegistered` and no `typeOptional` violation for the three
(FR-012). **Before this feature** each occurrence produced two violations.

**Expect** — `authors` and `author` both resolve, both `union` (research D2):

```sh
grep '^aligned:' _schema/Property/author.md _schema/Property/authors.md
# both: aligned: "schema:author"
```

---

## US5 — REMOVED

> **Removed 2026-08-23, post-implementation.** This section walked through building a graph
> with a previous binary and upgrading it with `arc upgrade`. That command was implemented, then
> removed by explicit decision: `arc` is pre-1.0/experimental, and the compatibility/migration
> machinery it required was judged unnecessary tech debt. A graph seeded by a previous release
> now simply fails to load once it declares the retired seventh merge value — there is no
> remedy; re-initialize instead.
