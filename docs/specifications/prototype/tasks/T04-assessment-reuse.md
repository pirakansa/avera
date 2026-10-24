# T04 — Matching, change detection, and assessment reuse

Status: not started. Prerequisite: T01 and T00 matching fixtures. Read [data contracts](../data-contracts.md).

## Outcome

Use previous inventory and advance assessments to produce a focused review queue while retaining explanations and making stale or conflicting decisions visible.

## Ownership

`internal/assessment`, its tests, and matching scenario fixtures. This package is a pure transformation of explicit inputs: no host access, file writes, UI, external lookups, or automatic human acceptance.

## Work steps

1. Implement target/schema compatibility checks and finding matching using the shared identity functions.
2. Compute change state independently from human decision and assessment applicability.
3. Implement every row of the contract's carry-forward matrix, including no prior snapshot, missing current items, partial scopes, and conflicts.
4. Evaluate advance expectations against supported observed fields. Missing fields or unsupported predicates must not become matches.
5. Preserve actor, date, explanation, and evidence references from the original assessment. Do not rewrite its provenance to the new collection time.
6. Generate updated editable assessment entries for new/current items and retain unmatched prior entries separately. Avoid duplicating a carried assessment on every run.
7. Produce deterministic review output and shared summary counts used by both CLI and reporting.
8. Expose reasons such as `facts_changed`, `coverage_incomplete`, or `multiple_assessments` as structured diagnostics, with human wording supplied by presentation code.

## Acceptance cases

- Unchanged meaningful facts preserve an applicable decision without asking the user to retype the explanation.
- Changed facts preserve the old rationale but require re-review; current acceptance counts exclude it.
- Generating editable output never silently rebases an old acceptance; explicit human reconfirmation produces a judgment tied to the current facts.
- PID-only changes do not force review when stable attribution and scope are sufficient; uncertain attribution is not silently treated as stable.
- Missing items with insufficient coverage remain unknown; complete comparable absence means not observed, not automatically remediated.
- Exact advance expectations can apply without a previous snapshot, with origin and date visible.
- An insufficient, changed, or conflicting advance assessment cannot suppress a finding.
- Target mismatch cannot transfer acceptance between machines.
- Repeating the same inputs produces the same result without duplicate entries or altered human text.
- Historical/unmatched items do not inflate current finding counts.

## Handoff

Supply the pure review API, reason codes, derived count definitions, and complete example outputs to T05/T06. List scenarios where v0 intentionally chooses extra re-review rather than guessing equivalence.
