# Phase 0 Research: Explicit Serve Context & Shareable Graph State

**Feature**: `034-serve-dir-public-state` | **Date**: 2026-09-11

**Scope directive from the user**: *"The feature does not require any structural
changes in the code base. Only impacted component has to be modified. Keep minimal
changes to interfaces, adjust only the behaviour of `serve` and `init` commands."*

Every decision below is measured against that directive first: a decision that
requires a new package, a new port interface, or a signature change to an existing
exported function is rejected unless nothing else satisfies the spec.

---

## D1 — Where the `serve <dir>` argument is resolved

**Decision**: `cmd/arc/graph/serve.go` only. `NewServeCmd` changes
`Args: cobra.NoArgs` → `cobra.MaximumNArgs(1)`, and the existing
`dir, err := filepath.Abs(".")` line becomes `dir, err := resolveGraphDir(args)`,
a five-line package-private helper in the same file.

**Rationale**: `buildServer(ctx, dir)` — and every one of the eight tool handlers
it registers — already takes `dir` as a parameter and closes over it. There is
literally nothing downstream of `RunE` that assumes the current working directory.
The whole of User Story 1 is one argument-resolution line plus Cobra metadata.

**Alternatives considered**:

- *A global `--graph` persistent flag on the root command.* Rejected: it would
  change the behaviour of every command, which the scope directive and the spec's
  Out of Scope section both forbid. Also worse UX for the MCP case — CLIG
  Principle IX prefers a single primary "subject" positional argument over a flag
  when the argument *is* the subject, which is exactly the case here.
- *An `ARC_GRAPH` environment variable.* Rejected: an MCP server entry can express
  a positional argument as naturally as an env var, so it buys nothing, and
  Constitution XI would then require it to slot into the full flag → env → config
  precedence chain — real interface surface for no gain.
- *Reusing `ctrl.resolveInitDir`.* Rejected: it lives in `package ctrl` and
  exporting it to share five lines across command packages is worse coupling than
  the duplication. `cmd/arc/graph/batch.go` already resolves its own
  `filepath.Abs(args[0])` inline — this follows the established local precedent.

---

## D2 — Distinguishing "missing", "not a directory" and "unreadable" from "not a graph"

**Decision**: extend `service.EnsureGraph` (`internal/app/graph/service/node.go`)
to stat `"."` through the already-mounted store *before* `guardIsGraph`, mapping
the three failure classes onto three new `faults.Safe1` sentinels in the same
package's `errors.go`. No signature changes; no new fsys API.

```
store.Stat(".")  →  nil                    → continue to guardIsGraph
                 →  fs.ErrNotExist         → ErrGraphDirNotFound
                 →  fs.ErrPermission       → ErrGraphDirUnreadable
                 →  otherwise (ENOTDIR, …) → ErrGraphDirNotDirectory
```

**Rationale**: `fsys.Local.Mount` performs no existence check by design, so today a
path that does not exist reaches `guardIsGraph` and is reported as *"… is not an
initialized graph"* — technically true, actively unhelpful, and precisely what
Constitution XII forbids. The spec's Edge Cases name all three classes. `Store.Stat`
already answers all three truthfully, and `internal/app/ctrl/service.existingTarget`
establishes the exact `Mount` + `Stat(".")` probe idiom in this codebase.

**Why not `fsys.ResolveLocalRoot`**: it *creates* the directory when absent. Calling
it from a read-only command would turn a typo'd path into a new empty directory —
the opposite of the required behaviour.

**Blast radius**: `EnsureGraph` is also the preflight for `arc stats` and
`arc apply`. Those always pass `filepath.Abs(".")`, which by construction exists and
is a directory, so the new branches are unreachable for them. No observable change
outside `serve`.

---

## D3 — Where the cache exclusion rule lives

**Decision**: `arc init` writes **both**:

| Path | Content | Tracked? | Purpose |
|------|---------|----------|---------|
| `.arc/.gitkeep` | *(empty)* | yes | unchanged graph marker |
| `.arc/.gitignore` | `cache/\n` | **yes** | the rule that survives a clone |
| `.arc/cache/.gitignore` | `*\n` | no (self-ignoring) | the rule the user asked for |

**Rationale**: `.arc/cache/.gitignore` containing `*` ignores *itself*, so git cannot
track it — which means **it does not exist in a clone**. A future cache writer
running in a clone would create `.arc/cache/<file>` with no rule covering it, and it
would show up as untracked content in `git status` and be committable by accident.
That is the exact failure this feature exists to prevent, reintroduced one release
later.

