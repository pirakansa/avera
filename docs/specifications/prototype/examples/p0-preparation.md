# P0 preparation — optional owner walkthrough

The fictional material uses contract revision `t00-2026-10-24.1`, prepared by an agent on 2026-10-24. No collector or proposed CLI command was executed. The owner has requested a simpler, task-oriented walkthrough and a handoff to T01; no completed usability trial is claimed.

**T01 can proceed with the prepared experimental baseline. Answers to these questions, a completed P0 session, and live Ubuntu target details are not prerequisites for T01.** Editing methods and detailed presentation will be tried in the executable prototype.

## What the owner is being asked to try

Start with one HTTP example from the [fictional report](fictional-report.md#http-endpoint-intended-public-static-documentation): a service accepts connections on port 80. The person explains its purpose, relevant protections, and why they accept the remaining risk. On a later collection the service runs as root instead of its previous service user. The previous explanation remains available, but the changed situation needs a new judgment.

Use that example to discuss only these three questions:

1. **If you received this result, would you know what to check and what explanation to write?** Point out any missing help, irrelevant question, or information you would have to obtain elsewhere.
2. **Can you find your previous explanation and tell what needs another look this time?** Explain where the presentation loses your work or makes the next action unclear.
3. **If you gave this report to your manager or requester, what would it explain, and what would still be unclear?** Look at the summary first and follow one detail/evidence link if useful.

“I cannot tell until I try it” is a useful response. It becomes a question for P1, not a demand for an immediate design decision. The owner's task is to evaluate whether the workflow helps them do the work; it is not to learn internal state names, inspect hash values, validate every JSON file, or approve an editing format in advance.

## What remains with the implementation agents

| Concern | Responsibility / next point |
|---|---|
| Field definitions, hashes, references, and artifact consistency | T01 uses the [contract and fixtures](../../../../test/fixtures/prototype/contracts/README.md) and verifies the baseline |
| Exact advance expectations and conflicting or stale decisions | T04 implements the experimental rules and tests |
| Internal states and their user-facing wording | T04/T05 make the next action understandable without teaching the internal model |
| Helpful prompts and report presentation | T05 prepares them; P1 use supplies feedback |
| JSON editing and replacing a judgment | Temporary implementation choice to try in P1, not an owner approval prerequisite |
| Disposable Ubuntu release and architecture | Gather separately before live validation; unnecessary for T01's fixture-based work |

The [input manifest](../../../../test/fixtures/prototype/contracts/scenarios.json) and the report's full scenario/count tables remain technical reference material. They are not an owner questionnaire. Technical consistency checks are not usability feedback or source-format validation.

## Recording feedback and proceeding

Use the [trial guide](../../../user-guides/prototype-trials.md#p0--read-a-concrete-fictional-example) to record actual observations when a walkthrough happens. The session may be deferred until a useful executable slice exists. Keep unanswered usability questions pending while implementation continues.

The contract revision, fixture semantics, and E01–E10 experiment defaults are unchanged by this questionnaire adjustment. A later semantic change requires coordinated updates to the contract, vectors, scenarios, and affected tasks. Proceed to [T01](../tasks/T01-model-storage.md) now; do not infer that this means the owner has validated the fictional report or approved a final product UI.
