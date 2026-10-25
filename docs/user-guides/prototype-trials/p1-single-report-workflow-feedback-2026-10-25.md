# P1 single-report workflow feedback — 2026-10-25

Historical feedback record for its original increment. Later decisions supersede
some behavior below; use the [current trial handoff](current-status.md) for the
implemented workflow and ongoing owner feedback status.

Status: actual owner feedback received in conversation; implementation revision follows the owner's explicit request. Owner acceptance of the new editor remains pending.

- Owner feedback (English paraphrase): preparing/editing the JSON is too complex, particularly when handling targets. Simplify to one report JSON including judgments: target-side collection consumes the previous report and emits the next; host-side browser viewing/editing consumes that report and exports HTML.
- Decision: make the portable single report and local editor the primary prototype flow, while retaining the old split artifacts for diagnostics/import.
- Observation boundary: facts/identity/basis are not browser-editable. Explicit save reconfirms current facts, retains superseded human judgments, and publishes a new private JSON before updating session state.
- Target handling: inherit logical target ID, compare a machine-ID digest, reject bound mismatches; explicitly adopt only verified unbound imports. Cloned IDs and conservative network namespace identity remain limitations.
- Files to carry: one latest saved/downloaded report JSON; HTML is a recipient deliverable, not the next collection input.
- Scope: loopback-only Japanese editor, bundled static assets, current findings/old judgments, saved-state HTML preview/export and JSON download. No target-side Web service, remote collection, automatic browser launch or multi-user authentication.
- Compatibility: `avera.report.v1` is a new outer format; original artifact schema/revision and T04 semantics are unchanged. Collection defaults to a single JSON; old directory collection requires `--legacy-output`. The user guide links the import route.
- Toolchain: minimum Go increased to 1.26.10 to fix reachable standard-library findings detected when adding net/http. No application module dependency was added. Devcontainer Go base was updated; a container rebuild was not performed as part of this change.
- Next owner trial: use the [single-report guide](../report-workflow.md), open the portable fictional report, edit/save a judgment, export HTML and download JSON, then judge whether file movement and target selection are understandable.
- Do not infer the owner performed a live collection, approved the UI, or accepted its security/readiness from this feedback alone.

## Agent verification

Setup, check, race tests and build passed with Go 1.26.10, as did the 29-JSON/13-scenario legacy fixture validator, new portable-fixture reproducibility test and diff checks. Final vulnerability scan: no vulnerabilities found. The initial check with the old toolchain exposed HTTP/crypto standard-library findings; the minimum Go version and devcontainer base were updated to resolve them.

Actual Chromium automation passed judgment editing/saving, preservation of a second item's draft, JSON/HTML downloads, Japanese report preview, escaped annotation text, mobile overflow checks and graceful shutdown. Screenshots were inspected. These were agent-run fictional tests, not an owner trial or live target inspection. The repeat-collection application test carries an edited bound report into an injected collector and checks target, accepted decision and archived history retention. Browser dependencies were temporary test tools only.
