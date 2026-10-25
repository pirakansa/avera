// Package sudo inventories a bounded subset of local configured sudo rules.
// It does not evaluate effective privileges or execute sudo policy commands.
package sudo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pirakansa/avera/internal/inventory"
)

const Version = "local-sudo-v1"
const defaultEntry = "/etc/sudoers"

// Limits bounds the whole run, including identity files and unsuccessful sources.
// Zero fields select defaults; negative values are rejected.
type Limits struct{ Bytes, Files, Entries, Lines, Depth, PathBytes, ContextBytes int }

type Collector struct {
	Source Source
	Now    func() time.Time
	Entry  string
	Limits Limits
}

type scan struct {
	ctx                          context.Context
	source                       Source
	result                       inventory.CollectorResult
	limits                       Limits
	bytes, files, entries, lines int
	timestamp                    string
	policy                       []string
	active, seen                 map[string]bool
	mainRead                     bool
}

func (c Collector) Collect(ctx context.Context, target string) (inventory.CollectorResult, error) {
	r := inventory.CollectorResult{Findings: []inventory.Finding{}, Evidence: []inventory.Evidence{}, Coverage: inventory.Coverage{
		Collector: inventory.Sudo, CollectorVersion: Version, Scope: "local configured sudoers and local passwd/group only", Status: inventory.Complete,
		AttemptedSources: []string{}, DiagnosticCodes: []string{}, Limitations: []string{
			"Configured rules only; effective permissions, policy backend selection, external identities and file ownership/mode eligibility are not resolved.",
			"Non-atomic local reads; no sudo commands or NSS lookups. Unsupported context limits all rule interpretations.",
		},
	}}
	if target == "" {
		return r, fmt.Errorf("target ID is required")
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if c.Source == nil {
		return r, fmt.Errorf("explicit local source is required")
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Entry == "" {
		c.Entry = defaultEntry
	}
	if !path.IsAbs(c.Entry) || path.Clean(c.Entry) != c.Entry {
		return r, fmt.Errorf("canonical absolute entry path is required")
	}
	values := []*int{&c.Limits.Bytes, &c.Limits.Files, &c.Limits.Entries, &c.Limits.Lines, &c.Limits.Depth, &c.Limits.PathBytes, &c.Limits.ContextBytes}
	defaults := []int{4 << 20, 256, 4096, 16384, 32, 4096, 16 << 20}
	for i, v := range values {
		if *v < 0 {
			return r, fmt.Errorf("negative source limit")
		}
		if *v == 0 {
			*v = defaults[i]
		}
	}
	s := scan{ctx: ctx, source: c.Source, result: r, limits: c.Limits, timestamp: c.Now().UTC().Format(time.RFC3339), policy: []string{}, active: map[string]bool{}, seen: map[string]bool{}}
	s.walk(c.Entry, 0)
	accounts, groups, identitiesOK := s.identities()
	if !identitiesOK {
		s.diagnose("local_identity_incomplete", "/etc/passwd,/etc/group", "")
	}
	// Context includes errors as well as ordered source hashes. It conservatively
	// invalidates every rule when any included source or local identity changes.
	for i := range s.result.Findings {
		f := &s.result.Findings[i]
		if facts, ok := f.Facts.(inventory.SudoFacts); ok {
			facts.LocalMembers, facts.LocalContextComplete = members(facts.Principal, accounts, groups, identitiesOK)
			for _, name := range facts.RunAsUsers {
				if name != "ALL" {
					if _, ok := accounts[name]; !ok {
						facts.LocalContextComplete = false
					}
				}
			}
			for _, name := range facts.RunAsGroups {
				if name != "ALL" {
					if _, ok := groups[name]; !ok {
						facts.LocalContextComplete = false
					}
				}
			}
			if !facts.LocalContextComplete {
				s.diagnose("local_principal_unresolved", f.Identity.(inventory.SourceIdentity).SourcePath, "")
			}
			f.Facts = facts
		}
	}
	contextTokens := append([]string{}, s.policy...)
	contextSize, rules := 0, 0
	for _, token := range contextTokens {
		contextSize += len(token)
	}
	for _, f := range s.result.Findings {
		if f.Kind == inventory.Rule {
			rules++
		}
	}
	// Bound serialized context amplification as well as input bytes. When the
	// complete ordered context cannot fit, no rule claims complete interpretation.
	if rules > 0 && contextSize > s.limits.ContextBytes/rules {
		s.diagnose("context_limit", c.Entry, "ordered context exceeds total output budget")
		contextTokens = []string{"context unavailable: context_limit"}
	}
	complete := len(s.result.Coverage.DiagnosticCodes) == 0
	for i := range s.result.Findings {
		f := s.result.Findings[i]
		if facts, ok := f.Facts.(inventory.SudoFacts); ok {
			facts.PolicyContext = slices.Clone(contextTokens)
			facts.InterpretationComplete = complete
			f.Facts = facts
		}
		normalized, err := inventory.NormalizeFinding(target, f)
		if err != nil {
			return s.result, err
		}
		s.result.Findings[i] = normalized
	}
	if !s.mainRead {
		s.result.Coverage.Status = inventory.Failed
	}
	slices.Sort(s.result.Coverage.AttemptedSources)
	s.result.Coverage.AttemptedSources = slices.Compact(s.result.Coverage.AttemptedSources)
	slices.Sort(s.result.Coverage.DiagnosticCodes)
	return s.result, ctx.Err()
}

func (s *scan) evidence(location, excerpt string) string {
	id := fmt.Sprintf("sudo-%d", len(s.result.Evidence)+1)
	s.result.Evidence = append(s.result.Evidence, inventory.Evidence{EvidenceID: id, SourceKind: "local_file", Location: location, CollectedAt: s.timestamp, Method: "bounded local configured-policy inventory; not effective permissions", Values: inventory.FileValues{Excerpt: excerpt}})
	return id
}

func (s *scan) diagnose(code, location, excerpt string) {
	s.result.Coverage.Status = inventory.Partial
	if !slices.Contains(s.result.Coverage.DiagnosticCodes, code) {
		s.result.Coverage.DiagnosticCodes = append(s.result.Coverage.DiagnosticCodes, code)
	}
	s.policy = append(s.policy, "diagnostic:"+code+":"+location)
	s.evidence(location, code+": "+excerpt)
}

func (s *scan) read(name string) ([]byte, bool) {
	if s.ctx.Err() != nil {
		return nil, false
	}
	if len(name) > s.limits.PathBytes || s.files >= s.limits.Files || s.bytes >= s.limits.Bytes {
		s.diagnose("source_limit", name, "")
		return nil, false
	}
	s.files++
	s.result.Coverage.AttemptedSources = append(s.result.Coverage.AttemptedSources, name)
	data, err := s.source.ReadFile(name, s.limits.Bytes-s.bytes)
	if err != nil {
		code := "source_unreadable"
		if errors.Is(err, errLimit) {
			code = "source_limit"
			s.bytes = s.limits.Bytes
		}
		s.diagnose(code, name, "")
		return nil, false
	}
	s.bytes += len(data)
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		s.diagnose("source_malformed", name, "")
		return nil, false
	}
	s.policy = append(s.policy, fmt.Sprintf("%s:sha256:%x", name, sha256.Sum256(data)))
	return data, true
}

func (s *scan) walk(name string, depth int) {
	if s.ctx.Err() != nil {
		return
	}
	if depth > s.limits.Depth {
		s.diagnose("include_depth_limit", name, "")
		return
	}
	if s.active[name] {
		s.diagnose("include_cycle", name, "")
		return
	}
	if s.seen[name] {
		s.diagnose("repeated_source", name, "")
		return
	}
	s.seen[name] = true
	data, ok := s.read(name)
	if !ok {
		return
	}
	if depth == 0 {
		s.mainRead = true
	}
	s.active[name] = true
	defer delete(s.active, name)
	s.evidence(name, "source inventoried")
	continued := false
	for n, raw := range strings.Split(string(data), "\n") {
		if s.ctx.Err() != nil {
			return
		}
		if s.lines >= s.limits.Lines {
			s.diagnose("line_limit", name, "")
			return
		}
		s.lines++
		line := strings.TrimSpace(raw)
		location := fmt.Sprintf("%s:%d", name, n+1)
		wasContinued := continued
		continued = strings.HasSuffix(line, `\`)
		if !wasContinued && !continued && (line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#include") && (len(line) < 2 || line[1] < '0' || line[1] > '9'))) {
			continue
		}
		s.policy = append(s.policy, raw)
		if !wasContinued && !continued {
			if inc, directory, recognized := include(line); recognized {
				s.evidence(location, line)
				if inc == "" {
					s.unparsed(name, n+1, raw, "unsupported_include")
					continue
				}
				if !path.IsAbs(inc) {
					inc = path.Join(path.Dir(name), inc)
				} else {
					inc = path.Clean(inc)
				}
				if directory {
					s.directory(inc, depth+1)
				} else {
					s.walk(inc, depth+1)
				}
				continue
			}
			if facts, understood := parseRule(line); understood {
				eid := s.evidence(location, line)
				s.result.Findings = append(s.result.Findings, inventory.Finding{Collector: inventory.Sudo, Kind: inventory.Rule, Identity: inventory.SourceIdentity{SourcePath: name, Line: int64(n + 1)}, Facts: facts, EvidenceIDs: []string{eid}})
				continue
			}
		}
		if line == "" {
			raw = "[empty continuation segment]"
		}
		s.unparsed(name, n+1, raw, "unsupported_construct")
	}
}

func (s *scan) unparsed(name string, line int, raw, code string) {
	location := fmt.Sprintf("%s:%d", name, line)
	s.diagnose(code, location, raw)
	eid := s.result.Evidence[len(s.result.Evidence)-1].EvidenceID
	s.result.Findings = append(s.result.Findings, inventory.Finding{Collector: inventory.Sudo, Kind: inventory.Unparsed, Identity: inventory.SourceIdentity{SourcePath: name, Line: int64(line)}, Facts: inventory.UnparsedFacts{Raw: raw, DiagnosticCode: code}, EvidenceIDs: []string{eid}})
}

func (s *scan) directory(name string, depth int) {
	if s.ctx.Err() != nil {
		return
	}
	if len(name) > s.limits.PathBytes || depth > s.limits.Depth || s.files >= s.limits.Files || s.entries >= s.limits.Entries {
		s.diagnose("directory_limit", name, "")
		return
	}
	s.files++
	s.result.Coverage.AttemptedSources = append(s.result.Coverage.AttemptedSources, name)
	names, err := s.source.ReadDir(name, s.limits.Entries-s.entries)
	if err != nil {
		code := "directory_unreadable"
		if errors.Is(err, errLimit) {
			code = "directory_limit"
		}
		s.diagnose(code, name, "")
		return
	}
	s.entries += len(names)
	slices.Sort(names)
	s.policy = append(s.policy, "directory:"+name)
	s.evidence(name, "include directory inventoried in lexical order")
	for _, item := range names {
		if strings.Contains(item, ".") || strings.HasSuffix(item, "~") {
			continue
		}
		if item == "" || strings.Contains(item, "/") {
			s.diagnose("source_malformed", name, "")
			continue
		}
		s.walk(path.Join(name, item), depth)
	}
}
