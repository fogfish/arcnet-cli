# Quickstart: `arc lint`

Validates spec.md's three user stories end-to-end against a real local graph. `arc lint` needs no network access and makes no filesystem or git-history changes (spec FR-014, SC-006) — every scenario below can be re-run repeatedly with identical results.

## Prerequisites

- `git` on `PATH`
- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- A graph created by `arc init` with at least one document ingested via `arc apply` (see `specs/003-apply-patch/quickstart.md` Scenario 1 for a ready-made setup)

## Scenario 1 — A fully conformant graph passes cleanly (spec.md User Story 1, Acceptance Scenario 1)

```sh
$ arc lint
✅ 2 nodes checked, 2 passing, 0 failing
$ echo $?
0
```

## Scenario 1b — Violations across multiple rules and files are all reported in one run (spec.md User Story 1, Acceptance Scenarios 2-3)

```sh
$ cat >> entities/Transport\ Layer\ Security.md <<'EOF'

## Mentions
- mentions:: [[Nonexistent Node]]
EOF

$ sed -i '' 's/category:.*/category: [independent, abstract, occurrent]/' entities/Transport\ Layer\ Security.md

$ arc lint
❌ entities/Transport Layer Security.md:12 — [linkResolves] target "Nonexistent Node" does not exist
❌ entities/Transport Layer Security.md:3 — [entityCategory] category must decode to exactly four Sowa words, found 3
❌ 2 nodes checked, 0 passing, 2 failing
$ echo $?
1
```

**Expected outcome**: both violations — from two different checklist rules, in the same file — are reported in the same invocation; the run does not stop after the first one found.

## Scenario 2 — A broken link is caught precisely, everything else stays clean (spec.md User Story 2)

```sh
$ git checkout entities/Transport\ Layer\ Security.md   # revert scenario 1b's edits

$ cat >> entities/Transport\ Layer\ Security.md <<'EOF'

## Mentions
- mentions:: [[Not A Real Node]]
EOF

$ arc lint
❌ entities/Transport Layer Security.md:12 — [linkResolves] target "Not A Real Node" does not exist
❌ 2 nodes checked, 1 passing, 1 failing
```

Introducing a second node with a colliding basename is reported the same way (spec.md User Story 2, Acceptance Scenario 3):

```sh
$ cp resources/rfc8446.md "entities/rfc8446.md"

$ arc lint
❌ [uniqueBasename] basename "rfc8446" is used by more than one file: resources/rfc8446.md, entities/rfc8446.md
❌ entities/Transport Layer Security.md:12 — [linkResolves] target "Not A Real Node" does not exist
❌ 3 nodes checked, 1 passing, 2 failing
```

## Scenario 3 — An unresolved merge conflict is caught (spec.md User Story 3)

```sh
$ printf '<<<<<<< HEAD\nkind: entity\n=======\nkind: entity\ncategory: [independent, abstract, occurrent, script]\n>>>>>>> feature-branch\n' > entities/broken.md

$ arc lint
❌ entities/broken.md:1 — [mergeConflict] unresolved git merge-conflict marker found
❌ 3 nodes checked, 2 passing, 1 failing
```

**Expected outcome**: the conflicted file is reported once, for the conflict itself — not also flooded with secondary "invalid front matter" noise from attempting to parse the still-conflicted content (research.md D13).

## Scenario 4 — Verbose mode shows every node's status (user's explicit `-v` requirement)

```sh
$ arc lint --verbose
✅ resources/rfc8446.md
✅ sources/rescorla-2026-tls13.md
❌ entities/Transport Layer Security.md:12 — [linkResolves] target "Not A Real Node" does not exist
❌ 3 nodes checked, 2 passing, 1 failing
```

Compare against the default (non-verbose) run above: only the failing node appears, the passing nodes are omitted, but the same overall summary line closes both.

## Verifying read-only behavior (spec SC-006)

```sh
$ git status --short > /tmp/before.txt
$ arc lint
$ git status --short > /tmp/after.txt
$ diff /tmp/before.txt /tmp/after.txt
# empty — arc lint changed nothing
```
