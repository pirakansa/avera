# Portable fictional report

`fictional-report.json` is a single-file offline viewer rehearsal, generated from T00's changed snapshot, first snapshot and reviewed assessments. No real host was inspected. HTTP now runs as UID 0; the earlier acceptance remains prior context until explicitly reviewed. The sudo group also changed, and unsupported policy interpretation remains visible.

Open it with `avera serve --input test/fixtures/prototype/reports/fictional-report.json --output-dir EXISTING_PRIVATE_DIRECTORY`. Edit judgments in the browser, save to a new file, download the latest JSON and export Japanese HTML. This file is unbound to any real installation; do not treat it as your machine's report.

Reproduce the fixture into a new temporary filename with:

```sh
avera review --single-report \
  --snapshot test/fixtures/prototype/contracts/snapshot-changed.json \
  --previous test/fixtures/prototype/contracts/snapshot-first.json \
  --assessments test/fixtures/prototype/contracts/assessments-reviewed.json \
  --output NEW_FILE.json
```

The original contract fixtures remain unchanged for domain regression tests. This preserved input fixture uses portable v1. Current saves use v3 and accept/migrate v1/v2; nested artifacts retain the original T00 contract revision. Current presentation groups the loopback endpoint as reference information and supports human-described links at its network entrance.
