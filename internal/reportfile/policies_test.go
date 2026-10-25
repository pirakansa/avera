package reportfile

import (
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

func TestSharedServicePolicyAcrossNamespacesAndCoverage(t *testing.T) {
	f := sample(t)
	edit := httpEdit(t, f)

	edit.Decision = i.Accepted
	shared, err := f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(shared.Policies) != 1 {
		t.Fatal("policy missing")
	}
	raw, err := shared.Encode()
	if err != nil {
		t.Fatal(err)
	}
	next, err := Decode(raw)
	if err != nil {
		t.Fatal("policy roundtrip", err)
	}
	previous, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	next.Previous = previous.Current
	next.Current.SnapshotID = "another-server"
	for n, old := range next.Current.Findings {
		if old.Kind == i.Endpoint {
			identity := old.Identity.(i.EndpointIdentity)
			identity.ScopeID = "another-machine-netns"
			old.Identity = identity
			facts := old.Facts.(i.NetworkFacts)
			facts.Owners = []i.Owner{{Executable: "/another/web-server", UID: 2000}}
			old.Facts = facts
			next.Current.Findings[n], err = i.NormalizeFinding(next.Current.TargetID, old)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for n := range next.Current.Coverage {
		if next.Current.Coverage[n].Collector == i.Network {
			next.Current.Coverage[n].Scope = "another-machine-netns"
			next.Current.Coverage[n].Status = i.Partial
		}
	}
	result, err := next.Review()
	if err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, item := range result.Review.Items {
		if item.Presence != i.Current {
			continue
		}
		for _, reason := range item.ReasonCodes {
			if reason == SharedJudgment {
				matched++
				if item.CurrentDecision != i.Accepted || item.Applicability != i.Applicable {
					t.Fatal("shared judgment lost")
				}
			}
		}
	}
	if matched != 1 {
		t.Fatal("policy did not match service on another server", matched)
	}
	// Other ports are not covered, while incomplete collection remains explicit.
	if result.Review.CoverageStatuses[i.Network] != i.Partial {
		t.Fatal("coverage hidden")
	}
	key := next.Policies[0]
	key.Port++
	next.Policies[0] = key
	if _, err := next.Encode(); err == nil {
		t.Fatal("mismatched selector accepted")
	}
	reuse := false
	edit.Reuse = &reuse
	edit.Rationale = "Updated shared explanation"
	updated, err := shared.Apply(edit, time.Now())
	if err != nil || len(updated.Policies) != 1 || updated.Policies[0].Assessment.Rationale != edit.Rationale {
		t.Fatal("legacy reuse=false must not disable shared saving", err)
	}
	if len(updated.History.Entries) == 0 {
		t.Fatal("old explanation lost")
	}
}
func TestSharedPolicyRejectsOldFormatAndUnknownFields(t *testing.T) {
	f := sample(t)
	edit := httpEdit(t, f)

	f, err := f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode([]byte(strings.Replace(string(raw), Version, previousVersion, 1))); err == nil {
		t.Fatal("new policies accepted under old format")
	}
	raw = []byte(strings.Replace(string(raw), `"policies": [`, `"policies": [{"unknown":true},`, 1))
	if _, err := Decode(raw); err == nil {
		t.Fatal("unknown policy fields accepted")
	}
	if assessment.IsReference(i.Finding{Kind: i.Forwarding}) {
		t.Fatal("forwarding classified as reference")
	}
}
