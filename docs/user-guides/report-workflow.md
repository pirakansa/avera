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

Run in an explicitly opened root shell on the selected Linux/amd64 target. Choose an existing private output directory and a new JSON filename:

```sh
./avera collect --target-id my-server --output /root/avera-run-001.json
```

The target name is optional on the first collection; omission generates an ID. Subsequent collection inherits the name from the report. AVERA does not invoke sudo, collect remotely or change target configuration. Live network collection reads sockets and existing iptables/nftables configuration; sudo remains explicitly uncollected. AVERA does not install missing firewall tools. Distribution/kernel metadata not inspected by the prototype remains unknown.

The JSON contains observed facts, previous context when available, editable judgments and superseded human judgment history. Exit 0 means successful collection; exit 2 means results were saved with incomplete/failed sources. Exit 1 means invalid input, wrong target, missing privilege or a fatal/output failure. Inspect incomplete coverage in the viewer rather than treating missing observations as absence.

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

Save explicitly reconfirms the judgment against the current facts and records the actor/time. Earlier human entries are archived verbatim; generated empty starters need not be archived. Saving accepted does not override incomplete collection or conflicting interpretation: the derived current decision can remain unreviewed. Use the unmatched-judgment section to inspect explanations without a current observation. Replaced judgment history is not displayed. The exported HTML includes active prior context and current review state; superseded archived entries remain in the portable JSON rather than becoming extra active judgments.

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

You do not need to re-enter the target ID or provide separate snapshot/assessment files. Then bring `avera-run-002.json` back to the host and open it with `serve` again. Old acceptance is not automatically reconfirmed after changed facts or incomplete collection.

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

New collected artifacts use contract revision `netfilter-2026-10-25.1` inside the
v3 portable container with reusable service policies. Existing T00 artifacts and v1/v2 reports remain
readable, including as previous collection input; original files are not modified.
Older executables cannot read the new nested contract revision.

Superseded judgment/relationship history is retained only in JSON, not in the
GUI, report preview or exported HTML/Markdown. Current accepted items and their
reasons remain visible. History storage itself is a [retirement candidate](../specifications/prototype/retirement-candidates.md#rc01--superseded-human-judgment-and-relationship-history), not removed in this revision.
