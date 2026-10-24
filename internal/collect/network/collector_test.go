package network

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/inventory"
)

const header = "sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"

type fakeSource struct {
	files   map[string]string
	dirs    map[string][]string
	links   map[string]string
	reverse bool
	cancel  context.CancelFunc
}

func (s *fakeSource) Open(path string) (io.ReadCloser, error) {
	if s.cancel != nil {
		s.cancel()
	}
	text, ok := s.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(text)), nil
}
func (s *fakeSource) ReadDir(path string, limit int) ([]string, bool, error) {
	a, ok := s.dirs[path]
	if !ok {
		return nil, false, os.ErrPermission
	}
	a = slices.Clone(a)
	if s.reverse {
		slices.Reverse(a)
	}
	more := len(a) > limit
	if more {
		a = a[:limit]
	}
	return a, more, nil
}
func (s *fakeSource) Readlink(path string, limit int) (string, error) {
	v, ok := s.links[path]
	if !ok {
		return "", os.ErrNotExist
	}
	if len(v) > limit {
		return "", errLimit
	}
	return v, nil
}

func fixture(t *testing.T) *fakeSource {
	t.Helper()
	s := &fakeSource{files: map[string]string{}, dirs: map[string][]string{"/proc": {"10", "20", "30"}}, links: map[string]string{}}
	for _, name := range []string{"tcp", "tcp6", "udp", "udp6"} {
		b, err := os.ReadFile("../../../test/fixtures/prototype/network/" + name)
		if err != nil {
			t.Fatal(err)
		}
		s.files["/proc/net/"+name] = string(b)
	}
	s.dirs["/proc/10/fd"] = []string{"1", "2", "3", "4", "5", "6", "7"}
	for i, inode := range []string{"101", "102", "104", "105", "106", "107", "110"} {
		s.links["/proc/10/fd/"+string(rune('1'+i))] = "socket:[" + inode + "]"
	}
	s.links["/proc/10/exe"] = "/usr/sbin/fictional-server"
	s.dirs["/proc/20/fd"] = []string{"8"}
	s.links["/proc/20/fd/8"] = "socket:[101]"
	s.links["/proc/20/exe"] = "/opt/fictional-worker"
	s.dirs["/proc/30/fd"] = []string{}
	return s
}

func collect(t *testing.T, s *fakeSource, limits Limits) inventory.CollectorResult {
	t.Helper()
	r, err := (Collector{Source: s, Limits: limits, ScopeID: "fictional-boot-a/net-a", Now: func() time.Time { return time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC) }}).Collect(context.Background(), "fictional-web-01")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCategoriesAddressesOwnersAndDeterminism(t *testing.T) {
	s := fixture(t)
	r := collect(t, s, Limits{})
	if r.Coverage.Status != inventory.Complete || len(r.Findings) != 6 {
		t.Fatalf("unexpected result: %+v", r)
	}
	addresses := map[string]bool{}
	for _, f := range r.Findings {
		id := f.Identity.(inventory.EndpointIdentity)
		addresses[id.LocalAddress] = true
		if id.LocalPort == 80 {
			facts := f.Facts.(inventory.NetworkFacts)
			if len(facts.Owners) != 2 || !facts.AttributionComplete || len(f.EvidenceIDs) != 5 {
				t.Fatalf("duplicate/owner aggregation: %+v", f)
			}
		}
		if id.LocalPort == 51966 || id.LocalPort == 49153 || id.LocalPort == 0 {
			t.Fatalf("excluded socket included: %+v", id)
		}
	}
	for _, a := range []string{"0.0.0.0", "127.0.0.1", "::", "::1"} {
		if !addresses[a] {
			t.Errorf("missing %s", a)
		}
	}
	s.reverse = true
	again := collect(t, s, Limits{})
	if !reflect.DeepEqual(r, again) {
		t.Fatal("directory ordering changed output")
	}
}

func TestCoverageFailuresAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*fakeSource)
		limits Limits
		status inventory.Status
		code   string
	}{
		{"missing table", func(s *fakeSource) { delete(s.files, "/proc/net/tcp") }, Limits{}, inventory.Partial, "source_unreadable"},
		{"malformed row", func(s *fakeSource) { s.files["/proc/net/tcp"] += "broken\n" }, Limits{}, inventory.Partial, "source_malformed"},
		{"missing process", func(s *fakeSource) { delete(s.dirs, "/proc/30/fd") }, Limits{}, inventory.Partial, "descriptors_incomplete"},
		{"missing executable", func(s *fakeSource) { delete(s.links, "/proc/10/exe") }, Limits{}, inventory.Partial, "executable_unreadable"},
		{"missing descriptor", func(s *fakeSource) { delete(s.links, "/proc/10/fd/1") }, Limits{}, inventory.Partial, "descriptor_unreadable"},
		{"process limit", func(s *fakeSource) {}, Limits{Processes: 1}, inventory.Partial, "process_limit"},
		{"descriptor limit", func(s *fakeSource) {}, Limits{Descriptors: 1}, inventory.Partial, "descriptors_incomplete"},
		{"row limit", func(s *fakeSource) {}, Limits{Rows: 1}, inventory.Partial, "source_limit"},
		{"byte limit", func(s *fakeSource) {}, Limits{SourceBytes: 150}, inventory.Partial, "source_limit"},
		{"path limit", func(s *fakeSource) {}, Limits{PathBytes: 16}, inventory.Partial, "executable_unreadable"},
		{"evidence limit", func(s *fakeSource) {}, Limits{Evidence: 2}, inventory.Partial, "evidence_limit"},
		{"all missing", func(s *fakeSource) { s.files = map[string]string{} }, Limits{}, inventory.Failed, "source_unreadable"},
		{"empty bytes", func(s *fakeSource) {
			for p := range s.files {
				s.files[p] = ""
			}
		}, Limits{}, inventory.Failed, "source_malformed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture(t)
			tc.mutate(s)
			r := collect(t, s, tc.limits)
			if r.Coverage.Status != tc.status || !slices.Contains(r.Coverage.DiagnosticCodes, tc.code) {
				t.Fatalf("coverage: %+v", r.Coverage)
			}
			if tc.name == "missing executable" && len(r.Findings) != 6 {
				t.Fatal("endpoint lost")
			}
		})
	}
}

