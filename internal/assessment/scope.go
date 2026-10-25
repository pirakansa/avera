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

// ReviewSummary excludes current reference endpoints from human-decision counts.
// Raw T04 artifacts remain an inventory of all observations for compatibility.
func ReviewSummary(current *i.Snapshot, review *i.Review) (i.Summary, int) {
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
		focused = append(focused, item)
	}
	return Summarize(focused, len(review.UnmatchedAssessmentIDs)), count
}
