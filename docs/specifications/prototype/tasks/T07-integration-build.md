# T07 — Single-binary build and integrated validation

Status: not started. Prerequisites: T06 and collectors implemented for the current increment. Read [architecture](../architecture.md) and the [trial guide](../../../user-guides/prototype-trials.md).

## Outcome

Give the owner a reproducibly built, locally runnable prototype plus an honest record of tested environments and known coverage limits.

## Ownership

Build-related changes in `vorbere.yaml`, integration fixtures/checks, and implemented development/trial instructions. Coordinate shared dependencies and command wiring with the integration owner. Do not publish binaries or create a release as part of this task.

## Work steps

1. Preserve existing `setup`, `check`, `test`, `build`, and `clean` entry points. Add or refine a prototype build task only when needed for the target and single-binary constraint.
2. Build the distributed executable with the proposed no-CGo configuration; embed required guidance/templates and a meaningful version/build label.
3. Inspect the produced executable using available development tools to verify architecture and dynamic/runtime dependencies. Record the actual result, not only environment variables used to build it.
4. Run fixture-based integration cases for first run, advance assessment, unchanged repeat, changed repeat, partial sources, stale decisions, and report counts.
5. Validate the executable in an explicitly selected disposable Ubuntu environment as root with outbound network unavailable, no SSH/service installation, no required external collector commands, and a minimal runtime search path.
6. Verify that saved-artifact review/report works without root. Do not infer Windows/macOS support from pure-Go code or cross-compilation alone.
7. Record OS/kernel/architecture, runtime conditions, collector limits, and all actual commands/results. If a live environment is unavailable, mark that validation pending while retaining successful fixture/build evidence.
8. Clean build/test artifacts according to repository policy after verification. Rebuild deliberately when preparing the owner's local trial.

## Validation matrix

| Scenario | Automated fixture coverage | Live/disposable validation |
|---|---|---|
| First run and repeat use | Required | At least one actual owner walkthrough |
| Advance assessment without prior snapshot | Required | Include in the P1 trial |
| Non-root collection rejection | Required via injected privilege boundary | Confirm when environment permits |
| Partial source visibility | Required | Record only if reproducible without unsafe host changes |
| No external runtime tools/network | Dependency-boundary tests | Execute under restricted conditions and record evidence |
| Read-only target behavior | Tests and implementation review | Observe controlled trial effects; acknowledge output-file writes |
| No-CGo standalone executable | Build verification | Execute the same artifact in the selected target environment |
| Sudo includes/unsupported constructs | Required at P2 | Disposable policy/reference environment only |

## Acceptance cases

- Required repository commands succeed, with vulnerability results reported accurately.
- Distributed binary runs in the tested target conditions without fetching templates, runtimes, or command utilities.
- No listener, resident process, remote connection, or configuration mutation is required by the tool.
- Current report and review counts agree across CLI and fixture scenarios.
- Smoke checks are reproducible and do not modify the user's actual sudo/firewall/service configuration.
- Trial instructions identify what is implemented, what is incomplete, and what remains untested.

## Handoff

Deliver a build recipe, artifact identity/checksum for the local trial, test evidence, known limitations, and exact trial invocation. Distinguish automated tests, live validation, and owner feedback; none substitutes for the other.
