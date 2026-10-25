# Reusable service explanations and judgments

Status: implemented from owner clarification on 2026-10-25. The primary goal is
preparing and reusing known explanations, including across equivalent servers or
a load-balanced group. Strict per-installation identity and mandatory difference
review are secondary. This supersedes the earlier machine-binding requirement.

## Workflow and policy scope

`collect --input REPORT --output NEW_REPORT` accepts previously bound or unbound
reports on any installation without consulting `/etc/machine-id`. The inherited
`target_id` names the logical inventory/group, not a physical PC. An explicit
conflicting target label remains an input error; omit it to reuse the group's
label. `--adopt-target` remains a compatibility no-op with input. New collection
emits an empty legacy `machine_binding`; old binding hashes remain readable but
are not enforcement. No machine ID is synthesized or written to the system.

In a network endpoint's editor, the human can save its explanation and decision
as a common service policy. The checkbox is initially selected, with its matching
scope shown alongside it; unselect it for an individual judgment. Reusable policy
keys are protocol, address family, local port and binding class: all interfaces
versus a specific non-loopback address. Specific address values, namespace,
installation, executable and UID are intentionally not matching conditions.
Loopback references do not receive standalone policies. The current increment
supports shared endpoint judgments, not general sudo/forwarding policy templates.

The report itself carries these policies, so a user can prepare static HTTP
explanations using a fixture or an existing server's report before collecting the
next server. Matching observations receive the shared human decision without
reconfirmation for process/PC differences. A service with a different port,
protocol, address family or binding class does not match. Coverage diagnostics
remain visible even when a shared judgment applies. Its meaning is a configured
human policy, not a host verification or a declaration that missing sources are
safe. Starting collection without input creates an inventory without shared rules.

A separate individual human judgment on the same finding takes precedence; raw
assessment conflicts are not silently resolved. Individual judgments and their
fact bases remain unchanged. Existing judgments are not silently promoted to
shared policies: save with the common-setting checkbox to opt in. Raw T00
assessment reuse semantics remain unchanged. Portable review adds the contextual
reason `shared_service_judgment`; this is not a raw T00 review artifact reason.
Explicit human relationship-change gating still applies to saved manual links.

Shared explanations retain original author/time/selector/basis. Shared-policy
sources do not appear as unmatched judgments just because another namespace was
collected. Saving a replacement preserves existing human history; disabling the
checkbox removes the matching shared policy while recording the individual save.

## Portable contract

New saves use `avera.report.v3` with optional `policies`. Each policy records
`transport`, `family`, `port`, `binding` (`all_interfaces` or `specific_address`),
and the complete original `assessment`. Keys must be unique and agree with their
source endpoint selector; judgments use strict inventory validation. Inconsistent
content sharing an assessment ID is rejected. Existing v1/v2 inputs remain
readable and migrate only on a new save; policy fields require v3. Old executables
cannot read v3. Existing nested contract revisions and no-clobber/16 MiB limits
remain unchanged.

Policies are preserved on repeat collection and used consistently in browser
counts, explanations and exported HTML/Markdown. They are not applied to absent
services or represented as newly collected evidence. Legacy split artifacts do
not carry this portable policy extension.

## Forwarding interaction

For literal DNAT destinations, match explicit loopback address, port and TCP/UDP
protocol to observed local endpoints. iptables syntax requiring quoted/escaped
interpretation, REDIRECT, ranges, nftables maps, absent protocols and ambiguous
syntax remain manual. The report/editor displays configured destination matches
separately from human-described paths, without a required checkbox or duplicated
path explanation. Matching configuration is not proof of effective reachability.
When no unambiguous match is available, human relationship selection is optional
supplemental context. Upstream protections belong in safeguards and rationale.

## Validation

Unit/integration tests cover ignored machine identity, unbound/bound reuse,
policy save/disable/history, strict JSON roundtrip, changed process/namespace with
incomplete coverage, nonmatching service classes, and old-format rejection.
Literal destination tests cover TCP/UDP separation, missing ports/protocols,
REDIRECT and quoted text. Browser checks cover automatic destination display,
shared-policy editing, v3 export, human provenance and HTML presentation.
