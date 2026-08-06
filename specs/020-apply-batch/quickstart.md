# Quickstart & Validation: `arc apply batch`

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Contract**: [contracts/cli-contract.md](contracts/cli-contract.md)

Runnable scenarios that prove the feature end-to-end. Each maps to acceptance
scenarios in [spec.md](spec.md) and to an E2E test in
`cmd/arc/graph/batch_test.go` (Principle VIII).

---

## Prerequisites

- Go 1.26.5 (per `go.mod`)
- `git` on `PATH` — `arc apply` shells out to a real git binary, and the E2E
  suite follows `cmd/arc/graph/apply_test.go`'s existing precedent of running
  against a real temporary repository
- No cloud credentials, no network: the feature is fully local and offline

```sh
go build ./...
go test ./...
```

---

## Fixture layout

Fixtures live under `cmd/arc/graph/testdata/batch/`, colocated with the test
(Principle VIII). A deliberately messy tree, so one directory exercises
discovery, ordering, and classification at once:

```text
cmd/arc/graph/testdata/batch/
├── README.md                     # not a patch — passed over (FR-003)
├── notes.md                      # front matter, but no `kind: patch` (FR-003)
├── bibliography.bib              # not Markdown — ignored entirely, uncounted
├── 2026/
│   ├── pqkex.patch.md            # published 2026-01-20
│   └── karpathy.patch.md         # published 2026-04-02
├── 2024/
│   └── tls13.patch.md            # published 2024-03-11
├── nested/deep/
│   └── legacy.patch.md           # published 2023-08-05  ← applies FIRST
├── broken/
│   └── truncated.patch.md        # declares kind: patch, no `published` (FR-020)
└── .hidden/
    └── ignored.patch.md          # never discovered (FR-019)
```

Expected plan order: `nested/deep/legacy` (2023) → `2024/tls13` →
`2026/pqkex` → `2026/karpathy`, then `broken/truncated` last as a
classification failure (research.md D5b) — note this contradicts the tree's
alphabetical order, which is the point. `README.md` and `notes.md` count as
`not_a_patch`; `bibliography.bib` appears in no count at all; `.hidden/` is
never walked.

A second bundle drives Scenario 4, where the failure must occur during
*application* rather than classification:

```text
cmd/arc/graph/testdata/batch-failfast/
├── a-early.patch.md              # published 2025-01-01 — commits
├── b-blocker.patch.md            # published 2025-03-14 — parses, then fails
└── c-late.patch.md               # published 2025-06-01 — unprocessed
```

`b-blocker` contributes an `Entity` node `Blocker`; the test seeds
`entities/Blocker.md` in the graph as a *directory*, so the merge read of that
existing node fails while the patch itself classifies cleanly and holds its
real position in the publication order.

---

## Scenario 1 — Bulk ingest in publication order

**Covers**: US1 scenarios 1-5 · FR-001, FR-004, FR-005, FR-006, FR-007

```sh
mkdir -p /tmp/g && cd /tmp/g
arc init
arc apply batch <fixtures>/batch
git log --oneline
```

**Expect**

- One commit per applied patch, none spanning two documents (FR-007)
- `git log` in reverse-chronological order shows `karpathy` newest and
  `legacy` oldest — i.e. applied oldest-publication-first (FR-004)
- Patches under `nested/deep/` and `2024/` are applied, proving recursion
  (FR-001) and that ordering is global rather than per-directory
- Summary on stdout reports applied / skipped / not-a-patch / failed counts
  (FR-013)
- Exit code `1` here, because the fixture tree deliberately contains
  `broken/truncated.patch.md` (FR-014). Point the command at a clean subtree
  to see a `0` exit.

**Verify ordering independently of filenames** — rename the fixtures so
alphabetical order contradicts publication order and re-run into a fresh
graph; the commit sequence must not change.

---

## Scenario 2 — Safe re-run over a partly-ingested directory

**Covers**: US2 scenarios 1-4 · FR-008, FR-009

```sh
arc apply batch <fixtures>/batch     # first run
arc apply batch <fixtures>/batch     # second run
git rev-list --count HEAD            # unchanged by the second run
```

**Expect**

- Second run: every previously applied document reported as skipped, zero new
  commits, exit status unchanged by the re-run itself (FR-008)
- Adding one new patch to the directory and re-running applies **only** that
  patch
- Two patches describing the same document in one run: the first applies, the
  second is skipped — proving the tracked check is evaluated when each patch
  is reached, not once up front (FR-009)

---

## Scenario 3 — One bad patch does not abandon the corpus

