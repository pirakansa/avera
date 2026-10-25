# Current T05/T06 trial status and session handoff

Status: implementation baseline complete; owner feedback/revision cycle in progress.
Final owner acceptance and release readiness have not been recorded. This is the
current entry point; dated feedback records describe earlier experiments and may
contain behavior superseded below.

## Current position

- T05: guidance and report baseline implemented/tested; Japanese wording,
  presentation and judgment usability are being refined with the owner.
- T06: CLI and browser workflow implemented/tested; the same feedback cycle covers
  collection, carrying one JSON, editing and report export.
- T08: recurring feedback work has started through actual owner discussion and
  requested revisions. Do not describe all P1 feedback as still unreceived.
- T07 delivery verification and live P2 sudo registration remain follow-up work.
  Local development builds/tests do not complete the distributed-build task.

## Decisions to preserve in a new session

| Owner direction | Current implemented behavior | Reference |
|---|---|---|
| Manual JSON preparation is too complex | One portable JSON; `collect` on target, `serve` on host; JSON/HTML export | [Workflow](../report-workflow.md) |
| Reports should be Japanese and graphical | Japanese editor/guidance and self-contained HTML; English documentation | [Guidance](../../specifications/prototype/guidance-report.md) |
| Make unfinished judgments prominent | Red action-required and amber unreviewed counts; accepted label is "追加対応不要" | [Workflow](../report-workflow.md) |
| Loopback services should not need duplicate standalone judgments | Reference grouping; network entrances own explanations; private/LAN listeners remain main items | [Scope](../../specifications/prototype/review-scope.md) |
| Inspect settings that forward to internal services | Read-only iptables/nftables candidates, evidence and explicit incomplete coverage | [Forwarding](../../specifications/prototype/forwarding-collector.md) |
| Reuse explanations across equivalent PCs/load-balanced groups; machine identity is unimportant | No machine-ID prerequisite; endpoint saves always update shared service judgments, with no opt-in checkbox | [Shared judgments](../../specifications/prototype/shared-service-judgments.md) |
| Avoid routine display of replaced decisions | Current accepted items remain visible; superseded judgments/relationships are JSON-only | [Retirement candidate RC01](../../specifications/prototype/retirement-candidates.md) |
| Use a diagram as a reporting/judgment aid | GUI/HTML diagram distinguishes configured destination matches from human relationships and unknowns | [Diagram](../../specifications/prototype/network-overview.md) |
| Upstream controls are not unfinished work for the tool | Diagram covers PC settings and invites upstream safeguards in the judgment rationale | [Diagram](../../specifications/prototype/network-overview.md) |

Shared endpoint matching uses protocol, IPv4/IPv6, port and all-interface versus
specific-address binding, intentionally ignoring PC, exact non-loopback address,
executable and UID. Use a separate report for a group needing different common
judgments. Current common settings support network endpoints; do not claim general
sudo/forwarding policy templates. Starting collection without input starts fresh.

## Reopen the prototype without root

From the repository root, use the installed supported Go toolchain, or allow Go to
select/download the required toolchain with `GOTOOLCHAIN=auto`:

```sh
GOTOOLCHAIN=auto vorbere run build
trial_dir=$(mktemp -d /tmp/avera-review.XXXXXX)
./bin/avera serve \
  --input test/fixtures/prototype/reports/fictional-report.json \
  --output-dir "$trial_dir"
```

Open the printed loopback URL. Stop an older `serve` before testing a rebuilt
binary; refreshing an old process does not replace its embedded assets. The
repository sample is fictional and includes socket/sudo examples, not the new
forwarding collector output. It can demonstrate Japanese guidance, current
judgments, shared endpoint saving, human links, the network diagram and HTML.
Actual forwarding observations require collection; developer browser tests also
create synthetic forwarding scenarios without changing firewall settings.

Keep the latest saved judgment JSON outside temporary storage before ending a
trial if it must survive cleanup/reboot. Neither `/tmp` samples, screenshots,
server processes nor previous tool sessions are required to continue. Repository
files and this record are the durable handoff. Inspect Git status before editing;
do not assume changes are uncommitted or reset someone else's work.

## Live collection and evidence limits

Follow the [primary workflow](../report-workflow.md). Collection requires root
and a new output path with a trusted parent owned by root/the effective user.
Create a `/tmp` collection directory as root, not as the ordinary host user.
Transfer the resulting private JSON into an ordinary-user-owned directory before
starting the host-side viewer. No installation binding/adoption is required.

New saves use portable v3; v1/v2 inputs remain readable. New live artifacts use
`netfilter-2026-10-25.1`, while T00 fixtures retain their original contract. Raw
split-artifact comparison tests are diagnostic compatibility cases, not the main
shared-setting workflow.

Agent verification has included check, race tests, build, legacy fixture/link
validation and Chromium tests for saving, export, diagram navigation, escaping,
hidden JSON-retained archives and narrow layouts. Local root collection succeeded
after removal of the machine-ID prerequisite. This is not a recorded complete
owner acceptance, T07 delivery verification, or full disposable-Ubuntu validation
of every forwarding backend. Sudo is still uncollected in the live CLI.

## Recipient report page revision

The owner requested a clearer reading boundary for report recipients. HTML now
opens on a report summary, with separate configuration-overview and
judgment/evidence pages inside the same offline file. The summary includes a
scoped conclusion, follow-up links and collection limitations; the diagram stays
available as background material. Printing includes every page. Owner acceptance
of this revised reading flow remains pending.

Validation for this revision: cached Go 1.26.10 with `GOTOOLCHAIN=auto` and
`GOPROXY=off` completed setup, race tests, build, fmt, vet and staticcheck.
`git diff --check` and both report-pages/shared-services Chromium checks passed.
The report-pages check opened a standalone local HTML with JavaScript disabled
and verified navigation, Back, keyboard, print and narrow layouts. The default
`check` could not complete the vulnerability scan with external access disabled;
a fresh database lookup remains pending network authorization. A trial using an
unavailable local database is not counted as vulnerability verification.


## Next conversation

Continue with concrete owner observations about wording, diagram usefulness,
judgment reuse or navigation. Read this record and the linked specification for
the affected behavior before changing it. Update the relevant docs and record
actual results; do not restore machine binding, opt-in shared saving or visible
archive sections merely because an older handoff describes them.
