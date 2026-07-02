# Phase 0 Research: `arc init`

## Naming assumption (flag for user confirmation)

**Decision**: The user input named the domain package `internal/app/ctrl` but the command package `cmd/arc/crtl`. Since the instruction explicitly says "maintain same hierarchy," this plan treats `crtl` as a typo for `ctrl` and uses `ctrl` consistently in both `internal/app/ctrl/` and `cmd/arc/ctrl/`.

**Rationale**: A deliberately different spelling between the mirrored command and domain package would defeat the stated goal of a matching hierarchy, and would be a permanent, awkward misspelling baked into the public command tree (`cmd/arc/ctrl` never becomes a user-facing subcommand name since `arc init` is a bare verb — see below — but the package name is still load-bearing for every future `ctrl`-domain command).

**Alternatives considered**: Using `crtl` literally as given. Rejected as very likely an unintentional transposition (`ctrl` → `crtl`), and propagating it would need to be corrected in a later PR anyway once noticed.

## D1: Git invocation strategy

**Decision**: Invoke the system `git` binary via `os/exec`, never a Go git library (e.g. `go-git`).

**Rationale**: CORE §11 mandates "Git MUST be used as the version-control system. No other system is used" — the intent is the real, canonical `git` tool, not a reimplementation. Later phases (`arc retract` via `git revert`, `arc history` via `git log --follow`, `arc locate` via `git log --grep`) depend on exact real-git semantics and porcelain output; committing to the real binary now avoids a divergent Go-native git implementation that has to track upstream git behavior forever. The constitution names no Go git library as mandatory, so this is a free choice, and the simplest one — shelling out — is also the one most compatible with every future phase in VISION.md.

**Alternatives considered**: `go-git` (pure-Go git implementation) — rejected: extra dependency, subtly different behavior from real git in edge cases, and every later VISION.md phase already assumes real `git` CLI semantics (`git log --grep`, `git revert`, `git ls-files`).

## D2: Reporting git progress to the user

**Decision**: Wrap each git invocation (`git init`, `git add -A`, `git commit -F <msgfile>`) with the ADR 002 DS-06 `Reporter` port: `Start(label)` before the subprocess runs, `Done(label, elapsed)` on success, `Error(label, err)` on failure. The real adapter writes to `stderr`; a Null Object (`silentReporter`) is used for `--quiet`/`--silent` and in unit tests.

**Rationale**: This is exactly what DS-06 exists for, and it directly satisfies the user's instruction to inform the user "about the git tool progress" without inventing a new, one-off progress mechanism. Because `git init`/`add`/`commit` complete in well under a second on a fresh empty directory, per-line streaming of git's own stdout is unnecessary (and git only prints anything interesting on `commit`, which we already summarize ourselves) — Start/Done framing per operation is sufficient and matches DS-06's flat (non-task-tree) mode for simple single-phase-per-call commands.

**Alternatives considered**: Streaming raw git stdout/stderr live to the terminal — rejected as noisy and inconsistent with `--quiet`/`--json`/`--plain` modes, which must suppress it cleanly; the Reporter port already handles that suppression by construction.

**Bugfix**: 2026-07-02 — BUG-001 revises this decision. Manual QA showed default (non-`--quiet`, non-`--verbose`) invocations printed Start/Done lines for all four git steps, ~~which is sufficient~~ which over-reports: this decision as originally written gated progress only behind `--quiet`/`--silent`, so it showed unconditionally otherwise. Revised: progress output is now gated behind `--verbose`/`-v` (silent by default; `--quiet` still forces silence and takes precedence over `--verbose` if both are set), styled faint/gray (`SCHEMA.Hint`-equivalent, not `SCHEMA.StatusOK` green), and consolidated to three steps instead of four — "Checking git availability", "Preparing git repository" (covers both `git init` and `git add -A`), and "Committing empty graph" — per the user's explicit verbose-mode list. Separately, BUG-001 also surfaced a genuine implementation bug independent of this gating decision: `stderrReporter.Done`/`.Error` passed a string with an embedded/trailing `\n` into `lipgloss.Style.Render(...)`. `lipgloss` treats multi-line input as a block and pads every line to the block's width, so a trailing `\n` is replaced with padding **spaces**, not a line break — the next Start/Done call's text then lands on the same terminal line, indented by the previous line's length. Fix: style the text alone, then write the newline outside the styled span (e.g. `w.Write(append(SCHEMA.Hint.Render(text), '\n'))`, never `SCHEMA.Hint.Render(text + "\n")`). This same rule applies to `humanInitPrinter.Show` in `cmd/arc/ctrl/init.go`.

