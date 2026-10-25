// Package accounts inventories bounded local account/group settings, never login access.
package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	i "github.com/pirakansa/avera/internal/inventory"
)

const Version = "local-accounts-v1"
const (
	sourceByteLimit        = 4 << 20
	defaultLineLimit       = 16384
	defaultAccountLimit    = 4096
	defaultMembershipLimit = 65536
	localNameByteLimit     = 256
	accountPathByteLimit   = 4096
	passwdPath             = "/etc/passwd"
	groupPath              = "/etc/group"
)

type Source interface {
	ReadFile(string, int) ([]byte, error)
}
type Collector struct {
	Source Source
	Now    func() time.Time
	// Zero limits select bounded defaults.
	Bytes, Lines, Entries, Memberships int
}
type accountRow struct {
	name  string
	facts i.AccountFacts
	line  int
}

func validName(name string) bool {
	return name != "" && len(name) <= localNameByteLimit && !strings.ContainsAny(name, "/:\x00\r\n\t ") && !strings.HasPrefix(name, "+") && !strings.HasPrefix(name, "-")
}
func number(raw string) (uint32, error) {
	for _, r := range raw {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid identifier")
		}
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	return uint32(v), err
}

func (c Collector) Collect(ctx context.Context, target string) (i.CollectorResult, error) {
	r := i.CollectorResult{Findings: []i.Finding{}, Evidence: []i.Evidence{}, Coverage: i.Coverage{Collector: i.Accounts, CollectorVersion: Version, Scope: "local passwd/group files", Status: i.Complete, AttemptedSources: []string{}, DiagnosticCodes: []string{}, Limitations: []string{"Local account settings only; shells do not establish login access. External identities, SSH authentication/access and non-sudo privilege paths are outside this review, not additional review tasks.", "Non-atomic bounded reads. Password fields, GECOS, shadow, keys and authentication data are not retained. Runtime groups are observed separately from account memberships."}}}
	if target == "" || c.Source == nil {
		return r, fmt.Errorf("target and explicit local source are required")
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	for _, pair := range []struct {
		value        *int
		defaultValue int
	}{{&c.Bytes, sourceByteLimit}, {&c.Lines, defaultLineLimit}, {&c.Entries, defaultAccountLimit}, {&c.Memberships, defaultMembershipLimit}} {
		if *pair.value < 0 {
			return r, fmt.Errorf("negative account limit")
		}
		if *pair.value == 0 {
			*pair.value = pair.defaultValue
		}
	}
	diagnose := func(code string) {
		r.Coverage.Status = i.Partial
		if !slices.Contains(r.Coverage.DiagnosticCodes, code) {
			r.Coverage.DiagnosticCodes = append(r.Coverage.DiagnosticCodes, code)
		}
	}
	accounts := []accountRow{}
	groups := []i.LocalGroup{}
	names := map[string]bool{}
	groupNames := map[string]bool{}
	remaining, lines, edges := c.Bytes, c.Lines, c.Memberships
	readable := 0
	for _, path := range []string{passwdPath, groupPath} {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		r.Coverage.AttemptedSources = append(r.Coverage.AttemptedSources, path)
		if remaining == 0 {
			diagnose("accounts_byte_limit")
			continue
		}
		data, err := c.Source.ReadFile(path, remaining)
		if err != nil || len(data) > remaining || !utf8.Valid(data) {
			diagnose("accounts_source_unreadable")
			continue
		}
		remaining -= len(data)
		readable++
		for n, line := range strings.Split(string(data), "\n") {
			if err := ctx.Err(); err != nil {
				return r, err
			}
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if lines == 0 {
				diagnose("accounts_line_limit")
				break
			}
			lines--
			fields := strings.Split(line, ":")
			expected := 7
			if path == groupPath {
				expected = 4
			}
			if len(fields) != expected || !validName(fields[0]) {
				diagnose("accounts_source_malformed")
				continue
			}
			id, err := number(fields[2])
			if err != nil {
				diagnose("accounts_source_malformed")
				continue
			}
			if path == passwdPath {
				if len(accounts) >= c.Entries {
					diagnose("accounts_entry_limit")
					break
				}
				gid, err := number(fields[3])
				if err != nil || names[fields[0]] || len(fields[5]) > accountPathByteLimit || len(fields[6]) > accountPathByteLimit {
					diagnose("accounts_source_malformed")
					continue
				}
				names[fields[0]] = true
				accounts = append(accounts, accountRow{name: fields[0], line: n + 1, facts: i.AccountFacts{UID: id, PrimaryGID: gid, Home: fields[5], Shell: fields[6], Groups: []i.AccountGroup{}}})
			} else {
				if len(groups) >= c.Entries {
					diagnose("accounts_entry_limit")
					break
				}
				if groupNames[fields[0]] {
					diagnose("accounts_source_malformed")
					continue
				}
				groupNames[fields[0]] = true
				members := []string{}
				if fields[3] != "" {
					for _, name := range strings.Split(fields[3], ",") {
						if edges == 0 {
							diagnose("accounts_membership_limit")
							break
						}
						edges--
						if !validName(name) {
							diagnose("accounts_source_malformed")
							continue
						}
						members = append(members, name)
					}
				}
				slices.Sort(members)
				members = slices.Compact(members)
				groups = append(groups, i.LocalGroup{Name: fields[0], GID: id, Members: members})
			}
		}
	}
	// Group member names absent from local passwd remain source observations;
	// external account resolution is outside scope and creates no review item.
	// The same global membership bound limits expansion of primary group aliases.
	for n := range accounts {
		primary := false
		for _, g := range groups {
			membership := ""
			if accounts[n].facts.PrimaryGID == g.GID {
				membership = "primary"
				primary = true
			} else if slices.Contains(g.Members, accounts[n].name) {
				membership = "supplementary"
			}
			if membership == "" {
				continue
			}
			if edges == 0 {
				diagnose("accounts_membership_limit")
				break
			}
			edges--
			accounts[n].facts.Groups = append(accounts[n].facts.Groups, i.AccountGroup{Name: g.Name, GID: g.GID, Membership: membership})
		}
		if !primary {
			diagnose("accounts_primary_group_unresolved")
		}
	}
	complete := r.Coverage.Status == i.Complete
	timestamp := c.Now().UTC().Format(time.RFC3339)
	r.Evidence = append(r.Evidence, i.Evidence{EvidenceID: "accounts-groups", SourceKind: "local_identity", Location: groupPath, CollectedAt: timestamp, Method: "bounded local group directory; password fields discarded", Values: i.AccountValues{Groups: groups}})
	for n, account := range accounts {
		account.facts.ContextComplete = complete
		id := fmt.Sprintf("accounts-user-%d", n+1)
		excerpt, err := json.Marshal(account.facts)
		if err != nil {
			return r, err
		}
		r.Evidence = append(r.Evidence, i.Evidence{EvidenceID: id, SourceKind: "local_file", Location: fmt.Sprintf("%s:%d", passwdPath, account.line), CollectedAt: timestamp, Method: "bounded local account settings; password and GECOS fields discarded; not a login test", Values: i.FileValues{Excerpt: string(excerpt)}})
		f, err := i.NormalizeFinding(target, i.Finding{Collector: i.Accounts, Kind: i.Account, Identity: i.AccountIdentity{Name: account.name}, Facts: account.facts, EvidenceIDs: []string{id, "accounts-groups"}})
		if err != nil {
			return r, err
		}
		r.Findings = append(r.Findings, f)
	}
	if readable == 0 {
		r.Coverage.Status = i.Failed
	}
	slices.SortFunc(r.Findings, func(a, b i.Finding) int { return strings.Compare(a.FindingID, b.FindingID) })
	slices.Sort(r.Coverage.DiagnosticCodes)
	return r, ctx.Err()
}
