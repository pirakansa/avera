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

func TestAccountReviewScopeUsesRuntimeServiceUID(t *testing.T) {
	root, web := uint32(0), uint32(33)
	snapshot := &i.Snapshot{Findings: []i.Finding{
		{FindingID: "root", Kind: i.Account, Facts: i.AccountFacts{UID: 0}},
		{FindingID: "web", Kind: i.Account, Facts: i.AccountFacts{UID: 33}},
		{FindingID: "system", Kind: i.Account, Facts: i.AccountFacts{UID: 999}},
		{FindingID: "operator", Kind: i.Account, Facts: i.AccountFacts{UID: 1000}},
		{FindingID: "web-listener", Collector: i.Network, Kind: i.Endpoint, Identity: i.EndpointIdentity{LocalAddress: "0.0.0.0"}, EvidenceIDs: []string{"web-process"}},
		{FindingID: "local-listener", Collector: i.Network, Kind: i.Endpoint, Identity: i.EndpointIdentity{LocalAddress: "127.0.0.1"}, EvidenceIDs: []string{"root-process"}},
	}, Evidence: []i.Evidence{
		{EvidenceID: "web-process", Values: i.ProcessValues{Observed: i.NetworkFacts{Owners: []i.Owner{{UID: 0}}}, Service: &i.ServiceObservation{EffectiveUID: &web, EffectiveGID: &root}}},
		{EvidenceID: "root-process", Values: i.ProcessValues{Service: &i.ServiceObservation{EffectiveUID: &root}}},
	}}
	references := AccountReferences(snapshot)
	if !references["root"] || !references["system"] || references["web"] || references["operator"] {
		t.Fatal("wrong account scope", references)
	}
	// A numeric socket UID/GID does not substitute for the process effective UID.
	snapshot.Evidence[0].Values = i.ProcessValues{Observed: i.NetworkFacts{Owners: []i.Owner{{UID: 0}}}}
	if !AccountReferences(snapshot)["root"] || !AccountReferences(snapshot)["web"] {
		t.Fatal("socket UID promoted an account")
	}
	snapshot.Evidence[0].Values = i.ProcessValues{Service: &i.ServiceObservation{EffectiveUID: &root}}
	if AccountReferences(snapshot)["root"] {
		t.Fatal("root service account omitted")
	}
}

func TestSystemAccountDecisionsAreOutsideFocusedCounts(t *testing.T) {
	snapshot := &i.Snapshot{Findings: []i.Finding{{FindingID: "root", Kind: i.Account, Facts: i.AccountFacts{UID: 0}}, {FindingID: "operator", Kind: i.Account, Facts: i.AccountFacts{UID: 1000}}}}
	for _, decision := range []i.Decision{i.Accepted, i.Unreviewed, i.ActionRequired} {
		review := &i.Review{Items: []i.ReviewItem{{FindingID: "root", Presence: i.Current, CurrentDecision: decision}, {FindingID: "operator", Presence: i.Current, CurrentDecision: i.Unreviewed}}}
		summary, references := ReviewSummary(snapshot, review)
		if summary.Current != 1 || summary.Unreviewed != 1 || summary.ActionRequired != 0 || summary.Accepted != 0 || references != 0 {
			t.Fatal(summary, references)
		}
		if review.Items[0].CurrentDecision != decision {
			t.Fatal("scope projection rewrote stored judgment")
		}
	}
}
