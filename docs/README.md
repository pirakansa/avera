# Documentation

AVERA currently has a Hello World CLI plus implemented internal model, validation, storage, and synthetic-tested network collector APIs. Collection/review/report commands remain planned.

## Product direction

- [Product direction](specifications/product-direction.md): agreed purpose, constraints, exclusions, and unresolved product choices.

## Prototype specifications and work plan

Start with the [prototype plan and task index](specifications/prototype/README.md). It provides the order of work and links to independently assignable agent tasks.

- [Architecture and execution](specifications/prototype/architecture.md)
- [Data contracts and assessment reuse](specifications/prototype/data-contracts.md)
- [Implemented model and storage APIs](specifications/prototype/model-storage.md): T01 handoff, limits, permissions, and portability
- [Network collector implementation](specifications/prototype/network-collector.md): T02 API, synthetic fixtures, scope, limits, and recorded Ubuntu comparison
- [Collection scope and evidence](specifications/prototype/collection.md)
- [Decision register](specifications/prototype/decisions.md)

These are proposed prototype defaults. They do not turn unresolved product choices into approved release requirements.

## Prototype trial guide

- [Trial cycles and feedback](user-guides/prototype-trials.md): how the project owner will try each increment and decide what to change.
- [Disposable network collector validation](user-guides/network-collector-validation.md): Ubuntu comparison procedure and interpretation of its results.
