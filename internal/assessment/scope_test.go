package assessment

import (
	i "github.com/pirakansa/avera/internal/inventory"
	"testing"
)

func TestReferenceScope(t *testing.T) {
	for _, tc := range []struct {
		address   string
		reference bool
	}{{"127.0.0.1", true}, {"127.3.2.1", true}, {"::1", true}, {"::ffff:127.0.0.1", true}, {"0.0.0.0", false}, {"::", false}, {"192.168.1.10", false}, {"10.0.0.1", false}, {"fe80::1", false}, {"not-an-address", false}} {
		f := i.Finding{Collector: i.Network, Kind: i.Endpoint, Identity: i.EndpointIdentity{LocalAddress: tc.address}}
		if IsReference(f) != tc.reference {
			t.Fatalf("wrong scope for %s", tc.address)
		}
	}
	if IsReference(i.Finding{Collector: i.Sudo, Kind: i.Rule}) {
		t.Fatal("sudo excluded from review")
	}
}
func TestFocusCountsDoNotCertifyReferenceSafety(t *testing.T) {
	s := &i.Snapshot{Findings: []i.Finding{{FindingID: "local", Collector: i.Network, Kind: i.Endpoint, Identity: i.EndpointIdentity{LocalAddress: "127.0.0.1"}}, {FindingID: "lan", Collector: i.Network, Kind: i.Endpoint, Identity: i.EndpointIdentity{LocalAddress: "192.168.1.1"}}}}
	for _, decision := range []i.Decision{i.Unreviewed, i.Accepted, i.ActionRequired} {
		raw := &i.Review{Items: []i.ReviewItem{{FindingID: "local", Presence: i.Current, CurrentDecision: decision}, {FindingID: "lan", Presence: i.Current, CurrentDecision: i.Unreviewed}}}
		summary, references := ReviewSummary(s, raw)
		if references != 1 || summary.Current != 1 || summary.Unreviewed != 1 || summary.Accepted != 0 || summary.ActionRequired != 0 {
			t.Fatalf("wrong focused counts: %+v", summary)
		}
		if raw.Items[0].CurrentDecision != decision {
			t.Fatal("classification rewrote a human judgment")
		}
	}
}
