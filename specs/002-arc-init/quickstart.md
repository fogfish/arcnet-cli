# Quickstart: Validating `arc init`

## Prerequisites

- Go 1.26 toolchain installed (matches `go.mod`)
- `git` installed and on `PATH`
- A shell in the repository root

## Build

```sh
go build -o /tmp/arc ./cmd/arc
```

## Scenario 1 — bootstrap in an empty current directory (spec US1)

```sh
mkdir -p /tmp/arc-graph-1 && cd /tmp/arc-graph-1
/tmp/arc init
```

**Expected**: exit code `0`; stdout prints a single confirmation line with the resolved path and a commit hash; `sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/` exist; `_meta/predicates.md` and `_meta/aliases.md` exist; `.arc/` exists; `.gitignore` contains `.arc/`.

```sh
git -C /tmp/arc-graph-1 log --oneline   # exactly one commit, subject "graph(init): empty knowledge graph"
git -C /tmp/arc-graph-1 status --short  # empty output (clean tree)
git -C /tmp/arc-graph-1 ls-files | grep '\.arc/'  # no output — .arc/ is not tracked
```

## Scenario 2 — bootstrap into a named, not-yet-existing directory (spec US2)

```sh
cd /tmp
/tmp/arc init ./arc-graph-2
```

**Expected**: exit code `0`; `/tmp/arc-graph-2` created with the full layout; current directory (`/tmp`) unaffected; stdout reports the resolved path `/tmp/arc-graph-2`.

## Scenario 3 — refuse to re-initialize an existing graph (spec US3, FR-014)

```sh
/tmp/arc init /tmp/arc-graph-1
```

**Expected**: exit code `1`; stderr states the target is already an initialized graph; `git -C /tmp/arc-graph-1 log --oneline` still shows exactly one commit (nothing lost or duplicated).

## Scenario 4 — refuse a non-empty, non-graph target directory (FR-015)

```sh
mkdir -p /tmp/arc-graph-4 && touch /tmp/arc-graph-4/unrelated.txt
/tmp/arc init /tmp/arc-graph-4
```

**Expected**: exit code `1`; stderr states the directory must be empty; `/tmp/arc-graph-4` still contains only `unrelated.txt` (no partial graph layout written).

## Scenario 5 — `--json` output contract

```sh
mkdir -p /tmp/arc-graph-5 && cd /tmp/arc-graph-5
/tmp/arc init --json
```

**Expected**: stdout is a single JSON object with `path`, `commit`, `foldersCreated` fields (see [contracts/cli-contract.md](contracts/cli-contract.md)); no hint text on stderr.

## Scenario 6 — missing `git` (FR-011)

Simulate by temporarily shadowing `PATH` without `git`:

```sh
mkdir -p /tmp/arc-graph-6 && cd /tmp/arc-graph-6
PATH=/usr/bin/nonexistent /tmp/arc init
```

**Expected**: exit code `1`; stderr explains `git` is required and was not found; `/tmp/arc-graph-6` remains empty (no partial writes).

## Automated coverage

All six scenarios above have a corresponding E2E test in `cmd/arc/ctrl/init_test.go` (constitution Principle VIII) — this quickstart is for manual/exploratory validation and CI smoke-testing, not a substitute for the `go test ./...` suite gating merges.
