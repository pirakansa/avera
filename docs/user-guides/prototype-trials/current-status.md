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
`usb-2026-10-25.1`, while earlier supported fixtures retain their original contract. Raw
split-artifact comparison tests are diagnostic compatibility cases, not the main
shared-setting workflow.

Agent verification has included check, race tests, build, legacy fixture/link
validation and Chromium tests for saving, export, diagram navigation, escaping,
hidden JSON-retained archives and narrow layouts. Local root collection succeeded
after removal of the machine-ID prerequisite. This is not a recorded complete
owner acceptance, T07 delivery verification, or full disposable-Ubuntu validation
of every forwarding backend. The live CLI now collects bounded sudo configuration and local accounts/groups; actual effective sudo privileges remain unverified.

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

## Port-based service inventory revision

The owner selected listening-port-based service identification before user and
privilege inventory. The live CLI now enriches only processes holding observed
TCP listeners / bound-unconnected UDP sockets. Editor and reports distinguish
socket UID from process effective UID, show executable/process/cgroup service
candidates, and leave unknowns explicit. Use the existing purpose explanation to
record service function, necessity and responsible team. Loopback-reference scope,
shared judgment matching and endpoint hashes remain unchanged. New service
evidence uses the services contract revision; older reports remain readable.
See [listening services](../../specifications/prototype/listening-services.md).
Owner acceptance and live systemd validation remain pending.

Validation for the service revision: Go 1.26.10 setup, full race tests, build,
fmt, vet, staticcheck and `git diff --check` passed. The opt-in `networklive`
test package compiled with `-run '^$'`; no live/root comparison was executed.
Chromium checks for listening services, shared services and report pages passed,
including offline exports and narrow layouts. The initial staticcheck finding
(S1016 field conversion) was fixed before the final checks. `vorbere run check`
remains incomplete only at vulnerability database fetching because external
access was disabled pending authorization. No successful fresh vulnerability scan
is claimed. Generated build artifacts were cleaned after validation.

Owner feedback: PID is unnecessary for an inventory rather than monitoring.
The service inventory now omits PID in the editor and HTML/Markdown item fields;
PID/source locations remain in technical evidence and JSON. Service identification
and matching do not use PID.

## Local account / sudo / runtime-group revision

Owner-approved scope: local accounts/groups, configured sudo and runtime groups
for already observed listening services. Management sockets and broader daemon
inventory stay deferred. External identities, actual SSH login access and other
non-sudo privilege paths are stated as out of scope only: no extra findings,
unreviewed counts or mandatory follow-up. Actual in-scope collection failures
remain diagnostics. Account purpose fields capture function, necessity and owner;
read-only links connect services, local numeric UID matches and configured sudo
subjects. Runtime groups remain distinct from configured memberships. PID remains
outside routine display. New artifacts use the accounts revision; earlier known
contracts and portable files remain readable.

The live sudo integration exposed fractional-second evidence timestamps that did
not satisfy the inventory contract. Evidence now uses second-precision UTC;
the collect/edit/repeat integration test exercises the live-source adapters with
fictional temporary files. See [accounts and privileges](../../specifications/prototype/accounts-privileges.md).
Owner acceptance and live target trial remain pending.

Validation for the account/runtime-group revision: cached Go 1.26.10 setup,
full race tests, build, fmt, vet, staticcheck and `git diff --check` passed.
Chromium checks passed for accounts/sudo, listening services, shared services and
report pages. The account and listening-service checks were rerun after the final
legacy-file display changes. The optional `networklive` package compiled without
executing a live comparison. No host account/sudo configuration or SSH login was
inspected by these tests. The fresh vulnerability lookup in `vorbere run check`
could not complete while external access was disabled pending authorization;
no successful fresh vulnerability scan is claimed. Generated build artifacts
were cleaned after verification.

## Network-authorized final verification

After the owner authorized network access, `GOTOOLCHAIN=auto GOPROXY=off
vorbere run check` completed successfully against the official vulnerability
database and reported `No vulnerabilities found.` This completes the previously
pending vulnerability check for the accounts/runtime-group revision. Earlier
notes about disabled access describe those earlier attempts, not an outstanding
verification requirement. Owner acceptance and live-target trials remain separate.

