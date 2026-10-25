package assessment

import (
	i "github.com/pirakansa/avera/internal/inventory"
	"net/netip"
)

// IsReference identifies loopback-only endpoints, not LAN/private/wildcard listeners.
// Unknown addresses remain in the main review scope. This is not a safety verdict.
func IsReference(f i.Finding) bool {
	if f.Collector != i.Network || f.Kind != i.Endpoint {
		return false
	}
	endpoint, ok := f.Identity.(i.EndpointIdentity)
	if !ok {
		return false
	}
	address, err := netip.ParseAddr(endpoint.LocalAddress)
	return err == nil && address.Unmap().IsLoopback()
}

// UserCandidateUID is a review heuristic, never a statement about login access.
const UserCandidateUID uint32 = 1000

// AccountReferences retains low-UID system accounts outside routine judgments,
// unless an in-scope observed endpoint identifies them by process effective UID.
// Socket UID alone and sudo run-as targets do not promote an account into scope.
func AccountReferences(current *i.Snapshot) map[string]bool {
	evidence := map[string]i.ProcessValues{}
	for _, entry := range current.Evidence {
		if value, ok := entry.Values.(i.ProcessValues); ok {
			evidence[entry.EvidenceID] = value
		}
	}
	serviceUsers := map[uint32]bool{}
	for _, finding := range current.Findings {
		if finding.Kind != i.Endpoint || IsReference(finding) {
			continue
		}
		for _, id := range finding.EvidenceIDs {
			value := evidence[id]
			if value.Service != nil && value.Service.EffectiveUID != nil {
				serviceUsers[*value.Service.EffectiveUID] = true
			}
		}
	}
	out := map[string]bool{}
	for _, finding := range current.Findings {
		if facts, ok := finding.Facts.(i.AccountFacts); ok && facts.UID < UserCandidateUID && !serviceUsers[facts.UID] {
			out[finding.FindingID] = true
		}
	}
	return out
}

// ReviewSummary excludes current reference endpoints and system accounts from human-decision counts.
// Raw T04 artifacts remain an inventory of all observations for compatibility.
func ReviewSummary(current *i.Snapshot, review *i.Review) (i.Summary, int) {
	accountReferences := AccountReferences(current)
	references := map[string]bool{}
	for _, f := range current.Findings {
		if IsReference(f) {
			references[f.FindingID] = true
		}
	}
	focused := make([]i.ReviewItem, 0, len(review.Items))
	count := 0
	for _, item := range review.Items {
		if item.Presence == i.Current && references[item.FindingID] {
			count++
			continue
		}
		if item.Presence == i.Current && accountReferences[item.FindingID] {
			continue
		}
		focused = append(focused, item)
	}
	return Summarize(focused, len(review.UnmatchedAssessmentIDs)), count
}
