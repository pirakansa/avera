package reportfile

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

func fixture(t *testing.T, name string) i.Artifact {
	t.Helper()
	a, err := artifact.Read(filepath.Join("../../test/fixtures/prototype/contracts", name))
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func sample(t *testing.T) *File {
	t.Helper()
	f, err := New(fixture(t, "snapshot-changed.json").(*i.Snapshot), fixture(t, "snapshot-first.json").(*i.Snapshot), fixture(t, "assessments-reviewed.json").(*i.Assessments), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func httpEdit(t *testing.T, f *File) Edit {
	t.Helper()
	for _, x := range f.Current.Findings {
		if x.Kind == i.Endpoint && x.Identity.(i.EndpointIdentity).LocalPort == 80 {
			return Edit{FindingID: x.FindingID, Decision: i.ActionRequired, Actor: "fictional-reviewer", Rationale: "Investigate changed privilege", Purpose: "Public content", Evidence: []i.EvidenceReference{{Reference: "https://evidence.invalid/record", Provenance: "human_reported"}}}
		}
	}
	t.Fatal("no HTTP finding")
	return Edit{}
}
func TestRoundTripAndExplicitReconfirmation(t *testing.T) {
	f := sample(t)
	data, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "report.json")
	if err = f.Write(context.Background(), out); err != nil {
		t.Fatal(err)
	}
	loaded, err := Read(out)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := loaded.Encode()
	if err != nil || !bytes.Equal(data, encoded) {
		t.Fatal("roundtrip changed report", err)
	}
	edit := httpEdit(t, f)
	updated, err := loaded.Apply(edit, time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.History.Entries) != 1 || updated.History.Entries[0].AssessmentID != "a-http" {
		t.Fatal("lost original judgment")
	}
	before, _ := i.Encode(loaded.Current)
	after, _ := i.Encode(updated.Current)
	if !bytes.Equal(before, after) {
		t.Fatal("facts mutated")
	}
	review, err := assessment.Evaluate(updated.Current, updated.Previous, updated.Assessments)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range review.Review.Items {
		if item.FindingID == edit.FindingID && (item.CurrentDecision != i.ActionRequired || item.Applicability != i.Applicable) {
			t.Fatal("reconfirmation not applied")
		}
	}
	original, _ := loaded.Encode()
	if !bytes.Equal(original, data) {
		t.Fatal("input mutated")
	}
	if err = updated.Write(context.Background(), out); err == nil {
		t.Fatal("input overwritten")
	}
	disk, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(disk, data) {
		t.Fatal("original file changed")
	}
	updated2, err := updated.Apply(edit, time.Now())
	if err != nil || len(updated2.History.Entries) != 2 {
		t.Fatal("repeated save lost history", err)
	}
}
func TestRejectInvalidContainer(t *testing.T) {
	f := sample(t)
	data, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{[]byte(`{"format":"avera.report.v1","format":"avera.report.v1"}`), bytes.Replace(data, []byte(Version), []byte("avera.report.v99"), 1), append([]byte(`{"unexpected":true,`), data[1:]...), append(data, []byte(`{}`)...)} {
		if _, err = Decode(bad); err == nil {
			t.Fatal("accepted malformed report")
		}
	}
	f.Assessments.TargetID = "other"
	if _, err = f.Encode(); err == nil {
		t.Fatal("cross-target input accepted")
	}
}
func TestPartialCoverageAndUnmatchedHistory(t *testing.T) {
	f, err := New(fixture(t, "snapshot-partial.json").(*i.Snapshot), fixture(t, "snapshot-first.json").(*i.Snapshot), fixture(t, "assessments-reviewed.json").(*i.Assessments), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	edit := httpEdit(t, f)
	edit.Decision = i.Accepted
	f, err = f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	review, err := assessment.Evaluate(f.Current, f.Previous, f.Assessments)
	if err != nil {
		t.Fatal(err)
	}
	if review.Review.Summary.Accepted != 0 || len(review.Unmatched.Entries) == 0 {
		t.Fatal("incomplete collection overrode applicability or lost context")
	}
	edit.FindingID = "unknown"
	if _, err = f.Apply(edit, time.Now()); err == nil {
		t.Fatal("edited nonexistent fact")
	}
}

func TestPortableFixtureMatchesSourceScenario(t *testing.T) {
	f := sample(t)
	expected, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("../../test/fixtures/prototype/reports/fictional-report.json")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Decode(actual)
	if err != nil {
		t.Fatal(err)
	}
	actual, err = loaded.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, actual) {
		t.Fatal("portable fixture drifted from its documented source scenario")
	}
	if _, err = Decode(actual); err != nil {
		t.Fatal(err)
	}
}

func TestHumanRoutesAndRelatedChangeReReview(t *testing.T) {
	f := sample(t)
	edit := httpEdit(t, f)
	edit.Decision = i.Accepted
	var local i.Finding
	for _, finding := range f.Current.Findings {
		if assessment.IsReference(finding) {
			local = finding
		}
	}
	edit.RelatedIDs = []string{local.FindingID}
	edit.RouteExplanation = "Public entrance forwards to the local admin service; external firewall restrictions are human-reported"
	updated, err := f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Routes) != 1 || !updated.RouteCurrent(updated.Routes[0]) {
		t.Fatal("missing observation-qualified path statement")
	}
	raw, err := updated.Encode()
	if err != nil {
		t.Fatal(err)
	}
	roundtrip, err := Decode(raw)
	if err != nil || len(roundtrip.Routes) != 1 {
		t.Fatal("route roundtrip failed", err)
	}
	missing, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	previousCopy, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	missing.Previous = previousCopy.Current
	missing.Current.SnapshotID = "fictional-route-missing"
	retained := []i.Finding{}
	for _, finding := range missing.Current.Findings {
		if finding.FindingID != local.FindingID {
			retained = append(retained, finding)
		}
	}
	missing.Current.Findings = retained
	missingReview, err := missing.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range missingReview.Review.Items {
		if item.FindingID == edit.FindingID && item.CurrentDecision != i.Unreviewed {
			t.Fatal("missing linked service retained acceptance")
		}
	}
	prior, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	roundtrip.Previous = prior.Current
	roundtrip.Current.SnapshotID = "fictional-route-next"
	// Current observation changes at the internal service, while the entrance is unchanged.
	for n := range roundtrip.Current.Findings {
		finding := &roundtrip.Current.Findings[n]
		if finding.FindingID != local.FindingID {
			continue
		}
		facts := finding.Facts.(i.NetworkFacts)
		facts.Owners = []i.Owner{{UID: 0, Executable: "/opt/changed-admin"}}
		finding.Facts = facts
		_, finding.FactFingerprint, err = i.Hashes(roundtrip.Current.TargetID, finding.Collector, finding.Kind, finding.Identity, finding.Facts)
		if err != nil {
			t.Fatal(err)
		}
	}
	review, err := roundtrip.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range review.Review.Items {
		if item.FindingID == edit.FindingID && (item.CurrentDecision != i.Unreviewed || item.Applicability != i.NeedsReview) {
			t.Fatal("stale linked context retained acceptance")
		}
	}
	if roundtrip.RouteCurrent(roundtrip.Routes[0]) {
		t.Fatal("changed internal facts treated as matched")
	}
	reconfirmed, err := roundtrip.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(reconfirmed.RouteHistory) != 1 || !reconfirmed.RouteCurrent(reconfirmed.Routes[0]) {
		t.Fatal("route reconfirmation did not preserve history")
	}
	review, err = reconfirmed.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range review.Review.Items {
		if item.FindingID == edit.FindingID && item.CurrentDecision != i.Accepted {
			t.Fatal("explicit reconfirmation did not apply")
		}
	}
	edit.RelatedIDs = []string{}
	cleared, err := reconfirmed.Apply(edit, time.Now())
	if err != nil || len(cleared.Routes) != 0 || len(cleared.RouteHistory) != 2 {
		t.Fatal("unlink lost history", err)
	}
}
func TestRoutesRejectInvalidTargetsAndReferenceJudgments(t *testing.T) {
	f := sample(t)
	edit := httpEdit(t, f)
	edit.RelatedIDs = []string{edit.FindingID}
	edit.RouteExplanation = "Invalid self-link"
	if _, err := f.Apply(edit, time.Now()); err == nil {
		t.Fatal("entrance used as local target")
	}
	for _, finding := range f.Current.Findings {
		if assessment.IsReference(finding) {
			edit.FindingID = finding.FindingID
		}
	}
	if _, err := f.Apply(edit, time.Now()); err == nil {
		t.Fatal("reference endpoint demanded a standalone judgment")
	}
}
func TestVersionOneMigration(t *testing.T) {
	data, err := os.ReadFile("../../test/fixtures/prototype/reports/fictional-report.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := f.Encode()
	if err != nil || !bytes.Contains(updated, []byte(Version)) {
		t.Fatal("old report did not migrate", err)
	}
}
