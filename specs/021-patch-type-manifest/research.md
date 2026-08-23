# Phase 0 Research: Patch Manifest Identity (`@type: patch`)

**Feature**: `021-patch-type-manifest` | **Date**: 2026-08-23 | **Plan**: [plan.md](./plan.md)

No `NEEDS CLARIFICATION` markers were carried in from `spec.md`. The decisions below resolve
the design questions the spec deliberately left to planning, plus one correction to the
technical context supplied with the `/speckit-plan` invocation.

---

## D1 — A bare `@type:` key is a hard YAML failure, verified not assumed

**Decision**: The manifest key MUST be written quoted (`"@type"` or `'@type'`). Both quoting
styles decode to the identical `@type` map key, so the parser needs no special handling for
them; only the *unquoted* form needs a translated error (FR-015).

**Rationale**: Verified empirically against this repo's actual parser stack
(`goldmark-meta` → `gopkg.in/yaml.v2`) rather than inferred from the YAML grammar. Four
front-matter forms were parsed through `core.parseDocument`:

| Front-matter form | Result |
|---|---|
| `@type: patch` (bare) | error — `yaml: found character that cannot start any token` |
| `"@type": patch` | `map[@type:patch document:x published:2024-01-01]` |
| `'@type': patch` | identical map — single quotes are equivalent |
| `"@type": patch` + `kind: patch` | both keys present in one map, `@` prefix intact |

This confirms the pasted constraint ("the `@` prefix survives `normalizeYAMLMap` unchanged")
and settles the spec's first edge case: the bare form fails inside `parseDocument`, *before*
`decodePatchManifest` is ever reached, so FR-015's error cannot live in the decode path.

**Consequence**: `ParsePatch` must inspect the raw source on the `parseDocument` error branch
and, when it finds a bare identity key, return `ErrIdentityQuoting` instead of wrapping the
raw yaml error in `ErrManifestInvalid`.

**Alternatives considered**: Pre-processing the front matter to auto-quote bare `@` keys —
rejected: it would silently accept a document ARCNET-CORE §14.2.1 does not permit, and it
would diverge from `arc lint`'s `RuleIdentityQuoting`, which reports the same construct as a
violation on nodes.

---

## D2 — No `Deprecations` field; the hexagonal constraint is met by a typed error

**Decision**: `core.Patch` gains **no new field**. The four new failure modes are
package-level `faults` constants in `internal/core/errors.go`, returned as ordinary errors.

**Rationale**: The technical context supplied with this `/speckit-plan` invocation reproduces
`CORE-FIX.md` §3.7, which was written against that document's §3.3 FR-003 — a one-release
transitional acceptance of `kind: patch` emitting a deprecation notice. `spec.md` supersedes
that: the user's specify input states compatibility "IS NOT REQUIRED", and the spec's first
Assumption records it. With no notice to emit, the `Deprecations []string` carrier has nothing
to carry.

The constraint that design was protecting — `internal/core` must not import `internal/bios`
(Principle III) — still binds and is still satisfied, by the simpler mechanism: a rejected
patch returns an error, and errors already cross that boundary freely.

**Consequence**: FR-011 (`--json` output unaffected) is satisfied *by construction* rather
than by a `json:"-"` tag. `internal/app/graph/service.Apply` needs no `bios.Reporter` change.

**Alternatives considered**:
- Adding `Deprecations []string` with `json:"-"` anyway, "for later" — rejected under YAGNI
  (Principle V). A field with no writer is dead weight on a type whose json tags are a public
  contract.
- Emitting the notice from `decodePatchManifest` via a package-level logger — rejected: it is
  exactly the `bios` import into `core` the constraint forbids.

---

## D3 — A minimal bare-key detector in `core`, not a shared locator with `lint`

**Decision**: `internal/core` gets its own small unexported detector answering one boolean
question: does this front matter contain a bare `@id`/`@type` key? `internal/app/lint/service`
keeps `locateUnquotedIdentityKey` unchanged.

**Rationale**: The two answer different questions for different consumers. `lint` needs a
**1-based line number** to populate a `Violation`; `core` needs a **yes/no** to choose which
error to return from a parse that has already failed. `lint/service` does import `core`, so
delegation is technically available — but FR-013 pins lint's behaviour as unchanged, and
routing a lint reporting concern through the core AST package inverts the dependency's intent:
`core` would grow an API shaped by `lint`'s needs.

