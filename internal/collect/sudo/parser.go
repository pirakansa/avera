package sudo

import (
	"path"
	"regexp"
	"strings"

	"github.com/pirakansa/avera/internal/inventory"
)

var localName = regexp.MustCompile(`^[a-z_][a-z0-9_-]*[$]?$`)
var rulePattern = regexp.MustCompile(`^(%?[a-z_][a-z0-9_-]*[$]?)\s+ALL\s*=\s*\(([^()]*)\)\s*(.+)$`)
var commandPath = regexp.MustCompile(`^/[a-zA-Z0-9_./+-]+$`)

// parseRule recognizes a configured rule only, never an effective permission.
func parseRule(line string) (inventory.SudoFacts, bool) {
	f := inventory.SudoFacts{RunAsUsers: []string{}, RunAsGroups: []string{}, Commands: []inventory.Command{}, LocalMembers: []inventory.Member{}, PolicyContext: []string{}}
	m := rulePattern.FindStringSubmatch(line)
	if m == nil {
		return f, false
	}
	f.Principal = inventory.Principal{Type: "user", Name: m[1]}
	if strings.HasPrefix(m[1], "%") {
		f.Principal.Type = "group"
		f.Principal.Name = m[1][1:]
	}
	f.Host = "ALL"
	runas := strings.Split(m[2], ":")
	if len(runas) > 2 {
		return f, false
	}
	for i, list := range runas {
		for _, name := range strings.Split(list, ",") {
			name = strings.TrimSpace(name)
			if name != "ALL" && !localName.MatchString(name) {
				return f, false
			}
			if i == 0 {
				f.RunAsUsers = append(f.RunAsUsers, name)
			} else {
				f.RunAsGroups = append(f.RunAsGroups, name)
			}
		}
	}
	tag := ""
	for _, spec := range strings.Split(m[3], ",") {
		spec = strings.TrimSpace(spec)
		for _, candidate := range []string{"NOPASSWD", "PASSWD"} {
			if strings.HasPrefix(spec, candidate+":") {
				tag = candidate
				spec = strings.TrimSpace(strings.TrimPrefix(spec, candidate+":"))
				break
			}
		}
		// A bare path permits arbitrary arguments in sudoers. Preserve that explicitly.
		// A trailing slash denotes a directory command specification and is unsupported.
		if spec != "ALL" && (!commandPath.MatchString(spec) || strings.HasSuffix(spec, "/") || path.Clean(spec) != spec || path.Base(spec) == "sudoedit") {
			return f, false
		}
		tags := []string{}
		if tag != "" {
			tags = append(tags, tag)
		}
		f.Commands = append(f.Commands, inventory.Command{Path: spec, Arguments: "", Tags: tags})
	}
	return f, true
}

func include(line string) (name string, directory, recognized bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	switch fields[0] {
	case "@include", "#include":
		recognized = true
	case "@includedir", "#includedir":
		recognized = true
		directory = true
	default:
		return
	}
	// Quoted/escaped paths, hostname expansion and inline comments remain unsupported.
	if len(fields) != 2 || strings.ContainsAny(fields[1], `\"'%#`) {
		return "", directory, true
	}
	return fields[1], directory, true
}
