# Quickstart: Validating Per-Predicate Merge Reconciliation

## Prerequisites

- A built `arc` binary (`go build ./cmd/arc`) from this feature's branch.
- A scratch directory outside this repo, with git available on `PATH` (arc apply shells out to `git commit`, per `cmd/arc/graph/apply_test.go`'s `TestMain`).

## Scenario A — a single patch drives three predicates by three different rules (spec User Story 1)

```sh
cd /tmp && mkdir arc-quickstart && cd arc-quickstart
arc init
```

Confirm `_schema/predicates/ref.md` declares `merge: immutable`, `_schema/predicates/status.md` declares `merge: lastWriteWin`, and `_schema/predicates/tags.md` declares `merge: union`.

Write `patch-1.md` contributing a `resource` node `example-book` with `ref: book`, `status: backlog`, `tags: [ai]`, then:

```sh
arc apply patch-1.md
cat resources/example-book.md   # ref: book, status: backlog, tags: [ai]
```

Write `patch-2.md` contributing to the same `example-book`: `ref: article` (a different value), `status: read`, `tags: [ml]`, then:

```sh
arc apply patch-2.md
cat resources/example-book.md
```

**Expected**: `ref` is still `book` (immutable — the patch-2 attempt to change it left no trace and produced no conflict marker); `status` is now `read` (lastWriteWin — the latest applied contribution won); `tags` is `[ai, ml]` (union — both values present, deduplicated).

## Scenario B — conflict flagging fires only for firstWriteWin (spec User Story 2)

Repeat the pattern above for a predicate declared `firstWriteWin` (e.g. `abstract`): apply one patch that sets `abstract: "First summary."`, then a second patch that sets `abstract: "A different summary."`.

**Expected**: `resources/example-book.md`'s `abstract` shows the conflict marker (`<<<<<<< existing` / `=======` / `>>>>>>> <patch-2's document id>`), and `arc apply`'s own reported outcome for that node is `merged (conflict flagged)`. Repeat with a `union`- or `append`-declared predicate contributing two different values instead — confirm no marker appears.

## Scenario C — idempotency and commutativity at the algebra level (spec User Story 3)

The CLI-level replay path is short-circuited by `arc apply`'s existing already-tracked check (re-applying the identical patch document is a no-op skip before any merge runs), so the merge algebra's own idempotency/commutativity is verified directly at the unit level instead:

```sh
go test ./internal/core/... -run TestMerge -v
```

**Expected**: every `MergeOp` has a passing idempotency case (`Merge(Merge(existing, incoming, index, id), incoming, index, id)` unchanged from one application) and a passing commutativity case for independent predicates, except `lastWriteWin`'s dedicated test, which asserts the documented order-sensitive outcome instead (research.md D5a) — applying two contributions to the same `lastWriteWin` predicate in reverse order changes the result, on purpose.

## Scenario D — full command-level regression

```sh
go build ./... && go test ./... -cover
```

**Expected**: all packages pass, including `cmd/arc/graph` (E2E, 1:1 with spec.md's acceptance scenarios per constitution Principle VIII) and `internal/app/lint/...` (renamed `MergeOp` fixture constants, no behavior change).
