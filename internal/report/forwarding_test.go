package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	"github.com/pirakansa/avera/internal/collect/forwarding"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func TestForwardingPortableJudgmentAndChange(t *testing.T) {
	a, err := artifact.Read("../../test/fixtures/prototype/contracts/snapshot-first.json")
	if err != nil {
		t.Fatal(err)
	}
	s := a.(*i.Snapshot)
	s.ContractRevision = i.ContractRevision
	var scope string
	for _, c := range s.Coverage {
		if c.Collector == i.Network {
			scope = c.Scope
		}
	}
	result, err := (forwarding.Collector{ScopeID: scope, Read: func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "nft" {
			return []byte(`{"nftables":[]}`), nil
		}
		if name == "iptables-save" {
			return []byte("*nat\n:PREROUTING ACCEPT [0:0]\n-A PREROUTING -p tcp --dport 8443 -j DNAT --to-destination 127.0.0.1:9000 --comment '<script>fictional</script>'\nCOMMIT\n"), nil
		}
		return []byte("*nat\nCOMMIT\n"), nil
	}}).Collect(context.Background(), s.TargetID)
	if err != nil {
		t.Fatal(err)
	}
	s.Findings = append(s.Findings, result.Findings...)
	s.Evidence = append(s.Evidence, result.Evidence...)
	history := &i.Assessments{Envelope: s.Envelope, Entries: []i.Assessment{}}
	history.Kind = "assessments"
	history.ContractRevision = i.LegacyContractRevision
	f, err := reportfile.New(s, nil, nil, history, "")
	if err != nil {
		t.Fatal(err)
	}
	if f.History.ContractRevision != i.ContractRevision || history.ContractRevision != i.LegacyContractRevision {
		t.Fatal("history envelope migration failed or mutated source")
	}
	entrance := result.Findings[0]
	var local string
	for _, finding := range s.Findings {
		if assessment.IsReference(finding) {
			local = finding.FindingID
		}
	}
	edit := reportfile.Edit{FindingID: entrance.FindingID, Decision: i.Accepted, Actor: "Fictional reviewer", Rationale: "Upstream firewall permits only the intended audience", Safeguards: "Human-reported router restriction", RelatedIDs: []string{local}, RouteExplanation: "DNAT to loopback; upstream protection documented by the reviewer"}
	updated, err := f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Routes) != 1 || !updated.RouteCurrent(updated.Routes[0]) {
		t.Fatal("forwarding entrance cannot own relationship")
	}
	raw, err := updated.Encode()
	if err != nil {
		t.Fatal(err)
	}
	updated, err = reportfile.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderFile(updated, Options{Format: "html", Language: "ja"})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"転送設定", "127.0.0.1:9000", "追加対応不要", "Upstream firewall", "&lt;script&gt;fictional&lt;/script&gt;"} {
		if !strings.Contains(rendered, text) {
			t.Fatalf("missing %q", text)
		}
	}
	if strings.Contains(rendered, "<script>fictional</script>") {
		t.Fatal("rule escaped incorrectly")
	}
	review, err := updated.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range review.Review.Items {
		if item.FindingID == entrance.FindingID && item.CurrentDecision != i.Accepted {
			t.Fatalf("judgment not applicable: %+v", item)
		}
	}
	previous, err := reportfile.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	updated.Previous = previous.Current
	updated.Current.SnapshotID = "forwarding-changed"
	for n, finding := range updated.Current.Findings {
		if finding.FindingID == entrance.FindingID {
			facts := finding.Facts.(i.ForwardingFacts)
			facts.Rule = strings.ReplaceAll(facts.Rule, "9000", "9001")
			finding.Facts = facts
			updated.Current.Findings[n], err = i.NormalizeFinding(s.TargetID, finding)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	review, err = updated.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range review.Review.Items {
		if item.FindingID == entrance.FindingID && (item.CurrentDecision != i.Unreviewed || item.Applicability != i.NeedsReview) {
			t.Fatalf("changed rule reused: %+v", item)
		}
	}
	// New forwarding judgment history must remain valid when old files are carried.
	next, err := reportfile.New(previous.Current, nil, previous.Assessments, previous.History, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = next.Apply(edit, time.Now()); err != nil {
		t.Fatal("cannot archive forwarding assessment", err)
	}
	legacy := *s
	legacy.ContractRevision = i.LegacyContractRevision
	if _, err := i.Encode(&legacy); err == nil {
		t.Fatal("new finding passed under old contract")
	}
}