A rule of `cache/` inside the tracked `.arc/.gitignore` is not self-ignoring, so it
is committed normally and is present in every checkout. The self-ignoring `*` file
the user asked for is kept as well: it is the belt-and-braces that still holds if
`.arc/.gitignore` is deleted, and it documents the directory's purpose in situ.

The file `.arc/.gitignore` is *reused, not introduced* — it exists today with content
`*`. Only its content changes, from `*` to `cache/`. That is the smallest possible
edit that both flips `.arc/` to public and keeps the invariant true in clones.

**Alternatives considered**:

- *`.arc/cache/.gitignore` with `*` alone (the literal request).* Rejected for the
  clone gap above. Flagged to the user as the one place this plan adds a file beyond
  the literal wording of the request.
- *`git add -f` the self-ignoring file so it becomes tracked.* Rejected: forcing a
  tracked-but-ignored file is a confusing repository state, and it would require a
  new `force` parameter on the `port.VCS` staging interface — exactly the interface
  change the scope directive rules out.
- *Drop `.arc/.gitkeep`.* It is now redundant (`.arc/.gitignore` already keeps the
  directory alive in git). Rejected anyway: removing it means touching
  `arcStateMarker`, `layoutPaths`, and four assertions for zero user-visible gain.

---

## D4 — What `footprint.Tracked()` must exclude

**Decision**: change the predicate from "anything under `.arc/`" to "anything under
`.arc/cache/`".

**Rationale**: `Tracked()` exists because `git add -- <ignored path>` fails outright
(not silently) when the pathspec *names* an ignored file — the reason is documented
on the method today. Under the new layout `.arc/.gitkeep` and `.arc/.gitignore` are
no longer ignored and must be in the pathspec (FR-011); `.arc/cache/.gitignore` still
is and must stay out. One constant swap in one predicate.

---

## D5 — FR-017: refusing when the host repository would exclude `.arc/`

**Decision**: add `StateIgnored bool` to `kernel.InitOpts`, populate it in
`cmd/arc/ctrl.resolveRepoContext` with a second `probe.IsIgnored` call for
`<rel>/.arc/.gitkeep`, and add one case to `guardRepositoryContext`
returning a new `ErrStateIgnored` sentinel.

**Correction (found implementing quickstart V6).** This decision originally
specified probing `<rel>/.arc`, the directory. That is wrong, and it made
the guard miss the single most likely case — a host `.gitignore` carrying
`.arc/`. A *directory-only* pattern (one with a trailing slash) matches a
path only when git knows that path is a directory; at probe time nothing
exists yet, by FR-005's requirement that a refusal precede every write, so
`git check-ignore notes/.arc` answers **not ignored** while
`git check-ignore notes/.arc/.gitkeep` answers **ignored**. Probing the
marker file init must actually stage is correct for directory-only patterns
and strictly more sensitive besides: any rule excluding the directory also
excludes what is under it. See the amended note under *Interface cost*.

**Rationale**: this is an exact structural clone of the existing R3 rule
(`opts.SkipGitInit && opts.TargetIgnored → ErrTargetIgnored`), which exists for the
same reason: *"without this the failure still happens, but late and
incomprehensibly."* Under the old layout `.arc/` was meant to be ignored, so nobody
asked the question. Now that init must commit `.arc/.gitkeep`, a host `.gitignore`
carrying `.arc/` makes `StagePaths` fail after the layout is on disk, triggering a
rollback with a raw git error. The guard runs before any write (FR-005's
never-touch-the-filesystem-on-refusal property is preserved).

**Reachability**: only in `--skip-git-init` mode. In standalone mode guard R1 already
refuses any target inside an existing repository, so there are no foreign ignore
rules to collide with. The new case is `opts.SkipGitInit && opts.StateIgnored`,
placed after R3 — a target that is wholly ignored is the more fundamental complaint
and should still win.

**Interface cost**: one bool field on an internal value struct, one extra `IsIgnored`
call on the existing `repoProbe` interface (no new method). `git check-ignore`
answers for paths that do not yet exist — but only for patterns whose match does not
itself depend on the path being a directory, which is exactly why the probe names
`<rel>/.arc/.gitkeep` rather than `<rel>/.arc` (see Correction above). Ordering
relative to directory creation remains a non-concern once the path is spelled that
way.

---

## D6 — Collateral: the case-folding probe writes into `.arc/`

