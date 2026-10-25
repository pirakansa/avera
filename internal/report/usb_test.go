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

func usbFile(t *testing.T) *reportfile.File {
	t.Helper()
	data, err := os.ReadFile("../../test/fixtures/prototype/reports/usb-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := reportfile.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func TestUSBCompositeReviewAndPhysicalMap(t *testing.T) {
	f := usbFile(t)
	owners := i.USBStorageOwners(f.Current)
	if len(owners) != 1 {
		t.Fatal(owners)
	}
	r, err := f.Review()
	if err != nil {
		t.Fatal(err)
	}
	summary, _ := assessment.ReviewSummary(f.Current, r.Review)
	var storageID, usbID string
	for s, u := range owners {
		storageID, usbID = s, u
	}
	if _, err = f.Apply(reportfile.Edit{FindingID: storageID, Actor: "reviewer", Rationale: "test", Decision: i.Accepted}, time.Now()); err == nil {
		t.Fatal("duplicate storage judgment allowed")
	}
	updated, err := f.Apply(reportfile.Edit{FindingID: usbID, Purpose: "○○アプリの入力と保存先", Safeguards: "上位提供の運用マニュアルで物理管理を定義済み", Actor: "fictional-reviewer", Rationale: "用途と保護策を担当者が確認", Decision: i.Accepted}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rr, err := updated.Review()
	if err != nil {
		t.Fatal(err)
	}
	after, _ := assessment.ReviewSummary(updated.Current, rr.Review)
	if after.Current != summary.Current || after.Accepted != summary.Accepted+1 {
		t.Fatal(summary, after)
	}
	for _, lang := range []string{"ja", "en"} {
		for _, format := range []string{"html", "markdown"} {
			output, err := RenderFile(updated, Options{Format: format, Language: lang})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(html.UnescapeString(output), "上位提供の運用マニュアル") {
				t.Fatal("human safeguards missing")
			}
			if format == "html" {
				if !strings.Contains(output, "physical-tree") || !strings.Contains(output, "Fictional desk hub") {
					t.Fatal("physical map missing")
				}
			}
		}
	}
	// Grouped storage changes invalidate the whole-device judgment.
	updated.Current.SnapshotID = "fictional-usb-next"
	for n, find := range updated.Current.Findings {
		if find.FindingID == storageID {
			facts := find.Facts.(i.StorageFacts)
			facts.ReadOnly = !facts.ReadOnly
			find.Facts = facts
			v, err := i.NormalizeFinding(updated.Current.TargetID, find)
			if err != nil {
				t.Fatal(err)
			}
			updated.Current.Findings[n] = v
		}
	}
	if _, err := updated.Encode(); err == nil {
		t.Fatal("stale grouped basis accepted")
	}
	if err := i.EnrichUSBStorage(updated.Current); err != nil {
		t.Fatal(err)
	}
	rr, err = updated.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range rr.Review.Items {
		if item.FindingID == usbID && item.CurrentDecision == i.Accepted {
			t.Fatal("changed storage inherited acceptance")
		}
	}
}
func TestUSBMetadataEscapingAndIncompleteCoverage(t *testing.T) {
	f := usbFile(t)
	f.Current.SnapshotID = "fictional-usb-changed"
	for n, e := range f.Current.Evidence {
		if v, ok := e.Values.(i.USBValues); ok {
			for k := range v.Nodes {
				v.Nodes[k].Facts.Product = `<script>window.injected=1</script>`
			}
			e.Values = v
			f.Current.Evidence[n] = e
		}
	}
	for n, find := range f.Current.Findings {
		if facts, ok := find.Facts.(i.USBFacts); ok {
			facts.Product = `<script>window.injected=1</script>`
			find.Facts = facts
			v, err := i.NormalizeFinding(f.Current.TargetID, find)
			if err != nil {
				t.Fatal(err)
			}
			f.Current.Findings[n] = v
		}
	}
	if err := i.EnrichUSBStorage(f.Current); err != nil {
		t.Fatal(err)
	}
	output, err := RenderFile(f, Options{Format: "html", Language: "ja"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "<script>") || !strings.Contains(output, "&lt;script&gt;") {
		t.Fatal("USB metadata not escaped")
	}
	for n, c := range f.Current.Coverage {
		if c.Collector == i.USB {
			c.Status = i.Partial
			f.Current.Coverage[n] = c
		}
	}
	r, err := f.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range r.Review.Items {
		for _, find := range f.Current.Findings {
			if find.Kind == i.USBDevice && find.FindingID == item.FindingID && item.CurrentDecision == i.Accepted {
				t.Fatal("incomplete USB reused acceptance")
			}
		}
	}
}