func TestSuccessfulEmpty(t *testing.T) {
	s := fixture(t)
	for p := range s.files {
		s.files[p] = header
	}
	s.dirs = nil
	r := collect(t, s, Limits{})
	if r.Coverage.Status != inventory.Complete || len(r.Findings) != 0 || len(r.Evidence) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestMissingOwnersRetainsEndpointAndUID(t *testing.T) {
	s := fixture(t)
	s.dirs["/proc"] = []string{}
	r := collect(t, s, Limits{})
	if len(r.Findings) != 6 || r.Coverage.Status != inventory.Partial {
		t.Fatalf("%+v", r)
	}
	for _, f := range r.Findings {
		facts := f.Facts.(inventory.NetworkFacts)
		if facts.AttributionComplete || len(facts.Owners) != 1 || facts.Owners[0].Executable != "" {
			t.Fatalf("%+v", facts)
		}
		if f.Identity.(inventory.EndpointIdentity).LocalPort == 80 && facts.Owners[0].UID != 33 {
			t.Fatal("socket UID lost")
		}
	}
	limited := collect(t, fixture(t), Limits{Evidence: 2})
	if len(limited.Evidence) > 2 || limited.Coverage.Status != inventory.Partial {
		t.Fatal("evidence limit lost")
	}
}

func TestChangedSocketUIDChangesFactsWithoutChangingIdentity(t *testing.T) {
	s := fixture(t)
	before := collect(t, s, Limits{})
	// The socket UID can change independently of PID and executable. Both
	// duplicate port-80 rows must contribute their own observed UID.
	s.files["/proc/net/tcp"] = strings.Replace(s.files["/proc/net/tcp"], " 33 0 101", " 44 0 101", 1)
	after := collect(t, s, Limits{})
	for i, f := range before.Findings {
		next := after.Findings[i]
		if f.FindingID != next.FindingID {
			t.Fatal("socket UID entered finding identity")
		}
		if f.Identity.(inventory.EndpointIdentity).LocalPort != 80 {
			if f.FactFingerprint != next.FactFingerprint {
				t.Fatal("unrelated facts changed")
			}
			continue
		}
		facts := next.Facts.(inventory.NetworkFacts)
		if f.FactFingerprint == next.FactFingerprint || len(facts.Owners) != 3 {
			t.Fatalf("socket UID change missed: %+v", facts)
		}
		if !slices.Contains(facts.Owners, inventory.Owner{Executable: "/usr/sbin/fictional-server", UID: 33}) ||
			!slices.Contains(facts.Owners, inventory.Owner{Executable: "/usr/sbin/fictional-server", UID: 44}) ||
			!slices.Contains(facts.Owners, inventory.Owner{Executable: "/opt/fictional-worker", UID: 44}) {
			t.Fatalf("incorrect per-socket attribution: %+v", facts)
		}
	}
}

func TestCancellationAndConfiguration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Collector{Source: fixture(t)}).Collect(ctx, "target"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	s := fixture(t)
	s.cancel = cancel
	if _, err := (Collector{Source: s}).Collect(ctx, "target"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := (Collector{}).Collect(context.Background(), ""); err == nil {
		t.Fatal("empty target")
	}
	if _, err := (Collector{Limits: Limits{Rows: -1}}).Collect(context.Background(), "target"); err == nil {
		t.Fatal("negative limit")
	}
	a, err := (Collector{Source: fixture(t)}).Collect(context.Background(), "target")
	if err != nil {
		t.Fatal(err)
	}
	b, err := (Collector{Source: fixture(t)}).Collect(context.Background(), "target")
	if err != nil {
		t.Fatal(err)
	}
	if a.Coverage.Scope == b.Coverage.Scope {
		t.Fatal("unproven scopes reused")
	}
}

func TestParserMalformedAndEndian(t *testing.T) {
	for raw, want := range map[string]string{"0100007F:0050": "127.0.0.1", "00000000000000000000000001000000:0050": "::1", "B80D0120000000000000000001000000:0050": "2001:db8::1"} {
		family := "ipv4"
		if len(raw) > 13 {
			family = "ipv6"
		}
		got, p, err := address(raw, family)
		if err != nil || got != want || p != 80 {
			t.Fatalf("%s: %s %d %v", raw, got, p, err)
		}
	}
	for _, line := range []string{"broken", "0: XYZ:0050 00000000:0000 0A 0 0 0 33 0 1", "0: 00000000:0050 00000000:0000 ZZ 0 0 0 33 0 1", "0: 00000000:0050 00000000:0000 0A 0 0 0 -1 0 1"} {
		if _, _, err := parseRow(line, "tcp", "ipv4", "scope", "source"); err == nil {
			t.Errorf("accepted %s", line)
		}
	}
}

func TestTransientAndChangedOwnerSnapshots(t *testing.T) {
	s := fixture(t)
	first := collect(t, s, Limits{})
	s.dirs["/proc"] = []string{"40", "20", "30"}
	s.dirs["/proc/40/fd"] = s.dirs["/proc/10/fd"]
	for p, v := range s.links {
		if strings.HasPrefix(p, "/proc/10/") {
			s.links[strings.Replace(p, "/10/", "/40/", 1)] = v
			delete(s.links, p)
		}
	}
	repeat := collect(t, s, Limits{})
	for i, f := range first.Findings {
		if f.FindingID != repeat.Findings[i].FindingID || f.FactFingerprint != repeat.Findings[i].FactFingerprint {
			t.Fatal("PID churn changed identity/facts")
		}
	}
	s.links["/proc/40/exe"] = "/opt/fictional-replacement"
	changed := collect(t, s, Limits{})
	different := false
	for i, f := range first.Findings {
		if f.FindingID != changed.Findings[i].FindingID {
			t.Fatal("owner changed identity")
		}
		different = different || f.FactFingerprint != changed.Findings[i].FactFingerprint
	}
	if !different {
		t.Fatal("owner change missed")
	}
	// Contract snapshots are checked in for T04; regenerate deliberately via the
	// opt-in environment variable and otherwise validate byte-for-byte output.
	for name, r := range map[string]inventory.CollectorResult{"first": first, "transient": repeat, "changed-owner": changed} {
		data, err := inventory.Encode(snapshot(r, name))
		if err != nil {
			t.Fatal(err)
		}
		path := "../../../test/fixtures/prototype/network/snapshot-" + name + ".json"
		if os.Getenv("AVERA_UPDATE_NETWORK_FIXTURES") == "1" {
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		stored, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(stored) != string(data) {
			t.Fatalf("fixture stale: %s", path)
		}
	}
}

func snapshot(r inventory.CollectorResult, name string) *inventory.Snapshot {
	return &inventory.Snapshot{Envelope: inventory.Envelope{SchemaVersion: inventory.SchemaVersion, ContractRevision: inventory.ContractRevision, Kind: "snapshot", TargetID: "fictional-web-01"}, SnapshotID: "network-" + name, CollectedAt: "2026-10-01T09:00:00Z", FinishedAt: "2026-10-01T09:00:00Z", ToolVersion: Version, Environment: inventory.Environment{OSID: "ubuntu", OSVersion: "fictional", Architecture: "amd64", KernelRelease: "fictional", ExecutionScope: "fictional current network namespace"}, Coverage: []inventory.Coverage{r.Coverage, {Collector: inventory.Sudo, CollectorVersion: "not-implemented", Scope: "local", Status: inventory.NotCollected, AttemptedSources: []string{}, DiagnosticCodes: []string{}, Limitations: []string{}}}, Findings: r.Findings, Evidence: r.Evidence}
}

func TestEvidenceTimestampUsesContractPrecision(t *testing.T) {
	result, err := (Collector{Source: fixture(t), ScopeID: "fictional-scope", Now: func() time.Time {
		return time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.FixedZone("test", 3600))
	}}).Collect(context.Background(), "fictional-web-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) == 0 {
		t.Fatal("missing evidence")
	}
	for _, evidence := range result.Evidence {
		if evidence.CollectedAt != "2026-10-24T11:00:00Z" {
			t.Fatalf("non-contract timestamp %q", evidence.CollectedAt)
		}
	}
}
