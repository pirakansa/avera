# CLI workflow reference

Status: T06 baseline implemented and validated; owner T05/T06 feedback and revisions are in progress, with final acceptance pending. See the [current handoff](../../user-guides/prototype-trials/current-status.md). E01–E10 remain experimental. See the [trial guide](../../user-guides/prototype-trials.md) for executable instructions.

The [single-report revision](report-workflow.md) supersedes split files as the primary workflow. The commands described below retain diagnostic compatibility; add `--legacy-output` to collect into a run directory. `serve`, `collect --input`, `review --single-report` and `report --input` implement the simpler workflow.

## Primary commands and flags

- `collect --output NEW_JSON [--input REPORT] [--target-id GROUP]`: root-only
  collection of network sockets and local iptables/nftables candidates into one
  portable report. The input carries common endpoint judgments across equivalent
  PCs; machine ID is not read. Sudo remains explicitly uncollected.
- `serve --input REPORT --output-dir EXISTING_PRIVATE_DIRECTORY
  [--listen 127.0.0.1:0] [--lang ja|en]`: local browser editing with private new
  saves, JSON download and HTML export. Editor controls are Japanese; the language
  flag selects report/guidance content. It reads saved files only.
- `report --input REPORT --output NEW_FILE [--format markdown|html]
  [--lang en|ja]`: offline portable report rendering. Default output is English
  Markdown; HTML includes the network diagram.
- `review --single-report --snapshot SNAPSHOT --output NEW_JSON
  [--previous SNAPSHOT] [--assessments ASSESSMENTS]`: import split files for the
  primary workflow. `review` is not the browser editor; use `serve`.

New saves use portable v3; v1/v2 inputs remain readable. `--target-id` is optional
on first collection and inherited from input thereafter. It is a logical group
label, not a physical-PC identity. `--adopt-target` is an obsolete compatibility
flag accepted with input, not required. Endpoint saves always update common
settings; there is no individual/shared checkbox. See the [portable contract](report-workflow.md)
and [shared judgments](shared-service-judgments.md) for exact matching and provenance.

## Legacy diagnostic commands and flags

- `collect --legacy-output --target-id ID --output NEW_DIRECTORY [--previous SNAPSHOT] [--assessments ASSESSMENTS]`: root-only live network collection in the current observable namespace. Sudo is explicitly `not_collected` for P1. Other domains are outside this increment.
- `review --snapshot SNAPSHOT --output NEW_DIRECTORY [--previous SNAPSHOT] [--assessments ASSESSMENTS]`: explicit-file review without privilege checks or collection.
- `report --snapshot SNAPSHOT --output NEW_FILE [--previous SNAPSHOT] [--assessments ASSESSMENTS] [--format markdown|html] [--lang en|ja]`: explicit-file report with embedded T05 guidance.

Value flags take values; `--legacy-output`, `--single-report`, `--adopt-target` and help are Boolean flags. Positional arguments, unknown commands/flags, and command-inappropriate flags are errors. `--help` describes each command. Offline processing supports Linux; other operating systems are unverified.

Legacy split-file collection and review publish `snapshot.json`, recomputed `review.json`, and editable `assessments.json` as one new private directory. The latter retains matching entries, generated unreviewed starters, and unmatched prior context. Generated starters are new inputs: reporting with them records their IDs, so reproduce the original comparison using the original assessment context rather than assuming starters are byte-equivalent inputs. Assessments never modify observations or silently reconfirm changed facts.

Report writes a single new Markdown or HTML file. The default remains English Markdown; use `--format html --lang ja` for the Japanese browser report. `report` accepts `--format`/`--lang`, and `serve` accepts `--lang`; unsupported values fail before publication. It recomputes review states and counts using T04; it does not accept a stored review file. Use the same current snapshot, previous snapshot and assessment context to reproduce a comparison. Omission of `--previous` explicitly labels comparison unavailable. Missing metadata about distribution/kernel is recorded as `unknown`; the application does not inspect extra host files to infer it.

## Validation and publication

Non-root collection fails without invoking a collector or sudo. Supplied artifacts are strictly validated, including version, kind, target compatibility and assessment basis, before privileged sources are read. All commands check destinations before work, including whether private staging can be created in the destination parent. Parents must exist, be owned by the effective user or root, have trusted write permissions, and have no symlink components. A root collector therefore rejects an ordinary-user-owned parent even if Unix permissions permit writing. Create the collection directory as root; use a separate ordinary-user-owned directory for host review. Existing destinations, including input collisions, are refused. Publication separately enforces no replacement against concurrent creators.

