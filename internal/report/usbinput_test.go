package report

import (
	"html"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func inputFile(t *testing.T) *reportfile.File {
	t.Helper()
	data, err := os.ReadFile("../../test/fixtures/prototype/reports/usb-input-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := reportfile.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func TestInputPrivilegeLinksAndJudgmentReuse(t *testing.T) {
	f := inputFile(t)
	var usb, root string
	for _, find := range f.Current.Findings {
		if facts, ok := find.Facts.(i.AccountFacts); ok && facts.UID == 0 {
			root = find.FindingID
		}
		if facts, ok := find.Facts.(i.USBFacts); ok && facts.Input != nil && len(facts.Input.Consumers) > 0 {
			usb = find.FindingID
		}
	}
	if usb == "" || root == "" {
		t.Fatal("fixture missing input consumer or root")
	}
	if assessment.AccountReferences(f.Current)[root] {
		t.Fatal("root input consumer excluded from account inventory")
	}
	links := PrivilegeConnections(f.Current)
	if !strings.Contains(strings.Join(links[usb], ","), root) {
		t.Fatal("input consumer account link missing")
	}
	r, err := f.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range r.Review.Items {
		if item.FindingID == usb && item.CurrentDecision == i.Accepted {
			t.Fatal("old device-only judgment became input approval")
		}
	}
	updated, err := f.Apply(reportfile.Edit{FindingID: usb, Purpose: "既存アプリの操作用", Safeguards: "上位の物理運用手順書で管理", Rationale: "入力口を開くプロセスと実行権限を確認", Actor: "fictional-reviewer", Decision: i.Accepted}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"ja", "en"} {
		for _, format := range []string{"html", "markdown"} {
			output, err := RenderFile(updated, Options{Language: lang, Format: format})
			if err != nil {
				t.Fatal(err)
			}
			visible := html.UnescapeString(output)
			for _, text := range []string{"Fictional-Xorg", "/dev/input/event5", "kbd", "/usr/bin/fictional-Xorg"} {
				if !strings.Contains(visible, text) {
					t.Fatal("input context missing", text, lang, format)
				}
			}
		}
	}
	updated.Current.SnapshotID = "fictional-next-input"
	// Incomplete input scope prevents reusing an otherwise matching whole-device judgment.
	for n, e := range updated.Current.Evidence {
		if v, ok := e.Values.(i.USBValues); ok {
			v.Input.Status = i.Partial
			v.Input.DiagnosticCodes = []string{"input_descriptors_unreadable"}
			for k := range v.Nodes {
				if v.Nodes[k].Facts.Input != nil {
					v.Nodes[k].Facts.Input.Status = i.Partial
					v.Nodes[k].Facts.Input.DiagnosticCodes = v.Input.DiagnosticCodes
				}
			}
			e.Values = v
			updated.Current.Evidence[n] = e
		}
	}
	for n, find := range updated.Current.Findings {
		if facts, ok := find.Facts.(i.USBFacts); ok {
			facts.Input.Status = i.Partial
			facts.Input.DiagnosticCodes = []string{"input_descriptors_unreadable"}
			find.Facts = facts
			normalized, err := i.NormalizeFinding(updated.Current.TargetID, find)
			if err != nil {
				t.Fatal(err)
			}
			updated.Current.Findings[n] = normalized
		}
	}
	r, err = updated.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range r.Review.Items {
		if item.FindingID == usb && item.CurrentDecision == i.Accepted {
			t.Fatal("incomplete input accepted")
		}
	}
}
func TestLegacyUSBInputIsExplicitlyUncollected(t *testing.T) {
	f := usbFile(t)
	output, err := RenderFile(f, Options{Format: "html", Language: "ja"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "入力経路は未収集") {
		t.Fatal("old file implies no input readers")
	}
}
