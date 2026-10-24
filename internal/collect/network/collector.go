package network

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pirakansa/avera/internal/inventory"
)

const Version = "proc-v1"

var errLimit = errors.New("source limit exceeded")

// Limits bounds all source enumeration. Zero fields use the documented defaults.
type Limits struct {
	SourceBytes int
	Rows        int
	Processes   int
	Descriptors int // Total descriptors across the run.
	PathBytes   int
	Evidence    int // Total evidence entries, including duplicate socket observations.
}

func defaultLimits() Limits {
	return Limits{SourceBytes: 4 << 20, Rows: 16384, Processes: 4096, Descriptors: 65536, PathBytes: 4096, Evidence: 65536}
}

// Collector returns domain observations only; privilege enforcement belongs to
// the application boundary. Tests can use synthetic sources without root.
type Collector struct {
	Source Source
	Now    func() time.Time
	Limits Limits
	// ScopeID is an assertion of comparable namespace incarnation by the caller.
	// Leave empty for live reads: proc namespace inode reuse cannot establish
	// stable incarnation, so each run conservatively gets a fresh scope.
	ScopeID string
}

func (c Collector) Collect(ctx context.Context, target string) (inventory.CollectorResult, error) {
	r := inventory.CollectorResult{Findings: []inventory.Finding{}, Evidence: []inventory.Evidence{}, Coverage: inventory.Coverage{
		Collector: inventory.Network, CollectorVersion: Version, Status: inventory.Complete,
		AttemptedSources: []string{}, DiagnosticCodes: []string{},
		Limitations: []string{"Current observable network namespace only; no reachability, firewall, dual-stack or executable-integrity inference.", "UDP includes only state 07, nonzero local port and unspecified remote endpoint; connected UDP is excluded.", "Collection is non-atomic; process and socket races may limit attribution. IPv6 zones are unavailable in these tables."},
	}}
	if target == "" {
		return r, fmt.Errorf("target ID is required")
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return r, fmt.Errorf("network collector supports linux/amd64 only")
	}
	if c.Source == nil {
		c.Source = ProcSource{}
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	l := c.Limits
	d := defaultLimits()
	values := []*int{&l.SourceBytes, &l.Rows, &l.Processes, &l.Descriptors, &l.PathBytes, &l.Evidence}
	defaults := []int{d.SourceBytes, d.Rows, d.Processes, d.Descriptors, d.PathBytes, d.Evidence}
	for i, v := range values {
		if *v < 0 {
			return r, fmt.Errorf("negative source limit")
		}
		if *v == 0 {
			*v = defaults[i]
		}
	}
	if c.ScopeID == "" {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return r, fmt.Errorf("create conservative scope: %w", err)
		}
		c.ScopeID = fmt.Sprintf("unverified-netns:%x", nonce)
		r.Coverage.Limitations = append(r.Coverage.Limitations, "Namespace incarnation cannot be proven from proc inode numbers; fresh scope requires re-review across runs.")
	}
	r.Coverage.Scope = c.ScopeID
	diagnose := func(code string) {
		r.Coverage.Status = inventory.Partial
		if !slices.Contains(r.Coverage.DiagnosticCodes, code) {
			r.Coverage.DiagnosticCodes = append(r.Coverage.DiagnosticCodes, code)
		}
	}
	timestamp := c.Now().UTC().Format(time.RFC3339)
	sockets := []socket{}
	usable := 0
	for _, table := range []struct{ name, transport, family string }{{"tcp", "tcp", "ipv4"}, {"tcp6", "tcp", "ipv6"}, {"udp", "udp", "ipv4"}, {"udp6", "udp", "ipv6"}} {
		path := "/proc/net/" + table.name
		r.Coverage.AttemptedSources = append(r.Coverage.AttemptedSources, path)
		rows, valid, code, err := readTable(ctx, c.Source, path, table.transport, table.family, c.ScopeID, l)
		if err != nil {
			return r, err
		}
		if valid {
			usable++
		}
		if code != "" {
			diagnose(code)
		}
		sockets = append(sockets, rows...)
	}
	owners, attributionOK, err := attribute(ctx, c.Source, sockets, l, diagnose)
	if err != nil {
		return r, err
	}
	if len(sockets) > 0 {
		r.Coverage.AttemptedSources = append(r.Coverage.AttemptedSources, "/proc/*/fd", "/proc/*/exe")
	}
	for i, s := range sockets {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		if len(r.Evidence) >= l.Evidence {
			diagnose("evidence_limit")
			break
		}
		observations := owners[s.inode]
		truncated := len(observations) > l.Evidence-len(r.Evidence)-1
		if truncated {
			observations = observations[:l.Evidence-len(r.Evidence)-1]
			diagnose("evidence_limit")
		}
		observations = slices.Clone(observations)
		for j := range observations {
			observations[j].owner.UID = s.uid
		}
		facts := inventory.NetworkFacts{Owners: []inventory.Owner{}, AttributionComplete: attributionOK && !truncated && len(observations) > 0 && s.inode != "0"}
		for _, owner := range observations {
			facts.Owners = append(facts.Owners, owner.owner)
			if owner.owner.Executable == "" {
				facts.AttributionComplete = false
			}
		}
		if len(observations) == 0 {
			facts.Owners = append(facts.Owners, inventory.Owner{UID: s.uid})
		}
		if !facts.AttributionComplete {
			diagnose("attribution_incomplete")
		}
		id := fmt.Sprintf("network-%d", i+1)
		r.Evidence = append(r.Evidence, inventory.Evidence{EvidenceID: id, SourceKind: "proc", Location: s.location, CollectedAt: timestamp, Method: "bounded proc socket row and inode-to-fd attribution; UID is observed socket UID in reader user namespace", Values: inventory.ProcessValues{Observed: facts}})
		ids := []string{id}
		for j, owner := range observations {
			pid := owner.pid
			eid := fmt.Sprintf("network-%d-owner-%d", i+1, j+1)
			r.Evidence = append(r.Evidence, inventory.Evidence{EvidenceID: eid, SourceKind: "proc", Location: fmt.Sprintf("/proc/%d/fd/%s", pid, owner.fd), CollectedAt: timestamp, Method: "socket inode link and bounded executable path; UID is observed socket UID in reader user namespace", Values: inventory.ProcessValues{Observed: inventory.NetworkFacts{Owners: []inventory.Owner{owner.owner}, AttributionComplete: facts.AttributionComplete}, PID: &pid}})
			ids = append(ids, eid)
		}
		r.Findings = append(r.Findings, inventory.Finding{Collector: inventory.Network, Kind: inventory.Endpoint, Identity: s.identity, Facts: facts, EvidenceIDs: ids})
	}
	r.Findings, err = inventory.MergeEndpoints(target, r.Findings)
	if err != nil {
		return r, err
	}
	if usable == 0 && len(r.Findings) == 0 {
		r.Coverage.Status = inventory.Failed
	}
	slices.Sort(r.Coverage.DiagnosticCodes)
	return r, ctx.Err()
}

