# Phase 0 Research: `arc apply batch`

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-08-06

All Technical Context unknowns are resolved below. No `NEEDS CLARIFICATION`
markers remain.

---

## D1 — Filesystem abstraction: stdlib `io/fs` via the existing `fsys.Local`

**Decision**: Traverse the patch directory with `fs.WalkDir` over the
`fsys.Store` returned by `fsys.Local{}.Mount(absPatchDir)`. Add **no**
dependency to `go.mod`.

**Rationale**: `internal/adapter/fsys.Store` already embeds `fs.FS`,
`fs.StatFS`, and `fs.ReadDirFS`
([types.go:32-38](../../internal/adapter/fsys/types.go#L32-L38)), which is the
complete read capability discovery needs. `fs.WalkDir` accepts any `fs.FS`, so
recursion (FR-001) is stdlib behaviour, not new code. Constitution Principle
VII's rule that `os.*` file/directory calls appear nowhere outside
`internal/adapter/fsys` is satisfied automatically: batch never names `os`.

**Alternatives considered**:

- **`github.com/fogfish/stream` + `s3://` prefix** (the original planning
  input). Rejected by explicit decision during planning. It contradicts
  constitution v1.5.0's mandate that `fsys` be built exclusively on stdlib
  `io/fs` with "no third-party filesystem or object-storage library," and the
  dependency cost the revert cited was re-measured and confirmed: 19 AWS SDK
  v2 modules. See [plan.md § Deferred: remote patch sources](plan.md#deferred-remote-patch-sources)
  for the full record, including the two code-level blockers a future remote
  feature must solve.
- **`filepath.WalkDir` directly.** Rejected: it calls `os` from outside
  `fsys`, a direct Principle VII violation, and gains nothing over the
  `fs.FS` route.

---

## D2 — Layering: orchestration in `internal/app/graph`, not `cmd/`

**Decision**: Discovery, classification, ordering, and iteration live in
`internal/app/graph/service/batch.go`, exposed through a thin
`appgraph.ApplyBatch` delegator in `component.go`.
`cmd/arc/graph/batch.go` contains only Cobra wiring and rendering.
`internal/app/graph/service/apply.go` is **not modified**.

**Rationale**: Constitution Principle III states `cmd/` "MUST NOT contain
business logic," and ADR 001 places use-case logic under `/internal/app` with
Cobra as the sole primary adapter. Tree walking, manifest classification,
date sorting, and outcome accumulation are business logic. Placing them
correctly also makes them unit-testable without Cobra — Principle III's
stated rationale — and lets `--json` render from a domain value instead of
being assembled inside a command handler.

This is a deliberate, documented divergence from the planning input's "only
cmd level concern" phrasing. The part of that instruction that carries the
real risk — *don't touch the patching algorithm* — is honoured exactly:
`service.Apply` is called unchanged, per patch.

**Alternatives considered**:

- **Everything in `cmd/arc/graph/batch.go`.** Rejected: Principle III
  violation with no offsetting benefit; would also force `--json` assembly
  into the handler.
- **A separate `internal/app/batch` use-case.** Rejected: ADR 001's port
  isolation rule 1 forbids one use-case importing another's internals, and
  batch's whole job is calling the `graph` use-case's own apply. It belongs
  inside `graph`, not beside it.

---

## D3 — Discovery: `fs.WalkDir`, skipping dot-directories and non-Markdown

**Decision**: Walk from `"."` on the mounted patch store. Skip any directory
entry whose name begins with `.` by returning `fs.SkipDir` (FR-019). Consider
only regular files whose name ends in `.md`; ignore everything else silently
and leave it out of the summary.

**Rationale**: FR-019 requires version-control metadata and other hidden
directories never be treated as patch input; a leading-dot test covers `.git`,
`.specify`, `.github`, and the general case in one rule with no allowlist to
maintain. The `.md` filter comes straight from the spec's `*.md` scope, and
the spec's edge cases say non-Markdown files do not even appear in the
summary.

**Symlinks**: `fs.WalkDir` over `os.DirFS` uses `ReadDir`, whose entries carry
`lstat`-derived modes — a symlink to a directory reports `ModeSymlink`, so
`IsDir()` is false and `WalkDir` does not descend into it. The spec's
assumption that a link cycle cannot make discovery run forever therefore holds
with no extra cycle-detection code.

**Nested patch directory inside the graph**: harmless. Graph node files are
`.md` but do not declare `kind: patch`, so D4's classifier passes them over
rather than applying them.

---

## D4 — Classification: reuse `core.LooksLikePatch`, then `core.ParsePatch`

**Decision**: For each discovered `.md` file, read its bytes and call the
**existing** `core.LooksLikePatch(raw)`
([markdown.go:95](../../internal/core/markdown.go#L95)):

- `false` → the file is **not a patch** (FR-003): counted as passed over,
  never applied, never a failure.
- `true` → call `core.ParsePatch(bytes.NewReader(raw), index)`. Success yields
  `Document` and `Published` for ordering; any error makes the file a
  **failed patch** carrying that error as its reason.

**Rationale**: `LooksLikePatch` already exists for precisely this
distinction — it checks only that front matter declares `kind: patch`,
"independent of whether the rest of the document is otherwise well-formed."
That is exactly the FR-003 / FR-020 split the spec asks for, with no new
parsing code and no risk of the two paths drifting apart.

FR-020 (absent or uninterpretable publication date ⇒ failure) falls out for
free: `decodePatchManifest`
([markdown.go:337-340](../../internal/core/markdown.go#L337-L340)) rejects a
manifest whose `published` value does not decode via `decodeManifestDate`, so
such a file is a `true`/parse-error case, never silently ordered somewhere
arbitrary.

**Alternatives considered**:

- **A new manifest-only parser** to avoid parsing the body twice. Rejected as
  premature (Principle V, YAGNI) — see D12 for the measured shape of the cost
  and the trigger that would justify revisiting it.

---

## D5 — Ordering: publication date ascending, path as tie-break

**Decision**: Sort the classified patches by `core.Patch.Published`
(`time.Time`) ascending; where two dates are equal, sort by the file's
slash-separated path relative to the patch directory, ascending, byte-wise.
The resulting plan is fixed **before** the first patch is applied.

**Rationale**: FR-004 requires oldest-first application so the timeline and
every first-writer-wins merge reflect publication order rather than
filesystem enumeration order — which is precisely the failure mode that
made manual ordering error-prone. FR-005 requires the tie-break be
deterministic across runs and machines; the relative path is stable, unlike
`fs.WalkDir`'s directory-entry order across filesystems, and unlike any
inode- or mtime-derived key.

Comparison is on the full `time.Time`. Dates parsed via the `"2006-01-02"`
layout land at midnight UTC, so a date-only manifest orders by calendar date
exactly as the spec assumes, and a full RFC3339 timestamp still orders
sensibly against it.

Fixing the plan before the first commit means the run's order cannot be
perturbed by files added to the directory mid-run.

---

## D5b — Where classification failures sit in the order

**Decision**: A candidate that declares `kind: patch` but fails to parse
(D4) has **no usable publication date**, so it is placed **after** every
successfully classified patch, ordered among its peers by relative path.

**Why this needs deciding at all**: `core.ParsePatch` decodes the manifest and
then the body, returning `Patch{}, err` on either failure
([markdown.go:63-84](../../internal/core/markdown.go#L63-L84)). A patch whose
manifest is fine but whose *body* is malformed therefore loses its
`Published` value along with everything else. Left naive, every such candidate
would carry the zero `time.Time` and sort to the very **front** of the plan —
ahead of every real patch, purely as an artefact of Go's zero value.

**Rationale for last rather than first**:

- FR-020 says a dateless patch must never be "applied at an arbitrary
  position in the order." Appending undated candidates after all dated ones is
  the one placement that is not a guess about where they belong. (They are
  never applied regardless — the position governs only where they are
  reported and, under `--fail-fast`, when the run stops.)
- It preserves the shape of User Story 4's independent test. Sorting failures
  first would mean `--fail-fast` produced **zero** commits whenever the
  directory contained any unparsable file, so "patches ordered before it are
  committed" could never be exercised. Placing them last keeps `--fail-fast`
  halting mid-order on the realistic case — a patch that classifies cleanly
  but fails during application (a merge or commit error).
- Under `--fail-fast` it does not discard work already known to be valid.

**Alternatives considered**:

- **Sort undated candidates first**, so `--fail-fast` refuses before producing
  any commit. Rejected: it makes `--fail-fast` behave as an all-or-nothing
  directory validation rather than the "halt at the first failing patch"
  FR-011 specifies, and it silently reorders the report so the user's own
  publication ordering no longer leads.
- **Expose a manifest-only decoder in `internal/core`** so a body-broken patch
  keeps its date and sorts naturally into the middle. Genuinely the most
  faithful ordering, and it would subsume D12's double-parse cost too — but it
  changes `internal/core`, which this feature otherwise leaves untouched.
  Deferred as the natural follow-up if either concern becomes real.

---

## D6 — Execution: call `service.Apply` once per patch, unchanged

**Decision**: Iterate the ordered plan and call the existing
`service.Apply(ctx, mounter, vcs, reporter, index, schema, graphDir, patchPath)`
for each entry, with `patchPath` reconstructed as
`filepath.Join(absPatchDir, filepath.FromSlash(rel))`.

**Rationale**: Three functional requirements are satisfied by reuse alone,
which is the strongest argument for not reimplementing anything:

- **FR-006 / FR-007** (identical semantics, exactly one commit per patch):
  guaranteed because it is literally the same function `arc apply` calls.
- **FR-008 / FR-009** (skip already-tracked, evaluated at the moment each
  patch is reached): `Apply` performs its own `vcs.IsTracked` check per
  invocation ([apply.go:240](../../internal/app/graph/service/apply.go#L240))
  and returns `ApplyResult{Skipped: true}`. Because the check runs when the
  patch is reached rather than up front, a document ingested earlier in the
  same run is correctly recognised by a later duplicate — FR-009 with no
  bookkeeping of our own.
- **FR-012** (no partial state on failure): `Apply` already rolls back
  newly-created files and commits only at the end.

`Apply` re-mounts the graph and re-runs its `guardIsGraph` /
`guardNoOldFormatNodes` preflight on every call. That redundancy is accepted
deliberately: it is the price of leaving the algorithm untouched, and it is
cheap next to the one `git commit` subprocess each applied patch already
costs.

**FR-002 preflight**: the batch itself validates the target path and the graph
*before* reading any patch, so a bad invocation fails immediately rather than
after a directory walk.

---

## D7 — Failure policy: continue by default, `--fail-fast` to halt

**Decision**: Record each failure and continue with the remaining patches
(FR-010). A `--fail-fast` boolean flag switches to halting at the first
failure (FR-011), leaving the remainder unprocessed and counted as such.

**Rationale**: Chosen by the user during specification. Continue-on-error is
the right default for a bulk corpus import, where extraction failures are
individually rare and collectively certain — halting would degrade batching
into a manual bisect loop. `--fail-fast` serves pipelines where a failure
signals something systematically wrong.

**Flag naming**: long form only. `-f` is reserved by Principle IX for
file/force and is not taken here.

An **unprocessed** outcome exists only under `--fail-fast`; in the default
mode every planned patch reaches a terminal outcome.

---

## D8 — Exit code: print the summary, then return `bios.ErrSilent`

**Decision**: The command prints its complete summary, then returns
`bios.ErrSilent` when at least one patch failed, and `nil` otherwise.

**Rationale**: This is the codebase's established mechanism for "exit non-zero
after the command has already printed its own complete result" —
`cmd/arc/main.go:42` recognises the sentinel and exits 1 without rendering a
second, redundant error line, and `arc lint`, `arc grep`, and `arc revert`
already use it. FR-014 is satisfied without inventing a parallel exit path.

A run in which every patch was skipped, or in which nothing applicable was
found (FR-022), is a **success** — it returns `nil`.

---

## D9 — Two reporters: batch progress on by default, per-node detail on `--verbose`

**Decision**: Construct two `bios.Reporter` values:

- **Batch reporter** — `bios.NewReporter(bios.Quiet, false)`: emits one line
  per patch as it completes (document, outcome). Visible by default,
  suppressed by `--quiet` (FR-015).
- **Per-patch reporter** — `bios.NewReporter(bios.Quiet, !bios.Verbose)`:
  handed to `service.Apply`, keeping its per-node and per-predicate detail
  `--verbose`-gated, exactly as `arc apply` does today
  ([apply.go:135](../../cmd/arc/graph/apply.go#L135)).

**Rationale**: FR-015 requires per-patch progress by default — a long batch
must not sit silent (Principle X). But `Apply`'s existing per-node reporting
is far too dense to enable by default across a hundred patches; it would bury
the per-patch line it is meant to accompany. Splitting the two keeps each at
its documented visibility and changes neither command's established
behaviour.

Both write to **stderr**, per Principle X's "diagnostics and progress go to
stderr" rule, leaving stdout carrying only the final summary — so
`arc apply batch ... --json | jq` stays clean.

---

## D10 — Aggregate conflicts and warnings into the summary

**Decision**: Accumulate every `ApplyResult.Conflicts` entry and every
`ApplyResult.Warnings` entry across the whole run into the batch result, and
render them once in the final summary.

**Rationale**: FR-016 exists because a conflict flagged at patch 3 of 100
scrolls out of view long before the run ends; the summary is the only place a
user reliably reads. FR-017 requires unregistered-kind warnings not to abort
the run — which is already `Apply`'s behaviour, so batch only has to carry
them forward rather than swallow them.

Conflicts are surfaced as file paths needing manual resolution, mirroring
`arc apply`'s existing `PostRunE` hint text so the two commands read the same
way.

---

## D11 — The patch directory is read-only by construction

**Decision**: Batch calls only `Open` / `ReadDir` / `Stat` on the patch
store, never `Create` or `Remove`.

**Rationale**: FR-021 makes the patch directory read-only input. Routing
through `fs.FS`'s read-only surface means the requirement is enforced by the
types rather than by discipline — there is no write call to review for.
Writes happen exclusively inside `service.Apply`, against the separately
mounted graph root.

---

## D12 — Accepted cost: each applicable patch is parsed twice

**Decision**: Accept that an applicable patch is parsed once during planning
(to obtain `Published` for ordering) and again inside `service.Apply`'s own
`readPatch`. Do not optimise now.

**Rationale**: Ordering by publication date cannot begin until every
candidate's date is known, so a planning pass is unavoidable; the only
question is whether it parses the manifest alone or the whole document.
Reusing `core.ParsePatch` costs a second body parse but adds zero new parsing
code and no second code path that could drift from the real parser — a
correctness argument that outweighs the cost here, since the run is already
dominated by one `git commit` subprocess per applied patch (D6), which is
orders of magnitude more expensive than a Markdown parse.

**Revisit if**: profiling on a realistically large corpus shows parse time is
material against commit time. The fix would then be a manifest-only parser in
`internal/core`, shared by both paths — not a bespoke one in the service.

---

## D13 — Errors declared as `faults` constants

**Decision**: Declare each new expected error condition once as a package-level
`faults.Type` / `faults.SafeN` constant in
`internal/app/graph/service`, and wrap with `.With(err, args...)`. Callers
that must branch use `errors.Is`.

**Rationale**: Constitution Principle XII and the Mandatory Libraries section
forbid ad hoc `fmt.Errorf("...: %w", err)` for this purpose. Conditions needing
a constant: target path does not exist; target path is not a directory; the
patch directory could not be read. Per-patch failure *reasons* are carried as
strings inside the batch result for reporting, not raised as errors — the run
continues past them by design (D7).

---

## Requirements coverage

| Requirement | Resolved by |
|---|---|
| FR-001 recursive discovery | D1, D3 |
| FR-002 preflight refusal | D6, D13 |
| FR-003 non-patch passed over | D4 |
| FR-004 publication-date order | D5 |
| FR-005 deterministic tie-break | D5, D5b |
| FR-006 identical apply semantics | D6 |
| FR-007 one commit per patch | D6 |
| FR-008 skip already tracked | D6 |
| FR-009 tracked check at reach time | D6 |
| FR-010 continue on failure | D7 |
| FR-011 `--fail-fast` halt | D7 |
| FR-012 no partial state | D6 |
| FR-013 summary counts and reasons | D2, D10 |
| FR-014 exit status | D8 |
| FR-015 per-patch progress | D9 |
| FR-016 aggregated conflicts | D10 |
| FR-017 warnings do not abort | D10 |
| FR-018 machine-readable output | D2, [contracts/batch-result.schema.md](contracts/batch-result.schema.md) |
| FR-019 skip hidden directories | D3 |
| FR-020 bad publication date is a failure | D4, D5b |
| FR-021 patch directory read-only | D11 |
| FR-022 nothing to apply succeeds | D8 |
