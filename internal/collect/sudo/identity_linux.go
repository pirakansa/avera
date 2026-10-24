package sudo

import (
	"strconv"
	"strings"

	"github.com/pirakansa/avera/internal/inventory"
)

type account struct{ uid, gid uint32 }
type group struct {
	gid   uint32
	names []string
}

func number(s string) (uint32, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(s, 10, 32)
	return uint32(n), err == nil
}

func (s *scan) identities() (map[string]account, map[string]group, bool) {
	accounts := map[string]account{}
	groups := map[string]group{}
	complete := true
	for _, name := range []string{"/etc/passwd", "/etc/group"} {
		data, ok := s.read(name)
		if !ok {
			complete = false
			continue
		}
		s.evidence(name, "local identity source inventoried; authentication fields omitted")
		for _, line := range strings.Split(string(data), "\n") {
			if s.ctx.Err() != nil {
				return accounts, groups, false
			}
			if line == "" {
				continue
			}
			if s.lines >= s.limits.Lines {
				s.diagnose("line_limit", name, "")
				complete = false
				break
			}
			s.lines++
			fields := strings.Split(line, ":")
			expected := 7
			if name == "/etc/group" {
				expected = 4
			}
			if len(fields) != expected || fields[0] == "" || strings.ContainsAny(fields[0], "+%#@! ") {
				complete = false
				continue
			}
			id, valid := number(fields[2])
			if !valid {
				complete = false
				continue
			}
			if name == "/etc/passwd" {
				gid, valid := number(fields[3])
				if !valid {
					complete = false
					continue
				}
				if _, exists := accounts[fields[0]]; exists {
					complete = false
					continue
				}
				accounts[fields[0]] = account{uid: id, gid: gid}
			} else {
				if _, exists := groups[fields[0]]; exists {
					complete = false
					continue
				}
				names := []string{}
				if fields[3] != "" {
					names = strings.Split(fields[3], ",")
				}
				groups[fields[0]] = group{gid: id, names: names}
			}
		}
	}
	return accounts, groups, complete
}

func members(principal inventory.Principal, accounts map[string]account, groups map[string]group, complete bool) ([]inventory.Member, bool) {
	out := []inventory.Member{}
	if principal.Type == "user" {
		a, exists := accounts[principal.Name]
		if exists {
			out = append(out, inventory.Member{Name: principal.Name, UID: a.uid, Membership: "primary"})
			primaryFound := false
			for _, g := range groups {
				if g.gid == a.gid {
					primaryFound = true
				}
			}
			complete = complete && primaryFound
		}
		return out, complete && exists
	}
	g, exists := groups[principal.Name]
	if !exists {
		return out, false
	}
	for name, a := range accounts {
		if a.gid == g.gid {
			out = append(out, inventory.Member{Name: name, UID: a.uid, Membership: "primary"})
		}
	}
	for _, name := range g.names {
		a, exists := accounts[name]
		if !exists {
			complete = false
			continue
		}
		out = append(out, inventory.Member{Name: name, UID: a.uid, Membership: "supplementary"})
	}
	return out, complete
}
