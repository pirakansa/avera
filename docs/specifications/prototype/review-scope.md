# Main review scope and human-described service paths

Status: owner-approved workflow refinement implemented for the prototype. The owner wants judgments to explain network entrances and their forwarding paths, rather than requiring a separate judgment for every loopback-only listener.

## Scope

- Keep every collected endpoint and its evidence. A validated network endpoint bound to IPv4 loopback (`127.0.0.0/8`), IPv6 loopback (`::1`), or an IPv4-mapped loopback address is reference information by default.
- Wildcard, LAN/private, link-local and other non-loopback bindings remain main review items. Unrecognized addresses remain main items conservatively; no external reachability is inferred.
- Rendered reports and the browser exclude current reference endpoints from current/accepted/action-required/unreviewed/re-review/conflict counts. Reference count is separate. Historical and unmatched context remain separate counts. A reference is not automatically accepted or certified safe.
- Reference sections retain observations, evidence and existing human judgments but do not ask for another standalone judgment or permit reference-only judgment edits in this increment. Entrances own the explanation of forwarding to them. Collection and raw T00 review artifacts still inventory all findings; their original schema, matching tests and all-observation counts are unchanged. The presentation projection is explicit and shared.

## Human-described paths

In an observed non-loopback network entrance's form, select current loopback services and explain the path and protections. Each relationship records entrance/internal finding IDs, both observed fact fingerprints, explanation, actor, timestamp and source snapshot ID. The tool generates bases/timestamps and rejects self-links, duplicate targets, missing explanation/actor and non-reference destinations.

These relationships are human statements. The prototype collects firewall translation candidates as specified in [Forwarding collection](forwarding-collector.md), but does not inspect reverse-proxy configuration, traverse a network path or infer reachability. Observed main network endpoints and forwarding candidates can own links. An unobserved ingress on another firewall/router is outside this relationship model; absence of a main listener is not proof of no ingress. Local-only references and scope limitations remain visible.

A relationship is observation-matched only while both endpoints and saved fact fingerprints still match the current observations. This does not verify forwarding. A changed/missing linked service gates the entrance's derived decision to unreviewed with re-review required, even if the entrance itself is unchanged. A conflict remains a conflict. Stored judgments are not modified. The portable view includes `related_service_changed_or_unavailable`; this contextual reason is not serialized as a raw T00 review artifact.

Explicitly saving the entrance reconfirms the selected relationship bases and its human judgment. Clearing selections archives the old relationships. Superseded relationship history is retained. An omitted relationship field preserves existing metadata for callers updating only judgment text; the browser always sends an explicit selection. Missing or changed finding identities are not automatically remapped to new services.

## Portable file compatibility

New saves use `avera.report.v2`, extending the outer container with optional `routes` and `route_history` arrays. Existing `avera.report.v1` inputs are accepted and migrated in memory; original files are unchanged and new output is private/no-clobber. Older tool versions cannot read v2. The initial reference-scope increment retained nested T00 contracts; the subsequent [forwarding increment](forwarding-collector.md) introduces a new nested contract revision while preserving T00 input compatibility. The aggregate 16 MiB byte limit and strict JSON validation still apply; history is never silently trimmed.

Repeat collection preserves relationship metadata and history in the same target-bound portable JSON. Export, browser summary and browser preview use the same contextual review. Legacy split-file exports apply the reference-scope projection but have no path metadata to attach. A standalone exported HTML retains the human statements and local evidence/related-service links without scripts or fetched references.

## Evidence

Tests cover loopback/private/wildcard/unknown-address classification, count exclusion without decision mutation, v1 migration, related-state roundtrip, invalid destinations, stale-related-service gating, explicit reconfirmation and unlink history. Existing raw T00 tests remain unchanged. Browser tests exercise read-only reference grouping, entrance selection, saved portable JSON, escaped linked HTML, draft preservation and narrow layouts. Repository quality gates are required; owner evaluation of the refined workflow remains pending.
