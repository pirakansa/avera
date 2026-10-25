package forwarding

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	i "github.com/pirakansa/avera/internal/inventory"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../test/fixtures/prototype/forwarding/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestRulesAndContext(t *testing.T) {
	for _, tc := range []struct {
		source, file string
		count        int
	}{{"iptables-save", "iptables-save.txt", 4}, {"nft", "nft.json", 4}} {
		t.Run(tc.source, func(t *testing.T) {
			data := fixture(t, tc.file)
			normalized, rules, err := Parse(tc.source, data)
			if err != nil || len(rules) != tc.count {
				t.Fatalf("rules=%v err=%v", rules, err)
			}
			changed := strings.ReplaceAll(strings.ReplaceAll(string(data), "12:456", "999:888"), `"packets":12,"bytes":456`, `"packets":999,"bytes":888`)
			repeat, r, err := Parse(tc.source, []byte(changed))
			if err != nil || repeat != normalized || !reflect.DeepEqual(r, rules) {
				t.Fatal("volatile counters affect context")
			}
			changed = strings.ReplaceAll(string(data), "9000", "9001")
			context, _, err := Parse(tc.source, []byte(changed))
			if err != nil || context == normalized {
				t.Fatal("destination change missing")
			}
			if !strings.Contains(normalized, "PREROUTING") && !strings.Contains(normalized, "prerouting") {
				t.Fatal("chain context lost")
			}
		})
	}
}
func TestCollectFailuresAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name  string
		read  ReadFunc
		want  i.Status
		count int
	}{
		{"complete", func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name == "nft" {
				return fixture(t, "nft.json"), nil
			}
			return fixture(t, "iptables-save.txt"), nil
		}, i.Complete, 20},
		{"missing", func(context.Context, string, ...string) ([]byte, error) { return nil, os.ErrNotExist }, i.Failed, 0},
		{"malformed", func(context.Context, string, ...string) ([]byte, error) { return []byte("bad"), nil }, i.Failed, 0},
		{"limit", func(context.Context, string, ...string) ([]byte, error) { return make([]byte, MaxSourceBytes+1), nil }, i.Failed, 0},
		{"one-backend", func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name == "nft" {
				return fixture(t, "nft.json"), nil
			}
			return nil, os.ErrPermission
		}, i.Partial, 4},
		{"empty", func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name == "nft" {
				return []byte(`{"nftables":[]}`), nil
			}
			return []byte("*nat\nCOMMIT\n"), nil
		}, i.Complete, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := (Collector{Read: tc.read, ScopeID: "fixture-scope", Now: func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }}).Collect(context.Background(), "fictional")
			if err != nil || r.Coverage.Status != tc.want || len(r.Findings) != tc.count {
				t.Fatalf("status=%s findings=%d err=%v", r.Coverage.Status, len(r.Findings), err)
			}
			for _, f := range r.Findings {
				if f.Kind != i.Forwarding || f.FactFingerprint == "" {
					t.Fatal("unnormalized finding")
				}
			}
			if tc.want != i.Complete && len(r.Coverage.DiagnosticCodes) == 0 {
				t.Fatal("missing diagnostics")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Collector{ScopeID: "scope"}).Collect(ctx, "fictional"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost")
	}
}
func TestMalformedSources(t *testing.T) {
	for _, data := range []string{"*nat\n-A PREROUTING -j DNAT", "COMMIT", "*nat\nCOMMIT\n-A BAD", "*nat\n*filter\nCOMMIT"} {
		if _, _, err := Parse("iptables-save", []byte(data)); err == nil {
			t.Fatalf("accepted %q", data)
		}
	}
	for _, data := range []string{`{}`, `{"nftables":null}`, `{"nftables":{}}`, `{"nftables":[{}]}`, `{"nftables":[]} trailing`} {
		if _, _, err := Parse("nft", []byte(data)); err == nil {
			t.Fatalf("accepted %q", data)
		}
	}
}
func TestBoundedWriterAndCommandAllowlist(t *testing.T) {
	var b boundedBuffer
	if _, err := b.Write(make([]byte, MaxSourceBytes)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("x")); err == nil || b.Len() != MaxSourceBytes {
		t.Fatal("bound ignored")
	}
	if _, err := ReadCommand(context.Background(), "sh", "-c", "false"); err == nil {
		t.Fatal("command allowlist ignored")
	}
}

func TestEmptySaveAndChangedFilterContext(t *testing.T) {
	for _, data := range []string{"", "# Generated empty save\n"} {
		context, rules, err := Parse("iptables-save", []byte(data))
		if err != nil || context != "" || len(rules) != 0 {
			t.Fatal("empty successful save rejected", err)
		}
	}
	original := fixture(t, "iptables-save.txt")
	before, rules, err := Parse("iptables-save", original)
	if err != nil {
		t.Fatal(err)
	}
	after, next, err := Parse("iptables-save", []byte(strings.Replace(string(original), "INPUT DROP", "INPUT ACCEPT", 1)))
	if err != nil || before == after || !reflect.DeepEqual(rules, next) {
		t.Fatal("filter context change missing or changed candidates", err)
	}
}
