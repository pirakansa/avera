# Prototype architecture and execution

Status: experimental architecture. T06 implements the commands below; see the [implemented CLI reference](cli-workflow.md). Other defaults remain proposals unless backed by recorded implementation evidence. See [the decision register](decisions.md).

## Runtime boundary

Build a Go executable for each supported Linux architecture with target-side runtime dependencies minimized. For the first validated build, propose `linux/amd64`; record the owner's Ubuntu release and architecture in T00 before claiming that environment is tested. Additional architectures require their own execution evidence.

Collection runs locally as root. Check effective privilege before opening host information sources. A non-root attempt fails with an actionable message; do not launch `sudo` automatically. Root does not imply unrestricted visibility in every namespace or mount environment.

Collectors read OS information directly. Do not shell out to `ss`, `netstat`, `sudo`, `visudo`, `getent`, or package managers as required production dependencies. Test/reference tools may be used in a disposable validation environment when explicitly available; they are not distributed-runtime requirements.

Collection opens no listening socket and performs no remote discovery, package installation, configuration change, or external lookup. Internal kernel queries, if selected later, must be distinguished from network traffic. Plan for cancellation and bounded work. The prototype is an observation at a point in time, not a continuous monitor or an atomic snapshot of the entire machine.

Propose `CGO_ENABLED=0` for the distributed build and verify the resulting binary rather than assuming it is self-contained. Keep development analysis tools out of the runtime path. Required templates and guidance should be compiled into the binary; source files may remain readable in the repository.

## Package responsibilities

| Package / location | Owns | Must not own |
|---|---|---|
| `cmd/avera/main.go` | Bootstrapping, process exit | Collection, matching, rendering |
| `internal/cli/commands/<name>` | Cobra definitions and argument mapping | Business rules or OS parsing |
| `internal/cli/shared` | CLI error and output helpers used across commands | Domain state or collector-specific parsing |
| `internal/app` | Use-case orchestration and dependency wiring | Linux format parsing or report templates |
| `internal/inventory` | Versioned domain types and domain validation | Filesystem access or Cobra imports |
| `internal/artifact` | Strict serialization, bounded reads, safe output creation | Deciding whether a risk is acceptable |
| `internal/collect/network` | Network source parsing and evidence | Human assessments |
| `internal/collect/sudo` | Sudo configuration traversal and supported parsing | Executing sudo or evaluating every policy backend |
| `internal/assessment` | Identity matching, changes, prior-assessment eligibility | Host access or UI |
| `internal/guidance` | Explanations, prompts, applicability guidance | Automatic safety certification |
| `internal/report` | Three-level Markdown rendering from a review model | Recollection or hidden decision changes |

Dependency direction: CLI → application → domain/collectors/artifacts/assessment/guidance/report. Collectors and pure processing packages may depend on domain types. Nothing outside the CLI command tree imports command packages. Do not create `internal/cli/manifest` unless a real manifest-specific responsibility appears; artifact handling belongs in `internal/artifact`. Do not create public `pkg/` APIs for this prototype without a concrete external consumer.

The integration owner adds and pins Cobra when CLI work begins, following repository dependency policy. Exact versions are implementation-time choices checked against the supported Go version. A compiled Go library dependency does not require the target user to install that library separately.

## Proposed command walkthrough

The commands and flags below are implemented for the T06 prototype and remain experimental pending P1 feedback. The T00 [fictional artifacts](../../../test/fixtures/prototype/contracts/README.md) and [report](examples/fictional-report.md) define the expected data relationships. `fictional-web-01` is an invented logical target, not a host to contact.

