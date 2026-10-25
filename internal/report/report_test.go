package report

import (
	"encoding/json"
	"fmt"
	i "github.com/pirakansa/avera/internal/inventory"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const fixtures = "../../test/fixtures/prototype/contracts"

func load(t *testing.T, name string) i.Artifact {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtures, name))
	if err != nil {
		t.Fatal(err)
	}
	a, err := i.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func TestScenarios(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(fixtures, "scenarios.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Scenarios []struct {
			Name, Snapshot, Expected string
			Previous, Assessments    *string
		}
	}
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, s := range m.Scenarios {
		t.Run(s.Name, func(t *testing.T) {
			current := load(t, s.Snapshot).(*i.Snapshot)
			var previous *i.Snapshot
			var annotations *i.Assessments
			if s.Previous != nil {
				previous = load(t, *s.Previous).(*i.Snapshot)
			}
			if s.Assessments != nil {
				annotations = load(t, *s.Assessments).(*i.Assessments)
			}
			before, err := json.Marshal([]any{current, previous, annotations})
			if err != nil {
				t.Fatal(err)
			}
			got, err := Render(current, previous, annotations)
			if err != nil {
				t.Fatal(err)
			}
			again, err := Render(current, previous, annotations)
			if err != nil || got != again {
				t.Fatal("nondeterministic", err)
			}
			after, err := json.Marshal([]any{current, previous, annotations})
			if err != nil || string(before) != string(after) {
				t.Fatal("mutated inputs", err)
			}
			expected, _ := focusedFixtureSummary(t, s.Name, s.Expected)
			for _, v := range []string{
				fmt.Sprintf("[Current items](#current-items): %d; accepted: %d; action required: %d; unreviewed: %d.", expected.Current, expected.Accepted, expected.ActionRequired, expected.Unreviewed),
				fmt.Sprintf("Needs re-review: %d; conflicts: %d", expected.NeedsReview, expected.Conflict),
				fmt.Sprintf("[Historical items](#historical-items): %d; [unmatched assessments](#unmatched-assessments): %d.", expected.Historical, expected.UnmatchedAssessments),
			} {
				if !strings.Contains(got, v) {
					t.Fatalf("missing totals %s", v)
				}
			}
			for _, c := range current.Coverage {
				if !strings.Contains(got, Text(string(c.Status))) {
					t.Fatal("missing coverage")
				}
			}
			for _, match := range regexp.MustCompile(`\[Underlying evidence\]\(#(evidence-\d+-\d+)\)`).FindAllStringSubmatch(got, -1) {
				heading := "### " + strings.ReplaceAll(match[1], "-", " ")
				if !strings.Contains(strings.ToLower(got), strings.ToLower(heading)) {
					t.Fatalf("broken evidence link %s", match[1])
				}
			}
			if s.Name == "changed" && !strings.Contains(got, "Prior context only") {
				t.Fatal("stale decision not labeled")
			}
		})
	}
}
func TestUntrustedText(t *testing.T) {
	attack := "|\n```\n<script>alert(1)</script>\r\x00[click](javascript:alert(1)) https://example.invalid <https://example.invalid> &lt;img&gt;\u202e"
	safe := Text(attack)
	for _, bad := range []string{"|", "\n", "`", "<", ">", "[", "]", ":", "\r", "\x00", "\u202e"} {
		if strings.Contains(safe, bad) {
			t.Fatalf("raw syntax %q", bad)
		}
	}
	current := load(t, "snapshot-first.json").(*i.Snapshot)
	annotations := load(t, "assessments-reviewed.json").(*i.Assessments)
	for n := range annotations.Entries {
		a := &annotations.Entries[n]
		a.Purpose = attack
		a.Exposure = attack
		a.Safeguards = attack
		a.Rationale = attack
		a.Actor = attack
		a.EvidenceRefs = []i.EvidenceReference{{Reference: attack, Provenance: "human_reported"}}
	}
	got, err := Render(current, nil, annotations)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, safe) || strings.Contains(got, "<script>") || strings.Contains(got, "```") {
		t.Fatal("unsafe annotation rendering")
	}
	if !strings.Contains(got, "Human safeguards (not tool verified)") {
		t.Fatal("assertions not distinguished")
	}
}
func TestBoundsAndInvalidInput(t *testing.T) {
	v, err := technical(i.FileValues{Excerpt: strings.Repeat("界", 3000)})
	if err != nil || !strings.Contains(v, "[truncated;") {
		t.Fatal("unbounded evidence", err)
	}
	current := load(t, "snapshot-first.json").(*i.Snapshot)
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	previous.TargetID = "different"
	if _, err := Render(current, previous, nil); err == nil {
		t.Fatal("incompatible input accepted")
	}
}

func TestCollectorOutputs(t *testing.T) {
	for _, name := range []string{"network/snapshot-first.json", "sudo/snapshot-understood.json", "sudo/snapshot-unsupported.json"} {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("../../test/fixtures/prototype", name))
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := i.Decode(b)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := artifact.(*i.Snapshot)
			got, err := Render(snapshot, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			links := strings.Count(got, "[Underlying evidence]")
			want := 0
			for _, f := range snapshot.Findings {
				want += len(f.EvidenceIDs)
			}
			if links != want {
				t.Fatalf("evidence routes %d want %d", links, want)
			}
			if !strings.Contains(got, "Next explanation:") {
				t.Fatal("next action missing")
			}
		})
	}
}

func TestPublicHTTPHumanStatement(t *testing.T) {
	current := load(t, "snapshot-repeat.json").(*i.Snapshot)
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	annotations := load(t, "assessments-reviewed.json").(*i.Assessments)
	annotations.Entries[0].Safeguards = "Authentication is inapplicable to public static content; operator reports an external firewall"
	got, err := Render(current, previous, annotations)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"accepted: 1", "Authentication is inapplicable", "Human safeguards (not tool verified)", "not a tool safety certification", Text("https://example.invalid/fictional-design")} {
		if !strings.Contains(got, s) {
			t.Fatalf("missing public-service context %q", s)
		}
	}
}
