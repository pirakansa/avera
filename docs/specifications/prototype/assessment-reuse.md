# Assessment reuse API

T04 implements baseline `t00-2026-10-24.1` in `internal/assessment`.
The package performs no host access, file I/O, external lookups, or human
acceptance. The authoritative matching rules remain in [data contracts](data-contracts.md).

This document describes raw T04 comparison, not the complete current portable
review. `reportfile.File.Review` additionally applies [common endpoint judgments](shared-service-judgments.md)
and human-link context; common judgments can apply despite process/namespace
changes or incomplete collection, with diagnostics retained separately.

## Pure transformation

`assessment.Evaluate(current, previous, assessments)` accepts a required
`*inventory.Snapshot` and optional previous snapshot and assessment envelope.
It validates linked inputs before returning `(*assessment.Result, error)`.
Incompatible targets, versions, revisions, and inconsistent referenced bases
return an error with no result. Callers must decode external artifacts with
`inventory.Decode` first; this retains T01's strict structural validation.

The result contains:

- `Review`: deterministic `inventory.Review` linking the explicit input artifacts.
- `Editable`: current matching assessments, plus one unreviewed starter for each
  current finding without an assessment. Existing assessments retain their basis,
  actor, date, human text, references, and decision. Conflicts retain all entries.
- `Unmatched`: supplied assessments without a current exact identity match,
  separate from the editable current entries and current finding counts.

No inputs are modified. Output arrays are sorted by their contract IDs. New
starter IDs use `starter-` followed by the finding ID; collisions with supplied
assessment IDs append `-new` until unique. Starters reference the current
snapshot and fingerprint but always have an unreviewed decision and blank human
fields. Reusing both output partitions as explicit inputs creates no duplicate
entries. Callers preserving a single active assessment artifact must combine both
partitions; omitting unmatched entries loses historical context. Preserve the
original immutable inputs separately for review provenance.

Regeneration does not reconfirm an existing decision. A human must explicitly
replace a stale entry with a judgment against the current basis. Keeping both
active creates a conflict. The original input artifact remains historical evidence.

## Rendering and counts

Presentation code resolves review assessment IDs against the supplied assessment
artifact to display original actor, date, rationale, and evidence references.
Review IDs deliberately exclude newly generated starters until those are supplied
as input to a subsequent evaluation. Resolve historical findings against the
previous snapshot. Render safeguards and external references as human assertions;
this package does not verify them.

`assessment.Summarize(items, unmatchedCount)` supplies the same summary used by
`Evaluate`. Current decision counts partition current items into accepted,
action-required, and unreviewed. Needs-review and conflict are independent
applicability subsets. Historical items only increment historical counts;
unmatched counts represent assessment entries. Render coverage alongside counts,
especially for empty or failed scans.

Exported reason constants are `FactsChanged` (`facts_changed`),
`CoverageIncomplete` (`coverage_incomplete`), `MultipleAssessments`
(`multiple_assessments`), `ExpectationMismatch`
(`expectation_mismatch_or_incomplete`), and `NotObserved` (`not_observed`).
They contain no human wording. Coverage reasons precede fingerprint/expectation
reasons for a sole assessment; conflicts always use multiple-assessment reasons.

## Examples and intentional extra review

The [scenario manifest](../../../test/fixtures/prototype/contracts/scenarios.json)
provides 13 complete input/output examples. `TestContractScenarios` executes all
of them, including explicit reconfirmation, missing items, conflicts, advance
expectations, and stored bases without a prior snapshot. Each scenario also
checks deterministic output, valid serialization, unchanged inputs, preserved
assessment content, and duplicate-free editable regeneration. Additional tests
cover incompatible inputs, uncertain attribution, changed scopes, and T02's
transient-evidence fixtures.

V0 intentionally requires extra review for incomplete domain coverage, uncertain
network attribution, noncomparable scopes, moved sudo source anchors, incomplete
local or policy context, unsupported sudo constructs, and partial advance
expectations. Owner order is normalized through shared hash functions; PID,
evidence, and collection-time churn alone do not invalidate stable facts.
Missing observations are never labeled remediated. An old acceptance remains
available through the assessment link but contributes no current acceptance
unless applicable.

T05 should consume this review and summary directly. T06 should retain named
inputs and serialize results through T01 storage APIs; publication and CLI
integration are outside T04. No shared contract changes were needed.

## Validation record

For this implementation, `vorbere run setup`, `vorbere run check`,
`vorbere run test`, and `vorbere run build` passed before changes. After changes,
`go test ./internal/assessment`, `vorbere run check`, `vorbere run test` (race
detector), and `vorbere run build` passed. Govulncheck reported zero reachable
vulnerabilities, with the existing informational findings of one vulnerability
in imported packages and thirteen in required modules. Generated build artifacts
were removed after validation. No live host collection or release was performed.
