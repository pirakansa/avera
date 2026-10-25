# Data contracts and assessment reuse

Status: experimental T00 baseline `t00-2026-10-24.1`, prepared by an agent and available for T01 implementation. Optional owner P0 feedback is pending and does not block that work. T01 implements this baseline; see the [model/storage API handoff](model-storage.md) for concrete limits and storage behavior. The baseline section below freezes serialization details for implementation handoff. Semantic changes require a contract revision and updated examples/tests.

The raw T00 contract below remains a regression baseline. The primary workflow
now uses a [portable v3 container](report-workflow.md), the [forwarding extension](forwarding-collector.md)
and [common service judgments](shared-service-judgments.md). In that workflow,
`target_id` labels a reusable service group across PCs; it does not enforce physical
machine identity. Do not apply the original per-run matching discussion as a
requirement to retype shared explanations or restore machine-ID binding.

## Separate roles, not necessarily permanent file formats

| Artifact | Role | Mutability |
|---|---|---|
| `snapshot.json` | Facts and collection coverage from one run | Immutable after collection |
| `assessments.json` | Human explanations, decisions, evidence references, and their applicability | Edited by the user; retain previous copies |
| `review.json` | Derived matching, change, reuse, and review-queue results | Regenerated from explicit inputs |
| Markdown report | Human-readable view of the same review result | Regenerated; not parsed as authoritative input in v0 |

JSON is proposed because it is straightforward to validate and exchange in Go. It is a prototype choice, not a conclusion about the best editing experience. Keep this separation in the domain model even if a later trial combines artifacts into one container or introduces another editor.

## Snapshot fields

| Field | Proposed meaning |
|---|---|
| `schema_version` | Exact supported serialization version, initially `0.1` |
| `kind` | `snapshot` |
| `snapshot_id` | Unique run identifier; independent of finding identity |
| `target_id` | Explicit user-supplied logical target, required for v0 |
| `collected_at`, `finished_at` | UTC timestamps; do not imply every source was read simultaneously |
| `tool_version` | Version/build label of the collector |
| `environment` | OS and kernel information, architecture, effective UID, observable execution/namespace scope |
| `coverage` | One record per requested or explicitly unavailable collection domain |
| `findings` | Typed observed review items |
| `evidence` | Bounded structured evidence entries referenced by findings |

An explicit target ID avoids silently treating a changed hostname as a new machine or reusing annotations across unrelated machines. It is an operator assertion, not hardware identity verification. Target mismatches are rejected for carry-forward. Cloned systems must receive intentionally chosen IDs; multi-host bulk matching is deferred.

### Coverage records

Each record includes collector ID/version, status, declared scope, attempted sources, diagnostic codes, and limitations. Proposed statuses are `complete`, `partial`, `failed`, and `not_collected`.

- `complete` means the collector finished within its stated scope, not that all security risks were inspected.
- `partial` includes missing process attribution, unreadable included policy, unsupported policy constructs, or a resource limit that limits the requested result.
- `failed` means the collector could not supply usable findings for its requested scope.
- `not_collected` means the domain was not attempted, for example because it is not implemented in P1.

Represent a successful empty scan explicitly. Do not infer successful coverage from an empty `findings` array. Completeness relevant to one item's comparison may be narrower than the collector's overall result; if v0 cannot establish that narrower coverage, use a conservative comparison outcome.

### Finding fields

Each finding includes `finding_id`, `collector`, `kind`, structured `identity`, structured `facts`, `fact_fingerprint`, and `evidence_ids`. Do not put user assessment fields into a finding.

Proposed kinds:

- `network.endpoint`: transport, address family, local address/zone, local port, observable namespace scope, socket category, process attribution and its completeness.
- `sudo.rule`: source location, principal/run-as/command information supported by the parser, options/tags where understood, and links to source evidence.
- `sudo.unparsed`: a relevant source construct whose meaning is not safely understood; never convert it into a verified grant or a harmless item.

Evidence entries carry source kind/location, collection time, method, and structured values or a minimal excerpt. An evidence ID links the review to the exact observation. Avoid complete configuration dumps when a bounded excerpt is sufficient. Do not collect process environments, command-line secrets, private keys, or password material as incidental evidence.

