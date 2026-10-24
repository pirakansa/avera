package sudo

import "testing"

func TestSupportedSubset(t *testing.T) {
	for _, line := range []string{
		"alice ALL=(root) ALL", "%operators ALL = (root,alice:root) NOPASSWD: /usr/bin/id, PASSWD: /bin/true",
	} {
		f, ok := parseRule(line)
		if !ok || len(f.Commands) == 0 {
			t.Fatalf("valid subset rejected: %s", line)
		}
	}
	for _, line := range []string{
		"alice ALL=(root) /usr/bin/sudoedit", "alice ALL=(root) sudoedit", "ALL ALL=(root) ALL", "ADMINS ALL=(root) ALL", "alice ALL=ALL", "alice ALL=() ALL", "alice ALL=(root:) ALL", "alice ALL=(root) /bin/id -u", "alice ALL=(root) /bin/id,", "alice ALL=(root) /bin/../id", "alice ALL=(root) SETENV: ALL", "alice ALL=(root) /bin/id \\",
	} {
		if _, ok := parseRule(line); ok {
			t.Fatalf("unsupported construct accepted: %s", line)
		}
	}
	for _, line := range []string{"@include local", "#include local", "@includedir /etc/policy", "#includedir /etc/policy"} {
		if name, _, ok := include(line); !ok || name == "" {
			t.Fatalf("include rejected: %s", line)
		}
	}
}
