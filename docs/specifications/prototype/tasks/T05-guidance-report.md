# T05 — Guidance and the three-level report

Status: implementation complete; owner P1 walkthrough pending and scheduled together with T06 after its completion. See the [combined trial procedure](../../../user-guides/prototype-trials.md#executable-t05--t06-rehearsal). See the [API handoff](../guidance-report.md). Prerequisites: T01 model and the T04 result contract. Development can begin against T00/T04 fixtures. Read [product direction](../../product-direction.md).

## Outcome

Help a beginner understand what to explain, then give a less technical recipient an accurate summary with traceable detail. Render the same meaning from fictional and real collector outputs.

## Ownership

`internal/guidance`, `internal/report`, their embedded content/templates, and report tests. Coordinate semantic field/count changes with T01/T04. Do not create a separate set of matching or acceptance rules inside templates.

## Guidance work

1. For each implemented finding kind, write a short observed-fact description, why review is useful, and a small set of contextual questions.
2. Port prompts cover purpose, intended audience, allowed access, applicable safeguards, and supporting evidence. Public static content and external firewalls must be representable without insisting every service needs user authentication.
3. Sudo prompts cover who uses the configured privilege, the required function, limits on target users/commands, safeguards, and supporting evidence. Unsupported interpretation needs a clear next question, not invented technical conclusions.
4. Distinguish a missing explanation from a tool-verified defect. Plain field completeness checks may prompt the user but must not certify adequacy.
5. Keep guidance bundled into the executable. No remote text service, AI dependency, or external template download is needed to run the prototype.
6. Record the language used for the owner's trial and any comprehension issues. Repository documentation remains English; English/Japanese report presentation is now implemented following owner feedback; additional languages remain undecided.

## Rendering work

The proposed summary contains target/run identity, observed scope, collection limitations, current review-item counts, outstanding decisions, and changes. Keep historical/not-observed items separate from current findings.

Individual items contain observed facts, applicable questions, human explanation and decision, applicability/re-review reason, actor/date, and evidence references. Accepted items remain visible. If a prior decision is stale, label it as prior context instead of presenting it as current acceptance.

The evidence level contains source/method/time and bounded technical values/excerpts. Link each item to its evidence; link summary categories to detail where practical. A reader should not need to decode raw `/proc` data to understand the next action.

Render untrusted values safely: control characters, Markdown tables/fences, embedded HTML, and URL-like text must not change report structure or become executable content. In v0, external references may be inert text; rendering must not fetch them. Define and test a consistent escaping policy.

## Acceptance cases

- Summary totals reconcile with T04's reviewed item set, including conflict, re-review, and unmatched history cases.
- A failed source is visible in the summary even if other collectors found no items.
- A public HTTP example can describe why authentication is inapplicable without automatic rejection or automatic safety certification.
- A reported external firewall remains a human statement with a reference, not a newly verified fact.
- Each report item has a usable route to the underlying evidence and limitations.
- Malicious-looking annotation text remains data and cannot break table/fence structure or inject active markup.
- Rendering the same model is deterministic apart from explicitly supplied metadata; generating a report never changes input decisions.
- The owner can identify the next needed explanation during the P1 walkthrough; record this as trial evidence, not a claim made by the implementation agent.

## Handoff

Provide report examples, guidance content, escaping tests, supported language, and unresolved usability questions. HTML and Japanese report presentation were selected by owner feedback and implemented as a follow-up; PDF, dashboards, scoring, and approval workflows remain deferred.
