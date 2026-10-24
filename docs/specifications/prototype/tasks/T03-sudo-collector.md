# T03 — Collect bounded local sudo configuration

Status: complete (2026-10-24). Collector, fictional-fixture tests and T05 snapshots implemented; Ubuntu 24.04/sudo 1.9.15p5 primary references and disposable syntax comparison verified; final check, test and build passed. See the [implementation handoff](../sudo-collector.md) for grammar, evidence limits, shared-contract snapshots, and the successful optional local and Ubuntu syntax comparisons. Prerequisites: T00 baseline; T01 interfaces before integration. P1 can be tried before this task finishes. Read [collection scope](../collection.md).

## Outcome

Show reviewable local sudo configuration with evidence and interpretation limits. Do not claim a complete map of effective root access.

## Ownership

`internal/collect/sudo`, its tests, and fictional policy/account/group fixtures under `test/fixtures/prototype/sudo/`. Use no real passwords, authentication material, or copied private policy.

## Work steps

1. Verify sudo source/include semantics for the selected Ubuntu/sudo version using primary references. Record a supported grammar table before implementing interpretation.
2. Start with source inventory and traversal: main file, supported include directives, ordering, filtering, path handling, bounded recursion, and diagnostics.
3. Implement a small explicitly supported rule subset with evidence locations. Preserve unsupported constructs as review items/diagnostics; explain how they limit conclusions about other rules.
4. Add local account/group context where needed, including primary group membership. Do not resolve external identity services.
5. Derive conservative stable anchors and context fingerprints through T01's shared functions. Ensure include/context and membership changes can trigger re-review.
6. Provide an understood rule and an unsupported construct to T05 for early presentation review. Expand grammar only when required for a useful trial, not to finish the whole sudo language.
7. Integrate with the existing review/report path; avoid collector-specific human-decision storage.
8. Validate against a disposable reference setup when available. An installed reference tool may aid tests, but must not become a required runtime dependency.

## Exclusions

No execution of sudo policy commands, complete alias/negation/argument evaluator unless explicitly added to the supported subset, remote directory lookups, all-privilege audit, modifications to `/etc/sudoers`, or automated privilege removal.

## Acceptance cases

- Supported explicit user and local-group rules yield accurate configured-rule facts and source evidence.
- A `sudo` group member alone is not presented as proof of all effective permissions.
- Missing includes, cycles, unsupported directives, and parse limits remain visible and affect coverage.
- A semantic construct affecting a rule cannot be ignored while presenting that rule as fully understood.
- Changed included context or local membership invalidates the relevant prior interpretation or conservatively requests re-review.
- The default test suite reads only temporary fictional policy roots; it does not require root or inspect the actual machine's policy.
- The report clearly labels configured rules versus fully resolved effective permissions, which this prototype does not promise.

## Handoff

Deliver the supported grammar table, source references, fixtures, interpretation limits, and a recommendation for the next trial. Any model extensions must go through the shared contract owner; do not encode extra semantics only in free-text evidence.