BUG-001 also revises two adjacent, previously under-specified points: (1) the commit hash shown to the user — both in the default human-mode confirmation line and in `--json` — MUST always be the **short** hash (`git rev-parse --short HEAD`), not the full 40-character SHA `contracts/vcs-port-contract.md`'s `Commit` description ambiguously allowed via "equivalent to `git rev-parse HEAD`"; and (2) the `PostRunE` next-step hint text changes from `(use "arc list" to see what's in your new graph)` to `(use "arc apply <patch.md>" to load content into your new graph)`, since a freshly initialized graph is empty and `arc list` has nothing useful to show — `arc apply` is the natural next step. Neither `arc list` nor `arc apply` exist as implemented commands yet (both are future VISION.md commands); the hint text is a forward-looking suggestion consistent with DS-12's existing pattern in this codebase, not a claim that the subcommand is implemented today.

## D3: Filesystem layout creation goes through stdlib `io/fs`/`io.Writer`, via `internal/adapter/fsys` (revised three times)

> **Correction 1**: This decision originally read "filesystem layout creation is not behind a port... uses `os.MkdirAll`/`os.WriteFile` directly." That was a mistake: `.arc/` and the canonical folders are exactly "the filesystem used as a state store" Principle VII already named.
>
> **Correction 2**: The first fix then adopted `github.com/fogfish/stream` and mixed plain-`os` root resolution (does the target exist? is it a directory? should it be created?) *inside* a `stream`-backed `Mount` call. That split resolution from mounting, which was right, but kept the third-party dependency.
>
> **Correction 3 (current)**: `github.com/fogfish/stream` itself is now dropped project-wide (`.specify/memory/constitution.md` v1.5.0 reverts v1.4.0). Two problems surfaced: (a) `stream`'s local and S3 backends share one Go package, so even purely local use unavoidably pulled in the full AWS SDK v2 tree (8 direct + 15 indirect modules — S3 client, STS, SSO, credential-chain resolution) for a command that touches no cloud API; (b) the S3 backend would not have delivered a real capability anyway — `arc init`'s contract is a git commit (CORE §11), and git needs a local working tree, which an S3 object store accessed via API calls is not. `internal/adapter/fsys` is rebuilt on stdlib `io/fs`/`io.Reader`/`io.Writer` only, keeping the resolve/mount split and the `.gitkeep`-creates-directory convention from Correction 2 (both were sound, backend-independent ideas), dropping only the third-party dependency and the S3 backend. This section replaces the previous text rather than layering a fourth decision on top.

**Decision**: Two clearly separated steps, both in the shared `internal/adapter/fsys` package, and `os`'s file/directory functions confined to this package alone (constitution Principle VII, Mandatory Libraries & Tooling: "Filesystem Abstraction").

**Step 1 — resolve the local root (`os`/`path/filepath`)**:

```go
// internal/adapter/fsys/resolve.go
package fsys

// ResolveLocalRoot ensures root exists as a local directory. Creates root
// if missing and reports whether it did so, so a caller can undo exactly
// that via RemoveLocalRoot on a later failure (FR-013). Also owns the
// FR-010 "target is a file, not a directory" check, since this is the one
// step that inspects the raw path before any Store exists.
func ResolveLocalRoot(root string) (created bool, err error) { /* os.Stat, os.MkdirAll */ }

// RemoveLocalRoot undoes a root ResolveLocalRoot created.
func RemoveLocalRoot(root string) error { /* os.RemoveAll(root) */ }
```

**Step 2 — mount an already-resolved root and perform all subsequent I/O through `io/fs`/`io.Writer` only**:

```go
// internal/adapter/fsys/types.go
package fsys

