package assessment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	i "github.com/pirakansa/avera/internal/inventory"
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
func TestContractScenarios(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(fixtures, "scenarios.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Scenarios []struct {
			Name, Snapshot, Expected string
			Previous, Assessments    *string
		}
	}
	if err = json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, s := range manifest.Scenarios {
		t.Run(s.Name, func(t *testing.T) {
			current := load(t, s.Snapshot).(*i.Snapshot)
			var previous *i.Snapshot
			var assessments *i.Assessments
			if s.Previous != nil {
				previous = load(t, *s.Previous).(*i.Snapshot)
			}
			if s.Assessments != nil {
				assessments = load(t, *s.Assessments).(*i.Assessments)
			}
			before, _ := json.Marshal([]any{current, previous, assessments})
			result, err := Evaluate(current, previous, assessments)
			if err != nil {
				t.Fatal(err)
			}
			expected := load(t, s.Expected).(*i.Review)
			if !reflect.DeepEqual(result.Review, expected) {
				t.Errorf("review mismatch\ngot: %+v\nwant: %+v", result.Review, expected)
			}
			after, _ := json.Marshal([]any{current, previous, assessments})
			if string(before) != string(after) {
				t.Fatal("mutated inputs")
			}
			again, err := Evaluate(current, previous, assessments)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result, again) {
				t.Fatal("nondeterministic result")
			}
			for _, a := range []i.Artifact{result.Review, result.Editable, result.Unmatched} {
				if _, err := i.Encode(a); err != nil {
					t.Fatal(err)
				}
			}
			// Recombining the two explicit assessment partitions must not duplicate starters.
			combined := &i.Assessments{Envelope: result.Editable.Envelope, Entries: append(append([]i.Assessment{}, result.Editable.Entries...), result.Unmatched.Entries...)}
			repeated, err := Evaluate(current, previous, combined)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Editable, repeated.Editable) || !reflect.DeepEqual(result.Unmatched, repeated.Unmatched) {
				t.Fatal("editable regeneration changed entries")
			}
			if assessments != nil {
				preserved := map[string]i.Assessment{}
				for _, a := range combined.Entries {
					preserved[a.AssessmentID] = a
				}
				for _, a := range assessments.Entries {
					if !reflect.DeepEqual(a, preserved[a.AssessmentID]) {
						t.Fatalf("assessment %s rewritten", a.AssessmentID)
					}
				}
			}
		})
	}
}

func TestCompatibilityAndBasisRejection(t *testing.T) {
	for _, field := range []string{"target", "schema", "revision", "basis"} {
		t.Run(field, func(t *testing.T) {
			current := load(t, "snapshot-first.json").(*i.Snapshot)
			a := load(t, "assessments-reviewed.json").(*i.Assessments)
			switch field {
			case "target":
				a.TargetID = "another-machine"
			case "schema":
				a.SchemaVersion = "99"
			case "revision":
				a.ContractRevision = "other"
			case "basis":
				b := a.Entries[0].Basis.(i.ReviewedBasis)
				b.FactFingerprint = a.Entries[1].Basis.(i.ReviewedBasis).FactFingerprint
				a.Entries[0].Basis = b
			}
			if result, err := Evaluate(current, nil, a); err == nil || result != nil {
				t.Fatal("incompatible input accepted")
			}
		})
	}
}

func TestScopeAndAttributionConservatism(t *testing.T) {
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	current := load(t, "snapshot-empty.json").(*i.Snapshot)
	for n := range current.Coverage {
		current.Coverage[n].Scope = "different-scope"
	}
	result, err := Evaluate(current, previous, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range result.Review.Items {
		if item.Change != i.Unknown {
			t.Fatal("scope change inferred disappearance")
		}
	}
	current = load(t, "snapshot-repeat.json").(*i.Snapshot)
	for n, f := range current.Findings {
		if facts, ok := f.Facts.(i.NetworkFacts); ok {
			facts.AttributionComplete = false
			f.Facts = facts
			f, err = i.NormalizeFinding(current.TargetID, f)
			if err != nil {
				t.Fatal(err)
			}
			current.Findings[n] = f
		}
	}
	a := load(t, "assessments-reviewed.json").(*i.Assessments)
	result, err = Evaluate(current, previous, a)
	if err != nil {
		t.Fatal(err)
	}
	if result.Review.Summary.Accepted != 0 {
		t.Fatal("incomplete attribution transferred acceptance")
	}
}

func TestCollectorTransientEvidence(t *testing.T) {
	read := func(name string) *i.Snapshot {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("../../test/fixtures/prototype/network", name))
		if err != nil {
			t.Fatal(err)
		}
		a, err := i.Decode(b)
		if err != nil {
			t.Fatal(err)
		}
		return a.(*i.Snapshot)
	}
	first := read("snapshot-first.json")
	repeat := read("snapshot-transient.json")
	r, err := Evaluate(repeat, first, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range r.Review.Items {
		if item.Change != i.Unchanged {
			t.Fatalf("transient evidence forced review: %+v", item)
		}
	}
}

func TestCurrentScopeMismatchIsUnknown(t *testing.T) {
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	current := load(t, "snapshot-repeat.json").(*i.Snapshot)
	for n := range current.Coverage {
		current.Coverage[n].Scope = "new-scope"
	}
	r, err := Evaluate(current, previous, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range r.Review.Items {
		if item.Change != i.Unknown {
			t.Fatalf("scope mismatch classified as %s", item.Change)
		}
	}
}