**Alternatives considered**:
- Export `core.LocateUnquotedIdentityKey(raw, key) int` and delete `lint`'s copy — cleaner on
  a DRY reading, and worth revisiting if a third consumer appears. Rejected here because it
  expands the diff into the lint package for zero behavioural gain, against a spec that
  explicitly fences lint out of scope.
- Have `core` skip FR-015 and let the raw yaml error through — rejected: FR-015 exists
  precisely because `yaml: found character that cannot start any token` is unactionable, and
  SC-007 requires every rejection to be self-correcting.

---

## D4 — One total decision over the (`@type`, `kind`) pair

**Decision**: `decodePatchManifest` delegates identity to a new unexported
`patchManifestType(manifest) error`, which evaluates the pair exactly once, in this order:

| # | `@type` | `kind` | Outcome | FR |
|---|---|---|---|---|
| 1 | present | present, **different value** | `ErrManifestTypeConflict` | FR-005 |
| 2 | `patch` | absent, or `patch` | **accept**; `kind` ignored | FR-001, FR-004 |
| 3 | present, not `patch` | absent, or equal | `ErrManifestNotAPatch` | FR-016 |
| 4 | absent | `patch` | `ErrManifestLegacyKind` | FR-003 |
| 5 | absent | absent, or any non-`patch` value | `ErrManifestInvalid` (unchanged) | FR-006 |

**Rationale**: Ordering the conflict check *first* is what makes the table total and
unambiguous. `@type: patch` + `kind: source` and `@type: source` + `kind: patch` are both
conflicts and neither should fall through to a "not a patch" or "legacy key" message that
would describe only half the problem. Rows 2 and 3 then split on the value, and rows 4 and 5
handle the no-`@type` world.

Extracting `patchManifestType` also keeps `decodePatchManifest` under Principle IV's 25-line
ceiling, which inlining the five cases would break.

**Consequence**: three new `faults` constants alongside the existing `ErrManifestInvalid`:

```go
ErrManifestTypeConflict = faults.Safe2[string, string](
    "manifest declares \"@type\": %s and legacy kind: %s — the two disagree")
ErrManifestLegacyKind = faults.Type(
    "manifest uses the retired \"kind: patch\" key — rewrite it as \"@type\": patch")
ErrManifestNotAPatch = faults.Safe1[string](
    "manifest declares \"@type\": %s, expected the literal lowercase \"patch\"")
```

`ErrManifestNotAPatch` carries the offending value, so `"@type": Patch` produces a message
that shows the casing error rather than merely asserting one (FR-016, SC-007).
`ErrManifestInvalid`'s own text and behaviour are untouched, satisfying FR-006's "unchanged in
wording" literally.

**Alternatives considered**: Checking `kind` first and short-circuiting to the legacy error —
rejected: it would report `@type: patch` + `kind: source` as a legacy-key problem, when the
file's real defect is that it contradicts itself.

---

## D5 — `readPatch` must attach the file path to parse errors

**Decision**: `internal/app/graph/service.readPatch` wraps `core.ParsePatch`'s error as
`ErrPatchRead.With(err, patchPath)`, matching what it already does for mount and open
failures.

**Rationale**: FR-003 and SC-007 require the rejection to name the offending file, and
`decodePatchManifest` structurally cannot — it receives a decoded map, never a filename
(the same asymmetry `ParseNode`'s doc comment already documents for `@id`-vs-basename
validation). Today `readPatch` wraps its two I/O failures with the path but returns parse
errors bare, so `arc apply broken.md` currently loses the filename on exactly the branch this
feature is about. `internal/app/schema/service.readPatchSource` already wraps parse errors
this way, so this aligns the two rather than inventing a convention.

**Consequence & risk**: this prefixes `failed to read patch file <path>: ` onto *every*
`arc apply` parse failure, including pre-existing ones such as spec 019's `ErrTypeCasing`.
Existing E2E assertions that match on the bare message will need their expectations widened.
`errors.Is` assertions are unaffected — `faults` wrapping preserves the chain. This is called
out as an implementation hazard for `/speckit-tasks`, not a silent behaviour change.

**Alternatives considered**: Threading a filename parameter into `ParsePatch` — rejected: it
changes a public `core` signature that `contracts/ast-contract.md` pins, and for the benefit
of an error message the caller can already supply.

---

## D6 — `revert`'s patch detection adopts the shared recognition rule

**Decision**: `internal/app/graph/service.isPatchDocument` switches from "`ParsePatch`
succeeds" to `core.LooksLikePatch(raw)`.

