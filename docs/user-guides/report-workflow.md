# Collect, review and carry one report

This is the primary prototype workflow after the owner's single-file/editor feedback. The browser editor is Japanese; JSON schemas, specifications and CLI diagnostics remain English. Reports preserve original evidence and human text.

## Build

Use Go 1.26.10 or later and Vorbere. If your installed Go is older and automatic toolchain downloads are allowed, prefix development commands with `GOTOOLCHAIN=auto`.

```sh
vorbere run setup
vorbere run build
```

Go 1.26.10 is the minimum because the local Web server makes standard-library vulnerabilities in the previous toolchain reachable. No browser, Python, npm or other runtime package is installed on the target by AVERA. The viewer's CSS/JavaScript and guidance are bundled in the executable.

## First collection on the target

Run in an explicitly opened root shell on the selected Linux/amd64 target. Choose an existing private output directory owned by root and a new JSON filename:

```sh
./avera collect --target-id my-server --output /root/avera-run-001.json
```

For a `/tmp` collection directory, run `mktemp -d /tmp/avera-collect.XXXXXX` in that root shell. A directory created by the ordinary user is rejected as a root collection parent, even with mode `0700`. Output paths must be new and have no symlink components.

The target name is optional on the first collection; omission generates an ID. Subsequent collection inherits the name from the report. AVERA does not invoke sudo, collect remotely or change target configuration. Live collection reads listening sockets and their service users/runtime groups, existing iptables/nftables configuration, local passwd/group settings, bounded local sudo configuration, and connected USB/removable non-root storage metadata, and current USB device/interface metadata. AVERA does not install missing firewall tools. Distribution/kernel metadata not inspected by the prototype remains unknown.

The JSON contains observed facts, previous context when available, editable judgments and superseded human judgment history. Exit 0 means successful collection; exit 2 means results were saved with incomplete/failed sources. Exit 1 means invalid input, a conflicting logical group label, missing privilege or a fatal/output failure. Inspect incomplete coverage in the viewer rather than treating missing observations as absence.

## Review on the host PC

Transfer just the report JSON through your existing chosen method to a private directory owned by your ordinary host user. AVERA does not transfer files or relax permissions. Start the local viewer as that ordinary user:

```sh
mkdir -m 700 ./review-saves
./bin/avera serve --input ./avera-run-001.json --output-dir ./review-saves
```

The overview emphasizes nonzero **action required** counts in red and **unreviewed** counts in amber, with larger numbers and explicit action/review text cues. The editor, preview and exported HTML share this treatment. Zero counts keep the neutral style. These are priorities for human follow-up, not a vulnerability severity score or proof that an unreviewed item is unsafe.

Open the exact `http://127.0.0.1:PORT` URL printed in the terminal. No other machine can connect; use the exact printed address rather than substituting a hostname. Leave this terminal running and stop with Ctrl+C when finished. The listener uses a free local port by default; `--listen 127.0.0.1:8080` selects a specific loopback port. Public addresses are refused. This is a local single-user workspace, not an authenticated multi-user Web service; other local processes can access its loopback endpoint.

1. Open a current item. Read the Japanese observation summary, questions, and earlier explanations. Technical observation fields are read-only.
2. Fill purpose, audience/exposure, safeguards, rationale, actor and optional evidence references. Pick unreviewed, action required or accepted. No selector, fingerprint, target ID, origin or timestamp needs manual editing.
3. Click the save button. The server creates a new `report-TIMESTAMP-RANDOM.json` in `review-saves` and shows its filename. The original input and earlier saves remain intact. Every save includes all previously saved judgments, not just that item. A failed save preserves the current session and leaves the form available for retry. Drafts in other items survive a successful save of one item; drafts are lost if the browser is closed/reloaded, with a navigation warning.
4. Use the report-preview tab to inspect the saved state. Use **HTML output** to download a standalone Japanese report for a recipient. It requires no server or network when opened later.
5. Use **Take JSON** to download the latest saved report as `avera-report.json`, or carry the newest filename displayed after saving. Downloads exclude unsaved fields and warn when drafts remain. If several tabs edit the same session, stale saves fail; retain your text and reload before retrying.

