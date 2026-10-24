# Prototype decision register

Status: initial planning register. Entries marked proposed have not been selected through user trials. Task owners update outcomes and reasons as evidence becomes available; do not retroactively label an experiment as a product requirement.

## Agreed constraints

| ID | Decision | Source / consequence |
|---|---|---|
| A01 | Help inexperienced users inventory concerns and explain treatment | Guidance is part of the product, not just a report decoration |
| A02 | Ubuntu first; Go implementation | Specific Ubuntu release and architecture still need a test record |
| A03 | Minimize target dependencies; aim for a single binary | No required SSH/service installation or opened port for inspection |
| A04 | Collection requires root | No need to build a non-root collection mode for this prototype |
| A05 | Ports and sudo are illustrative prototype subjects | They do not establish release scope or complete risk coverage |
| A06 | The project owner tests an unreleased prototype | Trial outcomes drive further work; no automatic publication |
| A07 | Summary, explanation/decision, and evidence serve distinct readers | Evidence also supports the initial user's judgment |
| A08 | Prior inventory and advance judgments are inputs to collection | Reuse is part of the workflow, not an optional afterthought |
| A09 | Target-side SBOM and version/CVE assessment are excluded | No package vulnerability engine in the collectors |

## Proposed experiment defaults

| ID | Starting proposal | Why try it | What could change it | Responsible task |
|---|---|---|---|---|
| E01 | JSON snapshot/assessments/review; Markdown output | Small implementation, inspectable artifacts | Owner cannot comfortably edit or understand them | T00, T01, T08 |
| E02 | Explicit target ID and exact matching only | Avoid accidental cross-host/broad reuse | Identity entry is impractical or template reuse is required | T00, T04 |
| E03 | Three decisions plus independent applicability/change state | Keep human judgment distinct from stale context | Users cannot distinguish action needed, accepted, and re-review | T00, T04, T05 |
| E04 | Non-root offline review/report on a Linux workstation | Minimize privileged time and target interaction | Actual workflow needs target-side editing or other OS support | T06, T08 |
| E05 | Direct `/proc` network sources for current namespace | No external command dependency | Source validation exposes a correctness/coverage issue | T02 |
| E06 | Local sudo source inventory with a bounded supported grammar | Useful trial without pretending full policy evaluation | Unsupported cases dominate or mislead the trial | T03, T08 |
| E07 | Exact advance expectations; changed facts request re-review | Reuse context without blind suppression | Too much churn or inadequate protection against stale decisions | T04, T08 |
| E08 | Embedded static prompts, no network/AI service | Deterministic offline guidance | Trial shows a need for richer local interaction | T05, T08 |
| E09 | `collect`, `review`, `report` commands | Makes responsibilities explicit | Too many steps or confusing artifact movement | T06, T08 |
| E10 | `CGO_ENABLED=0`, initial Linux architecture chosen in T00 | Testable single-binary delivery | Required source access or actual hardware needs a different approach | T07 |

## Questions to resolve at a specific point

| When | Question | Evidence to collect |
|---|---|---|
| T00 / before live testing | Which Ubuntu release, architecture, and disposable target are available? | Owner/environment record; do not infer production access |
| P0 | Does the example communicate the difference between observation and explanation? | Owner describes the example in their own words |
| P1 | Which questions are helpful, confusing, or irrelevant? | Time spent and misunderstandings while reviewing a port |
| P1 repeat | What should trigger re-review, and are old explanations still easy to find? | Unchanged/changed/missing-source scenarios |
| P2 | Is local sudo configuration interpretation useful at the proposed depth? | Supported and unsupported rule walkthroughs |
| P1/P2 | Is the editable artifact also the deliverable, or should the report be regenerated? | Compare actual editing/report-sharing friction |
| P3 | Which findings beyond these examples should be added next? | Unanswered real inventory questions from the trial |
| Before any release planning | What coverage, safeguards, usability, and support level are required? | Separate release proposal; prototype completion is insufficient |

## Decision record template

```text
ID:
Status: proposed / trying / adopted-for-prototype / revised / deferred
Question:
Current hypothesis:
Trial or implementation evidence:
Owner feedback (actual words or faithful summary):
Decision and rationale:
Affected contracts/tasks/artifacts:
Compatibility or migration consequence:
Next review point:
```

Keep observations from trials separate from assumptions. An agent may record technical evidence but must not invent owner feedback or silently answer product questions for the owner. Routine implementation choices within an assigned task can be made and documented without pausing for approval.


## T00 baseline record — 2026-10-24

- Revision: `t00-2026-10-24.1`; prepared by an agent, owner review pending.
- Agreed constraints A01–A09 remain unchanged. E01–E10 all remain **proposed**; fictional fixture consistency is not user-trial evidence for adoption.
- Frozen for the first implementation handoff: strict 0.1 envelopes, explicit contract revision, canonical SHA-256 preimages, exact identity/expectation matching, conservative coverage gating, independent change/applicability/decision states, and explicit replacement of an active judgment after human reconfirmation.
- Evidence: [contract](data-contracts.md), [fictional fixtures](../../../test/fixtures/prototype/contracts/README.md), [three-level report](examples/fictional-report.md), and [P0 preparation](examples/p0-preparation.md). These are design artifacts, not collector output.
- Unsupported Defaults affects the fictional sudo rule's interpretation; it limits reuse of that rule as well as producing a separate item. A second complete-context fixture demonstrates sufficient sudo advance matching.
- Live trial target: not provided. Ubuntu release and architecture are **unknown**. Ubuntu 24.04/amd64 in fictional JSON is invented scenario metadata. Linux/amd64 remains a proposed build target only.
- Owner feedback: not yet obtained. No completed trial record or owner approval is claimed. Reopen the experimental contract with a new revision if P0 feedback changes semantics.
- Remaining questions: can the owner distinguish stale acceptance from current decisions; are static-content and external-firewall prompts useful; is conservative sudo re-review too frequent; which disposable Ubuntu environment is available?
- Next: owner P0 walkthrough using the prepared report, then T01–T06 consume the checked-in handoff. No release decision is implied.
