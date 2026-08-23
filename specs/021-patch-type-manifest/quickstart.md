# Quickstart: Patch Manifest Identity (`@type: patch`)

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Contract**: [contracts/patch-manifest.md](./contracts/patch-manifest.md)

A runnable validation guide. Every scenario below is a real invocation with a stated expected
outcome; together they cover all 12 acceptance scenarios and all seven success criteria.

## Prerequisites

```bash
go build -o /tmp/arc ./cmd/arc
export PATH=/tmp:$PATH
cd "$(mktemp -d)"
arc init demo && cd demo
```

---

## Scenario 1 — A spec-conformant patch applies (US1.1, US1.2, SC-001, SC-006)

```bash
cat > shannon.patch.md <<'PATCH'
---
"@type": patch
document: shannon-1948-mathematical
published: 1948-07-01
title: "A Mathematical Theory of Communication"
stats: {sources: 1, entities: 1}
---
# Source

## shannon-1948-mathematical
```yaml
"@id": "shannon-1948-mathematical"
"@type": Source
```

Founding paper of information theory.

**Mentions**

- [[information-entropy]]

# Entity

## information-entropy
```yaml
"@id": "information-entropy"
"@type": Entity
category: "independent physical continuant object"
```

The average surprise of a random variable.
PATCH

arc apply shannon.patch.md
```

**Expected**: exit `0`, an ingest summary naming 1 source and 1 entity, one
`graph(ingest):` commit. `sources/shannon-1948-mathematical.md` and
`entity/information-entropy.md` exist.

```bash
arc apply shannon.patch.md
git log --oneline | wc -l
```

**Expected**: a skip message, exit `0`, and the commit count **unchanged** — idempotency
survives the key change (SC-006).

> Before this feature this first command failed with `manifest is missing a mandatory field`.
> That is the whole defect: the success rate for conformant patches was 0% (SC-001).

---

## Scenario 2 — Export emits a readable patch and round-trips (US2.1, US2.2, SC-002)

```bash
arc subgraph shannon-1948-mathematical > exported.patch.md
head -8 exported.patch.md
```

**Expected**: the first line after `---` is exactly `"@type": patch`, and:

```bash
grep -c '^kind:' exported.patch.md    # → 0
```

Round-trip into a fresh graph:

```bash
cd .. && arc init fresh && cd fresh
arc apply ../demo/exported.patch.md
```

**Expected**: exit `0`, the same nodes reproduced with zero manual edits (SC-002).

---

## Scenario 3 — `--json` is untouched (US2.3, FR-011)

```bash
cd ../demo
arc subgraph shannon-1948-mathematical --json | jq '.patch | keys'
```

**Expected**: `["document","nodes","published","stats","title"]` — `core.Patch`'s own fields,
unchanged. The identity key was never represented on `core.Patch` (it is a recognition
predicate consumed during decode, not carried data), and this feature adds no field, so
nothing new appears here.

```bash
arc subgraph shannon-1948-mathematical --json | grep -c '"kind"'   # → 0
```

The strongest form of this check is a golden comparison: capture `--json` output before and
after the change on the same graph and diff it. It must be byte-identical (FR-011).

---

## Scenario 4 — A retired-key patch is refused with actionable guidance (US3.1, SC-007)

```bash
sed 's/^"@type": patch/kind: patch/' shannon.patch.md > legacy.patch.md
arc apply legacy.patch.md; echo "exit=$?"
git log --oneline | wc -l
```

**Expected**: exit `1`, and on **stderr**:

```
failed to read patch file legacy.patch.md: manifest uses the retired "kind: patch" key — rewrite it as "@type": patch
```

The commit count is unchanged (FR-007). The message names the file, the offending key, and the
replacement, so the patch is correctable without opening ARCNET-CORE (SC-007).

> **Verified 2026-08-23.** The message above is what you get when the patch file lives *outside*
> the graph root — the ordinary case, and the one
> [contracts/patch-manifest.md §3.3](./contracts/patch-manifest.md) is normative for. When the
> file sits *inside* the graph tree (as it does in this walkthrough, since we wrote it into
> `demo/`), `guardNoOldFormatNodes`' whole-graph walk reaches it before `readPatch` does, and the
> same sentence arrives under a `failed to write legacy.patch.md:` prefix instead. Both name the
> file, the retired key and the replacement; only the prefix differs. Run
> `arc apply ../legacy.patch.md` from a sibling directory to see the contract's exact wording.

---

## Scenario 5 — A self-contradictory manifest is refused (US3.2)

```bash
sed 's/^"@type": patch/"@type": patch\nkind: source/' shannon.patch.md > conflict.patch.md
arc apply conflict.patch.md; echo "exit=$?"
```

**Expected**: exit `1`, and a message distinct from Scenario 4's:

```
failed to read patch file conflict.patch.md: manifest declares "@type": patch and legacy kind: source — the two disagree
```

---

## Scenario 6 — Both keys agreeing is accepted (US3.3)

```bash
cd ../fresh
sed 's/^"@type": patch/"@type": patch\nkind: patch/' ../demo/shannon.patch.md > both.patch.md
arc apply both.patch.md; echo "exit=$?"
```

**Expected**: exit `0`. A redundant retired key is not a contradiction; it is ignored.

---

## Scenario 7 — Batch names the retired file, never skips it (US1.3, US3.4, SC-005)

