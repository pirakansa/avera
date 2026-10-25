# Optional browser smoke check

This developer check needs Python Playwright and a Chromium browser in an isolated test environment. They are not target-side AVERA dependencies and are not needed for the default Go tests.

Import `snapshot-first.json` with `review --single-report` into a temporary report and start `serve` with that report and a temporary save directory. Then run:

```sh
python3 test/browser/report_viewer.py http://127.0.0.1:PORT/
```

Use a fresh first-snapshot report session. The script edits two items, saves HTTP, verifies that the other draft survives, downloads JSON/HTML, opens the preview, checks escaping, and checks mobile overflow. Screenshots are written to a printed temporary directory. Stop the viewer after the check. Run ordinary Go tests as well for revision conflicts, failed-save rollback, immutable fields, target binding and history preservation.

`python3 test/browser/related_services.py` starts its own fresh fixture session and checks focused counts, read-only loopback reference grouping, entrance-to-local relationship selection/save, v2 JSON roundtrip, escaped linked HTML and mobile overflow. It requires a built `bin/avera` and the same development browser dependencies. All report/screenshot outputs use a new temporary directory and the server is stopped after testing. The general smoke check now preserves a draft on a main sudo item because local reference services no longer have judgment forms.

`python3 test/browser/forwarding.py` creates a synthetic forwarding snapshot in a
new temporary directory, imports it through the CLI, and verifies the forwarding
rule's Japanese guidance, human judgment, loopback relationship, JSON/HTML exports,
escaping and mobile layout. It does not execute firewall commands.

`python3 test/browser/shared_services.py` verifies literal DNAT destination hints
without mandatory selection and automatically saving a reusable endpoint policy without a checkbox, v3 JSON and
the shared-judgment explanation in HTML. It starts/stops its own fixture viewer.
