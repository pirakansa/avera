# Destination forwarding collection

Status: implemented from the owner's approved request to inventory host forwarding
settings, with loopback forwarding as the primary use case. Upstream router/cloud
protections are human decision reasons, never a reason to discard observations.

## Collection boundary

The Linux/amd64 CLI registers `network.WithForwarding`, combining the existing
socket collector with `internal/collect/forwarding`. The socket-only collector
remains available to existing callers. There are no new Go dependencies.

Read existing system commands sequentially, without a shell:

- `iptables-save` and `ip6tables-save` (the system-selected backend).
- `iptables-legacy-save` and `ip6tables-legacy-save` (the explicit legacy backend).
- `nft --json list ruleset` (the native nftables view).

Resolve executables only from `/usr/sbin`, `/usr/bin`, `/sbin`, `/bin`, using a
fixed locale/environment. Do not install tools, restore rules, invoke sudo, probe
remote hosts, start services or change firewall settings. Bound each command to
10 seconds and 128 KiB stdout; do not expose command stderr. Bound normalized
context to 128 KiB as well. Cancellation is fatal and prevents publication;
backend errors, invalid UTF-8, malformed/oversized output are recorded as incomplete
coverage with usable results preserved. There is no fallback from an error to an
absence claim. The existing aggregate artifact limit still applies.

Successful empty iptables output and an empty nftables array represent an empty
observation from that source; other missing backends still remain unknown.

Read the complete available iptables tables, retaining DNAT, REDIRECT and TPROXY
candidate rules (including custom chains and OUTPUT). Read nftables JSON and
retain objects containing `dnat`, `redirect` or `tproxy` statements, including
indirect/map destinations. Preserve all source context as evidence, including
chain hooks, conditions, jumps and filtering rules. Direct loopback IPv4/IPv6
translations are included. Non-loopback/ambiguous candidates remain visible to
avoid incomplete destination reasoning; there is no automatic path assertion.
Source order is preserved; views across backends are not deduplicated.

Strip iptables generated comment headers and chain packet/byte counters. Strip
nftables metainfo and packet/byte values in counter objects. Preserve all other
configuration. Hash the normalized full source context so changed filtering,
chain conditions, maps or destinations conservatively invalidate prior judgments.
Ruleset reads are not atomic. Dynamic state other than stripped counters may
conservatively cause additional re-review.

The combined network coverage uses the socket collector's conservative namespace
scope. It is complete only when sockets and all configured backend reads are
complete, failed when both components failed, otherwise partial. Missing command
variants can therefore produce exit 2 on an otherwise functioning nft-only host.
This is an explicit incomplete-inspection signal, not a claim that a tool must be
installed or a vulnerability exists. All attempted sources and diagnostics are
shown in the editor and reports.

## Artifact contract

New collected artifacts use revision `netfilter-2026-10-25.1` with schema version
`0.1`; the reader also accepts `t00-2026-10-24.1`. Old T00 findings, hashes and
fixtures retain their semantics. The new kind/evidence are rejected under the old
revision. New executables can carry old reports into new collection, preserving
judgments/history and migrating their envelope when needed. Old executables reject
the new revision. The forwarding increment initially used `avera.report.v2`; current saves use [v3 shared service policies](shared-service-judgments.md).

- Collector: `network`; kind: `network.forwarding`.
- Identity: existing source identity `{source_path, line, ordinal}`. Source paths
  are synthetic `/netfilter/<command>` anchors, `line` is a one-based candidate
  ordinal, and `ordinal` is zero. They are not filesystem paths or source lines.
- Facts: `{rule, context_digest, scope_id}`. Rule is observed configuration text
  (canonical JSON for nftables); context digest is `sha256:<hex>` over normalized
  complete source configuration. Scope is the shared observable network namespace.
- Evidence: `source_kind: command_output`, existing `{excerpt}` values; location is
  `command:<name>`. Excerpt retains bounded normalized complete source context.
- Existing target-scoped identity/fact hashing applies. Source position changes
  can change identity; scope/context changes change the fact fingerprint. No
  automatic remapping, reachability certification or advance expectation reuse.

An observed forwarding candidate is a main editable review item even without a
matching listener. It can own human-described relationships to loopback reference
services. Judgment fields, archived history, route bases, missing-service gating,
Japanese guidance and HTML export behave as for observed network entrances.
An explicit judgment can apply when coverage is complete and bases match; the
rule is observed configuration, not a verified effective path. Full context and
human safeguards remain distinguishable. New collectors do not silently mark any
candidate accepted or remove it because protection might exist upstream.

## Acceptance evidence

Synthetic source fixtures in `test/fixtures/prototype/forwarding` cover IPv4/IPv6,
custom chains, OUTPUT/REDIRECT, TPROXY, maps and non-loopback candidates.
Collector tests cover empty-success, missing/denied/malformed/oversized sources,
cancellation, counter stability and meaningful context changes. Network wrapper
tests cover shared scope and incomplete coverage. The portable report integration
test covers judgment, loopback linking, JSON roundtrip, escaped Japanese HTML,
rule-change re-review, history and rejection under the old revision.

No live firewall was changed or inspected for this implementation's automated
verification. Live Ubuntu/root backend coverage and owner usability remain trial
work. No reverse-proxy, upstream router/cloud, packet-filter simulation, runtime
reachability testing or general multi-hop graph is included.

## Source references

- [Netfilter nft manual](https://netfilter.org/projects/nftables/manpage.html)
- [Netfilter ruleset operations](https://wiki.nftables.org/wiki-nftables/index.php/Operations_at_ruleset_level)
- [Netfilter iptables backend migration](https://wiki.iptables.org/wiki-nftables/index.php/Moving_from_iptables_to_nftables)
