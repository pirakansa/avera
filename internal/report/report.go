// Package report renders deterministic offline Markdown and HTML review reports.
package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/pirakansa/avera/internal/assessment"
	"github.com/pirakansa/avera/internal/guidance"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

const technicalLimit = 2000

// Text encodes punctuation as numeric entities so untrusted values cannot form
// Markdown or HTML syntax, links, or fences. Controls are visible Unicode escapes.
func Text(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			fmt.Fprintf(&b, "&#92;u%04X", r)
		} else if r == ' ' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "&#%d;", r)
		}
	}
	return b.String()
}

func technical(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	runes := []rune(string(b))
	if len(runes) > technicalLimit {
		return string(runes[:technicalLimit]) + " [truncated; consult original artifact]", nil
	}
	return string(b), nil
}

type writer struct{ strings.Builder }

func (w *writer) field(label, value string) {
	fmt.Fprintf(&w.Builder, "- %s: %s\n", label, Text(value))
}
func (w *writer) annotation(a i.Assessment, context string) {
	w.field("Assessment context", context)
	w.field("Assessment ID", a.AssessmentID)
	w.field("Origin", string(a.Origin))
	w.field("Human decision", string(a.Decision))
	w.field("Actor / date", a.Actor+" / "+a.AssessedAt)
	w.field("Human purpose", a.Purpose)
	w.field("Human exposure", a.Exposure)
	w.field("Human safeguards (not tool verified)", a.Safeguards)
	w.field("Human rationale", a.Rationale)
	for _, ref := range a.EvidenceRefs {
		w.field("Human evidence reference (inert; not fetched)", ref.Reference+" / "+ref.Provenance)
	}
	if missing := guidance.Missing(a); len(missing) > 0 {
		w.field("Missing explanation (field completeness only; not a verified defect)", strings.Join(missing, ", "))
	}
}

