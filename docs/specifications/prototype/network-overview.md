# Network configuration overview

Status: implemented from the owner's request for a graphical aid in the GUI and
recipient-facing report. This is a current single-PC configuration view, not an
inferred complete network topology or proof of reachability.

## Display contract

The editor and standalone HTML report show the same presentation model, built by
`internal/report.NetworkMapFor` from current network findings, derived judgments,
configured destination matches and active human-described relationships.

- The host boundary uses the report's logical target/group label. It represents
  the currently observed PC, not all hosts in that group.
- An upstream note invites reviewers to include router, load-balancer and cloud
  safeguards in the judgment rationale. The introduction identifies the diagram
  as settings observed within the PC; upstream inspection is outside tool scope,
  not an outstanding collection task. No actual upstream device, Internet path or protection is
  invented from a wildcard/LAN listener or a human safeguard statement.
- Network endpoint cards show transport, bind address, port, observed executable
  attribution when available, current decision and abbreviated purpose/safeguards.
  Safeguards are explicitly human statements. Action-required and unreviewed
  cards use the same red/amber emphasis as the summary.
- Forwarding cards show the source command/ordinal and observed rule text. A
  solid arrow means literal configured DNAT destination matching as defined in
  [shared service judgments](shared-service-judgments.md#forwarding-interaction).
  It does not establish filtering outcome or actual packet delivery.
- A dashed arrow means an active human-described relationship. Changed or missing
  observations are labeled for recheck. A missing service has a visibly unobserved
  placeholder, never a fabricated observed endpoint. Observed and human links
  remain separate even when they have the same source/destination.
- Without a determined relationship, show that the corresponding observed service
  is unidentified, without drawing a connection. Unconnected loopback services
  remain in the reference area; they are not automatically accepted or omitted.
- Collection status and diagnostic codes remain visible. Empty observations do
  not imply absence; a loopback-only result is distinguished from no observations.
- Current item cards link to their full judgment/evidence sections. Editor links
  open the matching details element. Long labels are abbreviated to 200 Unicode
  characters with an ellipsis; complete values remain in item details/JSON.
- Sort by source title and relation kind for stable display. Bound each diagram
  row list, reference list and per-row target list to 64 entries, with an explicit
  omitted-count notice directing users to the full inventory.

The graph contains no superseded judgment/relationship history. It does not
change matching, counts, collection, saved JSON or the portable schema. A fresh
view is recomputed after saves so common judgment changes appear in the graph.

## Rendering and export

The GUI uses DOM text properties and fixed fragment links. HTML export uses the
existing escaped text templates; shared bundled CSS draws cards and arrows.
No remote resources, JavaScript diagram library, image generation, external font
or new runtime/module dependency is required. The exported diagram works offline
and has a print layout. Narrow screens stack each row vertically. Markdown
remains a textual report; diagram export is HTML.

The editor state API adds the derived `Network` view model; downloaded report
JSON retains its existing schema and contents. Legacy HTML reports built from
split saved artifacts also show observed network configuration, without inventing
portable-only human links.

## Verification

Model tests cover literal configured and human relationships, accepted decisions,
human safeguard provenance, unknown/empty/partial views, explicit limits, escaped
content and unchanged source JSON. Existing report scenario tests verify local
fragment destinations and interpolation escaping. Browser checks verify the
editor/export diagram, reference services, item navigation, save refresh and
narrow layouts. No remote reachability or upstream configuration validation is
claimed by these checks.

Port-based service enrichment adds abbreviated process/service-candidate and
effective-UID observations to endpoint cards when present. These are derived
from the endpoint's linked proc evidence, not guessed from its port number.
Legacy reports retain executable-only cards. Full observations, explicit unknowns
and attribution diagnostics are available in item details; see
[listening services](listening-services.md).
