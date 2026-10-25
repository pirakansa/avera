package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func TestNetworkMapObservedAndHumanRelations(t *testing.T) {
	s := load(t, "snapshot-first.json").(*i.Snapshot)
	s.ContractRevision = i.ContractRevision
	finding, err := i.NormalizeFinding(s.TargetID, i.Finding{Collector: i.Network, Kind: i.Forwarding, Identity: i.SourceIdentity{SourcePath: "/netfilter/iptables-save", Line: 1}, Facts: i.ForwardingFacts{Rule: "-A PREROUTING -p tcp --dport 8443 -j DNAT --to-destination 127.0.0.1:9000", ContextDigest: "sha256:" + strings.Repeat("a", 64), ScopeID: s.Coverage[0].Scope}, EvidenceIDs: []string{s.Evidence[0].EvidenceID}})
	if err != nil {
		t.Fatal(err)
	}
	s.Findings = append(s.Findings, finding)
	f, err := reportfile.New(s, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	var web, local i.Finding
	for _, v := range s.Findings {
		if assessment.IsReference(v) {
			local = v
		}
		if v.Kind == i.Endpoint && v.Identity.(i.EndpointIdentity).LocalPort == 80 {
			web = v
		}
	}
	f, err = f.Apply(reportfile.Edit{FindingID: web.FindingID, Decision: i.Accepted, Actor: "Fictional owner", Purpose: "Static public web", Rationale: "Reviewed", Safeguards: `Human firewall statement <script>bad()</script>`, RelatedIDs: []string{local.FindingID}, RouteExplanation: "Human proxy path"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	review, err := f.Review()
	if err != nil {
		t.Fatal(err)
	}
	diagram := NetworkMapFor(f, review.Review, "ja")
	observed, human := 0, 0
	for _, row := range diagram.Rows {
		switch row.Mode {
		case "observed":
			observed++
			if len(row.Targets) != 1 || row.Targets[0].FindingID != local.FindingID {
				t.Fatal("configured destination mismatch")
			}
		case "human":
			human++
			if row.Source.Decision != "追加対応不要" || !strings.Contains(row.Source.Safeguards, "人の申告") {
				t.Fatal("judgment/provenance missing")
			}
		}
	}
	if observed != 1 || human != 1 || len(diagram.References) != 1 || !strings.Contains(diagram.Outside, "未調査") {
		t.Fatalf("bad diagram: %+v", diagram)
	}
	html, err := RenderFile(f, Options{Format: "html", Language: "ja"})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{`id="network-map"`, `map-row observed`, `map-row human`, "127.0.0.1:9000", "追加対応不要", "&lt;script&gt;bad()&lt;/script&gt;"} {
		if !strings.Contains(html, text) {
			t.Fatalf("missing %s", text)
		}
	}
	if strings.Contains(html, "<script>bad()") {
		t.Fatal("unescaped graph content")
	}
	after, err := f.Encode()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("diagram mutated report", err)
	}
}
func TestNetworkMapUnknownEmptyPartialAndLimits(t *testing.T) {
	s := load(t, "snapshot-first.json").(*i.Snapshot)
	f := &reportfile.File{Current: s}
	review := &i.Review{Items: []i.ReviewItem{}}
	s.Coverage[0].Status = i.Partial
	s.Coverage[0].DiagnosticCodes = []string{"source_unavailable"}
	diagram := NetworkMapFor(f, review, "en")
	if !strings.Contains(strings.Join(diagram.Coverage, " "), "source_unavailable") {
		t.Fatal("coverage hidden")
	}
	for _, row := range diagram.Rows {
		if row.Mode != "unknown" {
			t.Fatal("invented relationship")
		}
	}
	var entrance i.Finding
	for _, finding := range s.Findings {
		if finding.Kind == i.Endpoint && !assessment.IsReference(finding) {
			entrance = finding
			break
		}
	}
	f.Routes = []reportfile.Route{{EntranceID: entrance.FindingID, InternalID: "missing-service"}}
	missing := NetworkMapFor(f, review, "ja")
	foundMissing := false
	for _, row := range missing.Rows {
		if row.Mode == "human" {
			foundMissing = true
			if len(row.Targets) != 1 || row.Targets[0].FindingID != "" || !strings.Contains(row.Targets[0].Title, "未観測") {
				t.Fatal("missing service fabricated as observed")
			}
		}
	}
	if !foundMissing {
		t.Fatal("missing human relationship silently omitted")
	}
	f.Routes = nil

	for n := 0; n < networkMapLimit+10; n++ {
		s.Findings = append(s.Findings, i.Finding{FindingID: fmt.Sprintf("extra-%d", n), Collector: i.Network, Kind: i.Endpoint, Identity: i.EndpointIdentity{Transport: "tcp", LocalAddress: "0.0.0.0", LocalPort: uint16(1000 + n)}})
	}
	diagram = NetworkMapFor(f, review, "en")
	if len(diagram.Rows) != networkMapLimit || diagram.LimitNote == "" {
		t.Fatal("diagram limit not explicit")
	}
	s.Findings = []i.Finding{}
	diagram = NetworkMapFor(f, review, "ja")
	if len(diagram.Rows) != 0 || diagram.Empty == "" || len(diagram.Coverage) == 0 {
		t.Fatal("empty collection not explained")
	}
}
