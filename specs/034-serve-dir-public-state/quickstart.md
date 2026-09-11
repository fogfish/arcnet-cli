# Quickstart: Validating Explicit Serve Context & Shareable Graph State

**Feature**: `034-serve-dir-public-state` | **Date**: 2026-09-11

Runnable checks that prove the feature works end to end. Shapes and error strings
come from [contracts/cli-contract.md](contracts/cli-contract.md); the layout comes
from [data-model.md](data-model.md).

## Prerequisites

```bash
go version    # matches go.mod
git --version # arc init requires git on PATH
go build -o ./arc ./cmd/arc
```

---

## Automated gate

```bash
go test ./...
go test ./cmd/arc/graph/ -run 'Serve'   -v
go test ./cmd/arc/ctrl/  -run 'Init'    -v
go test ./internal/app/ctrl/service/    -v
staticcheck ./...
```

Every acceptance scenario in [spec.md](spec.md) maps 1:1 to a test
(Constitution VIII). Test placement is fixed by research D8.

---

## V1 — Serve a graph from an unrelated working directory (US1 #1, #3)

```bash
./arc init /tmp/v1-graph
cd /tmp                                   # deliberately NOT the graph
/path/to/arc serve /tmp/v1-graph          # blocks on stdio
```

**Expected**: the server starts and stays up. Ctrl-C exits promptly.

Prove tools answer from the named graph over HTTP instead:

```bash
/path/to/arc serve --http :8765 /tmp/v1-graph &
curl -s localhost:8765 -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize",
       "params":{"protocolVersion":"2025-06-18","capabilities":{},
                 "clientInfo":{"name":"qs","version":"0"}}}'
kill %1
```

**Expected**: an `InitializeResult` whose `instructions` begin with
*"This server exposes a knowledge graph, over eight tools…"*.

---

## V2 — No argument is unchanged (US1 #2, SC-004)

```bash
cd /tmp/v1-graph && /path/to/arc serve --http :8766 &
```

**Expected**: identical to V1. This is the regression guard — every existing
invocation must behave byte-for-byte as before.

---

## V3 — Refusals happen before any transport opens (US1 #4, SC-007)

```bash
./arc serve /tmp/does-not-exist   ; echo "exit=$?"
touch /tmp/a-file && ./arc serve /tmp/a-file ; echo "exit=$?"
mkdir -p /tmp/not-a-graph && ./arc serve /tmp/not-a-graph ; echo "exit=$?"
./arc serve /tmp/v1-graph /tmp/other ; echo "exit=$?"
```

**Expected**, each returning within a second with `exit=1`, nothing listening:

```
/tmp/does-not-exist does not exist
/tmp/a-file is not a directory
/tmp/not-a-graph is not an initialized graph
Error: accepts at most 1 arg(s), received 2      # plus usage
```

---

## V4 — Init produces a tracked `.arc/` and an ignored cache (US2 #1, #4; US3 #1-#3)

```bash
./arc init /tmp/v4-graph && cd /tmp/v4-graph
git status --porcelain            # expect: empty
git ls-files .arc                 # expect: .arc/.gitignore  .arc/.gitkeep
cat .arc/.gitignore               # expect: cache/
cat .arc/cache/.gitignore         # expect: *
git show --stat --name-only HEAD | grep -c '^\.arc/cache'   # expect: 0
echo scratch > .arc/cache/probe && git status --porcelain   # expect: empty
git check-ignore -v .arc/cache/probe                        # expect: a match
```

---

## V5 — A clone is a usable graph (US2 #2, #3; SC-003, SC-006)

```bash
cd /tmp/v4-graph
printf 'grep:\n  workers: 3\n' > .arc/config.yml
git add .arc/config.yml && git commit -q -m "graph: tune grep"

git clone -q /tmp/v4-graph /tmp/v5-clone
cd /tmp/v5-clone
cat .arc/config.yml               # expect: workers: 3 — config travelled
ls .arc/cache 2>&1                # expect: No such file or directory (data-model I6)
/path/to/arc stats                # expect: succeeds, zero setup steps
/path/to/arc serve --http :8767 & sleep 1; kill %1
```

**Expected**: every command succeeds in the clone with no `arc init` and no repair.
This is the failure the whole feature exists to remove.

---

## V6 — `--skip-git-init` into a host repo that ignores `.arc/` (FR-017)

```bash
mkdir -p /tmp/v6-host && cd /tmp/v6-host && git init -q
printf '.arc/\n' > .gitignore
git add .gitignore && git commit -q -m init
/path/to/arc init --skip-git-init notes ; echo "exit=$?"
ls notes 2>&1
git status --porcelain
```

**Expected**: `exit=1`, the message
`…/notes/.arc is excluded by the repository's ignore rules; a clone would not be a usable graph`,
**`notes/` does not exist** (refusal precedes every write), and the host repo's
working tree is untouched.

---

## V7 — A legacy graph still works, and migrates (US4 #1-#3; SC-008)

Fabricate the old layout:

```bash
./arc init /tmp/v7-graph && cd /tmp/v7-graph
git rm -q --cached .arc/.gitkeep .arc/.gitignore
rm -rf .arc/cache && printf '*\n' > .arc/.gitignore
git commit -q -am "simulate legacy layout"

/path/to/arc stats     # expect: succeeds — no warning, no error (FR-018, FR-020)
```

Then the documented migration:

```bash
printf 'cache/\n' > .arc/.gitignore
mkdir -p .arc/cache && printf '*\n' > .arc/cache/.gitignore
git add .arc/.gitkeep .arc/.gitignore && git commit -q -m "graph: publish .arc state"

git clone -q /tmp/v7-graph /tmp/v7-clone && /path/to/arc serve --http :8768 /tmp/v7-clone &
sleep 1; kill %1
```

**Expected**: the pre-migration `arc stats` is silent about the layout, and the
post-migration clone serves.

---

## V8 — The case-folding probe no longer dirties `.arc/` (research D6)

```bash
cd /tmp/v4-graph
/path/to/arc apply /path/to/some.patch.md >/dev/null 2>&1
git status --porcelain          # expect: no stray arc-case-probe-*.tmp under .arc/
```

**Expected**: any probe file that outlives a crash lands under `.arc/cache/`, where
the `*` rule keeps it invisible to git.

---

## Success criteria coverage

| SC | Validated by |
|----|--------------|
| SC-001 | V1 |
| SC-002 | V1 (HTTP handshake from `/tmp`) |
| SC-003 | V5 |
| SC-004 | V2 |
| SC-005 | V4 |
| SC-006 | V5 |
| SC-007 | V3 |
| SC-008 | V7 |

## Cleanup

```bash
rm -rf /tmp/v1-graph /tmp/v4-graph /tmp/v5-clone /tmp/v6-host \
       /tmp/v7-graph /tmp/v7-clone /tmp/not-a-graph /tmp/a-file
```
