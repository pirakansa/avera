# T06 — CLI and application workflow

Status: implementation and validation complete; owner P1 pending. Owner P1 will cover T05 and T06 together after completion. Prerequisites for P1 integration: T01, T02, T04, T05. Add T03 after the first trial. Read [architecture](../architecture.md).

## Outcome

Make the agreed fictional workflow executable: collect as root, supply prior context on the target, review saved results, and generate a report. Keep commands replaceable if the trial finds the workflow awkward.

## Ownership

- `cmd/avera/main.go` and `internal/cli/commands/<name>` for Cobra wiring.
- `internal/cli/shared` for shared CLI helpers.
- `internal/app` for use-case orchestration.
- Coordinate `go.mod`/`go.sum` changes with the integration owner; add a compatible pinned Cobra dependency when needed.

## Work steps

1. Implement the proposed commands/flags after checking T00's baseline. Keep command definitions thin and domain processing in existing packages.
2. Validate argument combinations, inputs, schema/target compatibility, and destination paths before privileged source reads or output publication.
3. Check root requirements only for live collection; saved-artifact processing must not accidentally read host state or require root.
4. For collection, invoke available collectors, combine their coverage, run T04 with optional prior/advance inputs, and publish a coherent output set.
5. For review/report, load only explicit files, invoke the same review semantics, and write new outputs. Do not trust arbitrary stored derived counts when they can be recomputed.
6. Implement cancellation and the proposed exit contract; source partial failures preserve usable output, fatal failures do not advertise a complete artifact set.
7. Produce concise stderr messages with output paths and next actions. Do not dump configuration or annotation contents in errors.
8. Add usage examples and one quick-start to the README only after these commands work. Put detailed implemented instructions in the trial/user guide rather than growing the README substantially.

## Acceptance cases

- A non-root collection attempt fails before host inspection and does not auto-invoke sudo.
- A root collection with prior snapshot and assessments produces current facts and a derived review queue; an advance-assessment-only invocation also works.
- Input/output path collisions and existing destinations fail without altering inputs.
- A partial collector produces persisted diagnostic results and exit code 2; an unreviewed item alone does not cause a command failure.
- Saved-file review/report succeeds without root, host-source access, external commands, or network access.
- Passing the same current snapshot, previous snapshot, and assessment context reproduces the same comparison across collect/review/report; omitted prior context is labeled as unavailable.
- Unsupported flags, invalid JSON, and incompatible target IDs produce actionable errors.
- Command tests use injected collectors and temporary directories; default tests do not need actual root access.
- Adding sudo later requires registration/integration work rather than duplicating the overall workflow.

## Handoff

Provide implemented command examples, exit/output behavior, fixture-based end-to-end evidence, and any changes to proposed flags. Update draft examples and decision entries to match the implemented prototype while preserving their experimental status.

## Implemented handoff — 2026-10-24

Implemented `internal/app` orchestration, thin Cobra command wiring, root-only network registration, private atomic artifact/report publication, cancellation and the 0/1/2 exit contract. No proposed flags changed. Cobra v1.10.2 was added for CLI dispatch using `go get`, followed by `go mod tidy`. See the [CLI reference](../cli-workflow.md) and [combined T05/T06 trial procedure](../../../user-guides/prototype-trials.md#executable-t05--t06-rehearsal).

Validation:

- Before changes: `vorbere run test` and `vorbere run build` passed with network disabled. Full initial check was blocked by the pending CLI dependency during implementation, then passed after it was installed.
- `vorbere run setup`, `vorbere run check`, `vorbere run test` (race detector), and `vorbere run build`: passed after implementation.
- The T00 fixture validator passed: 29 JSON files, 13 scenarios, hashes/history/counts and local links. `git diff --check` passed.
- App and command tests use injected collectors and temporary directories. They verify root/non-root behavior, prior and advance context, comparison reproduction, partial/all-failed collection, cancellation, collisions, invalid inputs and offline processing. Unknown-command handling initially returned success; the CLI test exposed it and the root handler was corrected.
- Actual binary rehearsal as UID 1000 passed: first review, six fictional report scenarios, edited human assessment, overwrite refusal, unsupported command/flag errors, and non-root collection refusal without output creation.
- Live integration exposed network evidence timestamps using subsecond precision although the artifact contract requires UTC seconds. The collector now emits contract precision; a fractional-clock regression test passes.
- Vulnerability scan: 0 reachable vulnerabilities; 1 finding in imported packages and 13 in required modules not apparently called by current code remain visible. No unrelated dependency upgrades were bundled.

The owner will try T05 guidance/reporting and T06 workflow together. No completed owner trial or live root collection is claimed. T07 delivery verification and P2 sudo registration remain follow-up work; the P1 binary intentionally registers network only and marks sudo uncollected. No contract revision or serialized field changes were needed.

## Owner-requested presentation follow-up

After this handoff, the owner requested Japanese support and a graphical HTML report. The report command now accepts optional `--format markdown|html` and `--lang en|ja`; existing invocations retain English Markdown. See the [presentation contract](../cli-workflow.md#p1-presentation-feedback-extension) and [feedback/trial record](../../../user-guides/prototype-trials/p1-report-presentation-feedback-2026-10-24.md). This extension does not change collection, review semantics or artifact schemas.
