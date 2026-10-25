# Prototype retirement candidates

This register records features or persisted settings that may no longer justify
their complexity after owner feedback. An entry is a candidate, not authorization
to delete stored data or change collection behavior. Record implemented reductions
separately from possible future removal.

## RC01 — superseded human judgment and relationship history

Status: hidden from GUI and rendered reports; JSON retention unchanged.
Owner direction: 2026-10-25.

The primary workflow is to explain the current inventory using common service
settings. A reviewer can reopen an item and update its current explanation or
decision. Routine display of replaced judgments adds ambiguity about which reason
is current, while audit/rollback history is not a stated primary requirement.

Implemented boundary:

- Remove the replaced-judgment history section and superseded relationship entries
  from the browser. The browser state response omits archive payloads.
- Omit superseded judgment and relationship history from HTML and Markdown reports
  and their previews. Continue showing current judgments, including accepted
  (Japanese: "追加対応不要") items and their purposes, safeguards and rationale.
- Preserve `assessment_history` and `route_history` in portable JSON, including
  file saves/downloads, read validation and repeat collection. Keep current
  matching, collection, archive writes and the portable schema unchanged.
- Keep previous observations and unmatched active explanations separate from
  replaced history; they are not removed by this change.

Possible later simplification: stop recording/retaining replacement history and
remove the associated model/storage/settings code. Reassess actual owner need for
rollback or explaining past decisions before that step. Any removal needs an
explicit compatibility/migration decision for existing files and tests proving
that current explanations and reusable judgments survive. No JSON history is
silently trimmed or discarded now; the existing aggregate size limit still applies.

Evidence: renderer tests cover English/Japanese HTML and Markdown with distinct
old/current markers and unchanged source bytes. HTTP/browser tests cover hidden
archive sections/state, retained JSON archives and visible current accepted
judgments. Collection and archive-storage tests remain in the full quality gates.