The Japanese display label for the stored `accepted` decision is now "追加対応不要" (no additional action required according to the human reviewer). This wording is used consistently in summary cards, judgment selectors, saved judgment displays and Japanese HTML/Markdown reports. It is a human judgment within the observed scope, not a tool certification of safety. JSON still stores `accepted`; existing saved files and matching rules are unchanged.

Save records the current observation basis, actor/time and, for network endpoints, updates the common service judgment. Earlier human entries are archived verbatim; generated empty starters need not be archived. Common endpoint judgments can apply even when network collection is incomplete; coverage and diagnostic information remain visible independently. Raw fact-based decisions, conflicts or stale manual relationships can still require review. A configured judgment is not verification of missing evidence. Use the unmatched-judgment section to inspect explanations without a current observation. Replaced judgment history is not displayed. The exported HTML includes active prior context and current review state; superseded archived entries remain in the portable JSON rather than becoming extra active judgments.

The preview and exported HTML open on **報告サマリー** (Report summary). A recipient
can read this page for the scoped conclusion, target and collection time, review
counts, required follow-up and collection limitations. Unfinished judgments and
zero main-scope observations are stated explicitly; the conclusion does not
certify overall safety or complete collection.

Use **構成の概要** (Configuration overview) for the network diagram and
**判断・根拠の詳細** (Judgments and evidence) for individual explanations, run
metadata and technical evidence. Diagram and follow-up links open the relevant
detail page. These pages remain within one offline HTML file. Browser Back works
with page navigation; printing includes all three pages. Markdown stays textual.

`--lang en` selects English report questions and export content; editor controls remain Japanese in this prototype.

## Network entrances and local reference services

The main review list focuses on non-loopback network entrances and other review domains. Local-only endpoints such as `127.0.0.1:9000` and `::1` appear under **local-only reference information**, outside the main unreviewed count. LAN/private and wildcard listeners remain review items. This is a scope choice, not an automatic additional-action-unnecessary decision. Existing reference judgments remain visible as saved context; new standalone reference judgments are not requested.

To describe forwarding, open the observed network entrance, select the relevant local service(s), and fill the forwarding/protection explanation. Save the entrance judgment. The tool records the association as a human statement, not a verified firewall/proxy route. Follow its related-service link in the report to inspect the observed local endpoint. If the linked service's facts change or it disappears, the entrance requests re-review even if its own listener is unchanged. Reconfirm after checking the path, or clear the selection if it no longer applies; earlier relationships stay in JSON history and are not displayed.

Only observed entrances can own these links. The collector reads local forwarding candidates but does not discover ingress on a separate firewall/router. No main listener observed does not mean no external ingress. Reference and coverage sections preserve that distinction.

New saves use portable report v3; old v1/v2 reports still open. Keep original files; older AVERA binaries cannot read v3. The raw split-artifact review JSON retains its all-observation counts for compatibility, while the browser and rendered reports show the focused counts described here.

## Repeat on the same target

Bring the latest judgment-containing JSON back to the same target. Run as root and choose a new output filename:

```sh
./avera collect --input ./avera-report.json --output ./avera-run-002.json
```

You do not need to re-enter the target ID or provide separate snapshot/assessment files. Then bring `avera-run-002.json` back to the host and open it with `serve` again. Shared endpoint judgments apply by service class across PC/process differences, independently of collection completeness. Raw fact-based context and stale manual relationships retain their review rules. Inspect coverage alongside the current human judgment.

The input report can be reused across servers with the same role, including a load-balanced group. AVERA no longer reads or requires `machine-id`; existing binding fields are ignored for reuse. Omit `--target-id` to inherit the logical group name. `--adopt-target` is no longer required.

