# Documentation

AVERA provides experimental `collect`, `serve`, `review` and `report` commands, with network/forwarding collection, shared service judgments and graphical Japanese HTML reports. T05/T06 owner feedback is in progress; final acceptance and T07 delivery verification remain pending. Start with the [current session handoff](user-guides/prototype-trials/current-status.md).

## Product direction

- [Product direction](specifications/product-direction.md): agreed purpose, constraints, exclusions, and unresolved product choices.

## Prototype specifications and work plan

Start with the [prototype plan and task index](specifications/prototype/README.md). It provides the order of work and links to independently assignable agent tasks.

- [Implemented CLI workflow](specifications/prototype/cli-workflow.md): flags, outputs, exit codes, safety boundaries and test evidence
- [Architecture and execution](specifications/prototype/architecture.md)
- [Data contracts and assessment reuse](specifications/prototype/data-contracts.md)
- [Implemented model and storage APIs](specifications/prototype/model-storage.md): T01 handoff, limits, permissions, and portability
- [Network collector implementation](specifications/prototype/network-collector.md): T02 API, synthetic fixtures, scope, limits, and recorded Ubuntu comparison
- [Guidance and HTML/Markdown report API](specifications/prototype/guidance-report.md): T05 content, rendering, escaping, examples, and ongoing owner feedback
- [Collection scope and evidence](specifications/prototype/collection.md)
- [Decision register](specifications/prototype/decisions.md)

These are proposed prototype defaults. They do not turn unresolved product choices into approved release requirements.

## Prototype trial guide

- [Single-report workflow](user-guides/report-workflow.md): primary collect/carry/browser-edit/export workflow
- [Review scope and related services](specifications/prototype/review-scope.md): reference grouping, entrance-owned human path statements, and v2 compatibility
- [Portable report contract](specifications/prototype/report-workflow.md): container, reusable service judgments, save and browser boundaries

- [Trial cycles and feedback](user-guides/prototype-trials.md): how the project owner will try each increment and decide what to change.
- [Disposable network collector validation](user-guides/network-collector-validation.md): Ubuntu comparison procedure and interpretation of its results.

- [Shared service judgments](specifications/prototype/shared-service-judgments.md): cross-PC reuse, common explanations, v3 policies and configured destination hints

- [Retirement candidates](specifications/prototype/retirement-candidates.md): implemented simplifications and deferred removal of obsolete prototype functionality

- [Network configuration overview](specifications/prototype/network-overview.md): graphical observations, configured versus human relationships, coverage and offline HTML export