## Narrowed account judgment scope

The owner clarified that individual account judgments should cover users of
main-scope listening services and UID 1000+ user-account candidates, not every
system account. Other low-UID accounts now remain separate read-only references,
excluded from current judgments/counts and explanation prompts. Observed socket
UID/GID or a sudo run-as name alone does not promote an account. UID 1000 is only
a candidate heuristic, not verified login access. Root is main-scope when a main
listening process actually has effective UID 0; otherwise a sudo run-as link can
open its reference. Source observations and existing judgments stay in JSON.

The owner's `/tmp/avera-review.AlwVXd/shared-services-demo.json` already has
appropriate fictional data: HTTP uses `websvc` UID 33, internal admin uses `alex`
UID 1001, and no observed service runs as root. The refreshed viewer therefore
moves root to references without altering that JSON; websvc, alex and old-maint
remain main account items. Restart serve after rebuilding; refresh alone does
not replace embedded assets.

Validation for the narrowed account scope: `vorbere run check`, full race tests,
build and `git diff --check` passed; the authorized vulnerability scan reported
`No vulnerabilities found.` Browser regressions for accounts/sudo, shared services
and report pages passed. A separate read-only check of the owner's exact demo
confirmed root as the sole account reference, websvc/alex/old-maint as main account
items, seven main items, three accepted, one action-required and three unreviewed.
The source JSON remained byte-for-byte unchanged, including root's stored prior
judgment. The rebuilt `bin/avera` is retained for the requested same-command demo.

## Connected storage revision

The owner requested connected external-storage inventory, including unmounted
candidates, resolved by a human purpose/reason such as an application's save
destination. The live CLI now reads bounded sysfs and mountinfo metadata for
USB/removable candidates, excludes rootfs backing disks and distinguishes
uncollected, complete-empty and incomplete observations. A concise storage form
asks for purpose, continued-connection reason, judgment and actor; it does not
require detailed permissions, contents scanning or unplugging/disable operations.
New files use the storage contract; earlier contracts and judgments remain readable.
See [external storage](../../specifications/prototype/external-storage.md).
Live hardware/owner acceptance remain separate from synthetic verification.

Storage verification: `vorbere run check`, full race tests, build and
`git diff --check` passed; the official vulnerability scan reported
`No vulnerabilities found.` Chromium regressions passed for external storage,
accounts/sudo, shared services and report pages. The storage check saved an
unmounted candidate as accepted using purpose, reason and actor only, preserved
immutable observations, and verified escaped offline HTML and narrow layouts.
Synthetic sources covered root/USB/stacked mappings and actual collection limits;
no real storage device was mounted, disabled or inspected.

The owner's `/tmp/avera-review.AlwVXd/shared-services-demo.json` now includes an
accepted application-save SSD and an unmounted transfer USB ready for judgment.
Its existing users, service policies and judgments were preserved. The pre-update
JSON is backed up in `/tmp/avera-storage-demo.djl4fqln/`. A durable entirely
fictional copy is `test/fixtures/prototype/reports/storage-demo.json`; the original
T00/v1 fixture remains unchanged. The rebuilt `bin/avera` is retained for the
same-command demo; restart serve to load both the new JSON and embedded assets.


## USB and physical overview increment

USB HID candidates now have a whole-device purpose/safeguards/reason workflow.
Hubs and other USB metadata provide reference topology. The editor and offline
HTML configuration page include a physical attachment tree and a human operating
boundary for manuals, permitted handlers and physical safeguards. These controls
are not verified or enforced. Reliably associated composite storage is judged in
the USB item; its changes invalidate the whole-device basis. Socket and daemon
expansion remains deferred.

Use `test/fixtures/prototype/reports/usb-demo.json` for a durable fictional trial.
It includes an accepted mouse with a fictional manual reference and a pending
HID/storage composite. Existing account/service/storage examples remain.
See the [contract](../../specifications/prototype/usb-physical-overview.md) and
[user workflow](../report-workflow.md#explain-usb-devices-and-physical-operation).