## Assessment fields

| Field | Proposed meaning |
|---|---|
| `schema_version`, `kind`, `target_id` | Versioned envelope; kind is `assessments` |
| `entries[].assessment_id` | Stable human-assessment identifier |
| `entries[].origin` | `reviewed_snapshot` or `advance` |
| `entries[].selector` | Exact target-scoped finding identity; no wildcard rules in v0 |
| `entries[].basis` | Snapshot ID and fact fingerprint, or an explicit advance expectation |
| `entries[].purpose` | What this configuration or capability is for |
| `entries[].exposure` | Intended access and operating context |
| `entries[].safeguards` | Reported controls, including external controls |
| `entries[].rationale` | Why the human chose the decision |
| `entries[].evidence_refs` | User-supplied references with provenance labels; not automatically fetched |
| `entries[].decision` | `unreviewed`, `action_required`, or `accepted` in the first experiment |
| `entries[].actor`, `entries[].assessed_at` | Who made the statement and when |

For a non-`unreviewed` decision, require nonempty actor, timestamp, and rationale as basic data completeness. These fields do not prove that the explanation is adequate. Show missing contextual explanations as prompts, without pretending that counting filled fields establishes safety.

Do not add an automatic `safe` decision. A separate `mitigated` decision may be tested later if users need it; initially mitigations are recorded as facts claimed by the assessor, with a decision about the remaining concern. Approval hierarchies, signatures, and expiry policies are deferred.

## Identity and meaningful changes

The T00 baseline below fixes identity/fingerprint definitions and vectors for these initial rules:

1. Namespace identity by target ID, collector, and kind. Encode structured values unambiguously; do not concatenate ambiguous IPv6 strings or delimiters.
2. Network identity uses protocol, family, normalized local endpoint, socket category, and declared network scope. Aggregate duplicate sockets with the same endpoint identity, retaining evidence and observed owners. Do not use PID, inode, or observation timestamp as stable identity.
3. A network fact fingerprint includes the relevant observed endpoint facts and sorted stable attribution available in the prototype (for example executable path and UID). PID churn alone is not a meaningful configuration change. Incomplete or newly changed attribution affects confidence in reuse. This is not executable-content verification or CVE analysis.
4. Sudo v0 uses a conservative source/rule anchor, including source path and location, plus a fingerprint of the supported normalized rule and its observed local group context. Moving a line may trigger re-review. Do not invent durable equivalence across complex policy rewrites.
5. Exclude incidental timestamps, evidence record ordering, and transient IDs from fingerprints. Preserve distinctions that change the meaning of a rule; do not normalize command arguments or quoting without proven semantic equivalence.
6. Version the normalization method. If semantics change, do not silently reuse assessments produced under the old rules.

An observation match is only a tool-level match. External firewall settings and other human assertions may have changed without being observable; show the original assessment date and provenance. Do not claim those controls were revalidated.

## Review outputs

Keep three independent concepts in `review.json`:

- Change: `new`, `unchanged`, `changed`, `not_observed`, `unknown`, or `not_compared`.
- Prior/advance assessment applicability: `none`, `applicable`, `needs_review`, or `conflict`.
- Human decision: the supplied decision, including its original basis, actor, and time.

An old `accepted` decision may remain visible while applicability is `needs_review`. It must not count as a currently accepted item. Derived counts must come from the same reviewed item set used by the detail renderer.

The T00 baseline defines explicit reconfirmation: the user reviews changed facts and records a new judgment against the current basis, preserving the old input artifact. Merely regenerating an assessment starter must never update an old acceptance's basis to current facts. Include source snapshot IDs in derived review metadata so repeat reports can reproduce the comparison.

### Matching and carry-forward matrix

