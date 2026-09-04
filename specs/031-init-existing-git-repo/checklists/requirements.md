# Specification Quality Checklist: Graph Initialization Inside an Existing Git Repository

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-04
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- **Validation result**: 16/16 items pass, unchanged from the pre-clarification
  pass (16/16). No item changed state; the clarification session resolved open
  design decisions rather than fixing checklist failures.
- **Deliberate exception to "no implementation details"**: the spec names the
  command (`arc init`), the flag (`--skip-git-init`), the local-state exclusion
  file (`.arc/.gitignore`), and the machine-readable field name (`repository`).
  These are the product's user-facing contract, not implementation choices, and
  the flag name was fixed by the request itself. All other language stays at the
  "versioned project" / "parent repository" level rather than naming git
  commands, internal paths, or packages.
- **Adjustment made during initial validation**: SC-008 (then SC-007) originally
  asserted "the existing test suite passes", which is a verification mechanism
  rather than a user-facing outcome. Rewritten to enumerate the observable
  behaviors that must remain unchanged.
- **Adjustment made during clarification**: requirement IDs added mid-session
  were briefly suffixed (FR-026a/b, SC-006a) and have been renumbered to a
  contiguous FR-001..FR-031 / SC-001..SC-008 sequence. No external references
  existed at the time.

## Clarification session 2026-09-04 — decisions recorded

Four of the five assumptions flagged after `/speckit-specify` are now settled
decisions in the spec's Clarifications section. The fifth (repairing
already-created nested repositories) stands as out-of-scope and was not
challenged.

| # | Decision | Requirements touched |
|---|----------|----------------------|
| 1 | Local-state exclusion is self-contained (`.arc/.gitignore` with `*`), never touching the host project's ignore-rules file, in both modes | FR-007, FR-016, FR-017, SC-008 |
| 2 | A pre-existing folder sharing a canonical layout name is refused, empty or not; recovery is to initialize into a subfolder, and the message must say so | FR-015, FR-031 |
| 3 | `--skip-git-init` with no enclosing repository is a usage error — no fallback, no history-less graph mode | FR-012 |
| 4 | Machine-readable output gains an unconditional `repository` field; existing fields keep their names and meanings | FR-027, FR-028, SC-007 |

**Deferred to planning** (not spec-level ambiguities):

- Which concrete commands must be audited to establish FR-021–FR-025. Some
  already hold via the BUG-002 fixes in the git adapter; the spec's obligation is
  that they be verified, not assumed.
- Exit-code taxonomy. Constitution IX permits distinct non-zero codes, but the
  codebase currently exits 1 uniformly for every failure. FR-029 matches existing
  practice; introducing a taxonomy would be a separate, tool-wide change.
