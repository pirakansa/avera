# Prototype trial cycles

Status: T06 commands and the fictional rehearsal have passed implementation validation. The owner plans to try T05 guidance/reporting together with the T06 workflow after implementation. P1 feedback remains pending. See the [CLI reference](../specifications/prototype/cli-workflow.md) for the implemented contract.

The [single-report and browser-editor guide](report-workflow.md) is now the primary trial entry point, following owner feedback about manual JSON editing. The split-file exercises below remain diagnostic scenarios and migration examples; collection examples need `--legacy-output` when requesting a run directory.

## Purpose

The project owner is the first test user. The goal is to learn how a person inventories concerns, supplies explanations, reuses prior work, and reports results. The trial is not a declaration of product safety or release readiness.

Keep each cycle small: choose a question, prepare a scenario, let the owner use it, record what actually happened, change one coherent part, and repeat.

## P0 — Read a concrete fictional example

Start with the HTTP example in the [short P0 walkthrough](../specifications/prototype/examples/p0-preparation.md). The full fictional report and scenario tables are available for technical follow-up; the owner need not work through every case.

Ask three task-oriented questions:

1. If you received this result, would you know what to check and what explanation to write?
2. Can you find your previous explanation and tell what needs another look this time?
3. If you gave this report to your manager or requester, what would it explain, and what would still be unclear?

Record confusing wording, missing help, irrelevant questions, and unclear next actions. Use evidence links when they help the owner answer, rather than making artifact navigation an examination. Do not ask the owner to validate hash algorithms, learn internal states, or approve JSON editing before trying it. Record “needs an executable trial” as a valid outcome.

P0 is optional and does not block T01. The technical handoff consists of the prepared contract and fixtures. Use those experimental defaults to build the prototype; gather workflow/editing feedback in P1 if it cannot be usefully obtained from a paper example. Request Ubuntu release/architecture separately when preparing live validation, not as part of this comprehension check.

## P1 — Try the complete port workflow

Preparation: a known disposable Ubuntu target, the T07-verified local binary, a separate output directory per run, and fictional or explicitly selected prior/advance inputs. Record the target release/architecture and implemented collection scope. Use root for collection. Use existing access or a local console; do not install SSH or change firewall rules to make the tool usable.

| Scenario | Owner action | What to observe |
|---|---|---|
| First inventory | Collect, inspect an unreviewed endpoint, write an explanation, generate a report | Can the owner find what to do next? |
| Advance knowledge | Supply an exact prepared assessment on the target and collect | Is its origin and applicability understandable? |
| Repeat unchanged | Supply prior snapshot/assessments and collect again | Is prior work retained without retyping? |
| Meaningful change | Use controlled fixtures or a disposable target change selected for the trial | Is old acceptance retained as history but clearly awaiting review? |
| Incomplete collection | Use the fault fixture rather than altering real permissions | Can the owner distinguish missing evidence from absence of concerns? |
| Management report | Read only the summary, then follow one item to evidence | Are the scope, unfinished work, and decision basis clear? |

Do not introduce production changes merely to create a changed-item scenario. Scripted fixtures are a valid way to test comparison semantics; record them as simulated rather than live evidence.

After P1, decide whether to keep separate files/commands for the next cycle. Consider a simpler editing surface only if the trial demonstrates that JSON or artifact movement is a problem. The owner has now requested Japanese content and a graphical HTML report. The presentation trial below evaluates that requested extension; an editing GUI remains undecided.

## Executable T05 + T06 rehearsal

Run from the repository root after `vorbere run setup` and `vorbere run build`. The following rehearsal uses only fictional files and needs no root, host collection, external commands inside AVERA, or network. It can be performed before a T07 distributed build is available.

```sh
trial_dir=$(mktemp -d /tmp/avera-trial.XXXXXX)
fixtures=./test/fixtures/prototype/contracts

./bin/avera review --snapshot "$fixtures/snapshot-first.json" \
  --output "$trial_dir/first"
./bin/avera report --snapshot "$fixtures/snapshot-first.json" \
  --output "$trial_dir/first.md"

./bin/avera report --snapshot "$fixtures/snapshot-first.json" \
  --assessments "$fixtures/assessments-advance.json" \
  --output "$trial_dir/advance.md"

./bin/avera report --snapshot "$fixtures/snapshot-repeat.json" \
  --previous "$fixtures/snapshot-first.json" \
  --assessments "$fixtures/assessments-reviewed.json" \
  --output "$trial_dir/unchanged.md"

./bin/avera report --snapshot "$fixtures/snapshot-changed.json" \
  --previous "$fixtures/snapshot-first.json" \
  --assessments "$fixtures/assessments-reviewed.json" \
  --output "$trial_dir/changed.md"

./bin/avera report --snapshot "$fixtures/snapshot-changed.json" \
  --previous "$fixtures/snapshot-first.json" \
  --assessments "$fixtures/assessments-reconfirmed.json" \
  --output "$trial_dir/reconfirmed.md"

./bin/avera report --snapshot "$fixtures/snapshot-partial.json" \
  --previous "$fixtures/snapshot-first.json" \
  --assessments "$fixtures/assessments-reviewed.json" \
  --output "$trial_dir/partial.md"
printf 'Open the Markdown reports under %s\n' "$trial_dir"
```

