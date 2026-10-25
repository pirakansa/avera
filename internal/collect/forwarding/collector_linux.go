// Package forwarding observes destination translation configuration without changing rules.
package forwarding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	i "github.com/pirakansa/avera/internal/inventory"
)

const MaxSourceBytes = 128 << 10
const commandTimeout = 10 * time.Second

// ReadFunc must bound output to MaxSourceBytes and honor cancellation.
type ReadFunc func(context.Context, string, ...string) ([]byte, error)

type Collector struct {
	Read    ReadFunc
	Now     func() time.Time
	ScopeID string
}

// Collect retains translation candidates, including ambiguous destinations, for human review.
// Coverage is merged into the network collector by the application registration.
func (c Collector) Collect(ctx context.Context, target string) (i.CollectorResult, error) {
	r := i.CollectorResult{Findings: []i.Finding{}, Evidence: []i.Evidence{}, Coverage: i.Coverage{Collector: i.Network, CollectorVersion: "netfilter-v1", Scope: c.ScopeID, Status: i.Complete, AttemptedSources: []string{}, DiagnosticCodes: []string{}, Limitations: []string{
		"Current observable namespace only; destination translation candidates are configuration, not proof of reachable or effective paths. No firewall simulation, reverse proxy, upstream router or cloud inspection.",
		"All DNAT/REDIRECT/TPROXY candidates are retained, including non-loopback and indirect destinations, to avoid silently omitting possible loopback paths. Humans confirm destinations and safeguards.",
		"iptables and nftables views may overlap; candidates are not deduplicated across backends. Rulesets are read sequentially and non-atomically. Missing backends are unknown, never absence.",
	}}}
	if target == "" || c.ScopeID == "" {
		return r, fmt.Errorf("target and namespace scope required")
	}
	if c.Read == nil {
		c.Read = ReadCommand
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	usable := 0
	for _, source := range []string{"iptables-save", "ip6tables-save", "iptables-legacy-save", "ip6tables-legacy-save", "nft"} {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		args := []string{}
		if source == "nft" {
			args = []string{"--json", "list", "ruleset"}
		}
		r.Coverage.AttemptedSources = append(r.Coverage.AttemptedSources, "command:"+source+" "+strings.Join(args, " "))
		data, err := c.Read(ctx, source, args...)
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		diagnose := func(code string) {
			r.Coverage.Status = i.Partial
			r.Coverage.DiagnosticCodes = append(r.Coverage.DiagnosticCodes, source+":"+code)
		}
		if err != nil {
			diagnose("source_unavailable")
			continue
		}
		if len(data) > MaxSourceBytes || !utf8.Valid(data) {
			diagnose("source_limit_or_encoding")
			continue
		}
		contextText, rules, err := Parse(source, data)
		if err != nil {
			diagnose("source_malformed")
			continue
		}
		if len(contextText) > MaxSourceBytes {
			diagnose("normalized_source_limit")
			continue
		}
		usable++
		digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(contextText)))
		eid := "forwarding-" + source
		stamp := c.Now().UTC().Format(time.RFC3339)
		r.Evidence = append(r.Evidence, i.Evidence{EvidenceID: eid, SourceKind: "command_output", Location: "command:" + source, CollectedAt: stamp, Method: "bounded read-only ruleset; volatile counters removed; full context retained, no reachability inference", Values: i.FileValues{Excerpt: contextText}})
		for n, rule := range rules {
			finding, err := i.NormalizeFinding(target, i.Finding{Collector: i.Network, Kind: i.Forwarding, Identity: i.SourceIdentity{SourcePath: "/netfilter/" + source, Line: int64(n + 1), Ordinal: 0}, Facts: i.ForwardingFacts{Rule: rule, ContextDigest: digest, ScopeID: c.ScopeID}, EvidenceIDs: []string{eid}})
			if err != nil {
				return r, err
			}
			r.Findings = append(r.Findings, finding)
		}
	}
	if usable == 0 {
		r.Coverage.Status = i.Failed
	}
	return r, nil
}

var chainCounter = regexp.MustCompile(`\[[0-9]+:[0-9]+\]`)