| Situation | Proposed output |
|---|---|
| First run, no input context | `not_compared`, `none`; item awaits a human decision |
| Prior snapshot has no matching identity | `new`; assessment absent unless an exact advance entry applies |
| Identity and meaningful facts match; needed coverage is sufficient | `unchanged`; preserve applicable explanation and its original date |
| Same identity, relevant facts changed | `changed`; preserve old explanation as context, mark `needs_review` |
| Previous item missing and comparable current scope completed | `not_observed`; retain history, do not call it automatically remediated |
| Previous item missing but scope changed or collection incomplete | `unknown`; no inference that the item disappeared |
| Multiple input assessments match one item | `conflict`; require human resolution, do not pick the last file entry |
| Target or schema mismatches | Reject input before collection/output publication; explain which input is incompatible |
| Advance entry with exact identity and sufficient explicit expectations that match | Mark origin `advance`; proposed applicability is `applicable`, not tool verification |
| Advance entry expectations fail, are incomplete, or cannot be evaluated | Preserve as context and mark `needs_review`; no silent acceptance |
| Supplied assessment has no current match | Retain it as unmatched context, outside current finding counts |

Advance expectations must be structured and evaluated only against observed supported fields. The T00 baseline specifies required fields per collector and an intentionally insufficient expectation. Do not support executable expressions, broad suppressions, or pattern-based allowlists in v0.

Input assessments without a previous snapshot can still be checked against their stored basis fingerprint or advance expectations. Change classification is `not_compared` if no previous observation is available. The T00 scenarios include this case because supplying advance judgments on the target is a required workflow.

## Validation and evolution

Reject malformed JSON, duplicate object keys, duplicate IDs, unsupported versions, unknown structural fields, invalid enum values, invalid timestamps, dangling evidence references, and inconsistent target relationships with actionable diagnostics. Bound input bytes, entry counts, string sizes, and nesting as appropriate; select and document concrete defaults in T01.

Do not discard unknown data and save a superficially valid truncated file. During the prototype, unsupported schema changes may require explicit regeneration rather than a migration engine. Keep old artifacts intact and explain the required version.

T00 provides complete, machine-parseable examples for first run, advance assessment, unchanged repeat, changed repeat, missing source, unmatched assessment, and conflict. T01/T04 turn those examples into meaningful validation and matching tests. Never fabricate an observation just to make an annotation match.

## T00 baseline: `t00-2026-10-24.1`

Prepared by an agent on 2026-10-24 for implementation handoff; **ready for T01, with optional owner usability feedback pending**. This freezes one experimental serialization/normalization revision for T01–T06, not the final product choices E01–E10. The definitions below resolve the earlier proposed details where they overlap. All artifacts in the [fixture directory](../../../test/fixtures/prototype/contracts/README.md) are fictional; none were collected from a host. The [illustrative report](examples/fictional-report.md) uses those exact artifacts.

### Envelopes and common validation

All listed fields are required unless explicitly marked optional. An empty array/string is a value, not an omitted field. `null` is permitted only for `review.previous_snapshot_id`. Objects reject unknown keys at every structural level, duplicate keys, unsupported versions, invalid enums/types, dangling references, and duplicate IDs. Reject before publishing outputs; never discard unknown fields and resave. Recognize exactly `schema_version: "0.1"` and `contract_revision: "t00-2026-10-24.1"`; normalization changes require a new contract revision even if schema shape is unchanged. No implicit migration or old-hash reuse.

Every runtime envelope has `schema_version`, `contract_revision`, `kind`, and `target_id`. Kind is exactly `snapshot`, `assessments`, or `review`. All linked inputs must have equal target/version/revision. Target IDs are case-sensitive, nonempty opaque Unicode strings; do not trim them or infer hardware identity. IDs are nonempty strings unique within their artifact and ID category; snapshot IDs identify immutable runs. All strings must be valid Unicode scalar sequences (reject unpaired surrogates). Numbers are integers only, within the exact JSON integer range −9007199254740991 through 9007199254740991; individual fields impose narrower bounds. T01 selects concrete byte/count/depth limits.

Timestamps are exactly `YYYY-MM-DDTHH:MM:SSZ`, valid Gregorian UTC dates, seconds 00–59, no fractions or offsets. `finished_at >= collected_at`. Assessment dates are human assertions and may precede collection. Fixture `fictional-*` labels are examples, not reserved runtime values.

