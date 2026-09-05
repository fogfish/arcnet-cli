# Phase 0 Research: Graph Statistics (`arc stats`)

**Feature**: 032-arc-stats | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

Every decision below is grounded in code that already exists in this
repository. Where the spec and the existing implementation disagree, the
disagreement is named explicitly and resolved here rather than deferred to
implementation.

---

## D1 — Two node populations, not one

**Decision**: `Stats` walks the graph **twice conceptually, once physically**,
producing two distinct populations from a single traversal:

1. **Census population** — every file the walk finds that `core.LooksLikeNode`
   recognizes, *including* `_schema/`. Feeds FR-002 (total nodes) and FR-004
   (per-type breakdown).
2. **Content population** — the census population minus schema documents.
   Feeds FR-006 (broken links), FR-014 (orphans/stubs), FR-015 (degree, hubs).

**Rationale**: The spec's FR-002 demands schema documents be counted, but
SC-003 demands the broken-link count match `arc lint` exactly — and
[`lint/service.walkNodeFiles`](../../internal/app/lint/service/lint.go#L265-L270)
deliberately excludes `_schema/` from the basename universe ("schema documents
are exempt from ordinary content rules and never enter the basename-uniqueness
index"). Resolving link targets against a universe that *includes* schema
documents would make a link to a schema document resolve in `arc stats` and
break in `arc lint` — a direct SC-003 violation.

These are genuinely different questions ("how many files does this graph
hold?" vs "is this graph's connective tissue intact?") and the report labels
them as such. FR-006a already requires the report to make its counting rules
legible; this is the same discipline applied to populations.

**Alternatives considered**:
- *One population including schema* — breaks SC-003.
- *One population excluding schema* — breaks FR-002, and hides schema
  documents from the census the user explicitly asked to be complete.
- *Make lint include schema* — changes `arc lint`'s behavior and its spec
  (004/031), out of scope, and would create violations in every existing graph.

---

## D2 — Extending `walkNodeFiles` without forking it

**Decision**: Refactor [`graph/service.walkNodeFiles`](../../internal/app/graph/service/grep.go#L35-L69)
into a shared `walkFiles(store, skipDir func(string) bool)` and express the
two callers as thin wrappers:

- `walkNodeFiles(store)` — skips `.arc` and `_schema` (**behavior unchanged**;
  Grep, Subgraph, Match, Context all keep their current universe).
- `walkGraphFiles(store)` — skips `.arc` only. New; used by Stats for the
  census population.

**Rationale**: Principle V forbids the copy-pasted third walker this would
otherwise become — the existing doc comment already warns that "a second,
copy-pasted walker in this package would be exactly the drift Principle V
exists to prevent." Parameterizing the skip predicate is the smallest change
that serves both universes, and it is a pure refactor: existing callers get
byte-identical results, provable by the existing Grep/Subgraph/Match tests
staying green untouched.

**Alternatives considered**:
- *A boolean `includeSchema` parameter* — a boolean parameter at a call site
  reads as `walkNodeFiles(store, true)`, which communicates nothing; a named
  wrapper does.
- *Stats walks the filesystem itself* — duplicates the walker and re-introduces
  the `.arc` exclusion by hand; exactly the drift D2 avoids.

---

## D3 — Node identity is content-derived, never folder-derived

**Decision**: A file is a node when `core.LooksLikeNode(raw)` says so, and its
type is `core.Node.Type` as parsed from its own front matter. No statistic
consults the file's path.

**Rationale**: FR-004a forbids deriving any figure from a path, because the
user established that folder layout is not fixed. This is achievable because
[`core.LooksLikeNode`](../../internal/core/markdown.go#L201-L212) is purely
content-based (it accepts any front matter carrying `@id`, `@type`, or `kind`),
and every population the census must separate is identifiable by declared type:
schema documents declare `@type: Class`/`Property`
([schema.go:21-22](../../internal/app/schema/service/schema.go#L21-L22)) and the
derived index declares `@type: Timeline`.

**Note on lint's heuristic**: `lint/service.isForeignFile` consults
`isNodeTypeFolder` *first* and falls back to `LooksLikeNode`. That folder check
is an accelerator, not the decision — a node moved to a user-created folder is
still recognized by the content fallback. Stats uses the content test alone, so
it needs no such fallback and inherits no folder assumption.

**Alternatives considered**:
- *Classify by folder* — directly violates FR-004a and silently misreports any
  graph whose owner reorganized it.

---

## D4 — Verbosity is a service parameter, not a render mode

**Decision**: `Stats(ctx, mounter, index, dir, detail bool) (kernel.StatsResult, error)`.
`kernel.StatsResult.Detail` is a `*StatsDetail` — `nil` when `detail` is false,
tagged `json:"detail,omitempty"`.

**Rationale**: Clarification Q2 chose "JSON mirrors verbosity", which the
existing output layer cannot express through rendering alone:
[`ResolveMode`](../../internal/bios/output.go#L44-L52) returns `ModeJSON`
*before* it tests `Verbose`, and `jsonPrinter` marshals the whole value
unconditionally. Making the detail section a nil-able pointer computed on
demand satisfies FR-020a **without touching `bios` at all** — the omitted
field simply disappears from the marshalled JSON, and `omitempty` on a nil
pointer is precisely the "absent, not null" distinction FR-020a requires.

It also satisfies FR-018's stronger reading ("verbosity gates what the tool
computes"): the hub ranking and degree distribution are never computed on a
default run.

**Alternatives considered**:
- *Always compute, render selectively* — violates FR-018 and puts detail
  fields into default `--json` output, violating FR-020a.
- *Teach `bios` about a JSON-verbose mode* — changes a shared component every
  command depends on, to serve one command. Rejected under Principle V.

---

## D5 — Ingestion figures read from timeline period nodes

**Decision**: Year and month figures come from parsing `@type: Timeline` nodes
found in the census walk, counting each period node's own entries. No source
node's `published` field is read for this purpose.

**Rationale**: The user's explicit direction (Clarification Q1 preamble;
FR-007). `core.TimelinePeriods` already establishes the period-code shapes
(`"2006"` yearly, `"2006-01"` monthly), and
[`apply.go`](../../internal/app/graph/service/apply.go#L675-L690) writes them to
`timeline/yearly/<YYYY>.md` and `timeline/monthly/<YYYY-MM>.md`. Reading the
index keeps `arc stats` and the graph's own timeline in agreement by
construction — any drift between index and sources is `arc lint`'s finding to
report, not a discrepancy for two commands to disagree about.

**Period identity is taken from the node's own `@id`, not its path**, per D3 —
a yearly period is one whose id parses as `2006`, a monthly one as `2006-01`.

**Alternatives considered**:
- *Derive from each Source node's `published`* — explicitly rejected by the
  user; would also silently disagree with the graph's own index.
- *Read paths `timeline/yearly/*`* — violates FR-004a.

---

## D6 — Reusing lint's link-resolution rule rather than restating it

**Decision**: Broken links are computed in `graph/service` over the **content
population** (D1) using the same rule
[`checkLinksResolve`](../../internal/app/lint/service/rules_links.go#L29-L48)
applies: flatten `HRefs` + `Edges`, count each **distinct target per node**
that resolves to no basename in the population.

`graph` MUST NOT import `internal/app/lint` — two sibling use-cases importing
each other is the cycle Principle V's "acyclic and wide" rule forbids. The rule
is small enough (a set-per-node over a basename map) that restating it in
`graph/service` is correct; what must not drift is the *population* and the
*counting rule*, both pinned by contract tests (see [contracts/](./contracts/)).

**Rationale**: SC-003 demands agreement, not shared code. A contract test that
runs both `Lint` and `Stats` over the same fixture and asserts the broken-link
count equals the `linkResolves` violation count enforces agreement far more
honestly than a shared helper would, because it fails when *either* side
drifts.

**Alternatives considered**:
- *Extract the rule into `internal/core`* — plausible, but premature: the rule
  needs lint's `Violation` shape on one side and a bare count on the other.
  YAGNI (Principle V). Revisit if a third caller appears.
- *`graph` imports `lint`* — forbidden import direction; also drags lint's VCS
  port into a command that needs no git.

---

## D7 — `Timeline` nodes and the derived-index skew

**Decision**: Timeline nodes participate in the census (FR-002) and in the
per-type breakdown, and their `cites::` edges **do** count toward the edge
total (FR-003) and the per-predicate breakdown (FR-012).

**Rationale**: FR-003 says "every structural edge", without exception, and
FR-012's counts must sum to it. Excluding the index would make `cites` appear
under-represented and would need a caveat in the report that FR-006a-style
transparency is meant to avoid. The per-type breakdown already shows how many
Timeline nodes exist, so a reader can see the index's contribution rather than
having it silently removed.

**Consequence, stated plainly**: in a graph with many sources, `cites` will
dominate the predicate breakdown because the timeline index cites every source.
That is a true fact about the graph's structure, not an artifact.

**Alternatives considered**:
- *Exclude timeline edges* — makes FR-012 not sum to FR-003, or requires a
  second "content edges" total. More figures, less clarity.

---

## D8 — Unreadable vs foreign files

**Decision**: FR-009's "could not read or interpret" count splits into two
reported figures:

- **Unreadable** — the file was found but could not be opened or parsed as a
  node (`core.ParseNode` failed).
- **Foreign** — the file is markdown that `core.LooksLikeNode` does not
  recognize as a node at all.

**Rationale**: Spec 031 established that a graph root may be shared with a host
project, so an ordinary `README.md` sitting beside the graph is *expected*, not
a defect. Folding it into an "unreadable" count would report a healthy graph as
damaged. Lint already draws exactly this distinction (`foreign` vs a read
error), and `arc stats` reporting a different number of "problem files" than
`arc lint` for the same directory is the kind of disagreement SC-003 exists to
prevent.

Foreign files are **excluded from every statistic** — they are not nodes.

**Alternatives considered**:
- *One combined count* — misreports a shared repository's ordinary files as
  graph damage.

---

## D9 — Degree, hubs, and determinism

**Decision**:
- Out-degree is computed over the **content population**, counting `Edges`
  occurrences per node, including zero-degree nodes in the denominator
  (FR-015).
- Median of an even-sized population is the mean of the two central values.
- In-degree ranks nodes by incoming `Edges` + `HRefs` count; ties break by
  ascending node id; the list is truncated to 10 (FR-015).
- Every map-derived list is sorted before it leaves the service (FR-005,
  FR-019, FR-021, SC-005).

**Rationale**: Go map iteration order is randomized per run, so any figure
derived from a map and rendered without an explicit sort would violate SC-005
non-deterministically — the worst kind of failure, since it passes locally and
flakes in CI. Sorting at the service boundary (not the printer) means the
`--json` contract is deterministic too, which FR-021 requires.

**Alternatives considered**:
- *Sort in the printer* — leaves `--json` non-deterministic, violating FR-021.

---

## D10 — Command surface

**Decision**: `arc stats`, `cobra.NoArgs`, no command-specific flags. Verbosity,
JSON, quiet, and color come from the existing root persistent flags. Registered
in [`root.go`](../../cmd/arc/root.go) alongside `lint`.

**Rationale**: Mirrors `arc lint` exactly — the closest existing sibling, also a
read-only whole-graph analysis over the working directory. Principle IX prefers
flags over positional arguments and this command has no subject argument;
Clarification Q3 removed the one candidate flag (a CI gate threshold).

`--plain` is **not** implemented: no existing command in this repository
implements it, and adding it here alone would be an inconsistent surface. Noted
as a project-wide gap, not a gap in this feature.

**Alternatives considered**:
- *`arc stats [path]`* — no other read-only command takes a path; would be the
  only one.
- *`arc graph stats`* — the command tree is flat; `lint`, `grep`, `subgraph`
  are all top-level.

---

## Resolved unknowns

| Unknown from Technical Context | Resolution |
|---|---|
| Which node universe feeds which figure | D1 — two populations, both named in the report |
| How to walk `_schema/` without forking the walker | D2 — parameterized skip predicate |
| How verbosity reaches `--json` | D4 — nil-able `*StatsDetail`, no `bios` change |
| Where ingestion figures come from | D5 — timeline period nodes, by declared id |
| How SC-003 agreement is guaranteed | D6 — cross-command contract test |
| Determinism strategy | D9 — sort at the service boundary |

No `NEEDS CLARIFICATION` markers remain.