Files use `0600`, directories `0700`. Complete artifact sets are staged privately and published atomically using Linux `renameat2(RENAME_NOREPLACE)`; reports use no-replacement hard-link publication. Unsupported filesystems fail rather than falling back to overwrite. Cancellation is propagated through collection and checked before publication. Fatal errors do not advertise a complete result. A post-publication sync failure can leave complete output present; inspect it and choose a new path for retries.

Diagnostics go to stderr. Invalid artifact errors identify the responsible flag and suggest checking JSON/schema; they omit raw decoder messages to avoid exposing annotation/configuration values. No annotations or evidence references are executed or fetched. Size/depth/count limits are those of the [model/storage reference](model-storage.md) and [network collector](network-collector.md). Report technical values are visibly bounded as specified by [T05](guidance-report.md).

| Exit | Meaning |
|---|---|
| 0 | Requested operation completed; unreviewed items and declared exclusions are permitted |
| 1 | Invalid inputs, privilege, cancellation, fatal collection or output failure |
| 2 | Live collection persisted usable results with partial/failed sources; inspect coverage diagnostics |

All sources failing still publishes diagnostic artifacts with exit 2. Offline review/report of a partial or failed snapshot returns 0 when processing succeeds; coverage remains visible in the output.

## Responsibility and evidence

`internal/cli/commands/root` owns thin Cobra mappings, `internal/app` orchestrates use cases, and `internal/artifact` owns publication. `app.Registration` supports adding another collector without duplicating review/report orchestration. P1 deliberately registers only network; enabling sudo later requires explicit integration and tests.

`internal/app/workflow_test.go` compares fixture-derived review output with T00 expected reviews and checks offline Markdown equality with T05 rendering. It verifies injected root/non-root collection, prior/advance context, partial/failed sources, cancellation, collisions, invalid JSON, target mismatch, and symlink parents. CLI tests pass for dispatch, flags, diagnostics and exit behavior with injected sources. These are simulated evidence, not owner feedback or a live-target validation record.

Historical baseline validation on 2026-10-24: setup, check, race tests, build, the T00 fixture validator and `git diff --check` passed. The actual binary completed the combined fictional rehearsal and assessment editing flow as UID 1000, rejected non-root collection before output creation, and refused overwrite. The vulnerability scan found 0 reachable vulnerabilities; it also reported 1 finding in imported packages and 13 in required modules that current code does not appear to call. Cobra is pinned to v1.10.2 to provide command/flag dispatch; `go.mod` and `go.sum` are synchronized. No live root collection or owner usability trial is claimed.

## P1 presentation feedback extension

The owner requested Japanese content and a more graphical HTML deliverable. `--lang ja` localizes report headings, summary labels, decision/change/applicability states, guidance questions, missing-field prompts and review reasons. Human statements, identifiers, schema keys, technical evidence, diagnostic codes and source limitations retain their original text. CLI help and stderr remain English. Presentation options do not change T04 assessment results or artifact schemas.

HTML is a UTF-8 single file with bundled CSS, responsive summary cards, section navigation, item details and expandable technical evidence. It requires no server, scripts, fonts, images, external stylesheet or network. Evidence references remain inert text. The fixed template HTML-escapes every interpolated value, controls are made visible, and a restrictive Content Security Policy blocks external resources and scripts. The existing private/no-clobber publication protocol also applies to HTML.

Tests cover all 13 fixture scenarios in both languages and formats, count reconciliation, deterministic output, input immutability, local anchor integrity, hostile annotations, option validation and CLI publication. English Markdown output is unchanged when no presentation flags are supplied.

## Current validation scope

The original dated results above do not describe the current full feature set or
vulnerability scan. Current quality gates use Go 1.26.10 or later and report no
vulnerabilities. Automated tests cover portable v3/common judgments, absent
machine identity, forwarding collection and its limits, current-only presentation
with JSON-retained archives, and linked GUI/HTML diagrams. Local root collection
has succeeded after removal of the machine-ID requirement; final owner acceptance
and T07 distributed-build verification remain pending. See the [current handoff](../../user-guides/prototype-trials/current-status.md).