Serialization: UTF-8 without BOM, two-space indentation and final LF for artifacts. Object key presentation order is irrelevant. Sort findings/review items by `finding_id`, evidence by `evidence_id`, coverage by `collector`, assessment entries and reference-ID arrays lexicographically by ID. Use Unicode scalar lexicographic order (equivalently UTF-8 byte order for valid strings). Duplicate members of set-like arrays are invalid. Preserve human text exactly, including whitespace; no normalization of rationale or URLs. URLs/references are never fetched. JSON files `scenarios.json` and `normalization-vectors.json` are test metadata, not runtime artifact kinds.

### Snapshot payload (complete field inventory)

In addition to the common envelope:

| Object | Fields / types |
|---|---|
| snapshot | `snapshot_id`, `collected_at`, `finished_at`, `tool_version`: strings; `environment`: object; `coverage`, `findings`, `evidence`: arrays |
| environment | `os_id`, `os_version`, `architecture`, `kernel_release`, `execution_scope`: strings; `effective_uid`: unsigned 32-bit integer |
| coverage record | `collector`: `network` or `sudo`; `collector_version`, `scope`: strings; `status`: `complete`, `partial`, `failed`, `not_collected`; `attempted_sources`, `diagnostic_codes`, `limitations`: string arrays |
| finding | `finding_id`, `collector`, `kind`, `fact_fingerprint`: strings; `identity`, `facts`: kind-specific objects below; `evidence_ids`: nonempty ID array |
| evidence | `evidence_id`, `source_kind`, `location`, `collected_at`, `method`: strings; `values`: typed object below |

There is one coverage record per requested domain, with explicit records for unavailable prototype domains. `collector` must agree with finding kind. `source_kind` is `proc` or `local_file` in this baseline. Network evidence values contain `observed` (the network facts type), and optional `pid` (positive integer). Local-file evidence values contain only `excerpt` (bounded string). Each finding evidence ID resolves within the same snapshot. Evidence and finding values are observations, never human assertions. `diagnostic_codes` are stable nonempty machine-readable strings, extensible without changing matching semantics; examples use `unsupported_defaults` and `source_unreadable`. Limitations contain human-readable explanations.

| Kind | Identity | Facts |
|---|---|---|
| `network.endpoint` | `transport`: `tcp`/`udp`; `family`: `ipv4`/`ipv6`; `local_address`, `zone`, `scope_id`: strings; `local_port`: integer 0–65535; `socket_category`: `listener` (TCP) or `bound_unconnected` (UDP) | `owners`: array of `{executable: string, uid: uint32}`; `attribution_complete`: boolean |
| `sudo.rule` | `source_path`: absolute string; `line`: positive integer; `ordinal`: nonnegative integer distinguishing constructs on the same logical line | `principal`: `{type: user/group, name: string}`; `host`: string; `run_as_users`, `run_as_groups`: string arrays; `commands`: ordered array of `{path: string, arguments: string, tags: string array}`; `local_members`: array of `{name: string, uid: uint32, membership: primary/supplementary}`; `local_context_complete`: boolean; `policy_context`: ordered string array; `interpretation_complete`: boolean |
| `sudo.unparsed` | Same source anchor as `sudo.rule` | `raw`, `diagnostic_code`: strings |

`executable` may be empty only when attribution is incomplete. Unknown owners remain incomplete, never an empty complete owner set for an observed endpoint. `scope_id` is an opaque comparable namespace incarnation identifier established by the collector (for example boot identity plus namespace identity); descriptive labels alone must not imply comparability. The fixture scope is an explicitly asserted fictional comparable scope. When comparability cannot be established, use a new scope identity and classify missing prior observations as `unknown`, even if the new namespace scan completed.

