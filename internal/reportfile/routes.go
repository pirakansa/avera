package reportfile

import (
	"fmt"
	"strings"
	"time"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

// Route records a human-described forwarding relationship, never an observed path.
type Route struct {
	EntranceID          string `json:"entrance_id"`
	InternalID          string `json:"internal_id"`
	EntranceFingerprint string `json:"entrance_fingerprint"`
	InternalFingerprint string `json:"internal_fingerprint"`
	Explanation         string `json:"explanation"`
	Actor               string `json:"actor"`
	RecordedAt          string `json:"recorded_at"`
	SnapshotID          string `json:"snapshot_id"`
}

func digest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, r := range value[7:] {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}
func validateRoutes(routes []Route) error {
	if len(routes) > i.MaxEntries {
		return fmt.Errorf("route count limit exceeded")
	}
	for _, r := range routes {
		stamp, err := time.Parse(time.RFC3339, r.RecordedAt)
		if !digest(r.EntranceID) || !digest(r.InternalID) || r.EntranceID == r.InternalID || !digest(r.EntranceFingerprint) || !digest(r.InternalFingerprint) || strings.TrimSpace(r.Explanation) == "" || strings.TrimSpace(r.Actor) == "" || r.SnapshotID == "" || err != nil || stamp.UTC().Format(time.RFC3339) != r.RecordedAt {
			return fmt.Errorf("invalid human route")
		}
	}
	return nil
}

// RouteCurrent only matches saved observation bases; it does not verify forwarding.
func (f *File) RouteCurrent(r Route) bool {
	var entrance, internal *i.Finding
	for n := range f.Current.Findings {
		v := &f.Current.Findings[n]
		if v.FindingID == r.EntranceID {
			entrance = v
		}
		if v.FindingID == r.InternalID {
			internal = v
		}
	}
	return entrance != nil && internal != nil && entrance.Kind == i.Endpoint && !assessment.IsReference(*entrance) && assessment.IsReference(*internal) && entrance.FactFingerprint == r.EntranceFingerprint && internal.FactFingerprint == r.InternalFingerprint
}
func (f *File) applyRoutes(edit Edit, found i.Finding, now time.Time) error {
	// Missing field preserves links for callers that only update judgment text.
	if edit.RelatedIDs == nil {
		return nil
	}
	if found.Kind != i.Endpoint || assessment.IsReference(found) {
		if len(edit.RelatedIDs) > 0 {
			return fmt.Errorf("only network entrances can own routes")
		}
		return nil
	}
	if len(edit.RelatedIDs) > 0 && (strings.TrimSpace(edit.RouteExplanation) == "" || strings.TrimSpace(edit.Actor) == "") {
		return fmt.Errorf("route explanation and actor are required")
	}
	targets := map[string]i.Finding{}
	for _, v := range f.Current.Findings {
		if assessment.IsReference(v) {
			targets[v.FindingID] = v
		}
	}
	additions := []Route{}
	seen := map[string]bool{}
	for _, id := range edit.RelatedIDs {
		v, ok := targets[id]
		if !ok || seen[id] {
			return fmt.Errorf("select distinct current loopback targets")
		}
		seen[id] = true
		additions = append(additions, Route{EntranceID: found.FindingID, InternalID: id, EntranceFingerprint: found.FactFingerprint, InternalFingerprint: v.FactFingerprint, Explanation: edit.RouteExplanation, Actor: edit.Actor, RecordedAt: now.UTC().Format(time.RFC3339), SnapshotID: f.Current.SnapshotID})
	}
	keep := []Route{}
	for _, r := range f.Routes {
		if r.EntranceID == found.FindingID {
			f.RouteHistory = append(f.RouteHistory, r)
		} else {
			keep = append(keep, r)
		}
	}
	f.Routes = append(keep, additions...)
	return nil
}

// RelatedContextChanged is a portable-view reason, not a raw T00 review artifact code.
const RelatedContextChanged = "related_service_changed_or_unavailable"

// Review additionally gates entrance judgments on their human-linked observation bases.
// Stored decisions remain unchanged; stale relationships never imply current acceptance.
func (f *File) Review() (*assessment.Result, error) {
	result, err := assessment.Evaluate(f.Current, f.Previous, f.Assessments)
	if err != nil {
		return nil, err
	}
	stale := map[string]bool{}
	for _, route := range f.Routes {
		if !f.RouteCurrent(route) {
			stale[route.EntranceID] = true
		}
	}
	for n := range result.Review.Items {
		item := &result.Review.Items[n]
		if item.Presence != i.Current || !stale[item.FindingID] {
			continue
		}
		if item.Applicability != i.Conflict {
			item.Applicability = i.NeedsReview
		}
		item.CurrentDecision = i.Unreviewed
		item.ReasonCodes = append(item.ReasonCodes, RelatedContextChanged)
	}
	result.Review.Summary = assessment.Summarize(result.Review.Items, len(result.Review.UnmatchedAssessmentIDs))
	return result, nil
}
