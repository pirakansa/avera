# Prototype trial cycles

Status: planned trial procedure. The commands described in the [architecture proposal](../specifications/prototype/architecture.md) are not implemented yet. This guide becomes an executable walkthrough increment by increment; do not present proposed examples as available functionality.

## Purpose

The project owner is the first test user. The goal is to learn how a person inventories concerns, supplies explanations, reuses prior work, and reports results. The trial is not a declaration of product safety or release readiness.

Keep each cycle small: choose a question, prepare a scenario, let the owner use it, record what actually happened, change one coherent part, and repeat.

## P0 — Read a concrete fictional example

Inputs: T00's fictional target, one public HTTP service, one undecided local endpoint, an example privilege rule, and an incomplete source.

Owner tasks:

1. Explain what the tool observed and what it does not know.
2. Locate the evidence for one item.
3. Describe why the endpoint exists and what further information is needed.
4. Identify what a manager would learn from the summary.
5. Identify which statements came from the person rather than the tool.

Record confusing labels, unnecessary questions, missing context, and misleading summaries. Adjust the sample and contract only where this trial provides a reason. P0 should not become a lengthy document approval process.

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

After P1, decide whether to keep separate files/commands for the next cycle. Consider a simpler editing surface only if the trial demonstrates that JSON or artifact movement is a problem. Do not assume a GUI or HTML is necessary before observing the difficulty.

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

Proceed when the owner has had a useful opportunity to try the current slice, important correctness problems are understood, and the next change has a clear purpose. Time estimates and fixed release dates are unnecessary at this stage.

If a live target or owner feedback is unavailable, continue independent fixture/unit work and mark the trial pending. Do not fabricate approval or imply the prototype was tested by its intended user.

Before any future public release, prepare a separate scope/readiness decision. This guide authorizes neither release publication nor submission of a security report to third parties.
