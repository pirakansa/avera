package app

import (
	"context"
	"fmt"
	"github.com/pirakansa/avera/internal/assessment"
	"path/filepath"
	"strings"
	"testing"
	"time"

	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

const fakeBinding = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestPortableCollectRepeatAndTargetBoundaries(t *testing.T) {
	called := 0
	identityCalls := 0
	svc := Service{EffectiveUID: func() int { return 0 }, HostIdentity: func() (string, error) { identityCalls++; return fakeBinding, nil }, Collectors: []Registration{{Name: i.Network, Collector: collectorFunc(func(context.Context, string) (i.CollectorResult, error) {
		called++
		return i.CollectorResult{Coverage: i.Coverage{Collector: i.Network, Status: i.Partial, Scope: "fictional", AttemptedSources: []string{}, DiagnosticCodes: []string{"injected"}, Limitations: []string{}}, Findings: []i.Finding{}, Evidence: []i.Evidence{}}, nil
	})}}}
	dir := t.TempDir()
	o := Options{TargetID: "fictional-target", Output: filepath.Join(dir, "first.json")}
	code, err := svc.CollectReport(context.Background(), o)
	if err != nil || code != 2 {
		t.Fatal("first failed", code, err)
	}
	first, err := reportfile.Read(o.Output)
	if err != nil || first.Binding != fakeBinding || first.Previous != nil {
		t.Fatal("bad first report", err)
	}
	o.Input = o.Output
	o.Output = filepath.Join(dir, "second.json")
	o.TargetID = ""
	code, err = svc.CollectReport(context.Background(), o)
	if err != nil || code != 2 {
		t.Fatal("repeat failed", err)
	}
	second, err := reportfile.Read(o.Output)
	if err != nil || second.Current.TargetID != "fictional-target" || second.Previous.SnapshotID != first.Current.SnapshotID {
		t.Fatal("repeat lost context", err)
	}
	initial := called
	o.Output = filepath.Join(dir, "mismatch.json")
	svc.HostIdentity = func() (string, error) { return strings.Replace(fakeBinding, "a", "b", -1), nil }
	if _, err = svc.CollectReport(context.Background(), o); err == nil || called != initial {
		t.Fatal("cross-host report collected")
	}
	o.TargetID = "different"
	svc.HostIdentity = func() (string, error) { t.Fatal("identity read before input validation"); return "", nil }
	if _, err = svc.CollectReport(context.Background(), o); err == nil {
		t.Fatal("cross-target report accepted")
	}
	svc.EffectiveUID = func() int { return 1000 }
	if _, err = svc.CollectReport(context.Background(), Options{Output: filepath.Join(dir, "nonroot")}); err == nil {
		t.Fatal("non-root collection accepted")
	}
	if identityCalls != 2 {
		t.Fatal("unexpected identity reads")
	}
}
func TestPortableImportAndExport(t *testing.T) {
	svc := Service{}
	out := filepath.Join(t.TempDir(), "report.json")
	o := Options{Snapshot: fixture(t, "snapshot-changed.json"), Previous: fixture(t, "snapshot-first.json"), Assessments: fixture(t, "assessments-reviewed.json"), Output: out}
	if err := svc.ImportReport(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	f, err := reportfile.Read(out)
	if err != nil || f.Binding != "" {
		t.Fatal(err)
	}
	o = Options{Input: out, Output: filepath.Join(t.TempDir(), "report.html"), Format: "html", Language: "ja"}
	if err = svc.Offline(context.Background(), o, true); err != nil {
		t.Fatal(err)
	}
	called := false
	svc = Service{EffectiveUID: func() int { return 0 }, HostIdentity: func() (string, error) { called = true; return "", fmt.Errorf("injected") }}
	o.Output = filepath.Join(t.TempDir(), "next.json")
	if _, err = svc.CollectReport(context.Background(), o); err == nil || called {
		t.Fatal("unbound report read host without explicit adoption")
	}
	o.AdoptTarget = true
	if _, err = svc.CollectReport(context.Background(), o); err == nil || !called {
		t.Fatal("explicit adoption not attempted")
	}
}

func TestCarryEditedReportIntoNextCollection(t *testing.T) {
	source := snapshot(t, "snapshot-first.json")
	a, err := load(fixture(t, "assessments-reviewed.json"), "assessments", "--assessments")
	if err != nil {
		t.Fatal(err)
	}
	f, err := reportfile.New(source, nil, a.(*i.Assessments), nil, fakeBinding)
	if err != nil {
		t.Fatal(err)
	}
	id := ""
	for _, finding := range source.Findings {
		if finding.Kind == i.Endpoint && finding.Identity.(i.EndpointIdentity).LocalPort == 80 {
			id = finding.FindingID
		}
	}
	f, err = f.Apply(reportfile.Edit{FindingID: id, Decision: i.Accepted, Actor: "fictional-owner", Rationale: "Explicit owner confirmation"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(t.TempDir(), "edited.json")
	if err = f.Write(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	svc := Service{EffectiveUID: func() int { return 0 }, HostIdentity: func() (string, error) { return fakeBinding, nil }, Collectors: []Registration{{Name: i.Network, Collector: collectorFunc(func(context.Context, string) (i.CollectorResult, error) {
		result := i.CollectorResult{Coverage: source.Coverage[0], Findings: []i.Finding{}, Evidence: []i.Evidence{}}
		used := map[string]bool{}
		for _, finding := range source.Findings {
			if finding.Collector != i.Network {
				continue
			}
			result.Findings = append(result.Findings, finding)
			for _, id := range finding.EvidenceIDs {
				used[id] = true
			}
		}
		for _, e := range source.Evidence {
			if used[e.EvidenceID] {
				result.Evidence = append(result.Evidence, e)
			}
		}
		return result, nil
	})}}}
	out := filepath.Join(t.TempDir(), "repeat.json")
	if code, err := svc.CollectReport(context.Background(), Options{Input: in, Output: out}); err != nil || code != 0 {
		t.Fatal("carry-back failed", code, err)
	}
	next, err := reportfile.Read(out)
	if err != nil {
		t.Fatal(err)
	}
	result, err := assessment.Evaluate(next.Current, next.Previous, next.Assessments)
	if err != nil {
		t.Fatal(err)
	}
	if result.Review.Summary.Accepted != 1 || len(next.History.Entries) != 1 || next.Current.TargetID != source.TargetID {
		t.Fatal("carry-back lost identity, decision or history")
	}
}
