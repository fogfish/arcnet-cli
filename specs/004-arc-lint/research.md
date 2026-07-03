# Phase 0 Research: `arc lint`

## D1: Package structure — `internal/app/lint`, `cmd/arc/lint`

**Decision**: One new use-case package, `internal/app/lint`, structured exactly like `internal/app/ctrl`/`internal/app/graph`/`internal/app/config` (`kernel/port/adapter/service/component.go`), hosted by a matching `cmd/arc/lint` Cobra wiring package. `internal/app/lint` gains one method on the already-shared `internal/adapter/git.Git` adapter (`CommitsMatching`) behind a new, lint-private `port.VCS` — no new adapter package.

**Rationale**: Direct implementation of the user's explicit instruction ("linter is own domain `internal/app/lint`. Also maintain same hierarchy in `cmd/arc/lint`"), and consistent with ADR 001's `componentX` layout every other use-case in this codebase already follows.

**Alternatives considered**: Folding lint into `internal/app/ctrl` (graph management) since both are "read the graph's local state" concerns — rejected: `internal/app/ctrl`'s own package doc already scopes it to "creates, and will later inspect and validate" the graph's *bootstrap* state (folder layout, git availability), not per-node content conformance; the user's own instruction is explicit that lint is its own domain, and ADR 001's Screaming Architecture guidance favors one cohesive use-case per package over a growing catch-all.

## D2: Node enumeration — what counts as a "node" to walk

