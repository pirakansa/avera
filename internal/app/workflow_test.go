package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/report"
)

type collectorFunc func(context.Context, string) (i.CollectorResult, error)

func (f collectorFunc) Collect(c context.Context, s string) (i.CollectorResult, error) {
	return f(c, s)
}
func fixture(t *testing.T, name string) string {
	t.Helper()
	p, e := filepath.Abs(filepath.Join("../../test/fixtures/prototype/contracts", name))
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func snapshot(t *testing.T, name string) *i.Snapshot {
	t.Helper()
	v, e := artifact.Read(fixture(t, name))
	if e != nil {
		t.Fatal(e)
	}
	return v.(*i.Snapshot)
}
func TestOfflineScenarios(t *testing.T) {
	for _, tc := range []struct{ name, current, previous, annotations string }{
		{"first", "snapshot-first.json", "", ""},
		{"advance", "snapshot-first.json", "", "assessments-advance.json"},
		{"unchanged", "snapshot-repeat.json", "snapshot-first.json", "assessments-reviewed.json"},
		{"changed", "snapshot-changed.json", "snapshot-first.json", "assessments-reviewed.json"},
		{"reconfirmed", "snapshot-changed.json", "snapshot-first.json", "assessments-reconfirmed.json"},
		{"partial", "snapshot-partial.json", "snapshot-first.json", "assessments-reviewed.json"},
		{"failed", "snapshot-failed.json", "snapshot-first.json", "assessments-reviewed.json"},
		{"unmatched", "snapshot-first.json", "", "assessments-unmatched.json"},
		{"conflict", "snapshot-first.json", "", "assessments-conflict.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := Service{EffectiveUID: func() int { t.Fatal("offline privilege read"); return 0 }, Collectors: []Registration{{Name: i.Network, Collector: collectorFunc(func(context.Context, string) (i.CollectorResult, error) {
				t.Fatal("offline collection")
				return i.CollectorResult{}, nil
			})}}}
			o := Options{Snapshot: fixture(t, tc.current), Output: filepath.Join(t.TempDir(), "review")}
			if tc.previous != "" {
				o.Previous = fixture(t, tc.previous)
			}
			if tc.annotations != "" {
				o.Assessments = fixture(t, tc.annotations)
			}
			if err := svc.Offline(context.Background(), o, false); err != nil {
				t.Fatal(err)
			}
			got, err := artifact.Read(filepath.Join(o.Output, "review.json"))
			if err != nil {
				t.Fatal(err)
			}
			expected, err := artifact.Read(fixture(t, "review-"+tc.name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			gb, _ := i.Encode(got)
			eb, _ := i.Encode(expected)
			if string(gb) != string(eb) {
				t.Fatalf("review differs from contract: %s", gb)
			}
			p, a, err := inputs(o)
			if err != nil {
				t.Fatal(err)
			}
			text, err := report.Render(snapshot(t, tc.current), p, a)
			if err != nil {
				t.Fatal(err)
			}
			o.Output = filepath.Join(t.TempDir(), "report.md")
			if err = svc.Offline(context.Background(), o, true); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(o.Output)
			if err != nil || string(b) != text {
				t.Fatalf("report mismatch: %v", err)
			}
		})
	}
}
func TestCollectBoundaries(t *testing.T) {
	base := snapshot(t, "snapshot-repeat.json")
	for _, tc := range []struct {
		name                  string
		uid                   int
		status                i.Status
		previous, annotations string
		existing, cancel      bool
		want                  int
		called                bool
	}{
		{name: "nonroot", uid: 1000, want: 1},
		{name: "existing", existing: true, want: 1},
		{name: "invalid input", previous: "assessments-reviewed.json", want: 1},
		{name: "cancelled", cancel: true, want: 1},
		{name: "complete", status: i.Complete, want: 0, called: true},
		{name: "with context", status: i.Complete, previous: "snapshot-first.json", annotations: "assessments-reviewed.json", want: 0, called: true},
		{name: "advance only", status: i.Complete, annotations: "assessments-advance.json", want: 0, called: true},
		{name: "partial", status: i.Partial, want: 2, called: true},
		{name: "all failed", status: i.Failed, want: 2, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := Service{EffectiveUID: func() int { return tc.uid }, Collectors: []Registration{{Name: i.Network, Collector: collectorFunc(func(context.Context, string) (i.CollectorResult, error) {
				called = true
				r := i.CollectorResult{Coverage: base.Coverage[0], Findings: []i.Finding{}, Evidence: []i.Evidence{}}
				r.Coverage.Collector = i.Network
				r.Coverage.Status = tc.status
				r.Coverage.DiagnosticCodes = []string{"injected"}
				return r, nil
			})}}}
			o := Options{TargetID: base.TargetID, Output: filepath.Join(t.TempDir(), "run")}
			if tc.previous != "" {
				o.Previous = fixture(t, tc.previous)
			}
			if tc.annotations != "" {
				o.Assessments = fixture(t, tc.annotations)
			}
			if tc.existing {
				if err := os.WriteFile(o.Output, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			code, err := svc.Collect(ctx, o)
			if code != tc.want || (err != nil) != (tc.want == 1) || called != tc.called {
				t.Fatalf("code=%d err=%v called=%v", code, err, called)
			}
			if tc.want == 1 {
				if !tc.existing {
					if _, err := os.Stat(o.Output); !os.IsNotExist(err) {
						t.Fatalf("fatal output: %v", err)
					}
				}
				return
			}
			s, err := artifact.Read(filepath.Join(o.Output, "snapshot.json"))
			if err != nil {
				t.Fatal(err)
			}
			if s.(*i.Snapshot).Coverage[1].Status != i.NotCollected {
				t.Fatal("sudo scope missing")
			}
			a, err := artifact.Read(filepath.Join(o.Output, "assessments.json"))
			if err != nil {
				t.Fatal(err)
			}
			if tc.annotations != "" {
				source, err := artifact.Read(o.Assessments)
				if err != nil {
					t.Fatal(err)
				}
				if len(a.(*i.Assessments).Entries) != len(source.(*i.Assessments).Entries) {
					t.Fatal("unmatched context lost")
				}
			}
		})
	}
}
func TestCollectReviewReportSameContext(t *testing.T) {
	s := snapshot(t, "snapshot-repeat.json")
	svc := Service{EffectiveUID: func() int { return 0 }}
	for _, coverage := range s.Coverage {
		c := coverage
		svc.Collectors = append(svc.Collectors, Registration{Name: c.Collector, Collector: collectorFunc(func(context.Context, string) (i.CollectorResult, error) {
			r := i.CollectorResult{Coverage: c, Findings: []i.Finding{}, Evidence: []i.Evidence{}}
			for _, f := range s.Findings {
				if f.Collector == c.Collector {
					r.Findings = append(r.Findings, f)
					for _, eid := range f.EvidenceIDs {
						for _, e := range s.Evidence {
							if e.EvidenceID == eid {
								r.Evidence = append(r.Evidence, e)
							}
						}
					}
				}
			}
			return r, nil
		})})
	}
	o := Options{TargetID: s.TargetID, Previous: fixture(t, "snapshot-first.json"), Assessments: fixture(t, "assessments-reviewed.json"), Output: filepath.Join(t.TempDir(), "collect")}
	if _, err := svc.Collect(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	collected, err := artifact.Read(filepath.Join(o.Output, "review.json"))
	if err != nil {
		t.Fatal(err)
	}
	o.Snapshot = filepath.Join(o.Output, "snapshot.json")
	o.Output = filepath.Join(t.TempDir(), "review")
	if err = svc.Offline(context.Background(), o, false); err != nil {
		t.Fatal(err)
	}
	reviewed, err := artifact.Read(filepath.Join(o.Output, "review.json"))
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := i.Encode(collected)
	rb, _ := i.Encode(reviewed)
	if string(cb) != string(rb) {
		t.Fatal("collect/review mismatch")
	}
	o.Output = filepath.Join(t.TempDir(), "report.md")
	if err = svc.Offline(context.Background(), o, true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(o.Output)
	if err != nil || !strings.Contains(string(b), "Current items") {
		t.Fatalf("missing report: %v", err)
	}
}
func TestOfflineRejectsInvalidAndPreservesInput(t *testing.T) {
	for _, tc := range []string{"collision", "invalid JSON", "target mismatch", "symlink parent"} {
		t.Run(tc, func(t *testing.T) {
			dir := t.TempDir()
			s := snapshot(t, "snapshot-first.json")
			in := filepath.Join(dir, "input.json")
			if err := artifact.Write(context.Background(), in, s); err != nil {
				t.Fatal(err)
			}
			o := Options{Snapshot: in, Output: filepath.Join(dir, "output")}
			switch tc {
			case "collision":
				o.Output = in
			case "invalid JSON":
				if err := os.WriteFile(in, []byte(`{"private":"DO_NOT_LOG"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "target mismatch":
				a := &i.Assessments{Envelope: s.Envelope, Entries: []i.Assessment{}}
				a.Kind = "assessments"
				a.TargetID = "another-target"
				o.Assessments = filepath.Join(dir, "a.json")
				if err := artifact.Write(context.Background(), o.Assessments, a); err != nil {
					t.Fatal(err)
				}
			case "symlink parent":
				link := filepath.Join(dir, "link")
				if err := os.Symlink(dir, link); err != nil {
					t.Fatal(err)
				}
				o.Output = filepath.Join(link, "out")
			}
			before, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			err = (Service{}).Offline(context.Background(), o, false)
			if err == nil || strings.Contains(err.Error(), "DO_NOT_LOG") {
				t.Fatalf("error=%v", err)
			}
			after, err := os.ReadFile(in)
			if err != nil || string(before) != string(after) {
				t.Fatal("input changed")
			}
		})
	}
}
func TestNoPreviousReport(t *testing.T) {
	s := snapshot(t, "snapshot-first.json")
	text, err := report.Render(s, nil, nil)
	if err != nil || !strings.Contains(text, "Unavailable") {
		t.Fatalf("comparison not labeled: %v", err)
	}
	r, err := assessment.Evaluate(s, nil, nil)
	if err != nil || r.Review.Summary.Unreviewed == 0 {
		t.Fatal("missing unreviewed")
	}
}