## Prepare common explanations before collection

Saving a network endpoint's form always saves its purpose, safeguards and decision as common settings for matching services. No checkbox or individual/shared mode selection is needed. Matching uses protocol, IPv4/IPv6, port and all-interface versus specific-address binding. It intentionally ignores the particular PC, specific non-loopback address, executable and UID. Only reuse the input report for the service group for which those explanations apply.

For example, open a report containing TCP/IPv4 port 80 on all interfaces (the fictional sample includes this), describe the static Web service and upstream controls, select **追加対応不要**, and save. Carry that JSON into `collect --input` on the intended server(s). Matching HTTP endpoints receive the common human judgment rather than asking for the same explanation again. Different ports or waiting/binding classes remain separate. For an older report without common settings, save the endpoint once to make its judgment reusable. The current common-setting editor supports network endpoints.

The editor and exported report identify a common-setting judgment as such; collection omissions remain visible independently. This is a human policy, not verification of safety on each server. Starting without `--input` begins with no shared settings. New saved files use v3, with v1/v2 inputs accepted.

## Try without root or a target

Open the prepared single-file fictional report directly:

```sh
trial_dir=$(mktemp -d /tmp/avera-single-report.XXXXXX)
./bin/avera serve \
  --input ./test/fixtures/prototype/reports/fictional-report.json \
  --output-dir "$trial_dir"
```

No manual JSON preparation or combination of snapshot files is required for this rehearsal. The portable fixture contains a changed observation and an older human explanation so you can try explicit reconfirmation. For migration of your existing legacy files only, use `review --single-report --snapshot CURRENT --previous PREVIOUS --assessments HUMAN_CONTEXT --output NEW_REPORT.json`.

These imported fixtures are fictional and unbound. They are for offline trial, not evidence about your computer. To reuse a real legacy report, pass it to `collect --input`; no installation binding or adoption step is required. Save common service judgments in the editor when explanations should apply across service instances.

Export saved reports without starting the viewer:

```sh
./bin/avera report --input ./avera-report.json \
  --format html --lang ja --output ./avera-report.html
```

For diagnostic compatibility, `collect --legacy-output` still writes the earlier three-file run directory, and legacy `review`/`report --snapshot ...` inputs remain supported. Do not mix `--input` with split context flags. The new single JSON is versioned separately and cannot be passed as a legacy `--snapshot` or `--assessments` file.

## Trial feedback

Check whether you can complete the collect → carry → edit → export → carry-back cycle without understanding JSON. Report confusing Japanese, unclear save/download behavior, difficulty finding old explanations, or uncertainty about which file to carry. The specification describes what the prototype implements; automated checks are not owner acceptance of this workflow.

## Inspect forwarding configuration

The same `collect` command now also reads destination translation candidates from
iptables (including the explicit legacy backend), IPv6 iptables and nftables.
No extra flag or firewall configuration change is needed. Existing system tools
are used when available; missing tools, permission errors, malformed output and
limits make network coverage incomplete (exit 2), with usable results still saved.
A missing backend never means that no forwarding exists. Inspect the collection
scope and diagnostics before making a decision.

In the editor, expand a **転送設定** item. Its observed rule and linked source evidence
retain ingress conditions and surrounding chain configuration. Literal DNAT destinations are matched to observed loopback services when their
address, port and protocol are explicit. These matches need no manual checkbox.
For ambiguous or indirect paths, optionally select the related service and explain
the relationship. Record protection by upstream routers or
cloud controls in the safeguards/reason fields. Then choose the human decision,
including **追加対応不要** when justified. Upstream safeguards do not remove the
observed setting from the inventory. Export HTML to include the rule, evidence,
judgment and human relationship explanation.