// Render derives the T04 review from validated explicit inputs. It never fetches
// references, reads the host, modifies decisions, or supplies implicit timestamps.
func Render(current, previous *i.Snapshot, annotations *i.Assessments) (string, error) {
	result, err := assessment.Evaluate(current, previous, annotations)
	if err != nil {
		return "", err
	}
	r := result.Review
	var w writer
	w.WriteString("# AVERA review report\n\n## Summary\n\n")
	w.field("Target", current.TargetID)
	w.field("Run", current.SnapshotID)
	w.field("Collection start / finish", current.CollectedAt+" / "+current.FinishedAt)
	w.field("Tool version", current.ToolVersion)
	environment, err := technical(current.Environment)
	if err != nil {
		return "", err
	}
	w.field("Observed environment and execution scope", environment)
	if r.PreviousSnapshotID != nil {
		w.field("Previous run", *r.PreviousSnapshotID)
	} else {
		w.field("Comparison", "Unavailable; no previous snapshot supplied")
	}
	s, referenceCount := assessment.ReviewSummary(current, r)
	fmt.Fprintf(&w.Builder, "\n[Current items](#current-items): %d; accepted: %d; action required: %d; unreviewed: %d.\n\nNeeds re-review: %d; conflicts: %d (applicability subsets).\n\n[Historical items](#historical-items): %d; [unmatched assessments](#unmatched-assessments): %d.\n\n", s.Current, s.Accepted, s.ActionRequired, s.Unreviewed, s.NeedsReview, s.Conflict, s.Historical, s.UnmatchedAssessments)
	fmt.Fprintf(&w.Builder, "Local-only references (outside review counts): %d.\n\n", referenceCount)
	w.WriteString("Counts describe review state, not overall safety. Outstanding decisions are identified in the current item details. Missing observations do not prove absence or remediation.\n\n### Collection scope and limitations\n\n")
	for _, c := range current.Coverage {
		w.field("Collector / status / scope", string(c.Collector)+" / "+string(c.Status)+" / "+c.Scope)
		w.field("Collector version", c.CollectorVersion)
		w.field("Attempted sources", strings.Join(c.AttemptedSources, ", "))
		w.field("Diagnostics", strings.Join(c.DiagnosticCodes, ", "))
		w.field("Limitations", strings.Join(c.Limitations, "; "))
	}
	findings := map[string]i.Finding{}
	if previous != nil {
		for _, f := range previous.Findings {
			findings[f.FindingID] = f
		}
	}
	for _, f := range current.Findings {
		findings[f.FindingID] = f
	}
	annotationsByID := map[string]i.Assessment{}
	if annotations != nil {
		for _, a := range annotations.Entries {
			annotationsByID[a.AssessmentID] = a
		}
	}
	// Use source-qualified numeric anchors, never untrusted IDs in Markdown syntax.
	evidenceLinks := map[string]string{}
	sources := []*i.Snapshot{current}
	if previous != nil {
		sources = append(sources, previous)
	}
	for si, source := range sources {
		for ei, e := range source.Evidence {
			evidenceLinks[fmt.Sprintf("%d/%s", si, e.EvidenceID)] = fmt.Sprintf("evidence-%d-%d", si, ei)
		}
	}
	for _, category := range []string{"current", "historical", "reference"} {
		presence := i.Current
		if category == "historical" {
			presence = i.Historical
		}
		if category == "reference" {
			w.WriteString("\n## Local-only reference services\n\nOutside the main review counts, not automatically accepted. Explain forwarding at the network entrance; forwarding is not verified by this tool.\n\n")
		} else if presence == i.Current {
			w.WriteString("\n## Current items\n\n")
		} else {
			w.WriteString("\n## Historical items\n\nPrior observations only; excluded from current counts.\n\n")
		}
		for n, item := range r.Items {
			if item.Presence != presence {
				continue
			}
			f := findings[item.FindingID]
			if presence == i.Current && assessment.IsReference(f) != (category == "reference") {
				continue
			}
			if assessment.IsReference(f) {
				fmt.Fprintf(&w.Builder, "### Item %d\n\n", n+1)
				w.field("Finding ID", f.FindingID)
				facts, err := technical(struct {
					Identity i.Identity
					Facts    i.Facts
				}{f.Identity, f.Facts})
				if err != nil {
					return "", err
				}
				w.field("Observed values (bounded)", facts)
				w.field("Change", string(item.Change))
				for _, id := range item.AssessmentIDs {
					w.annotation(annotationsByID[id], "Stored judgment outside the main review scope")
				}
				sourceIndex := 0
				if item.Presence == i.Historical {
					sourceIndex = 1
				}
				for _, id := range f.EvidenceIDs {
					fmt.Fprintf(&w.Builder, "- [Underlying evidence](#%s)\n", evidenceLinks[fmt.Sprintf("%d/%s", sourceIndex, id)])
				}
				continue
			}
			fmt.Fprintf(&w.Builder, "### Item %d\n\n", n+1)
			w.field("Finding ID", item.FindingID)
			w.field("Change", string(item.Change))
			w.field("Current decision", string(item.CurrentDecision))
			w.field("Applicability", string(item.Applicability))
			w.field("Review reason", reasonText(item.ReasonCodes))
			g := guidance.For(f.Kind)
			w.field("Observed fact", g.Observed)
			facts, err := technical(struct {
				Identity i.Identity
				Facts    i.Facts
			}{f.Identity, f.Facts})
			if err != nil {
				return "", err
			}
			w.field("Observed values (bounded)", facts)
			w.field("Why review", g.Why)
			for _, q := range g.Questions {
				w.field("Question", q)
			}
			if len(item.AssessmentIDs) == 0 {
				w.WriteString("- Next explanation: provide purpose, audience or privilege users, safeguards, rationale, and supporting evidence.\n")
			}
			for _, id := range item.AssessmentIDs {
				context := "Prior context only; does not establish current acceptance"
				if item.Applicability == i.Applicable && presence == i.Current {
					context = "Applicable human assessment; not a tool safety certification"
				}
				w.annotation(annotationsByID[id], context)
			}
			sourceIndex := 0
			if presence == i.Historical {
				sourceIndex = 1
			}
			for _, id := range f.EvidenceIDs {
				fmt.Fprintf(&w.Builder, "- [Underlying evidence](#%s)\n", evidenceLinks[fmt.Sprintf("%d/%s", sourceIndex, id)])
			}
			w.WriteString("- [Collection limitations](#collection-scope-and-limitations) apply; evidence below retains source run context.\n\n")
		}
	}
	w.WriteString("## Unmatched assessments\n\nHistorical context only; excluded from current decisions.\n\n")
	for _, id := range r.UnmatchedAssessmentIDs {
		w.annotation(annotationsByID[id], "Unmatched prior context")
	}
	w.WriteString("\n## Evidence\n\nTechnical values are bounded to 2000 Unicode characters per value; truncated values require the original artifact. External references are human statements and are never fetched.\n\n")
	for si, source := range sources {
		for ei, e := range source.Evidence {
			fmt.Fprintf(&w.Builder, "### Evidence %d %d\n\n", si, ei)
			w.field("Source run", source.SnapshotID)
			w.field("Evidence ID", e.EvidenceID)
			w.field("Source / location", e.SourceKind+" / "+e.Location)
			w.field("Method / time", e.Method+" / "+e.CollectedAt)
			value, err := technical(e.Values)
			if err != nil {
				return "", err
			}
			w.field("Technical values", value)
			for _, c := range source.Coverage {
				w.field("Source run coverage / limitations", string(c.Collector)+" / "+string(c.Status)+" / "+strings.Join(c.Limitations, "; "))
			}
			w.WriteString("\n")
		}
	}
	return w.String(), nil
}

// reasonText translates T04 diagnostics without introducing matching rules.
func reasonText(codes []string) string {
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		wording := map[string]string{
			reportfile.RelatedContextChanged: "A linked local service changed or is unavailable; review the entrance judgment and path explanation",
			assessment.FactsChanged:          "Observed facts changed; ask the decision owner to review the earlier explanation",
			assessment.CoverageIncomplete:    "Collection or attribution is incomplete; clarify the missing context before reusing a decision",
			assessment.MultipleAssessments:   "Multiple assessments conflict; ask their owners to resolve which judgment applies",
			assessment.ExpectationMismatch:   "The advance expectation is incomplete or differs from observations; review it with its author",
			assessment.NotObserved:           "Not observed in this comparable run; this does not establish remediation",
		}[code]
		if wording == "" {
			wording = "Ask the reviewer to interpret this diagnostic"
		}
		out = append(out, wording+" ("+code+")")
	}
	return strings.Join(out, "; ")
}
