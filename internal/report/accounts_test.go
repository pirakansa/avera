package report

import (
	"context"
	"fmt"
	"github.com/pirakansa/avera/internal/reportfile"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/collect/accounts"
	"github.com/pirakansa/avera/internal/collect/sudo"
	i "github.com/pirakansa/avera/internal/inventory"
)

func accountSnapshot(t *testing.T) *i.Snapshot {
	t.Helper()
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	snapshot.ContractRevision = i.ContractRevision
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "etc"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"passwd": "operator:x:1000:1000::/home/operator:/bin/sh\nalias:x:1000:1000::/alias:/bin/sh\n", "group": "operator:x:1000:\nmaint:x:3000:operator\nempty:x:4000:\n"} {
		if err := os.WriteFile(filepath.Join(root, "etc", name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := (accounts.Collector{Source: sudo.LocalSource{Root: root}}).Collect(context.Background(), snapshot.TargetID)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Coverage = append(snapshot.Coverage, result.Coverage)
	snapshot.Findings = append(snapshot.Findings, result.Findings...)
	snapshot.Evidence = append(snapshot.Evidence, result.Evidence...)
	uid, gid := uint32(1000), uint32(3000)
	groups := []uint32{3000}
	values := snapshot.Evidence[0].Values.(i.ProcessValues)
	values.Service = &i.ServiceObservation{ProcessName: "fictional-worker", EffectiveUID: &uid, EffectiveGID: &gid, SupplementaryGroups: &groups, UnitCandidate: "fictional.service", CgroupPath: "/fictional.service", Diagnostics: []string{}}
	snapshot.Evidence[0].Values = values
	return snapshot
}
func TestAccountLinksAndReviewBoundary(t *testing.T) {
	snapshot := accountSnapshot(t)
	connections := PrivilegeConnections(snapshot)
	endpoint := ""
	account := ""
	alias := ""
	for _, finding := range snapshot.Findings {
		if id, ok := finding.Identity.(i.AccountIdentity); ok {
			if id.Name == "operator" {
				account = finding.FindingID
			}
			if id.Name == "alias" {
				alias = finding.FindingID
			}
		}
		if id, ok := finding.Identity.(i.EndpointIdentity); ok && id.LocalPort == 80 {
			endpoint = finding.FindingID
		}
	}
	if !contains(connections[endpoint], account) || !contains(connections[endpoint], alias) {
		t.Fatal("UID aliases were guessed or omitted", connections)
	}
	for _, lang := range []string{"en", "ja"} {
		document, err := makeDocument(snapshot, nil, nil, lang)
		if err != nil {
			t.Fatal(err)
		}
		if document.Metrics[0].Count != 5 {
			t.Fatal("scope exclusions became findings", document.Metrics)
		}
		for _, format := range []string{"html", "markdown"} {
			output, err := RenderWithOptions(snapshot, nil, nil, Options{Format: format, Language: lang})
			if err != nil {
				t.Fatal(err)
			}
			visible := html.UnescapeString(output)
			if !strings.Contains(visible, AccountBoundary(lang)) || !strings.Contains(visible, "operator") || !strings.Contains(visible, "maint") {
				t.Fatal("missing account inventory or scope boundary")
			}
			if format == "html" {
				ids := map[string]bool{}
				for _, match := range regexp.MustCompile(` id="([^"]+)"`).FindAllStringSubmatch(output, -1) {
					if ids[match[1]] {
						t.Fatal("duplicate anchor", match[1])
					}
					ids[match[1]] = true
				}
				for _, match := range regexp.MustCompile(`href="#([^"]+)"`).FindAllStringSubmatch(output, -1) {
					if !ids[match[1]] {
						t.Fatal("broken relation link", match[1])
					}
				}
			}
		}
	}
	views := EndpointServices(snapshot, "ja")
	found := false
	for _, field := range views[endpoint].Fields {
		if field.Label == "実行中の実効GID" {
			found = strings.Contains(field.Value, "3000") && strings.Contains(field.Value, "maint")
		}
	}
	if !found {
		t.Fatal("runtime group label not resolved independently")
	}
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestPrivilegeNavigationIsBounded(t *testing.T) {
	snapshot := accountSnapshot(t)
	var template i.Finding
	for _, finding := range snapshot.Findings {
		if _, ok := finding.Facts.(i.AccountFacts); ok {
			template = finding
			break
		}
	}
	for n := 0; n < 100; n++ {
		item := template
		item.Identity = i.AccountIdentity{Name: fmt.Sprintf("alias-%03d", n)}
		normalized, err := i.NormalizeFinding(snapshot.TargetID, item)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.Findings = append(snapshot.Findings, normalized)
	}
	links := PrivilegeConnections(snapshot)
	for _, ids := range links {
		if len(ids) > privilegeLinkLimit {
			t.Fatal("unbounded privilege navigation")
		}
	}
	document, err := makeDocument(snapshot, nil, nil, "ja")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range document.Blocks {
		for _, f := range b.Fields {
			if f.Label == "関連項目の表示上限" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("navigation limit was silent")
	}
}

func TestSystemAccountReferencePresentationAndEditing(t *testing.T) {
	snapshot := accountSnapshot(t)
	// Add root as a configured sudo run-as target, with an existing accepted judgment.
	template := snapshot.Findings[len(snapshot.Findings)-1]
	for _, f := range snapshot.Findings {
		if _, ok := f.Facts.(i.AccountFacts); ok {
			template = f
			break
		}
	}
	template.Identity = i.AccountIdentity{Name: "root"}
	template.Facts = i.AccountFacts{UID: 0, PrimaryGID: 0, Home: "/root", Shell: "/bin/sh", Groups: []i.AccountGroup{{Name: "root", GID: 0, Membership: "primary"}}, ContextComplete: true}
	root, err := i.NormalizeFinding(snapshot.TargetID, template)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Findings = append(snapshot.Findings, root)
	file, err := reportfile.New(snapshot, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Apply(reportfile.Edit{FindingID: root.FindingID, Decision: i.Accepted, Actor: "fictional", Rationale: "not editable"}, time.Now()); err == nil {
		t.Fatal("reference account accepted an edit")
	}
	for n := range file.Assessments.Entries {
		entry := &file.Assessments.Entries[n]
		if identity, ok := entry.Selector.Identity.(i.AccountIdentity); ok && identity.Name == "root" {
			entry.Decision = i.Accepted
			entry.Actor = "fictional-root-reviewer"
			entry.Rationale = "Previously recorded context"
			entry.AssessedAt = "2026-10-25T00:00:00Z"
		}
	}
	for _, language := range []string{"ja", "en"} {
		document, err := makeDocument(snapshot, nil, file.Assessments, language)
		if err != nil {
			t.Fatal(err)
		}
		if document.Metrics[0].Count != 5 {
			t.Fatal("reference account entered counts", document.Metrics)
		}
		var rootBlock block
		for _, b := range document.Blocks {
			if b.FindingID == root.FindingID {
				rootBlock = b
			}
		}
		for _, field := range rootBlock.Fields {
			if strings.Contains(field.Label, "Missing") || strings.Contains(field.Label, "不足する説明") {
				t.Fatal("reference account prompts for an explanation")
			}
		}
		if len(rootBlock.Questions) != 0 {
			t.Fatal("reference account has judgment questions")
		}
		for _, followup := range document.Pages.FollowUp {
			if followup.ID == rootBlock.ID {
				t.Fatal("reference account requires follow-up")
			}
		}
		for _, format := range []string{"html", "markdown"} {
			output, err := RenderWithOptions(snapshot, nil, file.Assessments, Options{Format: format, Language: language})
			if err != nil {
				t.Fatal(err)
			}
			visible := html.UnescapeString(output)
			label := "System accounts (reference)"
			if language == "ja" {
				label = "システムアカウント（参考情報）"
			}
			if !strings.Contains(visible, label) || !strings.Contains(visible, "root") {
				t.Fatal("reference account lost from report")
			}
		}
	}
	// Retain root as a reference link from an explicitly named sudo run-as target.
	if len(PrivilegeConnections(snapshot)[root.FindingID]) == 0 {
		t.Fatal("sudo run-as context lost")
	}
}
