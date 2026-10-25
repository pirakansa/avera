package accounts

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/collect/sudo"
	i "github.com/pirakansa/avera/internal/inventory"
)

const fakePasswd = "root:password-must-not-persist:0:0:PRIVATE-GECOS:/root:/bin/sh\noperator:x:1000:1000:PRIVATE-GECOS:/home/operator:/bin/sh\nweb:x:2000:2000:PRIVATE-GECOS:/srv/web:/usr/sbin/nologin\n"
const fakeGroups = "root:group-password-secret:0:\noperator:x:1000:\nweb:x:2000:\nmaint:x:3000:operator\nempty:x:4000:\nexternal:x:5000:external-name\n"

func setup(t *testing.T) (string, Collector) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "etc"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"passwd": fakePasswd, "group": fakeGroups} {
		if err := os.WriteFile(filepath.Join(root, "etc", name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, Collector{Source: sudo.LocalSource{Root: root}, Now: func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }}
}
func snapshot(r i.CollectorResult) *i.Snapshot {
	return &i.Snapshot{Envelope: i.Envelope{SchemaVersion: i.SchemaVersion, ContractRevision: i.ContractRevision, Kind: "snapshot", TargetID: "fictional"}, SnapshotID: "accounts-first", CollectedAt: "2026-10-25T00:00:00Z", FinishedAt: "2026-10-25T00:00:00Z", ToolVersion: Version, Environment: i.Environment{}, Coverage: []i.Coverage{r.Coverage}, Findings: r.Findings, Evidence: r.Evidence}
}
func TestLocalInventoryAndSecretOmission(t *testing.T) {
	_, collector := setup(t)
	r, err := collector.Collect(context.Background(), "fictional")
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Status != i.Complete || len(r.Findings) != 3 {
		t.Fatal(r.Coverage, len(r.Findings))
	}
	for _, f := range r.Findings {
		if f.Identity.(i.AccountIdentity).Name == "operator" {
			facts := f.Facts.(i.AccountFacts)
			if !facts.ContextComplete || len(facts.Groups) != 2 || facts.Groups[1].Name != "operator" && facts.Groups[0].Name != "operator" {
				t.Fatal(facts)
			}
		}
	}
	directory := r.Evidence[0].Values.(i.AccountValues)
	if len(directory.Groups) != 6 || directory.Groups[4].Name != "empty" {
		t.Fatal("empty/external groups omitted", directory)
	}
	data, err := i.Encode(snapshot(r))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"password-must-not-persist", "group-password-secret", "PRIVATE-GECOS"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("retained private field")
		}
	}
	decoded, err := i.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := i.Encode(decoded)
	if err != nil || string(data) != string(again) {
		t.Fatal("unstable roundtrip", err)
	}
	old := snapshot(r)
	old.ContractRevision = i.ServicesContractRevision
	if _, err = i.Encode(old); err == nil {
		t.Fatal("new account data accepted under old contract")
	}
}
func TestAccountFailuresAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name, file, text, diagnostic string
		limit                        func(*Collector)
		remove                       bool
	}{
		{name: "missing group", file: "group", remove: true, diagnostic: "accounts_source_unreadable"},
		{name: "invalid passwd", file: "passwd", text: fakePasswd + "bad:x:not-a-uid:0::/:/bin/sh\n", diagnostic: "accounts_source_malformed"},
		{name: "oversized name", file: "passwd", text: fakePasswd + strings.Repeat("n", localNameByteLimit+1) + ":x:1001:1000::/:/bin/sh\n", diagnostic: "accounts_source_malformed"},
		{name: "duplicate name", file: "passwd", text: fakePasswd + "web:x:3000:0::/:/bin/sh\n", diagnostic: "accounts_source_malformed"},
		{name: "line bound", diagnostic: "accounts_line_limit", limit: func(c *Collector) { c.Lines = 2 }},
		{name: "entry bound", diagnostic: "accounts_entry_limit", limit: func(c *Collector) { c.Entries = 2 }},
		{name: "byte bound", diagnostic: "accounts_source_unreadable", limit: func(c *Collector) { c.Bytes = 20 }},
		{name: "membership bound", diagnostic: "accounts_membership_limit", limit: func(c *Collector) { c.Memberships = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, c := setup(t)
			if tc.file != "" {
				path := filepath.Join(root, "etc", tc.file)
				if tc.remove {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(path, []byte(tc.text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.limit != nil {
				tc.limit(&c)
			}
			r, err := c.Collect(context.Background(), "fictional")
			if err != nil {
				t.Fatal(err)
			}
			if r.Coverage.Status == i.Complete || !slices.Contains(r.Coverage.DiagnosticCodes, tc.diagnostic) {
				t.Fatal(r.Coverage)
			}
			for _, f := range r.Findings {
				if f.Facts.(i.AccountFacts).ContextComplete {
					t.Fatal("incomplete inventory claimed complete")
				}
			}
			if _, err = i.Encode(snapshot(r)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestAccountIdentityStabilityAndMembershipChange(t *testing.T) {
	root, c := setup(t)
	first, err := c.Collect(context.Background(), "fictional")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc/group"), []byte(strings.Replace(fakeGroups, "maint:x:3000:operator", "maint:x:3000:operator,web", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := c.Collect(context.Background(), "fictional")
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for n, f := range first.Findings {
		other := second.Findings[n]
		if f.FindingID != other.FindingID {
			t.Fatal("membership changed identity")
		}
		if f.FactFingerprint != other.FactFingerprint {
			changed++
		}
	}
	if changed != 1 {
		t.Fatal("unexpected affected accounts", changed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.Collect(ctx, "fictional"); err != context.Canceled {
		t.Fatal(err)
	}
	c.Entries = -1
	if _, err = c.Collect(context.Background(), "fictional"); err == nil {
		t.Fatal("accepted negative limit")
	}
}
func TestEmptyAndUIDAliases(t *testing.T) {
	root, c := setup(t)
	for _, name := range []string{"passwd", "group"} {
		if err := os.WriteFile(filepath.Join(root, "etc", name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	empty, err := c.Collect(context.Background(), "fictional")
	if err != nil || len(empty.Findings) != 0 || empty.Coverage.Status != i.Complete {
		t.Fatal(empty, err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc/passwd"), []byte(fakePasswd+"alternate:x:1000:1000::/alternate:/bin/sh\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc/group"), []byte(fakeGroups), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), "fictional")
	if err != nil || len(result.Findings) != 4 {
		t.Fatal(err)
	}
	before := result.Findings
	again, err := c.Collect(context.Background(), "fictional")
	if err != nil || !reflect.DeepEqual(before, again.Findings) {
		t.Fatal("nondeterministic account findings")
	}
}