**Decision**: `internal/app/lint/service.Lint` walks the entire graph root via `fsys.Store.ReadDir`, recursively, collecting every `*.md` file **except**: anything under `.arc/` (arc's own non-graph state directory, per `specs/002-arc-init`), and the two fixed registry stubs `_meta/predicates.md` and `_meta/aliases.md` (support files, not nodes — they carry no `kind` front-matter and are never intended to parse as one). Every other `*.md` file found is treated as a node and MUST satisfy FR-001 (valid front-matter + mandatory `kind`); a file that fails to parse for *any* reason (missing front matter, invalid YAML, missing `kind`) is reported as a FR-001 violation and excluded from every other rule's input set for that file (a node that isn't parseable has no `Links`/`Attrs` to check further) — but scanning of the rest of the graph continues (spec FR-013).

**Rationale**: CORE §3.2's "a node MAY be moved between folders without affecting any edge" means node identity and node validity cannot be inferred from directory position — the only reliable universal signal is "is this a Markdown file with a `kind`," so lint must not hard-code an allowlist of folder names (`sources/`, `entities/`, `resources/`, `timeline/*`) the way `arc init`'s static `DefaultLayout` does, since a real graph may add domain-specific folders for extension kinds the layout constant knows nothing about.

**Alternatives considered**: Restricting the walk to `arc init`'s fixed `kernel.DefaultLayout.Folders` list — rejected: would silently skip nodes of a domain/extension kind stored in a profile-specific folder, exactly the case spec User Story 3/FR-011 says lint must still validate.

## D3: Line-number attribution — a lint-private raw-text locator, not a change to `internal/core`

**Decision**: `internal/core.ParseNode` is reused unchanged for every *structural* field a rule needs (`Kind`, `Attrs`, `Text`, `Notes`, `HRefs`, `Edges`, `Links`) — it already discards Markdown source positions once it produces a `core.Node` (`specs/003-apply-patch`'s AST never needed them, and nothing about that contract changes here). A new, lint-private helper (`internal/app/lint/service/locate.go`) operates directly on the same file's raw `[]byte` source, independent of goldmark, to attribute a violation to a 1-based line number:

- **Front-matter/`kind` violations**: the line of the first `---` delimiter, or (for a YAML parse error) the line the underlying `yaml.v3` error itself reports, when available; otherwise line `1`.
- **Link violations**: a `regexp` search for the literal `[[<Target>` (and `[<Predicate>:: [[<Target>` for a predicate-qualified inline form) bounded to each line, returning the first line containing that exact bracketed occurrence — the same substring the violating `core.Link.Target` names, so no separate position-tracking parser is needed.
- **Predicate violations**: a `regexp` search for `<predicate>::` (structural edges) or the block's bold-label/heading line (`**Title**` / `## Title`) for a `Links` block key.
- **Conflict markers**: a direct line-by-line scan for lines beginning with `<<<<<<<`, `=======`, or `>>>>>>>` — this scan runs *before* `ParseNode` is attempted (see D13), since a conflicted file is not valid YAML/Markdown to begin with.

Where more than one line in a file matches a search (e.g. the same `[[Target]]` mentioned twice), the locator reports the first occurrence's line — sufficient for a human to find the offending reference; CORE §14 does not require enumerating every physical occurrence of the same violation.

**Rationale**: Extending `internal/core.Node`/`ParseNode` to carry positions for every field would ripple a new, apply/render-irrelevant concern into a package `specs/003-apply-patch` already shipped and stabilized, violating YAGNI (Principle V) for the sake of a feature-specific need. A small, deterministic regexp scan over already-known substrings (the exact `Target`/predicate/label strings `core.Node` already extracted) is cheap, testable in isolation, and does not require re-parsing Markdown a second time with a position-preserving tree.

**Alternatives considered**: Modifying `internal/core`'s goldmark-backed parser to retain `ast.Node.Lines()` byte offsets and translating them to line numbers on every field — rejected: `goldmark`'s AST types are explicitly confined to `internal/core/markdown.go` and never meant to leak into a second consuming package's design; threading position data through `core.Node`'s existing, already-tested shape for every one of `Text`/`Notes`/`HRefs`/`Edges`/`Links` is a much larger, riskier change than this feature needs, when a substring re-scan achieves the same user-facing result (a helpful line number) far more cheaply. Reporting only file paths, no line numbers, for link/predicate violations — rejected outright: the user's instruction and spec FR-015 both explicitly require a line number.

## D4: Basename uniqueness (§3.2)

**Decision**: A single pass builds `map[basename][]path` across every enumerated node file (D2); any basename with more than one path is a violation, reported once, naming every colliding path (spec User Story 2 Acceptance Scenario 3) — no single line number applies (spec FR-015's "clear indication that a line number is not applicable").

**Rationale**: Direct implementation of CORE §3.2 ("unique across the whole graph") using the same enumeration D2 already produces — no second filesystem walk needed.

## D5: Link resolution (§3.2's "every `[[link]]` resolves")

**Decision**: After D4's basename index is built, a second pass walks every node's `HRefs`, `Edges`, and every `Links[*].Seq` entry and checks each `Link.Target` against the basename index; an unresolved target is a violation located via D3's locator.

**Rationale**: Requires the complete basename set to exist first (a link may legitimately point forward to a node enumerated later in the walk order), so this is necessarily a second pass over already-parsed nodes, not a single combined walk.

## D6: `source` citekey identity (§6.2)

**Decision**: For every node whose `Kind == "source"`, compare `core.Node.ID` (which `ParseNode`'s `deriveNodeID` already resolves from the front-matter `id` field, preferentially, per `internal/core/markdown.go`) against the node file's actual basename (the filename without `.md`, obtained from the D2 walk, not from front matter) — a mismatch is FR-004's violation, located at the `id:` front-matter line via D3.

**Rationale**: Direct implementation of CORE §6.2 ("a `source` node's identity is a citekey... `id` equal to its basename"). `core.Node.ID` cannot be trusted alone as "the basename" for this check — `ParseNode` derives it from front matter precisely because it has no filename parameter (`specs/003-apply-patch/research.md` D2) — so the actual on-disk basename must come from the caller (lint's own walk), and the check is exactly this equality, not an assumption that they already match.

## D7: `entity` four-word Sowa category (§9.2.1)

**Decision**: `internal/app/lint/kernel` declares the fixed CORE §9.2.1 vocabulary as three positional word-sets plus one leaf set:

```go
var sowaPosition1 = map[string]bool{"independent": true, "relative": true, "mediating": true}
var sowaPosition2 = map[string]bool{"physical": true, "abstract": true}
var sowaPosition3 = map[string]bool{"continuant": true, "occurrent": true}
var sowaLeaf = map[string]bool{
    "object": true, "process": true, "schema": true, "script": true,
    "juncture": true, "participation": true, "description": true, "history": true,
    "structure": true, "situation": true, "reason": true, "purpose": true,
}
```

For every node whose `Kind == "entity"`, lint reads its `category` attribute — stored, per this codebase's own already-shipped convention (`internal/core/markdown_test.go`, `specs/003-apply-patch/quickstart.md`: `category: [independent, abstract, occurrent, script]`), as a literal four-element YAML sequence of the already-decoded words, not a compact `xyz:leaf` code — and checks: the field is present, decodes to a `[]any` of exactly four strings, and each position's word is a member of that position's fixed set above (position 4 against `sowaLeaf`). Any deviation (missing field, wrong length, a word not in its position's set) is FR-005's violation, located at the `category:` front-matter line via D3.

**Rationale**: CORE §9.2.1's prose example (`ipc:object` → `[independent, physical, continuant, object]`) explains the *code's meaning* for documentation purposes, but this codebase's own already-implemented parser, test fixtures, and worked quickstart examples (predating this feature, from `specs/003-apply-patch`) consistently store and consume `category` as the four-word array directly — confirmed by `internal/core/markdown_test.go`'s `entity.Attrs["category"].([]any)` assertion. Validating against the *actual, already-established on-disk representation* is correct; inventing a second, compact-code representation this codebase has never actually used would validate a shape no existing or future node produced by `arc apply` ever has. The four positional sets live in `internal/app/lint/kernel` (lint-private), not `internal/core`, since no other current use-case needs to validate a category — promoting it to the shared core package before a second consumer exists would be speculative (Principle V, mirroring how `specs/003-apply-patch/research.md` D5 kept `config`'s HTTP fetcher use-case-private until a real second need arises).

**Alternatives considered**: Requiring the compact `xyz:leaf` code form instead of the four-word array — rejected once checked against this codebase's own existing, shipped fixtures and parser test assertions, which already fix the on-disk shape as the word array; validating a shape nothing in the codebase actually produces would make every existing example fail lint incorrectly.

## D8: Derived-node provenance (§3.4)

**Decision**: For every node whose `Kind != "source"` (a "derived" node, spec Key Entities), lint checks whether at least one of its `HRefs`, `Edges`, or `Links[*].Seq` entries has a `Target` that resolves (via D5's basename index) to a node whose `Kind == "source"`. Zero such links is FR-006's violation — no single line number applies (the absence of a link is not localizable to one line; spec FR-015's "not applicable" case), so the violation is reported at the file level.

**Rationale**: Direct implementation of CORE §3.4 ("a node distilled from a document MUST link to the document node(s) it was derived from"). Requires D5's resolved-target index to already exist so "linking to a `source`" can be checked structurally rather than by name-guessing.

## D9: Predicate naming & registration (§7.3)

**Decision**: `_meta/predicates.md` (created as an empty stub `# Predicates\n` by `arc init`, `specs/002-arc-init`) is parsed as a Markdown bullet list, one predicate per list item, where the predicate name is the item's first inline-code span (`` `predicateName` ``) — e.g. `` - `mentions` — a document mentions an entity or resource ``. `internal/app/lint/service` parses this file with a small, dedicated regexp scan (consistent with D3's "no second Markdown parser" approach — list items and inline code spans are trivially line-oriented), building a `map[string]bool` of registered predicate names. Every predicate name lint encounters — every `Link.Predicate` across `HRefs`/`Edges`, and every `Links` map key (already camelCase-derived by `ParseNode`'s `camelizeTitle`/explicit `predicate::` form) — is checked against two independent rules: (1) does it match `^[a-z][a-zA-Z0-9]*$` (camelCase), and (2) is it present in the registered-predicates map. A predicate can fail either, both, or neither; FR-007 and FR-008 are reported as distinct violations so a user can tell "not camelCase" from "not registered" apart, per spec Edge Cases. An absent `_meta/predicates.md` file is treated as "every predicate is unregistered" (spec Edge Cases) rather than a hard lint failure of its own.

**Rationale**: No predicate-registry parsing convention exists anywhere in this codebase yet (the stub `arc init` writes has zero entries) — this decision fixes one, consistent with how a human would naturally write the registry (inline-code-spans in a bullet list is the same convention this project's own Markdown documentation already uses throughout, e.g. this very research document's code spans).

**Alternatives considered**: A structured YAML/front-matter list of predicates instead of a Markdown bullet list — rejected: CORE §7.3 names `_meta/predicates.md` as a `.md` file, and a bullet list of inline-code spans is both human-editable and trivially machine-parseable without inventing a second file format for one small registry.

## D10: Citation predicates (§8)

**Decision**: A citation is identified structurally, not by predicate name alone: CORE §8 states "citations are recorded inline, at the point of the statement they support," which is exactly what `core.Node.HRefs` already is (`specs/003-apply-patch/research.md` D2's inline-link cache, extracted from prose). Every `HRefs` entry with a non-empty `Predicate` is treated as a citation; lint checks that its `Predicate` is one of CORE §8's fixed `cito:`-aligned set (`cites`, `citesAsEvidence`, `citesAsAuthority`, `supports`, `confirms`, `extends`, `critiques`, `disputes`, `refutes`, `isCitedBy`) — membership in this fixed set is FR-009's check, reported as a distinct violation from the general predicate-registration check (D9), since citation predicates have their own required vocabulary independent of `_meta/predicates.md` registration. A bare `[[Target]]` `HRefs` entry (no predicate) is a plain mention, not a citation, and is not subject to this rule (it is still subject to D5's link-resolution check).

**Rationale**: Distinguishes an ordinary structural relation (a `Links` block or bare `Edges` entry — CORE §7's general-purpose predicate vocabulary, governed by D9) from a citation (an inline, prose-embedded, `cito:`-typed assertion about a supporting work — CORE §8's own, separate vocabulary) using a distinction `core.Node` already encodes structurally (`HRefs` vs. `Edges`/`Links`), rather than inventing a new heuristic (e.g. "target's `Kind == resource`") that CORE §8 does not itself specify.

## D11: Extension-kind profile checklist (§10/§14) — scoped to what's checkable today

**Decision**: For every node whose `Kind` is not one of CORE's four built-ins (`source`/`entity`/`resource`/`timeline`), lint checks it against the resolved `.arc/config.yml` merge-rule set (`internal/app/config.Resolve`, unchanged) exactly the way `arc apply` already does: recognized (present in the resolved `core.MergeRuleSet`) vs. unrecognized (FR-018's violation). A recognized extension kind's node otherwise runs through every *base* CORE §14 check identically to a built-in kind (front-matter validity, link resolution, provenance, predicates). No deeper, per-kind field-schema checklist is implemented, because no mechanism for a domain profile to *declare* such a schema exists anywhere in this codebase — `.arc/config.yml` carries only a kind→`MergeOp` map (`specs/003-apply-patch`).

**Rationale**: CORE §10/§14 describes a fuller profile contract (front-matter/body schema at three levels) than this codebase currently has any way to express or load; implementing FR-011 against a schema mechanism that would have to be invented from scratch, speculatively, for this feature alone contradicts YAGNI (Principle V) and this spec's own scope (spec.md's Assumptions: "this feature does not define new syntax for declaring a profile's checklist, only consumes what's already registered"). This gap is recorded in Complexity Tracking (plan.md) as a flagged, documented scope decision — not silently narrowed.

**Alternatives considered**: Designing a new profile-schema declaration format as part of this feature — rejected as scope creep beyond what the user's instruction (CORE §14's existing checklist) and spec.md actually ask for; deferred to a follow-up feature once a real domain profile needs field-level schema validation.

## D12: One `graph(ingest):` commit per document (§11.1)

**Decision**: `internal/adapter/git.Git` gains one new method:

```go
func (v VCS) CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)
```

wrapping `git log --all --fixed-strings --grep=<needle> --format=%H`, returning matching commit hashes (`--fixed-strings` so a citekey containing regex metacharacters, e.g. a title slug with a `.`, is matched literally, not as a pattern). A new, lint-private `internal/app/lint/port.VCS` interface declares only this one method — narrower than both `ctrl.port.VCS` (`IsAvailable`/`Init`/`StageAll`/`Commit`) and `graph.port.VCS` (`IsTracked`/`StageAll`/`Commit`), since lint never bootstraps, stages, or commits anything. For every `source` node lint enumerates, it calls `CommitsMatching(ctx, dir, "Source-Id: "+node.ID)` — the exact trailer format `specs/003-apply-patch/research.md` D9's commit-message convention already produces (`Source-Id: <id>`) — and reports FR-010's violation when the result has zero or more than one matching commit, naming the document and the commit hash(es) found (or none).

**Rationale**: The already-shipped `graph(ingest):` commit format (CORE §11.1, implemented by `specs/003-apply-patch`) already carries a `Source-Id:` trailer specifically so "`git log --grep=<id>` locates the ingestion commit for any source" — this decision uses exactly that documented, already-produced structure rather than inventing a new one. Extending the shared `internal/adapter/git.Git` (rather than adding a fourth, independent git client) follows `specs/003-apply-patch/research.md` D4's promotion precedent and constitution Principle VII's "duplicate clients for the same external system are forbidden" rule.

**Alternatives considered**: Parsing full commit messages/subjects instead of grepping the `Source-Id:` trailer — rejected: the trailer is precisely the structured, documented lookup key CORE §11.1 itself specifies for this purpose; grepping the free-form subject line risks false negatives if a title contains punctuation that alters the match. A dedicated `go-git` (pure-Go git library) dependency instead of shelling out — rejected: the existing `internal/adapter/git` already shells out to the system `git` binary for every other operation (Principle VII: no duplicate client for the same external system, and no second git-access strategy introduced alongside the first).

## D13: Merge-conflict markers — a pre-pass, before structural parsing

**Decision**: Before attempting `core.ParseNode` on a file's contents, lint scans its raw lines for `<<<<<<<`, `=======`, `>>>>>>>` markers (D3). If found, the file is reported as FR-012's violation (at the first marker's line) and is **excluded** from every other structural check (D2's "excluded from further checks" rule) — a file mid-merge-conflict is not valid Markdown/YAML to begin with, and attempting to also report a confusing, secondary "invalid front matter" violation for the same file would obscure the actual, actionable problem.

**Rationale**: Ordering the conflict-marker scan first prevents a single root cause (an unresolved `git merge`) from producing a wall of unrelated, misleading downstream violations for the same file, while still keeping the scan itself and the rest of the graph's checks fully independent (spec FR-013 — one broken file never stops the run).

## D14: Command grammar, output UX (ADR 002 compliance)

**Decision**: `arc lint` — a bare top-level verb (DS-01, continuing `arc init`/`arc apply`'s precedent), no positional arguments, no command-local flags. Output is resolved entirely through the existing `bios.Registry[kernel.LintResult]{Human: humanLintPrinter{}, Verbose: verboseLintPrinter{}}` (DS-04) — this is a direct, zero-new-mechanism implementation of the user's requested behavior:

- **`Human`** (default, normal verbosity): lists only nodes carrying at least one violation, each with its violated rule(s), file path, and line number, followed by one overall graph-status summary line (`N nodes checked, M passing, K failing` or an all-clear message).
- **`Verbose`** (`--verbose`/`-v`, already a global persistent flag — no new flag introduced): lists every node's individual status (pass, with `SCHEMA.StatusOK`/`IconOK`; or fail, with `SCHEMA.StatusFail`/`IconFail` and its violation detail), in the same file-walk order, followed by the identical overall summary line.
- **`--json`**: the generic `jsonPrinter[kernel.LintResult]{}` `bios.Registry` already supplies for every command, no bespoke wiring needed — the full structured result (every node's status, every violation) regardless of `--verbose`.

Exit code: `0` when `LintResult` carries zero violations, a distinct non-zero code (`1`, matching this codebase's existing convention — no other command yet uses a second distinct failure code, so no precedent to diverge from) otherwise (FR-016, DS-07) — set via a sentinel error returned from `RunE` after the result has already been printed, not via a bare `os.Exit` inside `RunE` (DS-07's "os.Exit is NEVER called from RunE" rule; the top-level `Execute()` still owns the actual process exit).

**Rationale**: The `Registry[T]{Human, Verbose}` pattern was built exactly for this shape of decision (ADR 002 DS-04) — "normal mode shows less, verbose mode shows more, both resolved from one shared flag state" is precisely what the registry already exists to make automatic. No new UX mechanism, flag, or options struct is needed.

**Alternatives considered**: A dedicated `--all`/`--show-passing` flag instead of reusing the global `--verbose` — rejected: the user's own instruction frames this exactly as the existing verbose/non-verbose distinction ("`arc lint -v` in the verbose mode shows status for each node"), and DS-03 already reserves `-v` project-wide for "reveal additional diagnostic detail" — introducing a second, overlapping flag for the same concept would violate DS-03's "MUST NOT reassign... to a different meaning" spirit by creating two flags that mean almost the same thing.