Sudo facts describe a configured rule, not an effective grant. The only baseline command tag is `PASSWD` or `NOPASSWD`; unsupported semantics become `sudo.unparsed` and restrict interpretation. `arguments: ""` means no explicit argument restriction in the source; it does **not** mean the command can only run with no arguments. Preserve any nonempty argument text exactly; unsupported argument semantics must not yield `interpretation_complete: true`. `policy_context` retains all relevant logical policy lines (including unsupported ones) in traversal order, so rule ordering, include context, and policy edits invalidate the fingerprint. T03 must include relevant include directives as context and mark completeness false when dependency/context is missing or unsupported. Local membership includes primary and supplementary membership; no remote lookup. For an explicit user principal, include that resolved local user in `local_members` with its actual local membership context; do not infer other users.

### Exact normalization and hash preimages

Two algorithms are frozen: `avera.identity.v1` and `avera.facts.v1`. Hash inputs use a constrained canonical JSON encoding: recursively sort ASCII object keys; no whitespace; UTF-8 Unicode scalar strings with no Unicode normalization; escape quote and backslash, use `\b`, `\t`, `\n`, `\f`, `\r` for those control characters and lowercase `\u00xx` for other U+0000–001F; emit all other characters literally (including `/` and non-ASCII). Booleans are `true`/`false`; integers use minimal decimal spelling, zero as `0`. No floats, BOM, or terminal newline in a preimage. Object keys in these typed preimages are ASCII. Hash with SHA-256; render lowercase hex prefixed by `sha256:`.

- Finding ID preimage: `["avera.identity.v1", target_id, collector, kind, identity]`.
- Fact fingerprint preimage: `["avera.facts.v1", kind, identity, facts]`.
- The fingerprint is target-independent; eligibility still requires an exact target-scoped finding match. Neither a matching hash nor an operator-provided target ID verifies a machine's identity.

Before encoding, normalize endpoints: IPv4 is four decimal octets without leading zeros; IPv6 is lowercase hexadecimal, no leading zeroes in a group, compress the longest run of at least two zero groups (leftmost on a tie), and use hexadecimal groups even for IPv4-mapped addresses. Never merge IPv4 and IPv6. Zone is a separate string, empty if absent; never infer a zone. Preserve opaque scope strings exactly. Merge duplicate endpoint identities and union owners/evidence. Sort owners by numeric UID then executable string; remove duplicate observed owners. Resulting attribution is complete only if all contributing socket owners and requested attribution sources were resolved.

For sudo, retain the absolute source path used by traversal without resolving symlinks into an equivalence claim; reject unresolved `.`/`..` path components. Keep the source line (first physical line of a logical construct) and ordinal. Sort run-as sets and tags lexicographically; sort local members by numeric UID, name, then membership. De-duplicate observed set members before emission. Preserve command order, argument quoting/spacing, raw unsupported text, and policy traversal order. Remove source line terminators only; do not otherwise rewrite unsupported text or argue semantic equivalence. Principals/names are case-sensitive. T03's parser subset remains its responsibility.

Every typed identity and facts field participates in its respective hash. PIDs, inode numbers, evidence order/IDs, snapshot ID, tool version, collection times, and explanations are outside the preimages. UID/executable changes, attribution completeness, local membership, raw policy changes, and policy completeness participate. PID-only repeat evidence is deliberately different with identical finding hashes in `snapshot-repeat.json`. The four finding [fixed vectors](../../../test/fixtures/prototype/contracts/normalization-vectors.json) include literal canonical preimages and expected digests; An additional encoding vector covers non-ASCII, HTML characters, a Unicode line separator, escapes, integers, and booleans. T01 must reproduce these vectors rather than copying generated IDs into implementation logic.

### Assessment payload and advance expectations

The assessments envelope adds `entries`. Every entry contains `assessment_id`, `origin`, `selector`, `basis`, `purpose`, `exposure`, `safeguards`, `rationale`, `evidence_refs`, `decision`, `actor`, `assessed_at`. Text fields are strings; `assessed_at` may be empty only for `unreviewed`, otherwise use the timestamp format above. Non-unreviewed entries require nonblank actor/rationale/date. Context text may be empty and generate prompts rather than automatic acceptance. Decision is exactly `unreviewed`, `action_required`, or `accepted`.

