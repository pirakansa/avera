# Data contracts and assessment reuse

Status: proposed v0 contract. T00 produces complete example artifacts and resolves the listed serialization details before dependent agents implement them. Changes after that baseline require a contract revision and updated examples/tests.

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

T00 owns the final identity/fingerprint definitions and examples. The proposed rules are:

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

T00 must define an explicit reconfirmation example: the user reviews changed facts and records a new judgment against the current basis, preserving the old input artifact. Merely regenerating an assessment starter must never update an old acceptance's basis to current facts. Include source snapshot IDs in derived review metadata so repeat reports can reproduce the comparison.

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

Advance expectations must be structured and evaluated only against observed supported fields. In T00 specify required fields per collector and at least one example of an intentionally insufficient expectation. Do not support executable expressions, broad suppressions, or pattern-based allowlists in v0.

Input assessments without a previous snapshot can still be checked against their stored basis fingerprint or advance expectations. Change classification is `not_compared` if no previous observation is available. T00 must include this case explicitly because supplying advance judgments on the target is a required workflow.

## Validation and evolution

Reject malformed JSON, duplicate object keys, duplicate IDs, unsupported versions, unknown structural fields, invalid enum values, invalid timestamps, dangling evidence references, and inconsistent target relationships with actionable diagnostics. Bound input bytes, entry counts, string sizes, and nesting as appropriate; select and document concrete defaults in T01.

Do not discard unknown data and save a superficially valid truncated file. During the prototype, unsupported schema changes may require explicit regeneration rather than a migration engine. Keep old artifacts intact and explain the required version.

T00 must provide complete, machine-parseable examples for first run, advance assessment, unchanged repeat, changed repeat, missing source, unmatched assessment, and conflict. T01/T04 turn those examples into meaningful validation and matching tests. Never fabricate an observation just to make an annotation match.