The patch directory must sit **outside** the graph root. `arc apply` guards the whole graph
against pre-0.5 node files before it writes anything, so a retired-key patch staged *inside* the
graph tree aborts every patch in the run, not just its own — you would see `applied: 0,
failed: 2` instead of the counts below. That guard is pre-existing behaviour, unrelated to the
manifest key; staging patches outside the graph is the layout this scenario is about.

```bash
cd .. && mkdir -p patches
cp demo/shannon.patch.md patches/good.patch.md
cp legacy.patch.md       patches/legacy.patch.md
cat > patches/notes.md <<'NOTES'
---
title: Reading notes
tags: [scratch]
---
# Reading notes
Front matter declaring no patch identity at all.
NOTES

arc init batchdemo && cd batchdemo
arc apply batch ../patches --json | jq '{applied, failed, not_a_patch}'
```

**Expected**:

- `applied: 1` — `good.patch.md`
- `failed: 1` — `legacy.patch.md`, **reported by name** with the Scenario 4 message
- `not_a_patch: 1` — `notes.md` only

`legacy.patch.md` must appear under `failed`, never under `not_a_patch`. That distinction is
SC-005: a file the user meant as a patch is never silently skipped.

---

## Scenario 8 — Wrong casing and unquoted keys (edge cases, FR-015, FR-016)

```bash
sed 's/^"@type": patch/"@type": Patch/' ../demo/shannon.patch.md > cased.patch.md
arc apply cased.patch.md
```

**Expected**: exit `1` —
`manifest declares "@type": Patch, expected the literal lowercase "patch"`.

```bash
sed 's/^"@type": patch/@type: patch/' ../demo/shannon.patch.md > bare.patch.md
arc apply bare.patch.md
```

**Expected**: exit `1` —
`"@type" must be a quoted YAML string key, found it unquoted` — **not** a raw
`yaml: found character that cannot start any token`, and not a generic manifest-invalid error.

---

## Scenario 9 — Schema patches, all three sources (US1.4)

```bash
cd ../demo
arc apply schema arcnet:core.md
```

**Expected**: the manifest is recognized. The catalog entry is `core.md`, not `core` — a bare
`arcnet:core` is a 404 from the catalog, not a manifest problem.

> **Verified 2026-08-23** against the live catalog. The published profile at
> `fogfish/arcnet-spec/schema/core.md` does declare `"@type": patch`, and this is the change that
> makes its manifest loadable at all: the `0.1.12` binary refuses it with
> `manifest is missing a mandatory field or uses the pre-0.5 node format`, while this build reads
> the manifest and proceeds. It then stops on a *separate*, out-of-scope conformance gap —
> `patch body does not follow the H1-kind/H2-node section structure` — which belongs to the
> in-flight body-format features (`CORE-FIX.md` §2: 022, 023, 024), not to this one. The manifest
> half of SC-001 is met; the profile is not yet end-to-end importable.

---

## Scenario 10 — `arc lint` is unaffected (FR-013)

```bash
arc lint; echo "exit=$?"
```

**Expected**: identical output and exit code to a run on the same graph before this feature.
Patches are never nodes and are never indexed, so no rule observes this change.

> **Verified 2026-08-23** by running the `0.1.12` binary and this build against the same graph
> and diffing. Every violation, its rule, its file, its line, the ordering and the exit code are
> identical. The only textual difference is the Go source line number `github.com/fogfish/faults`
> embeds in its error context (`core.ParseNode 120` → `core.ParseNode 193`), which moved because
> declarations were added above `ParseNode` in `markdown.go`. No rule was added, removed, or
> retimed; `internal/app/lint` is untouched.

---

## Repository-wide check (SC-003)

```bash
cd <repo root>
grep -rn 'kind: patch' --include='*.md' --include='*.go' \
  cmd/ internal/ ARCHITECTURE.md specs/VISION.md \
  specs/*/quickstart.md specs/*/contracts/
```

**Expected**: no match that *presents the retired key as a current, valid example*. Dated
records — past `spec.md`, `research.md`, `data-model.md`, `tasks.md`, `specs/CHANGELOG.md`,
`CORE-FIX.md`, and this feature's own `spec.md` — are deliberately excluded; see
[research.md](./research.md) D7.

Three further classes of match are expected inside the grep's own scope and are **not**
violations — every one of them names the retired key precisely in order to reject it:

| Match class | Where |
|---|---|
| The refusal message itself | `internal/core/errors.go` (`ErrManifestLegacyKind`), and the contract/spec text quoting it |
| Test fixtures constructing a retired-key document to assert it is refused | `cmd/arc/**/*_test.go`, `internal/core/markdown_test.go`, `cmd/arc/graph/testdata/batch/legacy-kind.patch.md` |
| Doc comments explaining that the key is retired | `internal/core/markdown.go`, `internal/app/graph/service/batch.go`, `ARCHITECTURE.md` |

The check that must return zero is the *valid-example* one:

```bash
grep -rn 'kind: patch' --include='*.md' --include='*.go' \
  cmd/ internal/ ARCHITECTURE.md specs/VISION.md \
  specs/*/quickstart.md specs/*/contracts/ \
  | grep -v 'specs/021-patch-type-manifest/' \
  | grep -viE 'retired|legacy|pre-0\.5|strings.Replace|withIdentity|manifestFixture'
```

**Verified 2026-08-23**: zero matches.

---

## Full suite

```bash
go build ./... && go test ./... -cover
```

**Expected**: all green. The five-case recognition table in `internal/core/markdown_test.go`
is the complete unit proof of §2 of the manifest contract.
