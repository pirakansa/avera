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

The target name is optional on the first collection; omission generates an ID. Subsequent collection inherits the name from the report. AVERA does not invoke sudo, collect remotely or change target configuration. P1 collects network only; sudo remains explicitly uncollected. Distribution/kernel metadata not inspected by the prototype remains unknown.

The JSON contains observed facts, previous context when available, editable judgments and superseded human judgment history. Exit 0 means successful collection; exit 2 means results were saved with incomplete/failed sources. Exit 1 means invalid input, wrong target, missing privilege or a fatal/output failure. Inspect incomplete coverage in the viewer rather than treating missing observations as absence.

## Review on the host PC

Transfer just the report JSON through your existing chosen method to a private directory owned by your ordinary host user. AVERA does not transfer files or relax permissions. Start the local viewer as that ordinary user:

```sh
mkdir -m 700 ./review-saves
./bin/avera serve --input ./avera-run-001.json --output-dir ./review-saves
```

Open the exact `http://127.0.0.1:PORT` URL printed in the terminal. No other machine can connect; use the exact printed address rather than substituting a hostname. Leave this terminal running and stop with Ctrl+C when finished. The listener uses a free local port by default; `--listen 127.0.0.1:8080` selects a specific loopback port. Public addresses are refused. This is a local single-user workspace, not an authenticated multi-user Web service; other local processes can access its loopback endpoint.

1. Open a current item. Read the Japanese observation summary, questions, and earlier explanations. Technical observation fields are read-only.
2. Fill purpose, audience/exposure, safeguards, rationale, actor and optional evidence references. Pick unreviewed, action required or accepted. No selector, fingerprint, target ID, origin or timestamp needs manual editing.
3. Click the save button. The server creates a new `report-TIMESTAMP-RANDOM.json` in `review-saves` and shows its filename. The original input and earlier saves remain intact. Every save includes all previously saved judgments, not just that item. A failed save preserves the current session and leaves the form available for retry. Drafts in other items survive a successful save of one item; drafts are lost if the browser is closed/reloaded, with a navigation warning.
4. Use the report-preview tab to inspect the saved state. Use **HTML output** to download a standalone Japanese report for a recipient. It requires no server or network when opened later.
5. Use **Take JSON** to download the latest saved report as `avera-report.json`, or carry the newest filename displayed after saving. Downloads exclude unsaved fields and warn when drafts remain. If several tabs edit the same session, stale saves fail; retain your text and reload before retrying.

Save explicitly reconfirms the judgment against the current facts and records the actor/time. Earlier human entries are archived verbatim; generated empty starters need not be archived. Saving accepted does not override incomplete collection or conflicting interpretation: the derived current decision can remain unreviewed. View the unmatched-judgment/history sections to inspect prior context. The exported HTML includes active prior context and current review state; superseded archived entries remain in the portable JSON/editor history rather than becoming extra active judgments.

`--lang en` selects English report questions and export content; editor controls remain Japanese in this prototype.

## Repeat on the same target

Bring the latest judgment-containing JSON back to the same target. Run as root and choose a new output filename:

```sh
./avera collect --input ./avera-report.json --output ./avera-run-002.json
```

You do not need to re-enter the target ID or provide separate snapshot/assessment files. Then bring `avera-run-002.json` back to the host and open it with `serve` again. Old acceptance is not automatically reconfirmed after changed facts or incomplete collection.

The report binds to a hash of `/etc/machine-id`, never the raw ID. A different installation fails before collector access; start a separate report for another target. Cloned systems sharing a machine ID cannot be distinguished by this check. Missing/invalid machine ID fails explicitly. A reinstall changes installation identity. There is no automatic migration or override for a bound mismatch.

Network namespace incarnation is still conservatively assigned per live run. Apparently unchanged live ports can therefore require new review and leave earlier explanations under unmatched context. This limitation is unchanged by the simpler file workflow; use fictional fixtures to explore exact-match unchanged reuse.

## Try without root or a target

Open the prepared single-file fictional report directly:

```sh
trial_dir=$(mktemp -d /tmp/avera-single-report.XXXXXX)
./bin/avera serve \
  --input ./test/fixtures/prototype/reports/fictional-report.json \
  --output-dir "$trial_dir"
```

No manual JSON preparation or combination of snapshot files is required for this rehearsal. The portable fixture contains a changed observation and an older human explanation so you can try explicit reconfirmation. For migration of your existing legacy files only, use `review --single-report --snapshot CURRENT --previous PREVIOUS --assessments HUMAN_CONTEXT --output NEW_REPORT.json`.

These imported fixtures are fictional and unbound. They are for offline trial, not evidence about your computer. To import a real legacy report into live use, verify the selected installation and explicitly pass `--adopt-target` once with `collect --input`; it only binds unbound reports and cannot override a different existing machine binding.

Export saved reports without starting the viewer:

```sh
./bin/avera report --input ./avera-report.json \
  --format html --lang ja --output ./avera-report.html
```

For diagnostic compatibility, `collect --legacy-output` still writes the earlier three-file run directory, and legacy `review`/`report --snapshot ...` inputs remain supported. Do not mix `--input` with split context flags. The new single JSON is versioned separately and cannot be passed as a legacy `--snapshot` or `--assessments` file.

## Trial feedback

Check whether you can complete the collect → carry → edit → export → carry-back cycle without understanding JSON. Report confusing Japanese, unclear save/download behavior, difficulty finding old explanations, or uncertainty about which file to carry. The specification describes what the prototype implements; automated checks are not owner acceptance of this workflow.
