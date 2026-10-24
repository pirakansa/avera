package guidance

import (
	i "github.com/pirakansa/avera/internal/inventory"
	"strings"
	"testing"
)

func TestBundledKinds(t *testing.T) {
	for _, k := range []i.Kind{i.Endpoint, i.Rule, i.Unparsed} {
		c := For(k)
		if c.Observed == "" || c.Why == "" || len(c.Questions) < 3 {
			t.Fatalf("missing guidance: %s", k)
		}
		c.Questions[0] = "changed"
		if For(k).Questions[0] == "changed" {
			t.Fatal("shared mutable content")
		}
	}
	if !strings.Contains(strings.Join(For(i.Endpoint).Questions, " "), "Authentication may be inapplicable") {
		t.Fatal("public content context missing")
	}
	if len(Missing(i.Assessment{Purpose: "  "})) != 5 {
		t.Fatal("blank fields not detected")
	}
	a := i.Assessment{Purpose: "public static content", Exposure: "public", Safeguards: "authentication inapplicable", Rationale: "owner judgment", EvidenceRefs: []i.EvidenceReference{{Reference: "external firewall"}}}
	if len(Missing(a)) != 0 {
		t.Fatal("context rejected")
	}
}
