# Specification Quality Checklist: ARCNET-CORE v0.10 — Schema Vocabulary Conformance

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

## Notes

- Items marked incomplete require spec updates before `/speckit-clarify` or `/speckit-plan`

### Validation record (iteration 1)

- **No implementation details** — PASS after revision. The first draft named Go identifiers
  (`CorePredicateDefs`, `core.TypeDef.Merge`) and file paths (`_schema/Property/`) carried over
  from `CORE-FIX.md`'s plan-level notes. Rewritten in domain language: "built-in vocabulary",
  "predicate definition", "vocabulary folders". Predicate and merge-operation *names*
  (`firstWriteWin`, `union`, `cites`) are retained deliberately — they are ARCNET-CORE's own
  normative vocabulary, i.e. the subject matter, not an implementation choice.
- **No [NEEDS CLARIFICATION] markers** — PASS. Three candidate questions were resolved against
  documented defaults rather than deferred, each recorded in Assumptions with its rationale:
  the analytics-score merge behaviour (`lastWriteWin`), the singular/plural authorship predicate
  (register both), and the shape of the migration path (specified by outcome in FR-017–FR-024,
  leaving the command surface to `/speckit-plan`).
- **Scope bounded** — PASS. The description's own enumeration skips its third item and miscounts
  its total; the omission is identified and resolved in Assumptions rather than silently dropped
  or silently included. An explicit Out of Scope section fences the four adjacent conformance
  findings tracked elsewhere.
- **Independently testable stories** — PASS. Each of the five stories names an Independent Test
  that exercises it without any other story's changes in place.
- **Success criteria technology-agnostic** — PASS. SC-001 through SC-007 are stated as observable
  graph and command outcomes; none names a package, type, or function.

### Validation record (iteration 2 — post-`/speckit-plan` amendment, 2026-08-23)

Re-run after `spec.md` was amended against the live upstream document. All 16 items still pass.

- **Requirements are testable and unambiguous** — was PASSING VACUOUSLY, now genuinely passes.
  Validation against ARCNET-CORE v0.11 found two spec premises that do not match the code:
  US1's "a third apply triples it" (the near-duplicate guard already prevents this) and US4's
  "silently creates a definition the tool guessed" (front-matter predicates never reach
  auto-registration). Four acceptance scenarios derived from those premises were green against
  `main`. Both stories were rewritten against real behaviour; see [plan.md](../plan.md) F2.
- **Scope is clearly bounded** — re-checked after nine additional defects were added
  (FR-026–FR-029). All nine are edits to the same two tables the feature already owns; none
  widens the surface. The one scope decision that *is* material — retracting the external-work
  type's requirements, which supersedes spec 022's recorded clarification — is flagged in
  plan.md F3 and listed as an open item rather than absorbed silently.
- **Dependencies and assumptions identified** — three assumptions were *retired* rather than
  restated: the authorship singular/plural hedge and the external-work requirements are now
  settled by v0.11 and cite it directly; the used-but-unregistered carve-out shrank from seven
  predicates to one.
- **No implementation details** — re-checked across the new FRs. FR-026–FR-029 name predicate
  roles and types in domain language ("the publication-year predicate", "the external-work
  type"), not Go identifiers.

**Note on Status**: the spec is marked amended, and the amendment is attributable — every change
traces to a numbered finding in plan.md's Validation Findings section.
