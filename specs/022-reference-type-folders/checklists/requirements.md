# Specification Quality Checklist: ARCNET-CORE v0.11 — `Reference` Type and Type-Named Folders

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-23
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
- [x] Feature scope reflects the clarification decisions of 2026-08-23

## Notes

**Iteration 1 (2026-08-23)** — 3 open `[NEEDS CLARIFICATION]` markers, all raised because the
authoritative sources disagreed with each other and no reasonable default settled them:
`Reference`'s folder name, `Reference`'s predicate lists, and the migration surface.

**Iteration 2 (2026-08-23)** — all three resolved by the user and recorded in the spec's
`## Clarifications` section. Resulting spec changes:

- Folder is `Reference/`; the naming rule applies universally, so `sources/`, `entities/`,
  `resources/`, `_schema/predicates/`, `_schema/types/` are renamed on the same grounds.
- `Reference` requires `title`/`ref`/`relevance`, offers
  `url`/`authors`/`year`/`doi`/`status`/`isCitedBy`/`notes` (FR-004).
- **Migration removed entirely.** The user accepted a clean break. Former User Story 4 and
  FR-021–FR-034 were deleted; FR-021/FR-022 now state the breaking-change position, and the
  union-cannot-retract rationale is retained as its justification rather than as a migration
  requirement. Migration-specific edge cases and success criteria were dropped, and SC-005–SC-007
  were rewritten around conformance, path stability, and cross-command path agreement.

**Open risk carried knowingly** (recorded in Assumptions, not a blocking defect): running a
post-feature build against a pre-feature graph is undefined, and the likely outcome is a graph
with both folder layouts side by side and no error raised. The user has accepted this; adding
detection would be scope the feature does not ask for. Worth revisiting as a separate feature
if it bites.

All checklist items pass. Spec is ready for `/speckit-plan`.
