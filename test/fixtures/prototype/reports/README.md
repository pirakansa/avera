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

## Storage and account demo

`storage-demo.json` is an entirely fictional v3 report using the storage contract.
It preserves shared service judgments and demonstrates local account scope,
service runtime groups, configured sudo links and connected storage. The backup
SSD is mounted at `/srv/fictional-app-data` with an accepted application-save
purpose. The transfer USB is connected but unmounted and unreviewed, ready to try
purpose/reason entry and saving. Root is a reference account; HTTP uses `websvc`
UID 33, internal admin uses `alex` UID 1001. No real host or storage was inspected.

```sh
./bin/avera serve \
  --input test/fixtures/prototype/reports/storage-demo.json \
  --output-dir EXISTING_PRIVATE_DIRECTORY
```

Use a new temporary copy to rehearse changes. Saves create new files; neither this
fixture nor the original T00/v1 report should be overwritten by a trial.
