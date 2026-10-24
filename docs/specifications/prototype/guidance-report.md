# Guidance and Markdown report API

T05 provides bundled English guidance in `internal/guidance` and a deterministic,
offline Markdown renderer in `internal/report`. CLI integration belongs to T06.

## API and semantics

`guidance.For(kind)` returns observed-fact wording, review motivation, and contextual
questions for `network.endpoint`, `sudo.rule`, and `sudo.unparsed`. Each call owns
its question slice. Public static content can explain why authentication is
inapplicable; unsupported sudo text asks for interpretation instead of inferring
privileges. `guidance.Missing(assessment)` reports blank explanatory fields and
missing reference entries only. It never evaluates adequacy or changes decisions.

`report.Render(current, previous, assessments) (string, error)` takes explicit
snapshot and annotation inputs and invokes T04 `assessment.Evaluate`. Previous
snapshot and assessments are optional. Incompatible inputs return an error.
There is no independent matching, acceptance, or summary policy in presentation.
The renderer has no host access, clock, remote service, or reference fetching.
The same input produces identical output without altering decisions or provenance.

The summary retains target/run identity, environment and execution scope,
collection times, tool version, previous run, T04 counts, collector statuses,
attempted sources, diagnostics, and limitations. Current decision counts partition
current findings; re-review and conflict are separate applicability subsets.
Historical findings and unmatched assessments appear separately.

Item details preserve observed values, guidance, change, applicability and readable
review reasons, human explanations, origin, decision, actor/date and references.
Only applicable current annotations are labeled applicable. Stale/conflicting
annotations remain prior context. Accepted items remain present. Historical
annotations appear in the unmatched section with their IDs rather than becoming
current decisions. Evidence links resolve to the appropriate current or previous
run. Evidence retains source, location, method, timestamp, bounded values, and
source-run coverage limitations. Human safeguards and references remain statements;
rendering does not independently verify an external firewall.

## Escaping and bounds

Every dynamic string passes through `report.Text`. Only ASCII letters, digits,
and spaces remain literal. All other printable characters become decimal numeric
entities. Control, formatting, and Unicode line/paragraph separator characters
become visible Unicode escape text. Newlines in values therefore cannot introduce
Markdown blocks. Markdown syntax, HTML, URL punctuation, and entity syntax from
input cannot form active markup or links. Render with a standard CommonMark
renderer; do not decode entities before parsing Markdown. Only fixed headings and
generated numeric evidence anchors enter Markdown structure directly. This policy
also keeps external references inert. The report performs no downloads.

Observed identity/facts and evidence values are JSON text, bounded to 2000 Unicode
characters per value before escaping. Truncation is explicitly labeled and points
to the original artifact. Human explanations are escaped but not truncated.
The report is a communication artifact, not a replacement for immutable inputs.

## Verification and handoff

`internal/report/report_test.go` exercises all 13 T00/T04 scenarios, reconciles
summary counts against fixture reviews, checks stale-context labeling, coverage,
evidence anchors, deterministic output, input immutability, hostile annotations,
technical bounds, and incompatible targets. Guidance tests cover all implemented
kinds, independent content slices, blank-field prompts, and public service context.
The [generated example](examples/generated-guidance-report.md) uses fictional
contract fixtures: `snapshot-repeat.json`, `snapshot-first.json`, and
`assessments-reviewed.json`. It is renderer output, not owner trial evidence.

Supported trial content language: English. No owner P1 walkthrough has been
performed as part of implementation; trial language actually used and comprehension
issues remain unrecorded until the owner tries it. Record them in the existing
[trial workflow](../../user-guides/prototype-trials.md). Open usability questions:
can the owner identify the next explanation, understand prior-context labels and
source limitations, and navigate technical evidence without reading raw proc data?
HTML/PDF output, dashboards, scoring, and approval workflows remain deferred.

## Implementation validation record

Before changes, setup, check, test, and build succeeded. After changes,
`vorbere run fmt`, `vorbere run vet`, `vorbere run staticcheck`,
`vorbere run test` (race detector), `vorbere run build`, and `git diff --check`
succeeded. After the owner authorized network access, the final
`vorbere run check` also succeeded. Govulncheck reported zero vulnerabilities
affecting called code, with informational findings of one vulnerability in
imported packages and thirteen in required modules whose vulnerable code does
not appear to be called. Generated binaries were removed. No owner usability
trial or CLI integration was performed.