func readTable(ctx context.Context, source Source, path, transport, family, scope string, l Limits) ([]socket, bool, string, error) {
	f, err := source.Open(path)
	if err != nil {
		return nil, false, "source_unreadable", nil
	}
	limited := &io.LimitedReader{R: f, N: int64(l.SourceBytes) + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 1024), l.SourceBytes+1)
	rows := []socket{}
	valid, code, line := false, "", 0
	consumed := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, false, "", err
		}
		line++
		consumed += len(scanner.Bytes()) + 1
		if consumed > l.SourceBytes || line > l.Rows+1 {
			code = "source_limit"
			break
		}
		if line == 1 {
			h := strings.Fields(scanner.Text())
			if len(h) < 4 || h[0] != "sl" || h[1] != "local_address" || (h[2] != "rem_address" && h[2] != "remote_address") || h[3] != "st" {
				code = "source_malformed"
				break
			}
			valid = true
			continue
		}
		s, include, err := parseRow(scanner.Text(), transport, family, scope, fmt.Sprintf("%s:%d", path, line))
		if err != nil {
			code = "source_malformed"
			continue
		}
		if include {
			rows = append(rows, s)
		}
	}
	if scanner.Err() != nil {
		code = "source_read_failed"
	}
	if limited.N == 0 {
		code = "source_limit"
	}
	if line == 0 {
		code = "source_malformed"
	}
	if err := f.Close(); err != nil {
		code = "source_read_failed"
	}
	return rows, valid, code, ctx.Err()
}

type ownerObservation struct {
	owner inventory.Owner
	pid   int64
	fd    string
}

func attribute(ctx context.Context, source Source, sockets []socket, l Limits, diagnose func(string)) (map[string][]ownerObservation, bool, error) {
	out := map[string][]ownerObservation{}
	if len(sockets) == 0 {
		return out, true, nil
	}
	wanted := map[string]uint32{}
	for _, s := range sockets {
		if s.inode != "0" {
			wanted[s.inode] = s.uid
		}
	}
	pids, more, err := source.ReadDir("/proc", l.Processes)
	complete := !more && err == nil
	if more {
		diagnose("process_limit")
	}
	if err != nil {
		diagnose("process_unreadable")
	}
	slices.Sort(pids)
	remaining := l.Descriptors
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			return out, false, err
		}
		n, err := strconv.ParseInt(pid, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		if remaining == 0 {
			complete = false
			diagnose("descriptor_limit")
			break
		}
		fds, more, err := source.ReadDir("/proc/"+pid+"/fd", remaining)
		if more || err != nil {
			complete = false
			diagnose("descriptors_incomplete")
		}
		remaining -= len(fds)
		slices.Sort(fds)
		seen := map[string]bool{}
		for _, fd := range fds {
			if err := ctx.Err(); err != nil {
				return out, false, err
			}
			link, err := source.Readlink("/proc/"+pid+"/fd/"+fd, l.PathBytes)
			if err != nil {
				complete = false
				diagnose("descriptor_unreadable")
				continue
			}
			if !strings.HasPrefix(link, "socket:[") || !strings.HasSuffix(link, "]") {
				continue
			}
			inode := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
			uid, ok := wanted[inode]
			if !ok || seen[inode] {
				continue
			}
			seen[inode] = true
			exe, err := source.Readlink("/proc/"+pid+"/exe", l.PathBytes)
			if err != nil || exe == "" || !utf8.ValidString(exe) {
				complete = false
				diagnose("executable_unreadable")
				exe = ""
			}
			out[inode] = append(out[inode], ownerObservation{inventory.Owner{Executable: exe, UID: uid}, n, fd})
		}
	}
	return out, complete, nil
}
