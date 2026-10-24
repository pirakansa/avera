# T06 — CLI and application workflow

Status: not started. Prerequisites for P1 integration: T01, T02, T04, T05. Add T03 after the first trial. Read [architecture](../architecture.md).

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
