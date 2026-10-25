package sudo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/inventory"
)

func write(t *testing.T, root, name, data string) {
	t.Helper()
	name = filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"sudoers", "sudoers.local", "passwd", "group"} {
		data, err := os.ReadFile("../../../test/fixtures/prototype/sudo/" + name)
		if err != nil {
			t.Fatal(err)
		}
		write(t, root, "etc/"+name, string(data))
	}
	write(t, root, "etc/sudoers.d/10_second", "bob ALL=(root) /usr/bin/true\n")
	write(t, root, "etc/sudoers.d/01_first", "alice ALL=(root) /usr/bin/id\n")
	write(t, root, "etc/sudoers.d/ignored.conf", "invalid ignored\n")
	write(t, root, "etc/sudoers.d/backup~", "invalid ignored\n")
	return root
}
func collect(t *testing.T, root string, limits Limits) inventory.CollectorResult {
	t.Helper()
	c := Collector{Source: LocalSource{Root: root}, Limits: limits, Now: func() time.Time { return time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC) }}
	r, err := c.Collect(context.Background(), "fictional-host")
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func firstRule(t *testing.T, r inventory.CollectorResult) inventory.Finding {
	t.Helper()
	for _, f := range r.Findings {
		if f.Kind == inventory.Rule {
			return f
		}
	}
	t.Fatal("missing rule")
	return inventory.Finding{}
}

func TestConfiguredRulesAndContext(t *testing.T) {
	root := fixture(t)
	r := collect(t, root, Limits{})
	if r.Coverage.Status != inventory.Complete || len(r.Findings) != 4 {
		t.Fatalf("unexpected result: %+v", r)
	}
	if !reflect.DeepEqual(r, collect(t, root, Limits{})) {
		t.Fatal("nondeterministic observations")
	}
	var group inventory.SudoFacts
	for _, f := range r.Findings {
		facts := f.Facts.(inventory.SudoFacts)
		if !facts.InterpretationComplete || !facts.LocalContextComplete {
			t.Fatal("incomplete rule")
		}
		if facts.Principal.Type == "group" {
			group = facts
		}
	}
	want := []inventory.Member{{Name: "alice", UID: 1001, Membership: "primary"}, {Name: "bob", UID: 1002, Membership: "supplementary"}}
	if !reflect.DeepEqual(group.LocalMembers, want) {
		t.Fatalf("members: %+v", group.LocalMembers)
	}
	cmds := firstRule(t, r).Facts.(inventory.SudoFacts).Commands
	if !reflect.DeepEqual(cmds[0].Tags, []string{"NOPASSWD"}) || !reflect.DeepEqual(cmds[1].Tags, cmds[0].Tags) || !reflect.DeepEqual(cmds[2].Tags, []string{"PASSWD"}) {
		t.Fatalf("tag inheritance: %+v", cmds)
	}
	paths := r.Coverage.AttemptedSources
	if slices.Index(paths, "/etc/sudoers.d/01_first") >= slices.Index(paths, "/etc/sudoers.d/10_second") {
		t.Fatal("incorrect lexical order")
	}
	for _, e := range r.Evidence {
		if e.SourceKind != "local_file" || e.CollectedAt != "2026-10-24T00:00:00Z" {
			t.Fatal("invalid evidence")
		}
	}
}

func TestUnsupportedContextLimitsEveryRule(t *testing.T) {
	constructs := []string{"alice ALL=(root) /usr/bin/sudoedit", "Defaults authenticate", "User_Alias ADMINS = alice", "alice ALL=(root) ALL, !/bin/false", "alice host=(root) ALL", "alice ALL=(root) /usr/bin/id -u", "alice ALL=(root) NOEXEC: /usr/bin/id", "@include \"space path\"", "@include /etc/%h", "@unknown /etc/policy", "alice ALL=(root) /usr/bin/id # comment", "alice ALL=(root) /usr/bin/", "alice ALL=(root) /usr/bin/*"}
	for _, raw := range constructs {
		t.Run(raw, func(t *testing.T) {
			root := fixture(t)
			write(t, root, "etc/sudoers.local", raw+"\n")
			r := collect(t, root, Limits{})
			if r.Coverage.Status != inventory.Partial {
				t.Fatal("expected partial")
			}
			found := false
			for _, f := range r.Findings {
				if f.Kind == inventory.Unparsed {
					found = true
					if len(f.EvidenceIDs) == 0 {
						t.Fatal("missing evidence")
					}
				}
				if facts, ok := f.Facts.(inventory.SudoFacts); ok && facts.InterpretationComplete {
					t.Fatal("unsupported context silently ignored")
				}
			}
			if !found {
				t.Fatal("missing unsupported review item")
			}
		})
	}
}

func TestContinuationCannotCreateIndependentRule(t *testing.T) {
	root := fixture(t)
	write(t, root, "etc/sudoers", "User_Alias ADMINS = \\\nalice ALL=(root) ALL\n")
	r := collect(t, root, Limits{})
	if len(r.Findings) != 2 {
		t.Fatalf("findings: %+v", r.Findings)
	}
	for _, f := range r.Findings {
		if f.Kind != inventory.Unparsed {
			t.Fatal("continuation interpreted as rule")
		}
	}
}

