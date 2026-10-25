package network

import (
	"context"
	"os"
	"testing"

	"github.com/pirakansa/avera/internal/collect/forwarding"
	i "github.com/pirakansa/avera/internal/inventory"
)

func TestWithForwardingCoverageAndScope(t *testing.T) {
	for _, missing := range []bool{false, true} {
		result, err := (WithForwarding{Sockets: Collector{Source: fixture(t), ScopeID: "fixture-scope"}, Forwarding: forwarding.Collector{Read: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if missing {
				return nil, os.ErrPermission
			}
			if name == "nft" {
				return []byte(`{"nftables":[]}`), nil
			}
			return []byte("*nat\n-A PREROUTING -j DNAT --to-destination 127.0.0.1:9000\nCOMMIT\n"), nil
		}}}).Collect(context.Background(), "fictional-host")
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, f := range result.Findings {
			if f.Kind == i.Forwarding {
				count++
				if f.Facts.(i.ForwardingFacts).ScopeID != result.Coverage.Scope {
					t.Fatal("namespace mismatch")
				}
			}
		}
		if missing && (count != 0 || result.Coverage.Status != i.Partial) {
			t.Fatal("unavailable source hidden")
		}
		if !missing && count != 4 {
			t.Fatal("forwarding findings missing")
		}
		if len(result.Coverage.AttemptedSources) < 9 {
			t.Fatal("source attempts not visible")
		}
	}
}
