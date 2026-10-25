# Implemented model and artifact storage (T01)

The internal Go APIs implement schema `0.1`, contract revision `t00-2026-10-24.1`, and the [T00 baseline](data-contracts.md#t00-baseline-t00-2026-10-241). No migration is performed. Unsupported versions, unknown or missing fields, null outside `previous_snapshot_id`, duplicate keys/IDs/set members, invalid Unicode, non-integer numbers, invalid enums/timestamps, incorrect hashes, and dangling references are errors. Human text and reference provenance are preserved; references are never fetched.

T01 owns the APIs below. Subsequent tasks have integrated network collection, assessment reuse, guidance/report rendering, and the CLI; see the [T06 CLI reference](cli-workflow.md) for current command behavior.

## Go API handoff

| Package/API | Responsibility |
|---|---|
| `internal/inventory` | Pure types and validation; no host or filesystem reads |
| `Snapshot`, `Assessments`, `Review` | Envelopes and typed findings, evidence, human statements, and review results |
| `EndpointIdentity`, `SourceIdentity` | Kind-specific identity payloads |
| `NetworkFacts`, `SudoFacts`, `UnparsedFacts` | Complete kind-specific observed facts |
| `ExpectedFacts` | Kind-restricted partial advance facts; pointers distinguish absent fields from explicit false/empty values |
| `CollectorResult` | Findings, evidence, and one coverage record; source failures belong in coverage/diagnostics, while the collector's separate returned error denotes a fatal application failure |
| `Decode([]byte)` | Strict parsing, shape checking, domain validation, and snapshot hash verification |
| `Encode(Artifact)` | Validated deterministic serialization without mutating the caller; two-space indentation and final LF |
| `Artifact.Header()` | Copy of the common version/kind/target envelope |
| `ValidateInputs(current, previous, assessments, review)` | Compatible envelopes/targets and supplied-input reference integrity; no assessment matching engine |
| `NormalizeIdentity`, `NormalizeFinding` | Shared versioned normalization; finding normalization computes both hashes |
| `Hashes` | Target-scoped finding ID and target-independent fingerprint |
| `MergeEndpoints` | Merge duplicate endpoint observations, union owners/evidence, and AND attribution completeness |
| `Canonical` | Exact constrained canonical JSON for hash preimages and fixture verification |
| `internal/artifact.Read` | Bounded regular-file reading followed by strict domain decoding |
| `internal/artifact.Write(ctx, path, artifact)` | Publish one new artifact without replacing an existing entry |
| `internal/artifact.WriteRun(ctx, path, artifacts)` | Publish a complete new directory of named artifacts with a common target |

T02/T03 should build typed observations and call `NormalizeFinding` or `MergeEndpoints` before publication. These observation helpers de-duplicate sets; input artifact decoding instead rejects duplicate set members. Supply IDs, timestamps, tool version, and target in the model from orchestration; the model generates no clock values or run IDs. `Encode` checks Go-created data with the same strict decoder used for input.

T04 should call `ValidateInputs` before matching and reuse the normalization/hash functions. A review passed to `ValidateInputs` requires its named current snapshot, previous snapshot if named, and assessment inputs. A reviewed assessment's historical basis is checked when that snapshot is supplied; absent historical bases are allowed. Structural review validation checks enums, links within the review, item/decision relationships, and summary counts. Computing or verifying matching outcomes is T04's responsibility.

T05/T06 should retain immutable named inputs alongside derived reviews. Call `ValidateInputs` with the explicit context before publication; storage does not discover or fetch omitted inputs. `WriteRun` validates each artifact and equal targets, but cannot validate links to historical inputs absent from its output map.

## Normalization and deterministic output

The exact preimages and participating fields remain those in the [baseline](data-contracts.md#exact-normalization-and-hash-preimages); T01 changes no contract revision or fixture hash. SHA-256 is rendered with lowercase hex and `sha256:`. No Unicode normalization or human-text trimming is applied.

IPv4 uses decimal octets without leading zeros. IPv6 uses lowercase hexadecimal groups, compressing the longest zero run of at least two groups, with the leftmost run winning ties; mapped IPv4 addresses retain hexadecimal IPv6 groups. Family, separate zone, and opaque scope remain distinct. Owners sort by UID then executable. Endpoint merging retains all evidence IDs and is complete only if every contributing observation is complete.

Sudo source anchors retain their absolute traversal path without resolving symlinks and reject `.`/`..` components. Run-as sets and command tags sort lexicographically; local members sort by UID, name, then membership. Commands and policy context retain their order. Only trailing source line terminators are removed from raw unsupported text and policy lines. Arguments, quoting, principal case, and traversal context remain significant.

`Encode` sorts findings/items by finding ID, evidence by evidence ID, coverage by collector, assessments by assessment ID, reference-ID arrays lexicographically, and the observed sets above. Coverage source/diagnostic/limitation sets and reason-code sets also sort for deterministic output. Partial advance expectations normalize only fields actually present. Text, evidence links, and human provenance survive round trips.

## Explicit resource limits

These exported constants in `internal/inventory` apply to input and output validation. An exceeded limit returns an error; no truncation or partial successful read occurs. T02/T03 must separately bound collection and represent source limits in coverage.

| Constant | Value | Rationale |
|---|---|---|
| `MaxArtifactBytes` | 16 MiB, including formatting | Bounds a local prototype artifact while leaving room for many bounded observations |
| `MaxStringBytes` | 256 KiB of decoded UTF-8 per string/key | Allows substantial human explanations and bounded excerpts without configuration dumps |
| `MaxEntries` | 100,000 per array/object | Bounds collection sizes independently of bytes; not a silent item filter |
| `MaxDepth` | 32 edges from the root | Exceeds every v0 structural nesting while bounding recursive decoding |
| `MaxNodes` | 1,000,000 values | Bounds aggregate parsing work across nested containers |
| `MaxInteger` | 9,007,199,254,740,991 (absolute value) | Exact JSON integer contract; UIDs, ports, lines, ordinals, and PIDs have additional domain bounds |

Boundary fixtures are generated in package-local tests to avoid checking in multi-megabyte example files. Byte, string, array, depth, and node boundaries are tested, as are exact integer bounds. Formatted output also obeys the byte limit so that a published artifact can be read back.

## Storage permissions and failure behavior

Artifact files have mode `0600`; staging/run directories have mode `0700`. Existing destinations, including symlinks, are refused. Input paths must be regular files and no path component may be a symlink. Every parent component is opened with `O_NOFOLLOW` and pinned by a directory descriptor. Root/sudo execution permits an explicitly selected ordinary-user-owned immediate parent; actual filesystem permissions still apply. Non-root callers retain the effective-user/root owner rule and sticky-bit requirement for group/other writes. Staging directories themselves are verified by the opened inode: directory type, effective writer UID, and no group/other access, before chmod or writes. The owner of a writable parent can rename staging entries; file publication uses the pinned private staging descriptor, and run-directory publication renames a child from inside that private staging descriptor. Neither relies on a replaceable source name in the output parent. Existing parent ownership/permissions are not changed; output files remain owned by the effective writer.

Both publication paths write and sync files inside a private temporary directory in the destination filesystem. Single-file publication uses a no-replacement hard link from that directory. Multi-file publication syncs the staged directory and uses `renameat2(RENAME_NOREPLACE)` to make the entire run visible. Concurrent publishers have at most one winner. Cancellation is checked during staging and immediately before publication. A cancellation arriving after that check may still publish a complete result.

Pre-publication errors remove staged files and preserve existing inputs/outputs. Abrupt process termination or power loss can leave a private `.avera-*` staging directory, which is not a complete output path. A directory-sync or cleanup error after publication can mean a complete destination already exists; inspect it and use a new path rather than retrying with overwrite. Parent-directory sync is requested for publication durability, but power-loss behavior depends on the filesystem.

Storage currently builds on Linux and requires hard-link support and, for runs, `renameat2` with `RENAME_NOREPLACE`. Unsupported filesystems fail without a replacing-rename fallback. The domain package remains platform-independent. No live Ubuntu host collection, filesystem fault injection, or Windows/macOS storage support is claimed by T01.

## Verification and fixture paths

`test/fixtures/prototype/contracts/` contains the unchanged T00 examples. The Go tests round-trip all 27 runtime artifacts, verify the four finding vectors plus the encoding vector in `normalization-vectors.json`, and validate relationships for all 13 entries in `scenarios.json`. The Python fixture validator remains an independent baseline check.

`internal/inventory/inventory_test.go` covers strict rejection, resource boundaries, stable incidental changes, meaningful facts changes, normalization, supplied-context validation, and exact human-text preservation. `internal/artifact/storage_linux_test.go` uses temporary directories for no-clobber, concurrency, symlink/FIFO rejection, unsafe parents, invalid output, cancellation before publication, permissions, bounded reads, and preservation checks. Tests need no root privileges and perform no host collection or network requests.