**Rationale**: FR-009 requires every command to apply an identical recognition rule.
`isPatchDocument` exists to make `arc revert` *skip* exchange files left sitting in the graph
tree — it must never treat one as a node to remove or rewrite. Under the current
`ParsePatch`-succeeds test, a patch whose body is malformed (or, after this feature, one still
keyed `kind`) stops being recognized as an exchange file and becomes eligible for node
handling by a destructive command.

**Consequence**: this is a behaviour change beyond the literal format migration — `arc revert`
will now skip a malformed patch it previously would have processed as a node. Given revert's
blast radius (Principle IX: "confirmation rigor MUST scale with the blast radius"), erring
toward skipping is the safe direction, and FR-009 asks for it explicitly.

**Alternatives considered**: Leaving `isPatchDocument` on `ParsePatch`-succeeds and accepting
a two-rule tool — rejected against FR-009. Recorded here rather than buried in a task so a
reviewer can challenge it: it is the one decision in this feature that changes behaviour for
input unrelated to the manifest key.

---

## D7 — Which repository occurrences migrate, and which are history

**Decision**: FR-014's "every patch example shipped in the repository" is scoped by whether an
occurrence is **executable or normative** (migrate) or a **dated record** (leave):

| Class | Files | Action |
|---|---|---|
| Test fixtures | `cmd/arc/graph/testdata/**` (10 files) | **Migrate** |
| Inline test constants | `apply_test.go` (33), `markdown_test.go` (27), `apply_schema_test.go` (9), `batch_test.go` (6), `revert_test.go` (5), `service/*_test.go` (11) | **Migrate** |
| Runnable validation guides | `specs/*/quickstart.md` — 003, 005, 009, 010, 011, 013, 018, 019, 020 | **Migrate** |
| Normative contracts | `specs/003-apply-patch/contracts/cli-contract.md`, `specs/007-arc-subgraph/contracts/cli-contract.md` | **Migrate** |
| Living architecture doc | `ARCHITECTURE.md` glossary rows **Patch** (`:244`), **Batch Plan** (`:273`) | **Migrate** |
| Living roadmap | `specs/VISION.md` (`arc apply` line) | **Migrate** |
| Production doc comments | `internal/core/markdown.go`, `internal/app/graph/service/{apply,batch}.go` | **Migrate** |
| Dated records | past `spec.md`, `research.md`, `data-model.md`, `tasks.md`, `bugs/*.md`; `specs/CHANGELOG.md` | **Leave** |
| This feature's own docs | `specs/021-*/spec.md`, `checklists/requirements.md` | **Leave** — they quote the retired key as the defect |
| Program brief | `CORE-FIX.md` | **Leave** — dated analysis |

**Rationale**: A `quickstart.md` is a *runnable* validation guide by this project's own
definition; leaving one on the retired key ships a guide that fails on execution, which is
what FR-014 is guarding against. A past feature's `research.md` is the opposite: it records
what was true and decided at that date, and rewriting it would falsify the record. `spec.md`'s
FR-014 says "none may present `kind: patch` as a **current, valid** example" — the dated
records do not, once this feature's own CHANGELOG entry lands.

**Alternatives considered**: Migrating every occurrence uniformly — rejected: it rewrites
specs 003/007/020's history to claim they specified a format they did not. Migrating nothing
outside `testdata/` — rejected: it leaves nine executable quickstarts broken.

---

## D8 — A new fixture proves FR-008, and `notes.md`'s prose is corrected

**Decision**: Add `cmd/arc/graph/testdata/batch/legacy-kind.patch.md` carrying `kind: patch`
and no `@type`. Update the prose inside `cmd/arc/graph/testdata/batch/notes.md`, which
currently explains itself as "front matter that does *not* declare `kind: patch`".

**Rationale**: FR-008 and SC-005 are the only requirements with no existing fixture that can
demonstrate them, and they guard the feature's one silent-failure mode. Without a legacy file
in the batch tree, nothing proves `LooksLikePatch`'s widened recognition actually routes such a
file to `candidate{err}` rather than incrementing `notAPatch`. Note that
`testdata/batch/nested/deep/legacy.patch.md` does **not** serve this purpose despite its name —
"legacy" there describes its subject matter (legacy key agreement), not its manifest key; it
migrates like every other fixture.

`notes.md` must stay a *non*-patch that is passed over; only its explanatory prose changes, so
that the fixture and its stated reason for existing do not drift apart.

**Consequence**: the batch E2E's expected counts shift by one — one additional candidate, one
additional failure, `notAPatch` unchanged. That count assertion *is* the SC-005 test.
