# Quickstart: Validating ARCNET-CORE v0.11 Conformance

**Feature**: `022-reference-type-folders` · **Date**: 2026-08-23

A runnable walkthrough proving the feature end to end. Details live in
[`contracts/folder-layout-contract.md`](contracts/folder-layout-contract.md) and
[`contracts/core-type-vocabulary-contract.md`](contracts/core-type-vocabulary-contract.md);
this file is the *procedure*.

## Prerequisites

```sh
export GOROOT="$(ls -d /opt/homebrew/Cellar/go/*/libexec | tail -1)"
export PATH="$GOROOT/bin:$PATH"
go version    # expect go1.27.0 or later
```

The shell's default `GOROOT` points at an uninstalled Go version and every `go` command fails
with `cannot find GOROOT directory`. Export it first or nothing below runs.

```sh
go build ./... && go test ./...    # must be green before starting
go build -o /tmp/arc ./cmd/arc
```

## Scenario 1 — Canonical layout (spec US3, contract C3)

```sh
cd "$(mktemp -d)" && /tmp/arc init .
find . -type d -not -path './.git*' -not -path './.arc*' | sort
```

**Expect exactly:**

```
./Entity
./Reference
./Resource
./Source
./_schema
./_schema/Class
./_schema/Property
./timeline
./timeline/monthly
./timeline/yearly
```

**Must NOT appear:** `sources`, `entities`, `resources`, `references`, `_schema/predicates`,
`_schema/types`.

> On macOS this listing cannot distinguish `Source` from `source`. It is a smoke check only —
> the binding assertion is the exact-string test required by contract C7.

## Scenario 2 — Corrected `Resource`, new `Reference` (spec US1, contract C2/C3)

```sh
cat _schema/Class/Resource.md
grep -E '^- (required|optional):: ' _schema/Class/Resource.md
```

**Expect** required `text`, `tags`, `mentionedIn`; optional `notes`.
**Expect no occurrence** of `ref`, `relevance`, `url`, `authors`, `year`, `doi`, `status`,
`isCitedBy`:

```sh
grep -cE 'ref|relevance|url|authors|year|doi|status|isCitedBy' _schema/Class/Resource.md   # 0
```

```sh
cat _schema/Class/Reference.md
```

**Expect** required `title`, `ref`, `relevance`; optional `url`, `authors`, `year`, `doi`,
`status`, `isCitedBy`, `notes`; and `subClassOf:: [[Node]]`.

Every predicate it names must already be registered:

```sh
for p in title ref relevance url authors year doi status isCitedBy notes; do
  test -f "_schema/Property/$p.md" || echo "MISSING: $p"
done
```

**Expect no output.**

## Scenario 3 — A clean graph lints clean (spec US1 scenario 5)

```sh
/tmp/arc lint .
```

**Expect** zero violations, exit 0. This is the check that catches a `Reference` declaring a
predicate nobody seeded.

## Scenario 4 — Prose lands on the right predicate (spec US2, contract C4)

Write a patch carrying one node of each affected type, each with leading body prose, apply it,
then read the files back:

```sh
/tmp/arc apply patch.md
cat Resource/<id>.md      # leading prose rendered bare — stored under `text`
cat Reference/<id>.md     # leading prose rendered bare — stored under `relevance`
```

**Expect** each node's opening paragraph rendered as bare prose directly under the H1, with no
`## Relevance` or `## Text` heading. A heading appearing there means the key was treated as a
non-slot predicate — the exact defect this feature fixes.

Confirm the key by round trip:

```sh
/tmp/arc subgraph <id> > /tmp/out.patch.md
grep -A2 '^## ' /tmp/out.patch.md
```

## Scenario 5 — Type-named filing, including an unknown type (spec US3 scenarios 2-3)

Apply a patch carrying `Source`, `Entity`, `Resource`, `Reference`, and a domain type the tool
does not know (e.g. `Thought`):

```sh
/tmp/arc apply mixed.patch.md
ls Source/ Entity/ Resource/ Reference/ Thought/
```

**Expect** `Thought/` — **not** `Thoughts/`, **not** `thoughts/`. This is the identity-derivation
check; the retired code would have produced `Thoughts/`.

## Scenario 6 — Timeline stays exempt (spec US3 scenario 4, contract C2)

```sh
ls timeline/yearly/ timeline/monthly/
ls -1 | grep -x 'Timeline' && echo "FAIL: flat Timeline/ folder created"
```

**Expect** period files bucketed, and **no** `Timeline/` folder.

> Do **not** use `test -d Timeline` here. On macOS/APFS it case-folds onto the
> `timeline/` that is supposed to exist and reports a false failure — verified during the
> T050 walkthrough. `ls -1 | grep -x` compares the name the filesystem actually stores,
> which is contract C7's rule applied to the shell.

## Scenario 7 — Revert round trip (spec US2 scenario 3, US3 scenario 6)

```sh
/tmp/arc revert <source-id>
ls Resource/ Reference/
```

**Expect** every node the ingest created removed from its type-named folder, backlink referrers
rewritten, and no prose orphaned. This exercises `revertLeadingKey` and `referrerPath` together
— the pair most likely to drift from `core`'s own table.

## Scenario 8 — The break fails closed (spec Assumptions)

Against a graph created by a **pre-feature** build:

```sh
/tmp/arc lint /path/to/old-graph
```

**Expect** a non-zero exit and an error naming `_schema/Property` as missing. **Expect no
writes**: the graph must be byte-identical afterwards.

```sh
git -C /path/to/old-graph status --porcelain    # empty
```

This is the verified behaviour that makes the no-migration decision safe — the failure is loud
and precedes any mutation.

Verified during the T050 walkthrough: `arc lint` and `arc apply` both refuse with
`schema folder _schema/Property is missing or unreadable`, exit 1, and leave `git status`
empty with `HEAD` unmoved. `arc grep` and `arc subgraph` exit non-zero without naming the
schema folder — they traverse content without resolving the schema first (research.md §1.3b)
— but both are strictly read-only, so the "no writes before the refusal" guarantee is
unaffected.

## Completion check

```sh
go test ./...
git grep -nE '"(sources|entities|resources|references)"|_schema/(types|predicates)' -- '*.go'
```

**Expect** all tests green and the grep to return nothing outside `specs/` and `CORE-FIX.md`.
A surviving hit in `internal/` or `cmd/` is a missed rename.
