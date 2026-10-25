// Package assessment derives deterministic reviews from explicit inventory inputs.
// It performs no I/O and never changes a human assessment's basis or provenance.
package assessment

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
)

const (
	FactsChanged        = "facts_changed"
	CoverageIncomplete  = "coverage_incomplete"
	MultipleAssessments = "multiple_assessments"
	ExpectationMismatch = "expectation_mismatch_or_incomplete"
	NotObserved         = "not_observed"
)

// Result separates editable current entries from unmatched historical context.
// Existing entries are preserved verbatim; generated entries are unreviewed.
type Result struct {
	Review    *i.Review
	Editable  *i.Assessments
	Unmatched *i.Assessments
}

// Evaluate validates all inputs before deriving any output. Nil previous and
// assessments inputs represent a first run without that context.
func Evaluate(current, previous *i.Snapshot, assessments *i.Assessments) (*Result, error) {
	if err := i.ValidateInputs(current, previous, assessments, nil); err != nil {
		return nil, err
	}
	// Encode/Decode also validates programmatically constructed values and gives
	// the result ownership of every mutable slice without modifying caller inputs.
	if assessments == nil {
		assessments = &i.Assessments{Envelope: envelope(current, "assessments"), Entries: []i.Assessment{}}
	}
	data, err := i.Encode(assessments)
	if err != nil {
		return nil, err
	}
	artifact, err := i.Decode(data)
	if err != nil {
		return nil, err
	}
	assessments = artifact.(*i.Assessments)
	r := &i.Review{Envelope: envelope(current, "review"), SnapshotID: current.SnapshotID, AssessmentIDs: []string{}, CoverageStatuses: map[i.Collector]i.Status{}, Items: []i.ReviewItem{}, UnmatchedAssessmentIDs: []string{}}
	if previous != nil {
		id := previous.SnapshotID
		r.PreviousSnapshotID = &id
	}
	for _, c := range current.Coverage {
		r.CoverageStatuses[c.Collector] = c.Status
	}
	out := &Result{Review: r, Editable: &i.Assessments{Envelope: envelope(current, "assessments"), Entries: []i.Assessment{}}, Unmatched: &i.Assessments{Envelope: envelope(current, "assessments"), Entries: []i.Assessment{}}}
	matches := map[string][]i.Assessment{}
	used := map[string]bool{}
	for _, a := range assessments.Entries {
		id, err := selectorID(current.TargetID, a.Selector)
		if err != nil {
			return nil, err
		}
		matches[id] = append(matches[id], a)
		used[a.AssessmentID] = true
		r.AssessmentIDs = append(r.AssessmentIDs, a.AssessmentID)
	}
	prior := map[string]i.Finding{}
	if previous != nil {
		for _, f := range previous.Findings {
			prior[f.FindingID] = f
		}
	}
	currentIDs := map[string]bool{}
	for _, f := range current.Findings {
		currentIDs[f.FindingID] = true
		item := i.ReviewItem{FindingID: f.FindingID, Presence: i.Current, Change: i.NotCompared, Applicability: i.None, AssessmentIDs: []string{}, CurrentDecision: i.Unreviewed, ReasonCodes: []string{}}
		sufficient := reusable(current, f)
		if previous != nil {
			old, ok := prior[f.FindingID]
			switch {
			case !ok:
				item.Change = i.New
			case old.FactFingerprint != f.FactFingerprint:
				item.Change = i.Changed
			case sufficient && reusable(previous, old) && comparable(current, previous, f.Collector):
				item.Change = i.Unchanged
			default:
				item.Change = i.Unknown
			}
		}
		entries := matches[f.FindingID]
		for _, a := range entries {
			item.AssessmentIDs = append(item.AssessmentIDs, a.AssessmentID)
			out.Editable.Entries = append(out.Editable.Entries, a)
		}
		switch len(entries) {
		case 0:
			id := "starter-" + f.FindingID
			for used[id] {
				id += "-new"
			}
			used[id] = true
			out.Editable.Entries = append(out.Editable.Entries, i.Assessment{AssessmentID: id, Origin: i.ReviewedSnapshot, Selector: i.Selector{Collector: f.Collector, Kind: f.Kind, Identity: f.Identity}, Basis: i.ReviewedBasis{SnapshotID: current.SnapshotID, FactFingerprint: f.FactFingerprint}, EvidenceRefs: []i.EvidenceReference{}, Decision: i.Unreviewed})
		case 1:
			a := entries[0]
			item.Applicability = i.NeedsReview
			reason := FactsChanged
			equal := false
			switch b := a.Basis.(type) {
			case i.ReviewedBasis:
				equal = b.FactFingerprint == f.FactFingerprint
			case i.AdvanceBasis:
				reason = ExpectationMismatch
				equal = expectationMatches(current.TargetID, f, b.Expectation)
			}
			if !sufficient {
				reason = CoverageIncomplete
			}
			if sufficient && equal {
				item.Applicability = i.Applicable
				item.CurrentDecision = a.Decision
			} else {
				item.ReasonCodes = append(item.ReasonCodes, reason)
			}
		default:
			item.Applicability = i.Conflict
			item.ReasonCodes = append(item.ReasonCodes, MultipleAssessments)
		}
		slices.Sort(item.AssessmentIDs)
		r.Items = append(r.Items, item)
	}
	if previous != nil {
		for _, f := range previous.Findings {
			if currentIDs[f.FindingID] {
				continue
			}
			change, reason := i.Unknown, CoverageIncomplete
			if comparable(current, previous, f.Collector) && coverageComplete(current, f.Collector) {
				change, reason = i.NotObserved, NotObserved
			}
			r.Items = append(r.Items, i.ReviewItem{FindingID: f.FindingID, Presence: i.Historical, Change: change, Applicability: i.NeedsReview, AssessmentIDs: []string{}, CurrentDecision: i.Unreviewed, ReasonCodes: []string{reason}})
		}
	}
	for id, entries := range matches {
		if currentIDs[id] {
			continue
		}
		for _, a := range entries {
			r.UnmatchedAssessmentIDs = append(r.UnmatchedAssessmentIDs, a.AssessmentID)
			out.Unmatched.Entries = append(out.Unmatched.Entries, a)
		}
	}
	slices.Sort(r.AssessmentIDs)
	slices.Sort(r.UnmatchedAssessmentIDs)
	slices.SortFunc(r.Items, func(a, b i.ReviewItem) int { return strings.Compare(a.FindingID, b.FindingID) })
	for _, a := range []*i.Assessments{out.Editable, out.Unmatched} {
		slices.SortFunc(a.Entries, func(a, b i.Assessment) int { return strings.Compare(a.AssessmentID, b.AssessmentID) })
	}
	r.Summary = Summarize(r.Items, len(r.UnmatchedAssessmentIDs))
	if err := i.ValidateInputs(current, previous, assessments, r); err != nil {
		return nil, err
	}
	return out, nil
}

