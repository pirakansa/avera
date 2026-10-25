package report

import (
	"html"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/guidance"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func storageFile(t *testing.T) *reportfile.File {
	t.Helper()
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	snapshot.ContractRevision = i.ContractRevision
	snapshot.Coverage = append(snapshot.Coverage, i.Coverage{Collector: i.Storage, CollectorVersion: "fictional", Scope: "fictional external metadata", Status: i.Complete, AttemptedSources: []string{"/sys/class/block", "/proc/self/mountinfo"}, DiagnosticCodes: []string{}, Limitations: []string{}})
	snapshot.Evidence = append(snapshot.Evidence, i.Evidence{EvidenceID: "fictional-storage", SourceKind: "local_file", Location: "/sys/class/block/fictional-usb", CollectedAt: snapshot.CollectedAt, Method: "Fictional metadata only", Values: i.FileValues{Excerpt: "Fictional connected, unmounted USB"}})
	finding, err := i.NormalizeFinding(snapshot.TargetID, i.Finding{Collector: i.Storage, Kind: i.ExternalStorage, Identity: i.StorageIdentity{Key: "wwid:fictional-storage"}, Facts: i.StorageFacts{Connection: "usb", Vendor: "Fictional", Model: `Storage <script>window.injected=1</script>`, Serial: "DEMO-STORAGE", SizeBytes: 32 << 30, ReadOnly: false, ObservationComplete: true, Volumes: []i.StorageVolume{{Partition: 0, Mounts: []i.StorageMount{}}}}, EvidenceIDs: []string{"fictional-storage"}})
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Findings = append(snapshot.Findings, finding)
	file, err := reportfile.New(snapshot, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	return file
}
func TestStorageReasonResolvesWithoutMountOrAdvancedPermissionFields(t *testing.T) {
	file := storageFile(t)
	var storage i.Finding
	for _, finding := range file.Current.Findings {
		if finding.Kind == i.ExternalStorage {
			storage = finding
		}
	}
	updated, err := file.Apply(reportfile.Edit{FindingID: storage.FindingID, Purpose: "○○アプリの保存先。架空の運用チームが管理", Rationale: "業務上必要な接続として確認", Actor: "fictional-reviewer", Decision: i.Accepted}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	review, err := updated.Review()
	if err != nil {
		t.Fatal(err)
	}
	accepted := false
	for _, item := range review.Review.Items {
		if item.FindingID == storage.FindingID {
			accepted = item.CurrentDecision == i.Accepted
		}
	}
	if !accepted {
		t.Fatal("reason did not resolve unmounted storage")
	}
	for _, entry := range updated.Assessments.Entries {
		if entry.Selector.Kind == i.ExternalStorage {
			if fields := guidance.Missing(entry); len(fields) != 0 {
				t.Fatal("advanced explanations required", fields)
			}
		}
	}
	for _, lang := range []string{"ja", "en"} {
		for _, format := range []string{"html", "markdown"} {
			output, err := RenderFile(updated, Options{Format: format, Language: lang})
			if err != nil {
				t.Fatal(err)
			}
			visible := html.UnescapeString(output)
			if strings.Contains(output, "<script>") || !strings.Contains(visible, "DEMO-STORAGE") || !strings.Contains(visible, "○○アプリの保存先") {
				t.Fatalf("storage output %s/%s: active script=%t, serial=%t, purpose=%t", lang, format, strings.Contains(output, "<script>"), strings.Contains(visible, "DEMO-STORAGE"), strings.Contains(visible, "○○アプリの保存先"))
			}
		}
	}
	data, err := updated.Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := reportfile.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	previous := decoded.Current
	next := *previous
	next.SnapshotID = "storage-repeat"
	next.Findings = append([]i.Finding{}, previous.Findings...)
	repeated, err := reportfile.New(&next, previous, decoded.Assessments, decoded.History, "")
	if err != nil {
		t.Fatal(err)
	}
	repeatReview, err := repeated.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range repeatReview.Review.Items {
		if item.FindingID == storage.FindingID && item.CurrentDecision != i.Accepted {
			t.Fatal("unchanged storage did not reuse judgment")
		}
	}
	for n, finding := range next.Findings {
		if finding.Kind != i.ExternalStorage {
			continue
		}
		facts := finding.Facts.(i.StorageFacts)
		facts.ReadOnly = true
		finding.Facts = facts
		next.Findings[n], err = i.NormalizeFinding(next.TargetID, finding)
		if err != nil {
			t.Fatal(err)
		}
	}
	changed, err := reportfile.New(&next, previous, decoded.Assessments, decoded.History, "")
	if err != nil {
		t.Fatal(err)
	}
	changedReview, err := changed.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range changedReview.Review.Items {
		if item.FindingID == storage.FindingID && item.Applicability != i.NeedsReview {
			t.Fatal("changed storage reused stale judgment")
		}
	}
}

func TestStorageCollectionStatusDoesNotInventAbsence(t *testing.T) {
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	if !strings.Contains(StorageStatus(snapshot, "ja"), "未収集") {
		t.Fatal("legacy input treated as empty")
	}
	snapshot.Coverage = append(snapshot.Coverage, i.Coverage{Collector: i.Storage, Status: i.Complete})
	if !strings.Contains(StorageStatus(snapshot, "ja"), "0台") {
		t.Fatal("empty success hidden")
	}
	snapshot.Coverage[len(snapshot.Coverage)-1].Status = i.Partial
	if !strings.Contains(StorageStatus(snapshot, "ja"), "不完全") {
		t.Fatal("partial input treated as absence")
	}
}
