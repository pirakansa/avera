# Optional browser smoke check

This developer check needs Python Playwright and a Chromium browser in an isolated test environment. They are not target-side AVERA dependencies and are not needed for the default Go tests.

Import `snapshot-first.json` with `review --single-report` into a temporary report and start `serve` with that report and a temporary save directory. Then run:

```sh
python3 test/browser/report_viewer.py http://127.0.0.1:PORT/
```

Use a fresh first-snapshot report session. The script edits two items, saves HTTP, verifies that the other draft survives, downloads JSON/HTML, opens the preview, checks escaping, and checks mobile overflow. Screenshots are written to a printed temporary directory. Stop the viewer after the check. Run ordinary Go tests as well for revision conflicts, failed-save rollback, immutable fields, target binding and history preservation.