func TestSourceFailures(t *testing.T) {
	for _, tc := range []struct{ name, policy, code string }{
		{"missing", "@include missing\n", "source_unreadable"},
		{"cycle", "@include sudoers\n", "include_cycle"},
		{"repeated", "@include sudoers.local\n@include sudoers.local\n", "repeated_source"},
		{"directory", "@includedir absent\n", "directory_unreadable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t)
			write(t, root, "etc/sudoers", "alice ALL=(root) ALL\n"+tc.policy)
			r := collect(t, root, Limits{})
			if r.Coverage.Status != inventory.Partial || !slices.Contains(r.Coverage.DiagnosticCodes, tc.code) {
				t.Fatalf("coverage: %+v", r.Coverage)
			}
			if firstRule(t, r).Facts.(inventory.SudoFacts).InterpretationComplete {
				t.Fatal("incomplete source ignored")
			}
		})
	}
	root := fixture(t)
	if err := os.Remove(filepath.Join(root, "etc/sudoers")); err != nil {
		t.Fatal(err)
	}
	if collect(t, root, Limits{}).Coverage.Status != inventory.Failed {
		t.Fatal("missing entry must fail")
	}
}

func TestLimitsAndCancellation(t *testing.T) {
	for _, l := range []Limits{{Bytes: 20}, {Files: 1}, {Entries: 1}, {Lines: 1}, {Depth: 1}, {PathBytes: 10}, {ContextBytes: 1}} {
		root := fixture(t)
		write(t, root, "etc/sudoers.local", "@include nested\n")
		write(t, root, "etc/nested", "alice ALL=(root) ALL\n")
		r := collect(t, root, l)
		if r.Coverage.Status == inventory.Complete || len(r.Coverage.DiagnosticCodes) == 0 {
			t.Fatalf("limit ignored: %+v", l)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Collector{Source: LocalSource{Root: fixture(t)}}).Collect(ctx, "host"); err != context.Canceled {
		t.Fatalf("cancellation: %v", err)
	}
	for _, c := range []Collector{{}, {Source: LocalSource{Root: fixture(t)}, Limits: Limits{Lines: -1}}, {Source: LocalSource{Root: fixture(t)}, Entry: "relative"}} {
		if _, err := c.Collect(context.Background(), "host"); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
}

func TestContextChangesFingerprint(t *testing.T) {
	for _, change := range []string{"include", "membership", "order", "missing", "move"} {
		t.Run(change, func(t *testing.T) {
			root := fixture(t)
			before := firstRule(t, collect(t, root, Limits{}))
			switch change {
			case "include":
				write(t, root, "etc/sudoers.local", "%operators ALL=(root) /usr/bin/true\n")
			case "membership":
				write(t, root, "etc/group", "root:x:0:\noperators:x:100:\nsudo:x:200:alice\n")
			case "order":
				write(t, root, "etc/sudoers", "# ordered fixture\nalice ALL=(root) NOPASSWD: /usr/bin/id, /usr/bin/whoami, PASSWD: /usr/bin/true\n@includedir /etc/sudoers.d\n@include sudoers.local\n")
			case "missing":
				if err := os.Remove(filepath.Join(root, "etc/sudoers.local")); err != nil {
					t.Fatal(err)
				}
			case "move":
				write(t, root, "etc/sudoers", "\n\nalice ALL=(root) NOPASSWD: /usr/bin/id, /usr/bin/whoami, PASSWD: /usr/bin/true\n@include sudoers.local\n@includedir /etc/sudoers.d\n")
			}
			after := firstRule(t, collect(t, root, Limits{}))
			if before.FactFingerprint == after.FactFingerprint {
				t.Fatal("context change did not invalidate fingerprint")
			}
			if change != "move" && before.FindingID != after.FindingID {
				t.Fatal("stable anchor changed")
			}
			if change == "move" && before.FindingID == after.FindingID {
				t.Fatal("movement retained identity")
			}
		})
	}
}

func TestIdentityBoundaries(t *testing.T) {
	for _, name := range []string{"passwd", "group"} {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			write(t, root, "etc/"+name, "malformed fictional identity\n")
			r := collect(t, root, Limits{})
			if r.Coverage.Status != inventory.Partial || firstRule(t, r).Facts.(inventory.SudoFacts).LocalContextComplete {
				t.Fatal("malformed identity treated as complete")
			}
		})
	}
	root := fixture(t)
	write(t, root, "etc/sudoers", "# no rules; sudo membership is not proof\n")
	r := collect(t, root, Limits{})
	if len(r.Findings) != 0 || r.Coverage.Status != inventory.Complete {
		t.Fatalf("membership invented grant: %+v", r)
	}
	write(t, root, "etc/passwd", "root:fictional-auth-marker:0:0:Fictional:/root:/bin/false\n")
	r = collect(t, root, Limits{})
	for _, e := range r.Evidence {
		if strings.Contains(e.Values.(inventory.FileValues).Excerpt, "fictional-auth-marker") {
			t.Fatal("authentication field retained")
		}
	}
}