Read the reports in this order:

1. `first.md`: from the summary, follow a current item to its questions and underlying evidence. Find what to explain about purpose, audience, safeguards and rationale. Check that a public port is not automatically called unsafe and that static-content/firewall explanations are prompts rather than verified safeguards. Comparison is unavailable without prior observations.
2. `advance.md`: distinguish prepared human expectations from observed facts and current applicability. An incomplete or mismatched expectation must remain a review question.
3. `unchanged.md` and `changed.md`: find the old explanation and current decision. Changed facts must not carry old acceptance into the current count. Missing/unsupported sudo context in these fictional examples illustrates limitations; it does not imply live sudo integration.
4. `reconfirmed.md`: see how an explicit replacement human judgment differs from silently updating an old basis.
5. `partial.md`: find collection diagnostics before drawing conclusions from missing observations. Offline rendering succeeds with exit 0; a live partial collection uses exit 2.

To try editing, copy `first/assessments.json` to a new file in the trial directory. Use an editor to fill the relevant entry's `purpose`, `exposure`, `safeguards`, `rationale`, `evidence_refs`, `decision`, `actor` and `assessed_at` (UTC `YYYY-MM-DDTHH:MM:SSZ`). Decision is `unreviewed`, `action_required`, or `accepted`. Keep the selector, origin and basis intact when judging those same facts. Regenerate a report with that file:

```sh
cp "$trial_dir/first/assessments.json" "$trial_dir/edited-assessments.json"
# Edit edited-assessments.json before running the next command.
./bin/avera report --snapshot "$fixtures/snapshot-first.json" \
  --assessments "$trial_dir/edited-assessments.json" \
  --output "$trial_dir/edited.md"
```

Missing explanation fields are prompts about completeness, not proof of a vulnerability. Keep the original assessment history. After a meaningful change, make an explicit replacement entry with the new snapshot/fingerprint only after human reconfirmation; the fictional reconfirmed input demonstrates this. Do not retain two conflicting active judgments for the same finding unless intentionally testing conflict handling.

For a live P1 trial, first select a disposable Linux/amd64 Ubuntu target and record the actual release, kernel and binary identifier. Use the T07-verified binary for target delivery. Run collection explicitly as root on that target, using a private directory owned by root (the shell below is a root shell):

```sh
live_dir=$(mktemp -d /tmp/avera-live.XXXXXX)
./bin/avera collect --legacy-output --target-id trial-target-01 --output "$live_dir/run-001"
# If exit is 2, inspect coverage and diagnostics; usable results are still saved.
./bin/avera review --snapshot "$live_dir/run-001/snapshot.json" \
  --output "$live_dir/review-001"
# Edit review-001/assessments.json, preserving selectors/basis.
./bin/avera report --snapshot "$live_dir/run-001/snapshot.json" \
  --assessments "$live_dir/review-001/assessments.json" \
  --output "$live_dir/report-001.md"
./bin/avera collect --legacy-output --target-id trial-target-01 \
  --previous "$live_dir/run-001/snapshot.json" \
  --assessments "$live_dir/review-001/assessments.json" \
  --output "$live_dir/run-002"
./bin/avera report --snapshot "$live_dir/run-002/snapshot.json" \
  --previous "$live_dir/run-001/snapshot.json" \
  --assessments "$live_dir/review-001/assessments.json" \
  --output "$live_dir/report-002.md"
```

Review/report can instead run as an ordinary user after explicitly transferring the selected artifacts to a private directory owned by that user. Root-created directories/files are private; changing users alone does not grant read access. Do not relax permissions globally to move artifacts. AVERA never invokes sudo, transfers files, or contacts evidence URLs.

Live network scope gets a fresh conservative namespace identity each run because `/proc` namespace inode numbers cannot prove incarnation. Therefore live repeat runs can require new review even when ports look unchanged. Use the fictional unchanged scenario to inspect exact-match reuse, and record live identity churn as P1 feedback. Live collection deliberately leaves sudo uncollected.

