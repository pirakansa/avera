# Fictional T00 contract fixtures

**All data are invented. No host was inspected.** Prepared by an agent for contract `t00-2026-10-24.1`; owner P0 review is pending. Ubuntu 24.04/amd64 here describes the invented scenario, not a validated trial environment. These JSON files are explicit design examples, not outputs from an implemented collector or matcher.

The [contract](../../../../docs/specifications/prototype/data-contracts.md#t00-baseline-t00-2026-10-241) defines all fields and algorithms. The [three-level report](../../../../docs/specifications/prototype/examples/fictional-report.md) uses the same facts and human explanations. No reference URL is fetched.

## Artifact relationships

`scenarios.json` is a test manifest: each row names a current snapshot, optional previous snapshot, optional assessments, and the exact expected review. Null inputs mean no prior observation/assessment. `normalization-vectors.json` contains literal hash preimages and digests; neither file is a runtime envelope. Retain snapshots and assessments with reviews because review IDs refer to those inputs rather than copying human text.

| Files | Purpose |
|---|---|
| `snapshot-first.json`, `review-first.json` | Four observed items, no judgments; every item references its own snapshot evidence |
| `assessments-first-starter.json` | Editable unreviewed starter for the first snapshot; no invented human text or automatic acceptance |
| `assessments-reviewed.json` | Human HTTP acceptance and sudo action-required statement against s1 |
| `snapshot-repeat.json`, `review-unchanged.json` | PID/time-only changes; HTTP acceptance reused with original actor/date/rationale |
| `snapshot-changed.json`, `review-changed.json` | HTTP UID becomes 0 and sudo local membership gains sam; old bases are unchanged and HTTP acceptance is stale |
| `assessments-reconfirmed.json`, `review-reconfirmed.json` | Human creates a new active HTTP judgment against s3, chooses action-required; old reviewed input remains intact |
| `assessments-advance.json`, `review-advance.json` | Exact HTTP advance expectation applies without a previous snapshot; sudo context differs and remains awaiting review |
| `snapshot-supported-sudo.json`, `review-advance-sudo.json` | Separate supported-only sudo policy with complete context, demonstrating sufficient sudo advance applicability |
| `assessments-insufficient.json`, `review-insufficient.json` | Exact endpoint selector with empty expected facts; valid context, insufficient to apply |
| `snapshot-partial.json`, `review-partial.json` | Only HTTP observed, sources partial; old HTTP acceptance needs review, missing prior items unknown |
| `snapshot-failed.json`, `review-failed.json` | No findings, both domains failed; four historical unknowns, no empty-success inference |
| `snapshot-empty.json`, `review-empty.json` | Complete comparable empty scope; historical items not observed, not automatically remediated |
| `assessments-unmatched.json`, `review-unmatched.json` | Previously reviewed endpoint on port 9999 is never fabricated as an observation |
| `assessments-conflict.json`, `review-conflict.json` | Two human decisions for HTTP; conflict independent of date/order, no current acceptance |
| `review-basis-only.json` | Stored reviewed basis applies without a previous snapshot; change still not-compared |

Evidence IDs e1/e2 describe HTTP/local endpoints; e3/e4 describe the configured sudo rule/unsupported Defaults. They are local to each snapshot. Finding IDs are hashes of stable identity. Hashes exclude evidence IDs, PID, run ID, and times. The current sudo policy is incomplete, so equal rule hashes alone cannot establish reuse. The separate complete-context sudo snapshot is a different fictional scenario, not a silent improvement to the first snapshot.

An explicit reconfirmation replaces the HTTP entry in the new active assessment set; do not append both old and new entries to simulate history, because that is a conflict. Historical rationale is preserved in `assessments-reviewed.json`. A starter generator must never update the old basis merely by copying current facts.

## Local verification and handoff

From the repository root, run:

```sh
python3 test/fixtures/prototype/contracts/validate.py
```

The standard-library validator checks JSON parsing with duplicate-key rejection, local Markdown links, traceability, canonical hash vectors, PID-only stability and meaningful changes, independent scenario outcomes, coverage distinctions, counts, and preserved human history. It reads only these fixtures and relevant repository documentation. It is not a substitute for T01's future strict runtime validation or T02/T03 source-format tests. No Go implementation or external service is required.

T01: reproduce normalization vectors and implement strict types/I/O. T02/T03: preserve the observation/evidence/coverage boundary, prove source fidelity, and confirm real namespace comparability. T04: turn manifest rows into matching tests. T05: use the common review counts and report prompts. T06: preserve explicit inputs and never rebase a decision automatically. This repository handoff notifies the future task owners without contacting anyone externally.