func envelope(s *i.Snapshot, kind string) i.Envelope { e := s.Envelope; e.Kind = kind; return e }
func selectorID(target string, s i.Selector) (string, error) {
	id, err := i.NormalizeIdentity(s.Kind, s.Identity)
	if err != nil {
		return "", err
	}
	b, err := i.Canonical([]any{i.IdentityVersion, target, s.Collector, s.Kind, id})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b)), nil
}
func coverage(s *i.Snapshot, c i.Collector) (i.Coverage, bool) {
	for _, v := range s.Coverage {
		if v.Collector == c {
			return v, true
		}
	}
	return i.Coverage{}, false
}
func coverageComplete(s *i.Snapshot, c i.Collector) bool {
	v, ok := coverage(s, c)
	return ok && v.Status == i.Complete
}
func comparable(a, b *i.Snapshot, c i.Collector) bool {
	x, ok := coverage(a, c)
	y, other := coverage(b, c)
	return ok && other && x.Scope == y.Scope
}
func reusable(s *i.Snapshot, f i.Finding) bool {
	if !coverageComplete(s, f.Collector) {
		return false
	}
	switch x := f.Facts.(type) {
	case i.USBFacts:
		if x.Input != nil && x.Input.Status != i.Complete {
			return false
		}
		if len(x.StorageFingerprints) > 0 && !coverageComplete(s, i.Storage) {
			return false
		}
		for _, other := range s.Findings {
			if facts, ok := other.Facts.(i.StorageFacts); ok && slices.Contains(x.StorageFingerprints, other.FactFingerprint) && !facts.ObservationComplete {
				return false
			}
		}
		return x.ObservationComplete
	case i.StorageFacts:
		return x.ObservationComplete
	case i.AccountFacts:
		return x.ContextComplete
	case i.ForwardingFacts:
		return true
	case i.NetworkFacts:
		return x.AttributionComplete
	case i.SudoFacts:
		return x.LocalContextComplete && x.InterpretationComplete
	}
	return false
}

// Summarize counts only current decisions and applicability; history is separate.
func Summarize(items []i.ReviewItem, unmatched int) i.Summary {
	s := i.Summary{UnmatchedAssessments: unmatched}
	for _, v := range items {
		if v.Presence == i.Historical {
			s.Historical++
			continue
		}
		s.Current++
		switch v.CurrentDecision {
		case i.Accepted:
			s.Accepted++
		case i.ActionRequired:
			s.ActionRequired++
		default:
			s.Unreviewed++
		}
		switch v.Applicability {
		case i.NeedsReview:
			s.NeedsReview++
		case i.Conflict:
			s.Conflict++
		}
	}
	return s
}