`selector` is `{collector, kind, identity}` using the exact finding types above; target is inherited from the envelope. `origin` is `reviewed_snapshot` or `advance`:

- `reviewed_snapshot` basis has exactly `snapshot_id`, `fact_fingerprint`. The original snapshot need not be supplied to compare the stored fingerprint. If supplied and referenced by ID, its finding must agree with the basis. A missing historical snapshot is not fabricated.
- `advance` basis has exactly `expectation: {normalization_version, facts}` with normalization version `avera.facts.v1`. `facts` is a subset of fields of the selected kind, typed as above. A subset is valid input but insufficient for automatic applicability unless it includes **all** facts fields, including completeness and context. Unknown fields/versions are errors; missing expectations are valid context requiring review. Equality is exact after the same typed normalization, not subset matching of arrays or wildcard matching.

`evidence_refs` is an array of `{reference: string, provenance: "human_reported"}`. References are external human context; never add them to snapshot evidence. Even when applicable, a reference or safeguard remains unverified by the tool.

Sufficient network advance expectations require exact normalized identity plus the full owner set and `attribution_complete: true`; coverage and namespace comparability must also be sufficient. Sufficient sudo expectations require exact source anchor, the full normalized rule, members and policy context, and both completeness flags true with complete relevant sudo coverage. Unsupported constructs cannot be accepted automatically through an advance entry. `assessments-advance.json` contains a sufficient port entry and a sufficient sudo entry for `snapshot-supported-sudo.json`; that same sudo entry needs review against the first snapshot's unsupported context. `assessments-insufficient.json` intentionally expects no facts: exact port identity alone never makes it applicable.

### Review payload, history, and counts

The review envelope adds these required fields:

| Field | Type / meaning |
|---|---|
| `snapshot_id` | Current immutable snapshot ID |
| `previous_snapshot_id` | Previous immutable snapshot ID, or null |
| `assessment_ids` | Sorted IDs of all supplied assessments, including unmatched ones |
| `coverage_statuses` | Object with requested collector keys (`network`, `sudo`) and coverage status values; exactly mirrors current snapshot |
| `items` | Current and historical items defined below |
| `unmatched_assessment_ids` | Supplied assessment IDs with no current exact identity match |
| `summary` | Integer counts defined below |

Each item has `finding_id`, `presence` (`current`/`historical`), `change`, `applicability`, `assessment_ids`, `current_decision`, `reason_codes`. Change/applicability enums are those listed earlier. Review items link to current findings or (for historical items) the previous snapshot; assessment IDs link to the explicit assessment input, preserving its actor/date/text. Reviews are not self-contained archives: retain the named immutable inputs. There is no generated timestamp in a deterministic review.

Priority is explicit:

1. Reject incompatible envelopes and invalid references before matching. Match target-scoped identities only.
2. Without a previous snapshot, change is `not_compared`, including when a stored basis or advance expectation applies. With a previous snapshot: identity absent means `new`; differing fingerprint means `changed`; equal fingerprint means `unchanged` only with sufficient comparable coverage, otherwise `unknown`.
3. Sufficient network coverage means complete relevant network collection/attribution and comparable scope. Sufficient sudo coverage means complete relevant collection, local context, and interpretation. The baseline conservatively uses the entire domain's coverage when narrower completeness is unavailable. `sudo.unparsed` is never sufficient for reuse. Coverage failure does not erase observed items.
4. Multiple exact assessment matches always mean `conflict` with `multiple_assessments`, regardless of dates or decisions. Do not select the newest. One match is `applicable` only with sufficient coverage and an equal stored fingerprint or sufficient equal advance expectation; otherwise `needs_review`. No matches means `none`.
5. Reasons for one inapplicable match: `coverage_incomplete` takes precedence; otherwise `facts_changed` for stored bases or `expectation_mismatch_or_incomplete` for advance entries. `current_decision` equals the sole applicable assessment's decision; otherwise it is `unreviewed`. Original non-current decisions remain visible through assessment links, never relabeled as current acceptance.
6. Missing previous identities are historical. Complete comparable current domain scope means `not_observed`; incomplete/mismatched scope means `unknown`. Historical items have `needs_review`, `unreviewed`, and reason `not_observed` or `coverage_incomplete`. Their assessment IDs are empty; their prior explanations remain accessible through `unmatched_assessment_ids` and original inputs. They are never counted as current or automatically remediated.

