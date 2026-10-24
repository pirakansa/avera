# T01 — Domain model, validation, and artifact storage

Status: not started. Prerequisite: T00 baseline. Read [data contracts](../data-contracts.md) and [architecture](../architecture.md).

## Outcome

Implement the shared language used by collectors, reuse logic, CLI, and reports, with artifact I/O that preserves prior work and rejects ambiguous input.

## Ownership

- `internal/inventory`: snapshot, finding, evidence, coverage, assessment, and review types; validation; identity/fingerprint normalization.
- `internal/artifact`: versioned JSON reading/writing and output publication.
- Package-local tests and shared contract fixtures coordinated with T00.

Do not put host reads, Cobra commands, matching policy, or Markdown rendering in these packages.

## Work steps

1. Implement only v0 fields in the contract baseline; use typed enums and finding payloads rather than unchecked maps for semantic fields.
2. Centralize identity and fingerprint functions so both collectors and reuse use the same versioned rules.
3. Define a small collector result contract: findings, evidence, coverage, and a clear distinction between source failures and fatal application failures.
4. Implement strict JSON input validation, including duplicate keys/IDs, unknown fields, sizes, target consistency, and reference integrity. Standard JSON decoding alone must not silently accept conflicting duplicate keys.
5. Select named size/count/depth limits with rationale and fixtures for boundary cases. Do not make arbitrary hidden limits a data-loss mechanism.
6. Implement private output creation and no-clobber publication. Refuse existing destinations and protect input files. Test symlink and interruption/failure paths relevant to root execution.
7. Expose deterministic serialization for tests and review; keep run IDs/times injectable instead of embedding nondeterminism throughout the model.
8. Document the supported schema/version behavior and examples actually accepted by the code.

## Acceptance cases

- All T00 valid artifacts round-trip without losing human text, evidence links, or provenance.
- Unsupported versions and invalid/unknown structural fields fail before publishing output.
- Duplicate IDs/keys, mismatched targets, and dangling references produce specific errors.
- An existing assessment input and an existing output file remain byte-for-byte intact after a failed write attempt.
- Partial writes are not exposed as complete artifacts; output permissions match the documented policy.
- Fingerprints are stable under ordering/timestamp/PID-only changes where the contract says they are incidental, and change under meaningful fields.
- Package tests run without root and use temporary directories only.

## Handoff

Provide Go APIs and fixture paths to T02–T06. Record exact normalization semantics, I/O limits, and known portability constraints. Shared-type changes after this handoff require coordinated fixture and consumer updates.