func TestArtifactContractAndRawContext(t *testing.T) {
	for _, partial := range []bool{false, true} {
		root := fixture(t)
		raw := "  Defaults authenticate  "
		if partial {
			write(t, root, "etc/sudoers.local", raw+"\n")
		}
		r := collect(t, root, Limits{})
		ts := "2026-10-24T00:00:00Z"
		snapshot := &inventory.Snapshot{
			Envelope:   inventory.Envelope{SchemaVersion: inventory.SchemaVersion, ContractRevision: inventory.LegacyContractRevision, Kind: "snapshot", TargetID: "fictional-host"},
			SnapshotID: "fictional-sudo", CollectedAt: ts, FinishedAt: ts, ToolVersion: Version,
			Environment: inventory.Environment{OSID: "fictional", OSVersion: "fixture", Architecture: "amd64", KernelRelease: "fixture", ExecutionScope: "fictional-policy-root"},
			Coverage:    []inventory.Coverage{r.Coverage}, Findings: r.Findings, Evidence: r.Evidence,
		}
		encoded, err := inventory.Encode(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := inventory.Decode(encoded); err != nil {
			t.Fatal(err)
		}
		if partial {
			found := false
			for _, f := range r.Findings {
				if f.Kind == inventory.Unparsed && f.Facts.(inventory.UnparsedFacts).Raw != raw {
					t.Fatal("unsupported raw text changed")
				}
				if facts, ok := f.Facts.(inventory.SudoFacts); ok && slices.Contains(facts.PolicyContext, raw) {
					found = true
				}
			}
			if !found {
				t.Fatal("raw unsupported context missing from rule")
			}
		} else {
			members := firstRule(t, r).Facts.(inventory.SudoFacts).LocalMembers
			if !reflect.DeepEqual(members, []inventory.Member{{Name: "alice", UID: 1001, Membership: "primary"}}) {
				t.Fatalf("explicit user context: %+v", members)
			}
			want := []string{"/etc/sudoers", "/etc/sudoers.local", "/etc/sudoers.d/01_first", "/etc/sudoers.d/10_second"}
			for i, f := range r.Findings {
				if f.Identity.(inventory.SourceIdentity).SourcePath != want[i] {
					t.Fatalf("traversal order: %+v", r.Findings)
				}
			}
		}
	}
}

func TestMissingAndDuplicateIdentity(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"group", "root:x:0:\noperators:x:100:missing\n"},
		{"passwd", "root:x:0:0:Fictional:/root:/bin/false\n"},
		{"passwd", "alice:x:1001:100:Fictional:/tmp:/bin/false\nalice:x:1002:100:Duplicate:/tmp:/bin/false\n"},
	} {
		root := fixture(t)
		write(t, root, "etc/"+tc.name, tc.data)
		r := collect(t, root, Limits{})
		if r.Coverage.Status != inventory.Partial {
			t.Fatal("identity gaps ignored")
		}
		for _, f := range r.Findings {
			if facts, ok := f.Facts.(inventory.SudoFacts); ok && facts.InterpretationComplete {
				t.Fatal("identity gap ignored by interpretation")
			}
		}
	}
}

// The two common-contract artifacts are the T05 presentation handoff. Updating
// them is explicit; default tests only read the fictional repository fixtures.
func TestPresentationSnapshots(t *testing.T) {
	for _, name := range []string{"understood", "unsupported"} {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			if name == "unsupported" {
				data, err := os.ReadFile("../../../test/fixtures/prototype/sudo/unsupported")
				if err != nil {
					t.Fatal(err)
				}
				write(t, root, "etc/sudoers.local", string(data))
			}
			r := collect(t, root, Limits{})
			ts := "2026-10-24T00:00:00Z"
			snapshot := &inventory.Snapshot{
				Envelope:   inventory.Envelope{SchemaVersion: inventory.SchemaVersion, ContractRevision: inventory.LegacyContractRevision, Kind: "snapshot", TargetID: "fictional-host"},
				SnapshotID: "fictional-sudo-" + name, CollectedAt: ts, FinishedAt: ts, ToolVersion: Version,
				Environment: inventory.Environment{OSID: "fictional", OSVersion: "fixture", Architecture: "amd64", KernelRelease: "fixture", ExecutionScope: "fictional-policy-root"},
				Coverage:    []inventory.Coverage{r.Coverage}, Findings: r.Findings, Evidence: r.Evidence,
			}
			data, err := inventory.Encode(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			path := "../../../test/fixtures/prototype/sudo/snapshot-" + name + ".json"
			if os.Getenv("AVERA_UPDATE_SUDO_FIXTURES") == "1" {
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			expected, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(expected) != string(data) {
				t.Fatalf("snapshot %s differs; regenerate explicitly after reviewing changes", name)
			}
		})
	}
}