```sh
# First collection; run this command as root on the target.
avera collect --target-id fictional-web-01 --output ./run-001

# Produce an editable assessment starter and a review report on a workstation.
avera review --snapshot ./run-001/snapshot.json --output ./review-001

# After editing review-001/assessments.json, regenerate the report.
avera report --snapshot ./run-001/snapshot.json \
  --assessments ./review-001/assessments.json --output ./report-001.md

# Next collection uses both previous observations and previous human context.
avera collect --target-id fictional-web-01 \
  --previous ./run-001/snapshot.json \
  --assessments ./review-001/assessments.json --output ./run-002

# Keep the same comparison context when reporting the repeat run.
avera report --snapshot ./run-002/snapshot.json \
  --previous ./run-001/snapshot.json \
  --assessments ./run-002/assessments.json --output ./report-002.md

# Advance assessments can also be supplied without a previous snapshot.
avera collect --target-id fictional-web-01 \
  --assessments ./prepared-assessments.json --output ./run-003
```

For an offline fixture rehearsal, the implemented report command maps directly to the unchanged scenario:

```sh
avera report --snapshot ./test/fixtures/prototype/contracts/snapshot-repeat.json \
  --previous ./test/fixtures/prototype/contracts/snapshot-first.json \
  --assessments ./test/fixtures/prototype/contracts/assessments-reviewed.json \
  --output ./fictional-repeat-report.md
```

Its review semantics must equal `review-unchanged.json`. For explicit reconfirmation, save a new assessment input as shown in `assessments-reconfirmed.json`, then report against `snapshot-changed.json` and the same first snapshot. Keep the old assessment file. Passing both old and new entries as active judgments creates a conflict.

`collect` proposes an output directory containing `snapshot.json`, `review.json`, and an updated `assessments.json` starter. Inputs are never overwritten. With no prior context it still generates an unreviewed queue. `review` recomputes these derived review/assessment outputs from saved inputs without accessing the host. `report` renders the same review semantics to Markdown and never writes a decision back to a snapshot. See [contracts](data-contracts.md) for artifact roles.

Both `review` and `report` accept optional `--previous` and `--assessments` inputs. To reproduce collection's comparison, pass the same previous snapshot and equivalent assessment context. Without a previous snapshot, display comparison as unavailable rather than guessing changes from human notes. Record the source snapshot IDs used by each derived output.

At P1 only the network collector is active; reports explicitly mark sudo and other domains as not implemented in that increment. At P2 the normal collection invokes both implemented collectors. A choice of individual collectors may be added only if the trial needs it; do not silently equate a deliberately omitted collector with successful coverage.

Offline review/report on Linux should not require root. Windows/macOS execution and a GUI are deferred, even though the processing packages should avoid unnecessary Linux dependencies. This workstation workflow is a testable initial arrangement, not a settled product UI.

## Error and output contract

Initial exit meanings frozen for experimental contract `t00-2026-10-24.1` (implemented by T06):

| Code | Meaning | Artifact handling |
|---|---|---|
| 0 | Requested operation completed within its declared scope | May still contain unreviewed human decisions and declared scope exclusions |
| 1 | Invalid input, insufficient invocation privilege, output failure, or other fatal error | Do not advertise a complete result; keep prior files intact |
| 2 | Collection completed with partial or failed sources | Persist the usable result and coverage diagnostics; tell the user where it is |

Unreviewed risks do not themselves make execution fail. All collectors failing must not produce an empty-success result. Interruption must not leave a file that appears to be a complete artifact. Diagnostics go to stderr; generated artifacts are explicitly written to the chosen destination.

Create new output paths by default and refuse overwrite. Serialize to a temporary file in the destination filesystem, then publish without replacing existing input/output. Use private permissions for inventory artifacts (proposed directories `0700`, files `0600`). Handle symlinks and existing destinations deliberately, especially when collecting as root. Test creation and failure behavior; a temporary-file name alone does not prove safe publication.

For a multi-file run, stage the complete artifact set in a private temporary directory and publish it as one new run directory, or implement an equally explicit incomplete-run protocol. Readers must not mistake a half-written set for a complete run. T01/T06 own this shared boundary.

No command interprets annotation text or evidence references as shell commands or fetches referenced URLs. Report generation escapes untrusted content for its output format. Document default size/count/depth limits as named constants; hitting a limit produces a visible diagnostic, not silent truncation.