Summary fields: `current`, `historical`, `accepted`, `action_required`, `unreviewed`, `needs_review`, `conflict`, `unmatched_assessments`. The three decision counts partition current items exactly; needs-review/conflict are independent applicability subsets of current items, not additional decisions. Unmatched count is assessment entries, not findings. A fully failed scan has zero current items plus failed coverage, potentially historical unknown items; an empty complete scan has complete coverage plus historical not-observed items. Neither statement establishes safety.

`assessments-reviewed.json` remains unchanged on repeat. On changed facts, an editable starter must preserve its old bases and rationales. The human creates `assessments-reconfirmed.json` as a new input: replace `a-http` in the active set with `a-http-reconfirmed`, explicitly referencing `fictional-s3` and its new fingerprint. The old input remains intact. The new decision is `action_required` after reviewing UID 0; it is applicable even though change relative to s1 remains `changed`. Keeping both assessments active would create a conflict, not a history ordering rule. Never rebase by merely regenerating a starter.

### Implementation handoff and remaining questions

The [scenario manifest](../../../test/fixtures/prototype/contracts/scenarios.json) names explicit inputs and expected reviews for first run, advance without prior snapshot, sufficient sudo advance, insufficient expectation, unchanged PID-only repeat, changed owner/group context, human reconfirmation, partial sources, total failure, complete empty coverage, unmatched annotations, conflicting annotations, and stored basis without prior snapshot. T01 owns strict validation and the hash vectors; T02/T03 own source fidelity and coverage; T04 owns scenario matching; T05 owns common counts, prompts, and report traceability; T06 owns CLI input preservation and output atomicity. This checked-in handoff is the notification to those future task owners; no external messages were sent.

Still pending for later trials: owner usability feedback, available disposable Ubuntu release/architecture before live testing, whether JSON editing is practical, and whether the conservative sudo/context churn is acceptable. These questions do not block T01's fixture-based model and storage work. Proposed first build remains Linux/amd64, not a tested environment. The [P0 preparation](examples/p0-preparation.md) separates optional owner feedback from technical implementation responsibilities. No live collection, parser, or commands were implemented by T00.

## Forwarding extension

New live collections use `netfilter-2026-10-25.1`. See the [forwarding contract](forwarding-collector.md#artifact-contract) for the additional kind and command evidence. Original T00 fixtures retain their baseline revision and remain accepted without rewriting.

## Listening-service evidence extension

New live artifacts use `services-2026-10-25.1`, adding an optional structured
`service` observation to PID-qualified proc evidence. This revision preserves
endpoint identity/fact normalization and the forwarding contract. The T00 and
netfilter revisions remain accepted, including mixed known revisions during
repeat collection, without rewriting old immutable snapshots or assessment bases.
Service metadata is forbidden under those earlier revisions. See the
[listening-service contract](listening-services.md#storage-and-compatibility)
for exact fields, unknown values and presentation boundaries.

## Accounts and runtime-group extension

New live artifacts use `accounts-2026-10-25.1`, adding `accounts/account.local`
findings, sanitized `local_identity` group-directory evidence and optional runtime
group fields in proc service evidence. Earlier T00, netfilter and services
revisions remain readable without rewriting immutable snapshots. Account
identities use exact local names; changed memberships affect account facts.
See [accounts and privileges](accounts-privileges.md#contract-compatibility) for
field types, normalization, scope boundaries and known-revision rules.

## Connected storage extension

`storage-2026-10-25.1` adds `storage/storage.external` findings with typed identity,
capacity/connection/read-only facts and partition/mount observations. Earlier
accounts, services, netfilter and T00 contracts remain readable. Account/runtime
group evidence remains permitted under the accounts and storage revisions.
Existing network/sudo/account hash behavior is unchanged. See the
[storage contract](external-storage.md#contract) for exact fields and normalization.
