# Phase 1 Data Model: Lint Conformance Gaps

**Feature**: `024-lint-conformance-gaps` · **Date**: 2026-08-23

Authority: ARCNET-CORE §10.2 (Sowa category) and §7.1 (identity charset). Decisions below derive
from [research.md](research.md).

---

## 1. The twelve legal Sowa combinations

`internal/app/lint/kernel/lint.go`, replacing `sowaPosition1`/`sowaPosition2`/`sowaPosition3`/`sowaLeaf`:

```go
var sowaCategories = [12][4]string{
	{"independent", "physical", "continuant", "object"},
	{"independent", "physical", "occurrent", "process"},
	{"independent", "abstract", "continuant", "schema"},
	{"independent", "abstract", "occurrent", "script"},
	{"relative", "physical", "continuant", "juncture"},
	{"relative", "physical", "occurrent", "participation"},
	{"relative", "abstract", "continuant", "description"},
	{"relative", "abstract", "occurrent", "history"},
	{"mediating", "physical", "continuant", "structure"},
	{"mediating", "physical", "occurrent", "situation"},
	{"mediating", "abstract", "continuant", "reason"},
	{"mediating", "abstract", "occurrent", "purpose"},
}
```

`ValidSowaCategory(words []string) (ok bool, reason string)` keeps its exported signature and its
existing wrong-length behavior (`len(words) != 4`, D1 in research.md). For a 4-word input, it
becomes an exact-row lookup against `sowaCategories` instead of four independent membership
checks; a non-match's `reason` names the rejected tuple and the closest legal row (D2).

**Replaces**: `sowaPosition1`, `sowaPosition2`, `sowaPosition3`, `sowaLeaf` (all four deleted).

## 2. `RuleIdentityCharset` — new lint rule

`internal/app/lint/kernel/lint.go`, `Rule` constant block:

```go
RuleIdentityCharset Rule = "identityCharset"
```

Reported by two check functions sharing this one `Rule` value:

| Check | Scope | Path | Line |
|-------|-------|------|------|
| `checkIdentityCharset` | content nodes | node's real file path | `locateFrontMatterField(raw, `\`"@id"\``)` (D4) |
| `checkSchemaIdentityCharset` | schema nodes (`Class`/`Property`) | synthesized `TypesDir`/`PredicatesDir` + name + `.md` (D3) | `0` |

## 3. Forbidden identity character set — one closed list

`internal/core` (new, alongside `isCamelCase`/`ErrTypeCasing`, D6):

```go
var forbiddenIdentityChars = []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|', '.'}
```

A pure scan function over `[]rune(id)` (D5, 1-indexed rune position) returns every offending
`(char, position)` pair found — zero pairs means the identity is legal. Both `arc lint` (reports,
keeps walking) and `arc apply` (rejects the whole operation) format their message from the same
pairs (D8), so the two surfaces never drift onto different wording for the same rule.

**Replaces**: nothing — this is new; no forbidden-character check exists today.

## 4. `Violation.Message` shape for `RuleIdentityCharset`

Single offending character:

```
identity "Handshake/Protocol" contains forbidden character "/" at position 10
```

Multiple offending characters — every one named, not just the first (spec.md US2 Acceptance
Scenario 5):

```
identity "v1.3: Handshake/Protocol" contains forbidden characters "." at position 3, ":" at position 4, "/" at position 17
```

## 5. `arc apply` rejection — new error

`internal/app/graph/service/errors.go`, alongside `ErrNodeWrite`:

```go
ErrIdentityCharset = faults.Safe2[string, string]("identity %q %s")
```

Second argument is the same detail fragment `contains forbidden character(s) ... at position ...`
built by the shared `internal/core` primitive (§3), so the message body matches the lint
violation's wording exactly (D8), differing only in the surrounding sentence CLI errors already
carry via `faults`.

## 6. Enforcement points inside `Apply`

Two additions to `internal/app/graph/service/apply.go` (research.md D7), both reusing the
existing `rollback(store, createdPaths)` unwind path already exercised by every other per-node
failure in `Apply`'s loop — no new rollback mechanism:

| Point | What it checks | When |
|-------|-----------------|------|
| Pre-scan (new guard, mirrors `guardNoOldFormatNodes`'s shape) | every `patch.Nodes[i].ID` | after `readPatch` succeeds, before any write |
| In-loop, before `schema.RegisterType` | `node.Type` (becomes a `Class` identity) | per node, only when the type is not yet recognized |
| In-loop, before `schema.RegisterPredicate` | each observed predicate name (becomes a `Property` identity) | per node, only for a not-yet-registered predicate |

`arc apply-schema` is explicitly out of scope (research.md D7) — it is a separate command and
patch format this feature does not touch.

## 7. Key Entities carried over from spec.md, made concrete

- **Entity category** → `[4]string` checked against `sowaCategories` (§1).
- **Node identity** → `node.ID string`, scanned rune-by-rune against `forbiddenIdentityChars` (§3);
  for a schema node, the identity is the `index.Types`/`index.Predicates` map key (no filename
  lookup, research.md D3).
- **Patch** → `patch.Nodes []core.Node`, iterated once by the new pre-scan guard (§6) before any
  node in it can be written.
