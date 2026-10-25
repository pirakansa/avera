# Prototype implementation plan

Status: proposed execution plan for an unreleased prototype. T00 has an agent-prepared [contract baseline](data-contracts.md#t00-baseline-t00-2026-10-241) and [fictional walkthrough](examples/fictional-report.md). T01 implements the shared domain model, strict validation, normalization, and artifact storage; see the [API handoff](model-storage.md). Optional P0 feedback remains pending and does not block subsequent implementation. T06 provides tested `collect`, `review`, and `report` commands with T04 assessment reuse and T05 guidance; see the [CLI handoff](cli-workflow.md) and [combined T05/T06 rehearsal](../../user-guides/prototype-trials.md#executable-t05--t06-rehearsal). Owner P1 is pending; T07 delivery verification is a separate follow-up.

The owner subsequently requested a simpler single-report browser workflow. Use the [current guide](../../user-guides/report-workflow.md) and [container/server contract](report-workflow.md); the task sequence and original split-file examples below provide historical/domain context.

## Objective and authority

Build a small, usable inventory workflow with listening ports and sudo configuration as examples. The project owner will use it as the first test user to discover what guidance, editing, reuse, and reporting should become. Completing these collectors does not authorize a release or establish complete security coverage.

[Product direction](../product-direction.md) is the source for agreed product requirements. This plan proposes concrete defaults so different agents can implement compatible increments. Choices marked **proposed** may change through trial. An assigned implementation task authorizes work within that task; agents need not seek repeated permission for routine choices already covered by it.

Read [architecture](architecture.md), [contracts](data-contracts.md), [collection](collection.md), and [decisions](decisions.md) before implementing a dependent task. A contract change must be recorded and communicated to dependent tasks rather than made silently.

## Required outcomes

| ID | Outcome | Main tasks | Evidence of completion |
|---|---|---|---|
| R1 | Explain review items to an inexperienced user | T05, T08 | User can describe the item and the next question without decoding raw evidence alone |
| R2 | Separate observed facts from human explanations and decisions | T00, T01, T04 | Fixtures and report clearly distinguish their provenance |
| R3 | Inspect Ubuntu locally as root without adding a service, port, or runtime | T02, T03, T06, T07 | Controlled offline trial of the built binary |
| R4 | Make omissions and collection failures visible | T01, T02, T03, T05 | Empty-success and failed-collection scenarios render differently |
| R5 | Use previous inventory and advance assessments on the target | T01, T04, T06 | Repeat run carries applicable context and identifies decisions still needed |
| R6 | Present summary, individual explanations, and evidence | T05, T08 | Three connected report levels with understandable counts |
| R7 | Preserve rationale and detect changes requiring review | T04, T08 | Changed and unchanged fixtures produce different reuse outcomes |
| R8 | Learn from use before defining release scope | T08 | Recorded trial feedback and explicit next-increment decisions |

SBOM, CVE matching for the inspected machine, remediation, a resident agent, remote scanning, and a hosted application are outside this plan.

## Increments and trial checkpoints

| Increment | Deliverable | Tasks | What the owner tries | Decision after the trial |
|---|---|---|---|---|
| P0: optional concrete example | Fictional input, observations, review output, and report example | T00 | Identify what to check/write, what needs another look, and what the recipient can understand | Usability feedback when available; no prerequisite for T01 |
| P1: one complete workflow | Port collector + reuse + editable review artifact + Markdown report | T01, T02, T04, T05, T06, T07 (port slice) | First collection, edit, report, repeat collection using prior context | Editing friction, useful guidance, surprising reuse, report readability |
| P2: second example | Bounded sudo configuration collector integrated into the same workflow | T03, T04–T07 (sudo extension) | Explain a privilege rule, an unsupported rule, and a changed rule | Whether the common model works beyond ports; required parser coverage |
| P3: revise and repeat | Only changes selected from P1/P2 feedback | T08 and affected owners | Repeat the same scenarios after a targeted change | Keep, change, or defer each experiment; decide the next prototype increment |

Do not wait for a complete sudo policy engine before testing P1. Do not finish every candidate feature before showing the owner something usable. Report the limits of each increment explicitly.

P0 is an optional lightweight rehearsal. The prepared T00 contract/fixtures allow T01 to proceed without P0 answers, final UI choices, or a live target. Internal status names, exact matching predicates, hash details, and JSON mechanics remain implementation responsibilities; the owner evaluates the work the tool helps them do. Editing and presentation choices can be tried in P1. Feedback can reopen a proposed contract, with a version change and explicit fixture updates where needed.

## Assignable tasks

| Task | Responsibility | Prerequisites | Primary ownership |
|---|---|---|---|
| [T00](tasks/T00-contract-baseline.md) | Contract baseline and fictional walkthrough | Product direction and this plan | Specification examples and decision register |
| [T01](tasks/T01-model-storage.md) | Domain model, validation, artifact I/O | T00 contract baseline | `internal/inventory`, `internal/artifact` |
| [T02](tasks/T02-network-collector.md) | Linux network collection | T00; T01 interfaces for integration | `internal/collect/network` |
| [T03](tasks/T03-sudo-collector.md) | Bounded sudo configuration collection | T00; T01 interfaces for integration | `internal/collect/sudo` |
| [T04](tasks/T04-assessment-reuse.md) | Matching, change detection, assessment reuse | T01 | `internal/assessment` |
| [T05](tasks/T05-guidance-report.md) | Guidance and three-level Markdown output | T01; T04 result contract | `internal/guidance`, `internal/report` |
| [T06](tasks/T06-cli-workflow.md) | CLI and application orchestration | T01, T02, T04, T05 for P1 | `cmd/avera`, `internal/app`, `internal/cli` |
| [T07](tasks/T07-integration-build.md) | Single-binary build and integrated checks | T06 and available collectors | Build tasks, integration fixtures, reproducible smoke checks |
| [T08](tasks/T08-trial-feedback.md) | User trial coordination and next decisions | T00 for P0; T07 for executable trials | Trial records and decision updates |

### Suggested sequence for one agent at a time

T00 → T01 → T02 → T04 → T05 → T06 → T07 → P1 trial through T08 → T03 and affected integrations → P2 trial through T08.

The [T02 network collector API and synthetic fixtures](network-collector.md) are available for [T04 — assessment reuse](tasks/T04-assessment-reuse.md), using the [T01 APIs](model-storage.md) and baseline `t00-2026-10-24.1`. T02 is complete: official Linux v6.8 source verification and the disposable Ubuntu comparison passed; the handoff records the tested userland/kernel combination and its limitations. T00's optional owner walkthrough may happen alongside implementation or be deferred to an executable trial; it is not another approval step in this sequence.

T04 now provides the [pure assessment reuse API](assessment-reuse.md), deterministic review counts, editable current entries, and separately retained unmatched context. Its tests execute every T00 matching scenario; T05/T06 can consume the documented result directly.

### Optional overlap

After the T00 baseline, T02/T03 can research their sources and build collector fixtures while T01 implements the shared model. After T01, T02, T03, and T04 have separate ownership boundaries. T05 may develop against agreed review fixtures while T04 is implemented. T06 integrates completed components; it must not duplicate their logic to bypass dependencies.

No task requires parallel execution. If work overlaps, appoint one integration owner to serialize changes to `go.mod`, `go.sum`, shared contracts, and CLI registration. Each agent follows repository branch policy; the integration owner combines branches before the trial. Do not use a shared working tree for concurrent edits to the same files.

## Shared definition of done

1. The task's acceptance cases pass and its implementation limits are visible.
2. Modified behavior has meaningful tests using temporary files and fictional fixtures. Default tests do not inspect the developer's real sudo configuration or modify the host.
3. `vorbere run check`, `vorbere run test`, and `vorbere run build` have actually been run; report exact failures and limitations. Follow repository setup and cleanup rules.
4. User documentation reflects implemented behavior, while draft interfaces remain labeled as proposed until implemented and verified.
5. The handoff lists changed files, commands run, observed results, deferred work, and any shared-contract changes.
6. No release, publication, or real third-party report submission occurs as part of task completion.

Implemented baseline: the CLI builds, T01 has package-local model/storage tests, and the vulnerability scan finds no reachable vulnerabilities but reports findings in dependency packages/modules. Record new results accurately; do not silently classify this baseline as a newly introduced defect or as a completely clean dependency scan.

## Copyable agent assignment

> Implement task Txx from `docs/specifications/prototype/tasks/`. First read `AGENTS.md`, `docs/specifications/product-direction.md`, the prototype plan, and the specifications linked by the task. Check that prerequisites are implemented or that the agreed fixtures/interfaces are available. Treat prototype defaults as experiments, not release promises. Stay within the task's ownership boundary. Coordinate shared-file changes with the integration owner. Preserve collected facts and human assessments separately. Use fictional fixtures and temporary directories, run the required checks, and report the resulting behavior, limitations, and next handoff. Do not publish a release.

Forwarding configuration collection and its compatibility boundaries are defined in the [forwarding collector specification](forwarding-collector.md).
