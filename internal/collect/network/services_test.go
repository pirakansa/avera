package network

import (
	"context"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	i "github.com/pirakansa/avera/internal/inventory"
)

type serviceSource struct {
	*fakeSource
	opened    []string
	statReads int
	changed   bool
}

func (s *serviceSource) Open(path string) (io.ReadCloser, error) {
	s.opened = append(s.opened, path)
	if strings.HasSuffix(path, "/stat") {
		s.statReads++
		if s.changed && s.statReads == 2 {
			return io.NopCloser(strings.NewReader(serviceStat("999"))), nil
		}
	}
	return s.fakeSource.Open(path)
}
func serviceStat(start string) string {
	return "10 (fictional worker) S " + strings.Repeat("0 ", 18) + start + "\n"
}
func servicesFixture(t *testing.T) *serviceSource {
	s := &serviceSource{fakeSource: fixture(t)}
	for _, pid := range []string{"10", "20"} {
		s.files["/proc/"+pid+"/status"] = "Name:\tfictional-worker\nUid:\t1000\t2000\t1000\t1000\n"
		s.files["/proc/"+pid+"/stat"] = serviceStat("123")
		s.files["/proc/"+pid+"/cgroup"] = "0::/system.slice/fictional.service\n"
	}
	return s
}
func collectServices(t *testing.T, s Source) i.CollectorResult {
	t.Helper()
	r, err := (Collector{Source: s, ScopeID: "fictional-boot-a/net-a", IncludeServices: true}).Collect(context.Background(), "fictional-web-01")
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestServicesFollowOnlyObservedSocketHolders(t *testing.T) {
	s := servicesFixture(t)
	baseline := collect(t, s.fakeSource, Limits{})
	r := collectServices(t, s)
	if r.Coverage.Status != i.Complete {
		t.Fatalf("coverage: %+v", r.Coverage)
	}
	for _, path := range s.opened {
		if strings.HasPrefix(path, "/proc/30/") {
			t.Fatal("unrelated process inspected", path)
		}
	}
	for _, pid := range []string{"10", "20"} {
		for _, file := range []string{"status", "cgroup"} {
			if n := slicesCount(s.opened, "/proc/"+pid+"/"+file); n != 1 {
				t.Fatalf("metadata read %d times", n)
			}
		}
	}
	count := 0
	for _, e := range r.Evidence {
		v := e.Values.(i.ProcessValues)
		if v.Service == nil {
			continue
		}
		count++
		if v.Service.EffectiveUID == nil || *v.Service.EffectiveUID != 2000 || v.Service.UnitCandidate != "fictional.service" {
			t.Fatalf("service: %+v", v.Service)
		}
	}
	if count == 0 {
		t.Fatal("missing service evidence")
	}
	for n, f := range r.Findings {
		if f.FindingID != baseline.Findings[n].FindingID || f.FactFingerprint != baseline.Findings[n].FactFingerprint || !reflect.DeepEqual(f.Facts, baseline.Findings[n].Facts) {
			t.Fatal("metadata changed endpoint matching")
		}
	}
	snapshot := snapshot(r, "services")
	snapshot.ContractRevision = i.ContractRevision
	encoded, err := i.Encode(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := i.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	again, err := i.Encode(decoded)
	if err != nil || string(encoded) != string(again) {
		t.Fatal("service evidence lost in roundtrip", err)
	}
	snapshot.ContractRevision = i.ForwardingContractRevision
	if _, err = i.Encode(snapshot); err == nil {
		t.Fatal("service evidence accepted under old revision")
	}
}
func slicesCount(values []string, want string) int {
	n := 0
	for _, v := range values {
		if v == want {
			n++
		}
	}
	return n
}
func TestServiceFailuresRetainEndpointsAndExplicitUnknowns(t *testing.T) {
	for _, tc := range []struct {
		name, path, code string
		value            *string
		changed          bool
	}{
		{"missing status", "/proc/10/status", "service_status_unreadable", nil, false},
		{"bad status", "/proc/10/status", "service_status_malformed", new("Name: worker\nUid: bad\n"), false},
		{"missing cgroup", "/proc/10/cgroup", "service_cgroup_unreadable", nil, false},
		{"bad cgroup", "/proc/10/cgroup", "service_cgroup_malformed", new("broken\n"), false},
		{"oversized status", "/proc/10/status", "service_status_unreadable", new(strings.Repeat("x", serviceSourceBytes+1)), false},
		{"process replaced", "", "service_process_changed", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := servicesFixture(t)
			s.changed = tc.changed
			if tc.path != "" {
				if tc.value == nil {
					delete(s.files, tc.path)
				} else {
					s.files[tc.path] = *tc.value
				}
			}
			r := collectServices(t, s)
			if len(r.Findings) != 6 || r.Coverage.Status != i.Partial || !slices.Contains(r.Coverage.DiagnosticCodes, tc.code) {
				t.Fatalf("lost endpoint or diagnostic: %+v", r.Coverage)
			}
			if tc.changed {
				for _, e := range r.Evidence {
					v := e.Values.(i.ProcessValues)
					if v.PID != nil && *v.PID == 10 && (v.Service.EffectiveUID != nil || v.Service.UnitCandidate != "") {
						t.Fatal("mixed process metadata retained")
					}
				}
			}
		})
	}
}
func TestServiceCgroupCandidates(t *testing.T) {
	for _, tc := range []struct{ text, unit, path string }{
		{"0::/system.slice/web@blue.service\n", "web@blue.service", "/system.slice/web@blue.service"},
		{"2:cpu:/unrelated.service\n1:name=systemd:/system.slice/web.service\n", "web.service", "/system.slice/web.service"},
		{"0::/user.slice/user@1000.service/app.slice/worker.service\n", "worker.service", "/user.slice/user@1000.service/app.slice/worker.service"},
		{"0::/docker/fictional\n", "", "/docker/fictional"},
	} {
		unit, path, err := serviceCgroup(tc.text)
		if err != nil || unit != tc.unit || path != tc.path {
			t.Fatalf("%q: %s %s %v", tc.text, unit, path, err)
		}
	}
	for _, text := range []string{"", "bad::/a.service\n", "0::/a.service\n0::/b.service\n"} {
		if _, _, err := serviceCgroup(text); err == nil {
			t.Fatal("accepted empty, malformed or ambiguous hierarchy")
		}
	}
}

func TestServiceCollectionDoesNotEnumerateDaemons(t *testing.T) {
	s := servicesFixture(t)
	for _, pid := range []string{"10", "20"} {
		s.files["/proc/"+pid+"/cgroup"] = "0::/fictional-container\n"
	}
	s.files["/proc/10/status"] += "Unrelated: fictional-secret-must-not-be-retained\n"
	r := collectServices(t, s)
	if r.Coverage.Status != i.Complete {
		t.Fatal(r.Coverage)
	}
	sample := snapshot(r, "services")
	sample.ContractRevision = i.ContractRevision
	data, err := i.Encode(sample)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "fictional-secret-must-not-be-retained") {
		t.Fatal("unrelated status data retained")
	}
	s = servicesFixture(t)
	for _, name := range []string{"tcp", "tcp6", "udp", "udp6"} {
		s.files["/proc/net/"+name] = header
	}
	empty := collectServices(t, s)
	if len(empty.Findings) != 0 || len(s.opened) != 4 || empty.Coverage.Status != i.Complete {
		t.Fatal("empty sockets caused process inventory", s.opened, empty.Coverage)
	}
}