import (
    "io"
    "io/fs"
)

// File is the writable counterpart to fs.File. Stdlib io/fs is read-only by
// design (fs.FS only defines Open); this is the minimal write-side
// extension a graph-writing use-case needs, expressed in stdlib terms only.
// Discard is the one addition borrowed directly from stream.Canceler ("cancel
// effect of file system i/o, before file is closed"): a caller that hits a
// write error partway through MUST call Discard instead of Close, so a
// half-written file is not left behind looking like a completed one.
type File interface {
    io.Writer
    io.Closer
    Stat() (fs.FileInfo, error)
    Discard() error
}

// Store is the capability a use-case needs to read and write a mounted
// graph root: stdlib fs.FS (+ fs.StatFS, fs.ReadDirFS) for reads, plus
// Create/Remove for writes, which io/fs itself does not define.
type Store interface {
    fs.FS
    fs.StatFS
    fs.ReadDirFS
    Create(name string) (File, error)
    Remove(name string) error
}

// Mounter mounts an already-resolved root as a Store. It performs no
// existence checks and no root creation — that is ResolveLocalRoot's job.
type Mounter interface {
    Mount(root string) (Store, error)
}
```

`fsys.Local`, the sole `Mounter` implementation, wraps `os.DirFS(root)` (which already satisfies `fs.FS`/`fs.StatFS`/`fs.ReadDirFS`/`fs.GlobFS` as of the stdlib version this project pins) for the read side, and adds `Create`/`Remove` — `io/fs`'s one genuine gap — as thin methods backed directly by `os.Create`/`os.MkdirAll`/`os.Remove`, joined against the stored root path. `Create` wraps the resulting `*os.File` in a small unexported type that adds `Discard()` (`Close()` then `os.Remove()` the just-created file) — `*os.File` alone satisfies `Write`/`Close`/`Stat` natively, so this wrapper's only job is the one method stdlib doesn't provide. This is "inject learnings from `stream` into the adapter" per the user's instruction: the same `File`-as-`io.Writer`+`io.Closer`+`Stat`+cancel shape `stream.File`/`stream.Canceler` used, and the same resolve-then-mount separation, reimplemented on pure stdlib with no external dependency and no S3 backend.

`internal/app/ctrl/service.Init` depends on `fsys.Mounter`/`fsys.Store` directly (constructor-injected from `cmd/arc/ctrl/init.go`, per Dependency Inversion) — intentionally **not** re-declared as a separate ctrl-private port type, unlike `port.VCS`. Filesystem access is the cross-use-case concern the user explicitly asked to centralize in `internal/adapter/fsys`; every future use-case (`apply`, `lint`, `index`, ...) will mount the same way, so the interface belongs at the shared adapter tier, not duplicated per use-case (a deliberate, documented exception to ADR 001 port-isolation rule 1 — here the *adapter* is shared by design, matching ADR 001's own "phase 2: `/internal/adapter`... grouped by technology dependency" evolution stage).

**No S3 backend exists anymore.** Real S3-backed graph storage, if it becomes a genuine future requirement, needs its own design effort when a concrete, non-git-coupled use-case actually needs it (see the git-working-tree argument in Correction 3) — not a library adopted speculatively "for later" that turned out not to solve the actual problem.

Path/directory conventions follow stdlib `io/fs`'s own contract (`fs.ValidPath`), not any third-party library's convention: relative, slash-separated, no leading or trailing slash, `"."` for the root. `ArcNetCoreLayout.Folders` entries are therefore `"sources"`, `"entities"`, `"resources"`, `"timeline/yearly"`, `"timeline/monthly"`, `"_meta"` (no trailing slash) and file paths are e.g. `"_meta/predicates.md"`, `"sources/.gitkeep"` (no leading slash).

Directories have no separate existence from files: a canonical folder like `sources` comes into existence purely because `sources/.gitkeep` was written under it via `Store.Create` — this is required regardless of storage backend anyway, since **git itself does not track empty directories**; `.gitkeep` (or any placeholder file) is the standard workaround for that git limitation, independent of whatever is mounting the filesystem underneath. `io/fs` has no `Mkdir`/`MkdirAll` operation on `Store` for the same reason resolution stayed a separate `os`-based step: creating a *canonical subfolder* is never actually needed as its own operation, only as a side effect of `Create`.

**Forward note (not built now, YAGNI)**: VISION.md's "Graph Root" discovery — walking up from the current directory to find an existing `.arc/`, the way git finds `.git/` — is a *different* `os`-based operation than `ResolveLocalRoot` (which creates; discovery only searches and fails if nothing is found) needed by future non-`init` commands, not this one. It would live alongside `ResolveLocalRoot` in `internal/adapter/fsys` when a command that needs it is planned.

**Rationale**: Confining `os`'s file/directory functions to one package, and expressing everything above that boundary in terms of `io/fs`/`io.Reader`/`io.Writer`, is the actually load-bearing property here (constitution Principle VII) — it is what keeps `service.Init` unit-testable with a fake `Store` and keeps the codebase portable to a different concrete backend later without touching domain code. A third-party library was never required to get that property; stdlib's own interfaces (plus one small, project-owned `File` extension for the write side `io/fs` omits) deliver it directly, with no dependency weight and no unrealized promises.

**Alternatives considered**: `github.com/fogfish/stream` — superseded, see Correction 3. The original "no port, direct `os.*` calls everywhere" decision, and the "`Mount` does both resolution and mounting" decision — both superseded, see Corrections 1 and 2. A ctrl-private `port.Store`/`port.Mounter` duplicate of the same interfaces (matching how `port.VCS` is handled) — rejected because filesystem mounting is explicitly cross-use-case, so the interface declaration is centralized in `internal/adapter/fsys` rather than copy-declared per use-case.

## D4: Safety guards and failure cleanup (FR-013, FR-014, FR-015) (revised for the stdlib-only resolve/mount split)

**Decision**: `service.Init` calls `fsys.ResolveLocalRoot(dir)` first (FR-010 "not a directory" surfaces here, before any `Store` exists). Then `local.Mount(dir)` (`local` is the injected `fsys.Mounter`, always `fsys.Local{}`). With a `Store` in hand, `Init` runs the FR-014 (`Store.Stat(".arc")` succeeds → already a graph → fail) and FR-015 (`Store.ReadDir(".")` non-empty and not a graph → fail) guards before writing anything. All three guard failures produce zero writes by construction — no cleanup needed.

For a failure *after* guards pass (a mid-run I/O error while writing a layout file or during a git step, FR-013): if `ResolveLocalRoot` reported `created=true`, `Init` calls `fsys.RemoveLocalRoot(dir)` — one call undoes everything, since the whole directory was this run's own creation. If `created=false` (the target pre-existed and the FR-015 guard already proved it was empty), `Init` instead removes only the fixed, statically-known set of paths `ArcNetCoreLayout` describes (the same list it just attempted to write), calling `Store.Remove(path)` for each and tolerating not-found errors for any that were never reached — no generic recursive-delete extension is needed since the exact write set is known in advance from `ArcNetCoreLayout`, not discovered by walking the filesystem. A single in-flight `Store.Create` write that fails partway is unwound via `File`'s own `Close`/discard contract, not `Store.Remove` (see `contracts/fsys-port-contract.md`).

**Rationale**: This keeps the "minimal extension" surface (constitution: "additive and minimal... never a parallel reimplementation") to exactly the two `os`-based functions in `resolve.go` plus the small `Create`/`Remove` pair `io/fs` omits, and avoids inventing a generic recursive `RemoveAll` walker that the fixed, small, already-known `ArcNetCoreLayout` file list makes unnecessary. It also cleanly separates "we own this whole directory, undo it in one call" from "we only own the specific files we wrote inside a pre-existing empty directory" — the two rollback shapes FR-013/FR-014/FR-015 actually require, no more.

**Alternatives considered**: A generic `fsys.RemoveAll(store, root)` that walks `ReadDir` recursively and removes everything found — rejected as unneeded complexity: the service always knows exactly what it tried to write (`ArcNetCoreLayout` is a static value), so a walk-and-discover approach would be solving a problem the design doesn't have. Writing to a temp staging location and renaming into place — rejected as over-engineered for this failure mode.

## D5: Shared output/UX kernel introduced now

**Decision**: Introduce `internal/bios` as the shared kernel package housing DS-04 (`Mode`, `ResolveMode()`, `Registry[T]`, `Printer[T]`), DS-05 (`Schema`, `SCHEMA_PLAIN`/`SCHEMA_COLOR`), and DS-06 (`Reporter` port + `stderrReporter` + `silentReporter`). `github.com/charmbracelet/lipgloss` becomes an active dependency starting with this feature.

**Rationale**: `specs/001-cli-infrastructure/plan.md` explicitly deferred lipgloss ("no styled output exists yet"). `arc init` is the first command that produces styled/state-changing output (git progress, success confirmation, errors), so this is the natural point to stand up the shared kernel exactly once, per ADR 002's implementation note that these types "SHOULD live in one small shared package (e.g. `internal/bios`... )." Every subsequent command reuses this package rather than re-deriving output-mode handling.

**Alternatives considered**: Inlining `fmt.Println`/ad hoc styling directly in `cmd/arc/ctrl/init.go` for this one command, deferring the shared kernel to a later feature — rejected: ADR 002 DS-04/DS-05/DS-06 are binding (constitution Principle I: accepted ADRs are BINDING), and root.go already needs the DS-03 persistent flags (`--quiet`, `--verbose`, `--json`, `--color`) wired regardless, since `init` is the first command to need them.

## D6: Command grammar — bare verb, no noun prefix

**Decision**: `arc init [<dir>]` stays a bare top-level verb, not `arc graph init`.

**Rationale**: DS-01 permits a bare top-level verb "only when the tool has a single, obvious subject" — true here: the entire `arc` tool operates on exactly one kind of subject, a knowledge graph. VISION.md's full command list (`arc init`, `arc apply`, `arc list`, `arc lint`, ...) is uniformly bare-verb, so this is not a new decision so much as continuity with the already-established (if not yet ADR-recorded) pattern every other planned command will need to follow.

**Alternatives considered**: None seriously — nesting under a `graph` noun would break every other command name already fixed in VISION.md's command list, which this feature does not have license to change.

## D7: Error annotation via `github.com/fogfish/faults`

**Decision**: Every expected failure this feature can produce (`git` missing, target is a file, target already a graph, target non-empty, a mid-run I/O error during layout creation or git invocation) is declared once as a package-level `faults.Type` or `faults.SafeN` constant — `faults.SafeN[string]` where the message needs to embed the resolved target path — and wrapped at the point of failure with `.With(err[, args...])`. Callers (primarily tests) that need to distinguish failure classes use `errors.Is(err, <constant>)`, never string matching. This applies in `internal/app/ctrl/service` (guard failures, layout-write failures) and `internal/app/ctrl/adapter/git` (subprocess failures for `IsAvailable`/`Init`/`StageAll`/`Commit`).

**Rationale**: This is a project-wide mandate added to `.specify/memory/constitution.md` (Principle XII, Mandatory Libraries & Tooling) alongside this feature, and `arc init` is the first command with real expected-error paths to apply it to. `faults.Type`'s declared message *is* the human-readable guidance constitution Principle XII already required — the same construct satisfies both rules rather than requiring a separate `fmt.Errorf` context string and a separate design for "is this error type X" checks. Table-driven unit tests for the guard logic (D4) get a precise, type-safe way to assert which specific guard fired (`errors.Is(err, service.ErrAlreadyInitialized)`) instead of matching on rendered message substrings, which is exactly the failure mode `faults` was built to eliminate.

**Alternatives considered**: `fmt.Errorf("<context>: %w", err)` at each layer (the pattern shown in ADR 002 DS-07's own example, predating this mandate) — rejected per the new constitution rule: it is untyped, cannot be matched with `errors.Is` against a specific *semantic* failure (only against the wrapped underlying error, if any), and drifts as message text is copy-pasted across call sites. This plan notes, but does not itself resolve, that ADR 002 DS-07's example code should eventually be updated to match (see constitution Sync Impact Report's "Known tension" note) — out of scope for this feature.