**Decision**: change `probeFoldsCase` in `internal/adapter/fsys/local.go` to prefer
`.arc/cache`, falling back to `.arc`, then to the graph root. Three lines plus the
now-stale comment.

**Rationale**: this is the one place outside `serve` and `init` that the layout
change actually reaches, and leaving it alone is a real defect rather than a
cosmetic one. The function writes a throwaway temp file into `.arc/`, and its own
comment justifies that choice with *"already gitignored, so a probe file that
somehow outlives a crash is never mistaken for graph content."* That premise is
exactly what this feature retires. A leaked probe file in a now-tracked `.arc/`
shows up in `git status` and is committable.

`.arc/cache` is the correct home for it — machine-local, reproducible, disposable —
and is precisely what the user described the cache as being for.

**Why the fallback chain matters**: a clone has no `.arc/cache` (D3: the directory
carries only a self-ignoring file, so git cannot materialise it). Without a fallback,
`os.CreateTemp` fails and the function returns its `safeDefault` of *case-insensitive*
— wrong on Linux, and it would silently change node-identity merge behaviour in every
clone. The fallback to `.arc` keeps today's behaviour exactly.

**Scope note**: this touches `internal/adapter/fsys`, outside the literal
"`serve` and `init`" directive. It is included because it is a direct consequence of
making `.arc/` tracked, not an independent improvement, and it is called out
explicitly for the user rather than folded in quietly.

---

## D7 — Migration for existing graphs

**Decision**: documentation only, per the spec's resolved clarification (FR-020).
`README.md` gains a short migration block; no detection, no warning, no conversion
code, no lint rule.

**Rationale**: the tool has no code path that inspects `.arc/.gitignore`'s *content*
— `guardIsGraph` stats the `.arc/` directory and nothing else. A legacy graph
therefore keeps working with zero compatibility shims, which is what FR-018 requires.
FR-020 is satisfied by *not writing code*, which is the cheapest possible
implementation of a requirement.

**The documented migration** (three commands, reversible):

```bash
printf 'cache/\n' > .arc/.gitignore
mkdir -p .arc/cache && printf '*\n' > .arc/cache/.gitignore
git add .arc/.gitkeep .arc/.gitignore && git commit -m "graph: publish .arc state"
```

---

## D8 — Test strategy

**Decision**: no new test packages. Scenarios land in the four existing files that
already cover these commands.

| File | Tier | Covers |
|------|------|--------|
| `cmd/arc/graph/serve_test.go` | E2E (`sut`/`connectServeSession`) | US1 scenarios 1-5, edge cases |
| `cmd/arc/ctrl/init_test.go` | E2E (`sut`) | US2 scenarios 1-4, US3 scenarios 1-4, FR-017 |
| `internal/app/ctrl/service/init_test.go` | unit (fake store) | layout paths, `Tracked()`, guard R4 |
| `internal/app/graph/service/*_test.go` | unit (`fstest.MapFS`) | D2's three error classes |

**Rationale**: Constitution VIII requires E2E tests colocated with the Cobra command
and exercised through `RunE` via the shared `sut` helper — both files already do
exactly that. `serve_test.go` additionally has `connectServeSession`, which builds the
*real* registered `mcp.Server` over in-memory transports, so "tools answer from the
named graph" is assertable without a subprocess or a socket.

**Existing tests that must change** (all in
`internal/app/ctrl/service/init_test.go`): the assertions at lines ~223-228 pinning
`.arc/.gitignore` to `"*\n"`, and the two written-path lists at ~248 and ~473. These
are the pin points of the old layout and their change is the point of the feature,
not churn.

---

## D9 — Documentation surface

**Decision**: `ARCHITECTURE.md` glossary (**Arc State Directory** rewritten, **Arc
Cache Directory** added), `README.md` (three sites: the bootstrap prose, the folder
tree, and the "current working directory" sentence under Usage), and Cobra
`Long`/`Example` on both commands.

**Rationale**: Constitution I makes the glossary update mandatory when a domain
concept changes, and Constitution XII forbids help text drifting from behaviour. The
README's *"`arc` operates on the graph in the current working directory"* sentence
becomes false for `serve` specifically and must say so.

**No ADR required**: no architectural pattern changes. A command gains an optional
positional argument (ADR 002's established UX grammar), and a static layout constant
gains an entry. ADR 003 (MCP server adapter) is unaffected — the server's
construction, transports, and tool registration are untouched.

---

## Unresolved

None. No `NEEDS CLARIFICATION` markers remain in the spec or in this document.