The collector retains destination translation candidates even when they are not
literal loopback addresses: REDIRECT, maps and chain traversal can require human
interpretation. A candidate is not proof that external traffic can reach a service.
The tool does not simulate packet filtering, inspect upstream devices or collect
reverse-proxy configuration. The iptables compatibility and native nftables views
can duplicate a setting; they remain labeled by source rather than being merged
on an unproven equivalence. Their counts describe review items, not unique holes.

Counters do not trigger re-review. Configuration changes anywhere in a candidate's
source context do, conservatively. Missing or changed linked services also require
re-review. Live namespace incarnation is still not proven across runs, so existing
conservative namespace re-review rules continue to apply.

New collected artifacts use contract revision `usb-2026-10-25.1` inside the
v3 portable container with reusable service policies. Existing T00 artifacts and v1/v2 reports remain
readable, including as previous collection input; original files are not modified.
Older executables cannot read the new nested contract revision.

Superseded judgment/relationship history is retained only in JSON, not in the
GUI, report preview or exported HTML/Markdown. Current accepted items and their
reasons remain visible. History storage itself is a [retirement candidate](../specifications/prototype/retirement-candidates.md#rc01--superseded-human-judgment-and-relationship-history), not removed in this revision.

## Read the network diagram

The editor and exported HTML include **ネットワーク構成の概要**. It shows observed
listeners/forwarding rules and their related local services within the currently
observed PC. Solid arrows identify literal configured destination matches; dashed
arrows identify human-described relationships. An unspecified relationship is not
proof that no forwarding exists. The diagram covers settings within the PC. Its upstream note invites you to
supplement the judgment rationale with safeguards provided by routers, load
balancers or cloud services. Those controls support your explanation; the note
does not indicate unfinished work for the tool.

Cards show current judgment state and available purpose/safeguards, with human
provenance. Select a card's title to open its full item; the editor expands the
matching judgment form. After saving, the diagram updates. Use HTML export to
share the same diagram offline or print it. Markdown remains textual. The diagram
is an overview with explicit abbreviation/size limits; see the [display contract](../specifications/prototype/network-overview.md).

## Identify the service behind a listening port

New live collections include a **待受サービスの棚卸し** (listening service inventory)
inside each endpoint's details. Use the observed process name, executable,
numeric process effective UID and systemd cgroup service candidate to identify
what holds that port. Multiple holders remain separate. The socket UID is a
separate observation and can differ from the process effective UID.

The systemd name is a candidate from the observed cgroup, not a verified service
or socket-activation target. Unknown/uncollected values remain explicit. Existing
JSON files still open, but need a new collection to acquire the added metadata.
The tool inspects only processes holding observed sockets; it does not inventory
all daemons or users.

For a main endpoint, record the service's purpose, necessity and responsible
person/team in **サービスの用途・必要性・管理担当**. Continue to explain its audience,
safeguards and judgment separately. Loopback-only endpoints remain read-only
references. The configuration diagram shows abbreviated service observations;
open its linked details for the complete inventory. Saved JSON and HTML/Markdown
exports preserve observations separately from human explanations.

PID is omitted from the listening-service inventory because it changes across
process restarts. It remains in technical collection evidence and JSON, not as a
service identity or a field to review.

## Review local accounts and configured sudo

New collections retain local accounts, but individual judgments cover observed
main listening-service users and UID 1000+ user-account candidates. Other low-UID
system accounts appear under **システムアカウントの参考情報** without a judgment
form or contribution to current/unreviewed counts. UID 1000 is a selection
heuristic, not proof of SSH login access. Record each in-scope account's
function, continuing necessity and responsible person/team in
**アカウントの用途・必要性・管理担当**. Check configured memberships and related sudo
settings against those duties; a shell is displayed as configuration, not proof
that login is allowed. Groups with no explicit members remain visible; the group
directory distinguishes primary membership from explicit members.

Open the related account from a service, then follow its configured sudo links.
Service links use numeric effective UID matches; multiple account aliases remain
visible. The service's **実行中の実効GID** and **実行中の補助グループGID** describe the
observed process, independently of configured account memberships. Local group
names are shown alongside numeric GIDs when available. PID is not displayed as
an inventory identity.

Sudo fields show the configured subject, run-as users/groups, commands, argument
restrictions and password tags. These are configured rules, not a complete grant
calculation. Review accounts and sudo items separately. Unsupported sudo syntax
and unreadable in-scope sources remain visible as actual collection limitations.

The scope paragraph states that external identities, actual SSH login access,
management sockets and non-sudo privilege paths are outside this review. These
exclusions require no extra judgment, create no unreviewed counts and do not
prevent completing the in-scope report. Existing JSON files remain readable;
recollect to obtain accounts and runtime groups that were not collected before.

A root account listed as a reference does not mean a service runs as root. It
becomes a main account item when a main listening service's observed process
effective UID is 0. Sudo run-as links can open reference accounts without asking
for another account judgment. Existing stored decisions on reference accounts
remain in JSON and are not included in report follow-up counts.

## Explain connected external storage

New collections show **外付けストレージ** items for USB/removable storage candidates,
including unmounted devices. Rootfs backing disks are excluded; another internal
fixed disk is not treated as external just because it is outside rootfs. Product,
capacity, serial and mount/read-only observations help identify the candidate.
These observations are not a file-permission or content-safety evaluation.

In **接続の用途・必要性・管理担当**, explain the application/work and responsible
team, for example “○○ application save destination.” Record the reason for
continued connection, select **追加対応不要** when appropriate and save with the
reviewer's name. You do not need to mount, unplug or disable it, or fill in a
detailed access-control checklist. Unknown/unnecessary devices can instead be
left unreviewed or judged to require action with a human explanation.

The storage-status text distinguishes **未収集**, a complete inventory with zero
candidates, and incomplete reads/rootfs identification. Older JSON stays readable;
recollect to acquire storage observations. The backend observes USB ancestry or
removable block-device flags in the current namespace; it does not inventory every
possible external transport or past connection. See the
[storage reference](../specifications/prototype/external-storage.md).

For a fictional rehearsal with accounts, existing service judgments and two USB
examples, open `test/fixtures/prototype/reports/storage-demo.json` with `serve`.
The backup SSD is already explained; the unmounted transfer USB is ready to judge.


## Explain USB devices and physical operation

New collections include **USB HID機器** for devices declaring HID functions.
Use **物理構成の概要** in the editor or the HTML configuration page to follow
PC → root hub → hub → attached device. Open a device card to review its item.
Hubs and other USB functions provide context without extra judgment tasks.

Record the intended application/use and responsible team, then explain safeguards.
For example: “Used to operate the existing application; the upstream operating
manual defines who may connect devices and how the equipment is physically
managed.” A manual reference and operating rule are valid human explanation;
AVERA does not verify their implementation or enforce a mouse-only USB port.
Record a reason and select **追加対応不要** when the responsible person agrees.

A reliably associated HID/storage composite device is judged once in the USB
item, covering all its functions. Its storage detail remains in a reference
section, and prior storage explanations remain in JSON. Changing the associated
storage observation requires fresh USB review. Without reliable association,
storage remains independently reviewable. Never interpret an old storage-only
judgment as approval of HID behavior.

USB uncollected/partial/failed states are explicit in the diagram. Old files do
not imply no USB devices; recollect for attachment metadata. No input events,
keystrokes, storage contents or device configuration writes are involved.
See the [USB contract and limits](../specifications/prototype/usb-physical-overview.md).

For a fictional rehearsal, open `test/fixtures/prototype/reports/usb-demo.json`.
It contains a hub, a mouse accepted with a manual-based human safeguard, a
pending HID/storage composite device and the earlier storage/account/service
examples. Stop an older `serve` and restart after rebuilding to load new assets.
