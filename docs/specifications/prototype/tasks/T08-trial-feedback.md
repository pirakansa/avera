# T08 — Run the prototype learning cycle

Status: recurring owner-feedback/revision cycle in progress for T05/T06; final P1 acceptance and T07 delivery verification remain pending. See the [current handoff](../../../user-guides/prototype-trials/current-status.md). Prerequisites: T00 for the paper walkthrough; implemented T05/T06 for local fictional/development feedback; T07 for distributed-build live validation. Local feedback does not wait for T07. Read [trial cycles and feedback](../../../user-guides/prototype-trials.md).

## Outcome

Turn actual owner use into a small, prioritized next increment. This task can recur; it is not a one-time sign-off or release gate.

## Ownership

Trial scenarios and records under `docs/user-guides/prototype-trials/`, plus updates to the [decision register](../decisions.md). Coordinate contract changes with T00/T01 and behavioral changes with their task owners.

## Work steps

1. State the question being tested and the implemented scope before the session. Use the fictional example when live behavior is not available yet.
2. Prepare the scenario and artifact inputs; verify that no previous owner work will be overwritten.
3. Ask the owner to perform the task, noting where explanation or manual intervention is needed. Do not silently repair the workflow and report it as easy to use.
4. Record observed behavior, the owner's actual comments, and agent interpretations separately. Use anonymized excerpts rather than copying sensitive machine data into the repository.
5. Classify issues as correctness, comprehension, workflow friction, missing coverage, or presentation. Prioritize lost decisions, misleading coverage, and unsafe reuse before cosmetic changes.
6. Choose a small next experiment, update affected task acceptance cases, and record which decisions remain provisional.
7. After implementation, replay the same scenario to check whether it improved. Keep the observation history so regressions are visible.

## Acceptance cases

- Every trial record names the artifact/build, scenario, observed result, owner feedback, and next decision.
- Unavailable owner input is marked pending; the agent never invents validation or usability approval.
- A decision to change schema or matching rules identifies affected saved data and compatibility consequences.
- New collection items are added because they answer an observed inventory need, not because they appeared in the original candidate list.
- Completion of a trial does not trigger release/publication.

## Handoff

Provide a short backlog ordered by learning/correctness value, assign each item to an existing task owner or a new clearly scoped task, and state the next trial question. Stop adding features once the selected experiment is implementable; do not expand into an unbounded product roadmap.
