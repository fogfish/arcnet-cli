# Phase 1 Data Model: Patch Manifest Identity

**Feature**: `021-patch-type-manifest` | **Plan**: [plan.md](./plan.md) | **Research**: [research.md](./research.md)

This feature adds no type and changes no field. What it changes is the **decision** that maps
a front-matter map onto a `core.Patch` or an error. That decision is the model.

---

## 1. Entities

### 1.1 `core.Patch` — unchanged

```go
type Patch struct {
    Document  string         `json:"document"`
    Published time.Time      `json:"published"`
    Title     string         `json:"title,omitempty"`
    Stats     map[string]any `json:"stats,omitempty"`
    Nodes     []Node         `json:"nodes"`
}
```

No field is added, removed, or retagged. The identity key was never represented on this type —
it is a *recognition predicate* consumed during decode, not carried data — which is why FR-011
(`--json` unaffected) holds by construction rather than by a `json:"-"` tag. See
[research.md](./research.md) D2.

### 1.2 Patch Manifest — the front-matter map

The value `parseDocument` hands to `decodePatchManifest`, after `normalizeYAMLMap`:

| Key | Type | Mandatory | Role | Changed here |
|---|---|---|---|---|
| `@type` | `string` | **yes** | Identity. Literal lowercase `patch`. Quoted key. | **new — the only recognition gate** |
| `kind` | `string` | no | Retired pre-0.5 identity. Not a valid declaration. | **retired — recognized only to refuse** |
| `document` | `string` | yes | Source citekey this patch contributes | no |
| `published` | date / RFC3339 string | yes | Drives timeline derivation | no |
| `title` | `string` | no | Human label | no |
| `stats` | `map` | no | Carried through, not validated | no |

Any other key is ignored, as today. **`kind` is ignored too, once `@type` is present and
agrees** — it is not an error to carry a redundant retired key, only to carry a contradictory
one (FR-004 / FR-005).

---

## 2. The recognition decision

Evaluated once, top to bottom, first match wins. `T` = `manifest["@type"]` as string,
`K` = `manifest["kind"]` as string; "present" means the key exists with a non-empty string.

| # | Condition | Result | FR | Error constant |
|---|---|---|---|---|
| 1 | `T` present ∧ `K` present ∧ `T ≠ K` | reject | FR-005 | `ErrManifestTypeConflict(T, K)` |
| 2 | `T = "patch"` | **accept** | FR-001, FR-004 | — |
| 3 | `T` present ∧ `T ≠ "patch"` | reject | FR-016 | `ErrManifestNotAPatch(T)` |
| 4 | `T` absent ∧ `K = "patch"` | reject | FR-003 | `ErrManifestLegacyKind` |
| 5 | otherwise | reject | FR-006 | `ErrManifestInvalid` *(unchanged)* |

### Worked cases

| Manifest | Row | Outcome |
|---|---|---|
| `"@type": patch` | 2 | accept |
| `"@type": patch` + `kind: patch` | 2 | accept, `kind` ignored |
| `"@type": patch` + `kind: source` | 1 | conflict |
| `"@type": source` + `kind: patch` | 1 | conflict |
| `"@type": Patch` | 3 | not-a-patch, message shows the casing |
| `"@type": source` | 3 | not-a-patch |
| `kind: patch` | 4 | legacy key |
| `kind: source` | 5 | manifest invalid |
| *(neither)* | 5 | manifest invalid |
| bare `@type: patch` (unquoted) | — | never reaches the table; `ErrIdentityQuoting` from `ParsePatch` (D1) |

Row 1 precedes rows 2–4 deliberately: a self-contradictory manifest must be reported as a
contradiction, not as whichever half the check happened to read first.

The table is **total** — every (`T`, `K`) pair lands in exactly one row — which is what makes
the five-case unit table in `internal/core/markdown_test.go` a complete test of recognition.

---

## 3. Detection vs. acceptance

Two distinct predicates, deliberately not the same function:

| | `LooksLikePatch(raw) bool` | `decodePatchManifest(m) (Patch, error)` |
|---|---|---|
| Question | "Was this file *meant* as a patch?" | "Is this a valid patch?" |
| True for `"@type": patch` | yes | accept |
| True for `kind: patch` | **yes** | **reject** (row 4) |
| True for `"@type": patch` + `kind: source` | yes | reject (row 1) |
| Purpose | route the error to the right reporter | the sole gate on acceptance |

`LooksLikePatch` widening to recognize the retired key is **not** backward compatibility — no
document is accepted because of it. It is what stops `arc apply batch` from counting a legacy
patch as ordinary Markdown and skipping it in silence (FR-008), and what lets
`guardNoOldFormatNodes` surface the actionable legacy-key error instead of `ParseNode`'s
generic old-format heuristic.

Consumers of `LooksLikePatch` after this feature:

| Call site | Uses it to |
|---|---|
| `graph/service.classifyPatchFiles` | separate failed candidates from passed-over Markdown |
| `graph/service.guardNoOldFormatNodes` | choose the patch error over the node error |
| `graph/service.isPatchDocument` (revert) | skip exchange files (newly adopts it — D6) |

---

## 4. Errors

All four are `internal/core` package-level `faults` constants (Principle XII). None carries a
file path; the path is attached by the service-layer caller that knows it (D5).

| Constant | Shape | Message |
|---|---|---|
| `ErrManifestTypeConflict` | `faults.Safe2[string, string]` | `manifest declares "@type": %s and legacy kind: %s — the two disagree` |
| `ErrManifestLegacyKind` | `faults.Type` | `manifest uses the retired "kind: patch" key — rewrite it as "@type": patch` |
| `ErrManifestNotAPatch` | `faults.Safe1[string]` | `manifest declares "@type": %s, expected the literal lowercase "patch"` |
| `ErrIdentityQuoting` | `faults.Safe1[string]` | `%q must be a quoted YAML string key, found it unquoted` |
| `ErrManifestInvalid` | *(existing, unchanged)* | `manifest is missing a mandatory field or uses the pre-0.5 node format` |

`ErrIdentityQuoting`'s wording intentionally matches `lint`'s existing `RuleIdentityQuoting`
message so a user meets the same sentence whichever command surfaces the problem — the two are
independent implementations of one message (D3), and that pairing is worth a test.

Every constant is reachable via `errors.Is` for callers that need to branch; no call site may
string-match (constitution, Mandatory Libraries & Tooling).

---

## 5. Rendering

`renderPatchManifest` emits, in fixed order:

1. `"@type": patch` — via `appendQuotedKeyYAMLPair`, the same helper `renderAttrYAML` already
   uses for node `@id`/`@type`, so the double-quote styling is shared rather than reimplemented
2. `document`
3. `published` — `2006-01-02`
4. `title` — when non-empty
5. `stats` — when non-empty, flow style

The only change is step 1 replacing `appendYAMLPair(root, "kind", "patch")`. Ordering,
styling, and every other field are untouched, which is what makes FR-010's byte-for-byte
round-trip property hold.
