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
| Before live testing | Which Ubuntu release, architecture, and disposable target are available? | Owner/environment record; not a prerequisite for T01 |
| Optional P0 / P1 | Does the owner know what to check/write, what needs another look, and what the recipient can understand? | Task-oriented feedback; no internal-state or artifact-format examination |
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

- Revision: `t00-2026-10-24.1`; prepared by an agent, available for implementation with optional owner usability feedback pending.
- Agreed constraints A01–A09 remain unchanged. E01–E10 all remain **proposed**; fictional fixture consistency is not user-trial evidence for adoption.
- Frozen for the first implementation handoff: strict 0.1 envelopes, explicit contract revision, canonical SHA-256 preimages, exact identity/expectation matching, conservative coverage gating, independent change/applicability/decision states, and explicit replacement of an active judgment after human reconfirmation.
- Evidence: [contract](data-contracts.md), [fictional fixtures](../../../test/fixtures/prototype/contracts/README.md), [three-level report](examples/fictional-report.md), and [P0 preparation](examples/p0-preparation.md). These are design artifacts, not collector output.
- Unsupported Defaults affects the fictional sudo rule's interpretation; it limits reuse of that rule as well as producing a separate item. A second complete-context fixture demonstrates sufficient sudo advance matching.
- Live trial target: not provided. Ubuntu release and architecture are **unknown**. Ubuntu 24.04/amd64 in fictional JSON is invented scenario metadata. Linux/amd64 remains a proposed build target only.
- Report usability feedback: not yet obtained. No completed trial record or owner validation of the report is claimed. Reopen the experimental contract with a new revision if later trial feedback changes semantics.
- Remaining questions: can the owner distinguish stale acceptance from current decisions; are static-content and external-firewall prompts useful; is conservative sudo re-review too frequent; which disposable Ubuntu environment is available?
- Next: T01 consumes the checked-in handoff now; T02–T06 follow their task dependencies. The optional P0 walkthrough may happen alongside implementation or be deferred to P1. No release decision is implied.

## Workflow clarification after T00

- The owner requested simpler P0 questions and intends to assign T01 next. The preparation material now asks about completing the inventory/reporting task rather than understanding implementation details.
- The three feedback questions concern what to check/write, retaining prior explanations and spotting re-review work, and what the recipient can understand.
- Unanswered P0 questions, final editing/presentation choices, and live-target details do not block T01. JSON and the internal state model remain experimental implementation defaults.
- This is a workflow decision, not a completed usability trial or adoption of E01–E10 as final product requirements. Contract revision `t00-2026-10-24.1`, serialized fields, normalization vectors, and scenario outcomes remain unchanged.

## T06 technical evidence — 2026-10-24

E01, E04, E08 and E09 are implemented for a prototype trial, not adopted as final product decisions. Cobra maps the proposed flags to shared application use cases; no proposed flags changed. Collection registers network only at P1 and marks sudo uncollected. Future sudo integration adds a registration and integration evidence. Saved-file processing recomputes T04 review and T05 Markdown without live dependencies. Output artifacts retain unmatched assessment context.

Fixture and injected-source tests exercise comparison, advance context, failure handling, privilege boundaries and safe publication. See the [CLI reference](cli-workflow.md) and [combined T05/T06 trial](../../user-guides/prototype-trials.md#executable-t05--t06-rehearsal). Owner feedback is pending and will be gathered in the combined P1 trial after T06. No trial completion or release readiness is claimed.

## P1 presentation feedback — 2026-10-24

The owner requested Japanese support and a graphical HTML output. E01/E08/E09 are revised for this experiment: preserve English Markdown as the default and add explicit `report --format html --lang ja` presentation options with bundled text/styles. This is an owner-requested trial extension, not a completed acceptance of its layout. JSON schemas and T04 rules are unchanged. Original human/evidence text is preserved. CLI translation, a browser editor, PDF, dashboards and scoring remain outside this revision. See the [feedback record](../../user-guides/prototype-trials/p1-report-presentation-feedback-2026-10-24.md) and [trial steps](../../user-guides/prototype-trials.md#japanese-html-presentation-trial).

## P1 single-report/editor revision — 2026-10-25

The owner found manual JSON preparation/editing too complex, especially across targets, and requested the one-report → target collect → host browser edit → JSON carry-back/HTML export flow. E01/E04/E09 are revised for this trial. The [portable report contract](report-workflow.md) defines a new `avera.report.v1` container, target installation binding, generated edit provenance, history retention and local-only HTTP boundaries. The [user guide](../../user-guides/report-workflow.md) is now the primary entry point; split artifacts remain import/diagnostic interfaces. This is an explicitly authorized workflow change, not a claim of completed owner acceptance. Go 1.26.10 is required to pass vulnerability scanning with the standard-library HTTP server. See the [feedback record](../../user-guides/prototype-trials/p1-single-report-workflow-feedback-2026-10-25.md).

## Entrance-focused review revision — 2026-10-25

The owner agreed to keep loopback services as reference information and perform judgment at network entrances, linking internal forwarding targets as human-described context. The [scope/path specification](review-scope.md) and [current user guide](../../user-guides/report-workflow.md#network-entrances-and-local-reference-services) define this revision. Non-loopback/private-LAN listeners remain review items; reference does not mean automatically accepted. Changed/missing linked observations gate entrance judgment reuse. New portable saves are v2, v1 inputs migrate without overwriting originals, and raw T00 artifacts remain unchanged. Automated fixture/browser evidence is not owner acceptance of the revised layout.

## Host forwarding inspection (2026-10-25)

The owner requested collection of iptables/nftables settings, primarily to expose
entrances forwarding to loopback references. Upstream firewall/router/cloud
restrictions belong in human judgment reasons; they must not suppress observed
configuration. Implemented as [destination translation candidates](forwarding-collector.md),
with conservative incomplete coverage, full source context, Japanese guidance and
human-selected relationships. No automatic reachability or upstream verification.
New collection uses a versioned nested contract extension; T00 inputs remain readable.

## Common service explanations over installation binding (2026-10-25)

The owner clarified that explanations should be prepared before inventory and
reused across equivalent servers, including a load-balanced group. Machine ID is
not an important boundary; strict per-run comparison must not force repeated
explanation of known services. Implemented [shared service judgments](shared-service-judgments.md)
and removed collection's identity prerequisite. Earlier machine-binding/adoption
decisions are superseded. Literal configured destinations are shown directly;
manual service selection remains optional for ambiguous paths.

### Common saving is unconditional

The owner removed the need to choose individual versus shared endpoint judgments.
Every endpoint save now updates the common service policy; the editor has no
reuse checkbox. Matching scope and human provenance remain visible. Older API
clients may send `reuse`, but it no longer selects a mode. Existing policy-free
files acquire a policy when their endpoint is saved.

## Superseded history display and possible retirement

The owner requested hiding replaced judgments from GUI/reports while leaving JSON
and collection unchanged. Superseded relationship entries follow the same display
boundary. Current accepted judgments remain visible. History storage is recorded
as [RC01](retirement-candidates.md#rc01--superseded-human-judgment-and-relationship-history),
not deleted or automatically migrated away.
