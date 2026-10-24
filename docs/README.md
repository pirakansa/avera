# Documentation

AVERA provides experimental `collect`, `review`, and `report` commands, with network collection, assessment reuse, and bundled guidance. Owner P1 feedback and T07 build delivery verification remain separate follow-ups.

## Product direction

- [Product direction](specifications/product-direction.md): agreed purpose, constraints, exclusions, and unresolved product choices.

## Prototype specifications and work plan

Start with the [prototype plan and task index](specifications/prototype/README.md). It provides the order of work and links to independently assignable agent tasks.

- [Implemented CLI workflow](specifications/prototype/cli-workflow.md): flags, outputs, exit codes, safety boundaries and test evidence
- [Architecture and execution](specifications/prototype/architecture.md)
- [Data contracts and assessment reuse](specifications/prototype/data-contracts.md)
- [Implemented model and storage APIs](specifications/prototype/model-storage.md): T01 handoff, limits, permissions, and portability
- [Network collector implementation](specifications/prototype/network-collector.md): T02 API, synthetic fixtures, scope, limits, and recorded Ubuntu comparison
- [Guidance and Markdown report API](specifications/prototype/guidance-report.md): T05 content, rendering, escaping, examples, and pending owner trial
- [Collection scope and evidence](specifications/prototype/collection.md)
- [Decision register](specifications/prototype/decisions.md)

These are proposed prototype defaults. They do not turn unresolved product choices into approved release requirements.

## Prototype trial guide

- [Trial cycles and feedback](user-guides/prototype-trials.md): how the project owner will try each increment and decide what to change.
- [Disposable network collector validation](user-guides/network-collector-validation.md): Ubuntu comparison procedure and interpretation of its results.
