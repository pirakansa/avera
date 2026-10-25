# P1 entrance scope feedback — 2026-10-25

Historical feedback record for its original increment. Later decisions supersede
some behavior below; use the [current trial handoff](current-status.md) for the
implemented workflow and ongoing owner feedback status.

Status: actual owner discussion and implementation request; evaluation of the revised UI remains pending.

The owner questioned why a loopback-only service required judgment in a security-concern inventory. They agreed to focus normal review on network entrances, keep local-only services as reference, and explain forwarding/protections at the entrance to avoid duplicated judgments. The owner explicitly requested this revision after discussion; no claim is made that loopback services are inherently safe or that LAN services are unreachable.

## Implemented experiment

- Shared scope projection: IPv4/IPv6 loopback endpoints are read-only reference information, excluded from main current-decision counts. LAN/private and wildcard endpoints remain main items. Observations, evidence and existing human judgments are retained. Raw T00 inventory/review artifacts remain compatible.
- The entrance form selects related observed local services and records a human-described path/protections with actor, time and observation bases. No firewall/proxy inspection or reachability discovery was added. Unobserved external ingress is outside this relationship model.
- Changed/missing linked observations gate derived entrance acceptance pending re-review. Explicit reconfirmation updates bases; replacement/removal archives old relationships. Repeat collection retains metadata without automatically remapping identities.
- Portable v2 saves include relationship metadata/history; v1 files are accepted and saved to new v2 files. Nested artifacts are unchanged. Older binaries cannot read v2; originals are not overwritten.

## Agent evidence

Before and after implementation, check, race tests and build passed. The vulnerability scan reported no vulnerabilities. Classification, raw/focused count distinction, immutable judgments, v1 migration, relationship roundtrip, changed/missing internal service gating, explicit reconfirmation, unlink history, and collection carry-back are covered by package tests. The original 29-JSON/13-scenario contract validator still passes; the generated Markdown example was synchronized with focused counts.

Actual Chromium checks passed: three main items plus one read-only reference in the fictional first snapshot; path selection and saved v2 JSON; escaped standalone HTML with internal evidence/related-service links; mobile overflow; ordinary judgment saves and retained drafts; downloads/preview; clean server shutdown. Browser helpers were updated to use stable finding IDs for locating the second edited item after rerender. These are agent-run fictional checks, not owner acceptance or a live root collection.

## Next owner trial

Open the existing portable fictional report with the normal `serve` command. Confirm that `127.0.0.1:9000` is reference information rather than an unreviewed main item. Open the HTTP entrance, select that local service, describe the hypothetical forwarding/protections, save, and export HTML. Check whether the recipient can distinguish observed sockets from a human-described route and whether duplicate explanation work has been reduced. See the [user guide](../report-workflow.md#network-entrances-and-local-reference-services) and [scope contract](../../specifications/prototype/review-scope.md).