Record actual observations and owner comments using the feedback template below. T05/T06 implementation and fixture tests are not a completed owner P1 trial.

## Japanese HTML presentation trial

The owner requested Japanese content and HTML after the initial T05/T06 handoff. Build the current binary, then generate a single-file browser report from fictional saved inputs:

```sh
vorbere run build
html_trial=$(mktemp -d /tmp/avera-html-trial.XXXXXX)
./bin/avera report \
  --snapshot ./test/fixtures/prototype/contracts/snapshot-repeat.json \
  --previous ./test/fixtures/prototype/contracts/snapshot-first.json \
  --assessments ./test/fixtures/prototype/contracts/assessments-reviewed.json \
  --format html --lang ja --output "$html_trial/unchanged.html"
./bin/avera report \
  --snapshot ./test/fixtures/prototype/contracts/snapshot-changed.json \
  --previous ./test/fixtures/prototype/contracts/snapshot-first.json \
  --assessments ./test/fixtures/prototype/contracts/assessments-reviewed.json \
  --format html --lang ja --output "$html_trial/changed.html"
printf 'Open the HTML files in a browser: %s\n' "$html_trial"
```

Open the files directly in a browser; no local server or network is required. Start with the overview cards, follow the current-item navigation, read the Japanese questions and decision context, then follow an evidence link and expand the technical details. Compare the unchanged and changed reports: stale acceptance must remain prior context. Repeat with a narrow browser window to assess readability. Evidence/source text and fictional human statements may remain English because their original content is preserved.

Use `--format markdown --lang ja` for Japanese Markdown or `--format html --lang en` for English HTML. Omitting both flags preserves the original English Markdown. CLI help/errors and JSON editing remain in English; this increment localizes the report rather than introducing translated artifact fields or a browser editor.

Feedback to gather: does Japanese wording clarify what to do next; are the cards and item headings understandable; can the owner distinguish observations from human statements and current acceptance from prior context; are the layout, density, mobile view and technical-detail expansion useful? Record actual feedback separately from automated checks. The first request is recorded in [the presentation feedback record](prototype-trials/p1-report-presentation-feedback-2026-10-24.md); acceptance of this new layout is still pending.

## P2 — Add the sudo example

Owner tasks:

1. Explain a supported local configured privilege rule using the supplied evidence.
2. Review an unsupported construct and identify what the tool could not determine.
3. Reuse an assessment, then compare a meaningful rule/context change.
4. Check whether the summary remains understandable with different item kinds and partial coverage.

Decide whether additional parsing is necessary for a useful inventory, whether guidance is specific enough, and whether a common assessment structure still works. Avoid forcing sudo explanations into port-specific fields.

## P3 — Focused revision cycles

Choose one or a few related questions per cycle, for example:

- Would structured prompts reduce “required by the product” explanations without forcing irrelevant controls?
- Does re-review happen too often because identities or fingerprints are too unstable?
- Does any reuse happen too broadly, hiding a meaningful change?
- Can the report itself become the working document, or is separate structured input preferable?
- Which additional observation would have answered a question the owner could not resolve?

Translate the selected question into a concrete change and a repeatable scenario. Update the [decision register](../specifications/prototype/decisions.md), affected contracts, and task acceptance cases. Preserve previous artifacts; do not silently migrate the owner's working history.

## Feedback template

Create one English record under `docs/user-guides/prototype-trials/` per actual trial. Use a descriptive filename such as `p1-network-repeat-YYYY-MM-DD.md`. Do not create completed-looking records before a session takes place.

```text
Trial / date:
Increment and task versions:
Build/artifact identifier:
Target and scope (sanitized):
Question or hypothesis:
Inputs and scenario (live / fictional / injected fault):
Steps actually performed:
Observed behavior:
Owner feedback:
Assistance required:
What remains unclear:
Decision: keep / change / investigate / defer
Next experiment and responsible task:
Affected contracts and compatibility consequences:
Validation to repeat:
```

## Readiness for another increment

For executable trial cycles, offer the owner a useful opportunity to try the current slice and use actual feedback to choose revisions. T01's initial fixture-based implementation can start as soon as the T00 baseline is available; it does not require a completed P0 walkthrough. Time estimates and fixed release dates are unnecessary at this stage.

If a live target or owner feedback is unavailable, continue independent fixture/unit work and mark the trial pending. Do not fabricate approval or imply the prototype was tested by its intended user.

Before any future public release, prepare a separate scope/readiness decision. This guide authorizes neither release publication nor submission of a security report to third parties.
