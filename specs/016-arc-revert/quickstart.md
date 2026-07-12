# Quickstart: Validating `arc revert`

## Prerequisites

- A built `arc` binary (`go build ./cmd/arc`) from this feature's branch.
- A scratch directory outside this repo, with git available on `PATH`.

## Scenario A — undo the patch just applied (spec User Story 1)

```sh
cd /tmp && mkdir arc-quickstart-a && cd arc-quickstart-a
arc init
```

Write `patch-1.md` contributing a `resource` node `rfc-9110` (an entirely new document, nothing shared with anything else in the empty graph), then:

```sh
arc apply patch-1.md
git log --oneline
arc revert rfc-9110
git log --oneline
ls resources/ 2>/dev/null   # empty or absent
```

**Expected**: `resources/rfc-9110.md` no longer exists, exactly one new commit exists on top of `arc apply`'s own, and `arc revert`'s own summary reports `approach: whole-commit` (D3/D4 — nothing has touched this patch's files since it was applied, so it takes the fast path even though it's phrased generically, not specifically "is this literally HEAD").

## Scenario B — retract an old patch nothing has touched since (spec User Story 2)

Repeat Scenario A's setup, then apply a second, unrelated patch `patch-2.md` (a different document, no shared entities/resources) before reverting the first:

```sh
arc apply patch-1.md
arc apply patch-2.md
arc revert rfc-9110
```

**Expected**: `resources/rfc-9110.md` is removed, `approach: whole-commit` is still reported (D3's per-path eligibility test, not literal-HEAD), and every file `patch-2.md` touched is byte-for-byte unchanged.

## Scenario C — retract a patch whose node was later enriched (spec User Story 3, the crux case)

```sh
cd /tmp && mkdir arc-quickstart-c && cd arc-quickstart-c
arc init
```

Write `patch-1.md` contributing an `entity` node `tls-1.3` with a `notes` paragraph "Introduced in RFC 8446." Write `patch-2.md` contributing to the same `tls-1.3` entity: an additional `notes` paragraph "Widely deployed by 2026." plus a new `tags` value.

```sh
arc apply patch-1.md
arc apply patch-2.md
cat entities/tls-1.3.md
arc revert <patch-1's document id>
cat entities/tls-1.3.md
```

**Expected**: before the revert, `entities/tls-1.3.md` shows both `notes` paragraphs and the `tags` value. After the revert, `approach: per-node` is reported (patch-2 touched the same node file since patch-1's ingest), the file still exists, "Introduced in RFC 8446." is gone, "Widely deployed by 2026." and the `tags` value are unchanged, and `arc revert --verbose`'s per-node line for this node reads `reconciled (1 paragraph stripped)`.

## Scenario D — conflict-marker provenance (spec User Story 3, Acceptance Scenario 3)

Extend Scenario C: declare `notes`' merge behavior as `firstWriteWin` in `_schema/predicates/notes.md` before applying either patch, and have `patch-2.md` contribute a *different* `notes` value for `tls-1.3` (not an additional paragraph — a genuinely diverging scalar).

```sh
arc apply patch-1.md
arc apply patch-2.md
cat entities/tls-1.3.md   # shows a <<<<<<< existing / ======= / >>>>>>> conflict marker
arc revert <patch-1's document id>
cat entities/tls-1.3.md
```

**Expected**: patch-1 is the marker's frozen "existing" side (it wrote `notes` first); reverting it resolves D8(b) — the marker is replaced by patch-2's own "incoming" text, promoted now that patch-1's original is retracted. Repeat with the revert order swapped (revert patch-2 instead, re-running from a fresh copy) to exercise D8(a): the marker is replaced by patch-1's "existing" text instead, since patch-2 is self-documented as the marker's own incoming `sourceID`.

## Scenario E — already-retracted is a safe no-op (spec Edge Cases, SC-008)

```sh
arc revert <patch-1's document id>   # run again, same id
```

**Expected**: `skipped: true`, "nothing to retract," zero new commit, `git log --oneline` unchanged from before this second invocation.

## Scenario F — destructive-operation confirmation (research.md D10)

```sh
arc revert <patch-1's document id>          # interactive TTY: prompts y/N before removing anything
arc revert <patch-1's document id> --force  # non-interactive: no prompt
echo | arc revert <patch-1's document id>   # piped stdin, no --force: refuses rather than hanging or silently proceeding
```

## Scenario G — full command-level regression

```sh
go build ./... && go test ./... -cover
```

**Expected**: all packages pass, including `cmd/arc/graph` (E2E, 1:1 with spec.md's acceptance scenarios per constitution Principle VIII) and the widened `internal/app/graph/adapter/mock.VCS` fake used by both `apply_test.go` and the new `revert_test.go`.