**Covers**: US3 scenarios 1-4 · FR-010, FR-012, FR-013, FR-014, FR-020

```sh
arc apply batch <fixtures>/batch
echo "exit=$?"
```

**Expect**

- Every valid patch applied and committed despite `broken/truncated.patch.md`
  (FR-010)
- The failure reported by path with its reason (FR-013)
- `exit=1` (FR-014)
- `git status` clean — no partial node files, no dangling commit (FR-012)
- A patch whose manifest lacks a usable `published` date is reported as failed,
  never applied at a guessed position (FR-020)

---

## Scenario 4 — `--fail-fast` halts mid-plan

**Covers**: US4 scenarios 1-3 · FR-011

Use a fixture whose failure occurs during *application* rather than
classification, so it holds a real position in the publication order
(research.md D5b).

```sh
arc apply batch --fail-fast <fixtures>/batch-failfast
echo "exit=$?"
```

**Expect**

- Patches ordered before the failing one are committed; none after it are
  applied (FR-011)
- The remainder reported as unprocessed, with a count
- `exit=1`
- Fixing the offending patch and re-running skips the already-committed
  documents and resumes from the repaired one (US4 scenario 3, via FR-008)

---

## Scenario 5 — Nothing to apply

**Covers**: Edge cases · FR-003, FR-022

```sh
mkdir -p /tmp/empty && arc apply batch /tmp/empty
echo "exit=$?"                       # 0

arc apply batch <fixtures>/batch/.hidden   # explicit hidden dir as target
```

**Expect**

- Empty directory: succeeds, no changes, says plainly there is nothing to
  apply (FR-022), `exit=0`
- A directory of nothing but non-patch Markdown: same, with a non-zero
  `not_a_patch` count and zero failures (FR-003)

---

## Scenario 6 — Preflight refusals

**Covers**: Edge cases · FR-002

```sh
arc apply batch /does/not/exist      # refuse: not found
arc apply batch ./some-file.md       # refuse: not a directory
cd /tmp/not-a-graph && arc apply batch <fixtures>/batch   # refuse: not a graph
```

**Expect**: each refuses before reading any patch, makes no filesystem or git
change, and renders a `faults`-annotated human-readable message through
`cmd/arc/main.go` (FR-002).

---

## Scenario 7 — Machine-readable output

**Covers**: FR-018 · [contracts/batch-result.schema.md](contracts/batch-result.schema.md)

```sh
arc apply batch --json <fixtures>/batch | jq '.'
arc apply batch --json <fixtures>/batch | jq '.applied, .failed, .not_a_patch'
arc apply batch --json <fixtures>/batch | jq -r '.patches[] | "\(.outcome) \(.path)"'
```

**Expect**

- stdout carries **only** the JSON document — progress and hints go to stderr,
  so the pipe stays clean (research.md D9)
- `applied + skipped + failed + unprocessed == (.patches | length)`
- `not_a_patch` is outside that sum
- `conflicts` and `warnings` are arrays, never `null`
- Assertions validate against the documented schema, not merely "non-empty"
  (Principle VIII)

---

## Scenario 8 — Read-only over the patch directory

**Covers**: FR-021

```sh
cp -R <fixtures>/batch /tmp/patches
find /tmp/patches -type f | sort > /tmp/before
arc apply batch /tmp/patches
find /tmp/patches -type f | sort > /tmp/after
diff /tmp/before /tmp/after           # no output
```

**Expect**: the patch directory is byte-identical afterwards — nothing moved,
renamed, deleted, or rewritten.

---

## Scenario 9 — Progress visibility

**Covers**: FR-015, FR-016, FR-017

```sh
arc apply batch <fixtures>/batch 2>&1 >/dev/null     # stderr only
arc apply batch --quiet <fixtures>/batch 2>&1 >/dev/null
arc apply batch --verbose <fixtures>/batch 2>&1 >/dev/null
```

**Expect**

- Default: one progress line per patch as it completes (FR-015)
- `--quiet`: no progress lines; summary still printed
- `--verbose`: additionally the per-node/per-predicate detail `arc apply`
  already emits, unchanged
- Conflicts flagged anywhere in the run appear once in the final summary, not
  only inline where they occurred (FR-016)
- An unregistered node type produces a warning and does **not** abort the run
  (FR-017)

---

## Regression guard

`arc apply <patch.md>` and `arc apply schema` are untouched by this feature.
The existing suites in `cmd/arc/graph/apply_test.go` and the schema tests MUST
continue to pass unmodified — if either needs editing, the change has leaked
past the intended boundary (plan.md § Structure Decision).
