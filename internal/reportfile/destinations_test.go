package reportfile

import (
	i "github.com/pirakansa/avera/internal/inventory"
	"testing"
)

func TestDetectedTargetsOnlyExplicitLiteralDNAT(t *testing.T) {
	f := sample(t)
	for _, tc := range []struct {
		rule string
		want int
	}{
		{"-A PREROUTING -p tcp --dport 8443 -j DNAT --to-destination 127.0.0.1:9000", 1},
		{"-A PREROUTING -p udp -j DNAT --to-destination 127.0.0.1:9000", 0},
		{"-A PREROUTING -p tcp -j REDIRECT --to-ports 9000", 0},
		{"-A PREROUTING -p tcp -j DNAT --to-destination 127.0.0.1:9001", 0},
		{"-A PREROUTING -p tcp --comment 'fake -j DNAT --to-destination 127.0.0.1:9000'", 0},
		{`{"rule":{"expr":[{"match":{"left":{"payload":{"protocol":"tcp","field":"dport"}},"right":8443}},{"dnat":{"addr":"127.0.0.1","port":9000}}]}}`, 1},
		{`{"rule":{"expr":[{"dnat":{"addr":"127.0.0.1","port":9000}}]}}`, 0},
	} {
		if targets := f.DetectedTargets(i.Finding{Kind: i.Forwarding, Facts: i.ForwardingFacts{Rule: tc.rule}}); len(targets) != tc.want {
			t.Fatalf("%s: %d targets", tc.rule, len(targets))
		}
	}
}