// Parse preserves complete configuration context. Rule positions are candidate ordinals,
// not source line numbers. It deliberately does not evaluate chain reachability.
func Parse(source string, data []byte) (string, []string, error) {
	rules := []string{}
	if source == "nft" {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(data, &doc); err != nil {
			return "", nil, err
		}
		var entries []map[string]json.RawMessage
		if raw, ok := doc["nftables"]; !ok || json.Unmarshal(raw, &entries) != nil || entries == nil {
			return "", nil, fmt.Errorf("missing nftables array")
		}
		normalized := []any{}
		for _, entry := range entries {
			if len(entry) != 1 {
				return "", nil, fmt.Errorf("invalid nftables entry")
			}
			if _, ok := entry["metainfo"]; ok {
				continue
			}
			var value any
			raw, err := json.Marshal(entry)
			if err != nil {
				return "", nil, err
			}
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return "", nil, err
			}
			stripCounters(value)
			raw, err = json.Marshal(value)
			if err != nil {
				return "", nil, err
			}
			normalized = append(normalized, value)
			if hasTranslation(value) {
				rules = append(rules, string(raw))
			}
		}
		out, err := json.Marshal(map[string]any{"nftables": normalized})
		return string(out), rules, err
	}
	lines := []string{}
	inTable, complete := false, false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "*") && len(strings.Fields(line)) == 1 && len(line) > 1 && !inTable {
			inTable = true
			lines = append(lines, line)
			continue
		}
		if line == "COMMIT" && inTable {
			inTable = false
			complete = true
			lines = append(lines, line)
			continue
		}
		if !inTable || (!strings.HasPrefix(line, ":") && !strings.HasPrefix(line, "-A ")) {
			return "", nil, fmt.Errorf("invalid save structure")
		}
		if strings.HasPrefix(line, ":") {
			line = chainCounter.ReplaceAllString(line, "[0:0]")
		}
		lines = append(lines, line)
		fields := strings.Fields(line)
		for n, v := range fields {
			if (v == "-j" || v == "--jump") && n+1 < len(fields) && (fields[n+1] == "DNAT" || fields[n+1] == "REDIRECT" || fields[n+1] == "TPROXY") {
				rules = append(rules, line)
				break
			}
		}
	}
	if len(lines) == 0 {
		return "", rules, nil
	}
	if !complete || inTable {
		return "", nil, fmt.Errorf("incomplete save")
	}
	return strings.Join(lines, "\n"), rules, nil
}
func stripCounters(v any) {
	switch x := v.(type) {
	case map[string]any:
		if counter, ok := x["counter"].(map[string]any); ok {
			delete(counter, "packets")
			delete(counter, "bytes")
		}
		for _, child := range x {
			stripCounters(child)
		}
	case []any:
		for _, child := range x {
			stripCounters(child)
		}
	}
}
func hasTranslation(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for key, child := range x {
			if key == "dnat" || key == "redirect" || key == "tproxy" {
				return true
			}
			if hasTranslation(child) {
				return true
			}
		}
	case []any:
		for _, child := range x {
			if hasTranslation(child) {
				return true
			}
		}
	}
	return false
}

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > MaxSourceBytes {
		return 0, errors.New("command output limit")
	}
	return b.Buffer.Write(p)
}

// ReadCommand uses fixed system directories rather than the caller's PATH and
// never invokes a shell, writes firewall rules, or exposes command stderr.
func ReadCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	allowed := name == "iptables-save" || name == "ip6tables-save" || name == "iptables-legacy-save" || name == "ip6tables-legacy-save" || name == "nft"
	if !allowed {
		return nil, fmt.Errorf("unsupported command")
	}
	var path string
	for _, dir := range []string{"/usr/sbin", "/usr/bin", "/sbin", "/bin"} {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
			path = candidate
			break
		}
	}
	if path == "" {
		return nil, fmt.Errorf("command unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LC_ALL=C"}
	cmd.WaitDelay = time.Second
	var out boundedBuffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cannot read ruleset")
	}
	return out.Bytes(), nil
}
