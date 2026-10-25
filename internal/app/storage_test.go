package app

import (
	"context"
	"path/filepath"
	"testing"

	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func TestStorageCollectionAcceptsLegacyContext(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "legacy-report.json")
	if err := (Service{}).ImportReport(context.Background(), Options{Snapshot: fixture(t, "snapshot-first.json"), Output: input}); err != nil {
		t.Fatal(err)
	}
	service := Service{EffectiveUID: func() int { return 0 }, Collectors: []Registration{{Name: i.Storage, Collector: collectorFunc(func(ctx context.Context, target string) (i.CollectorResult, error) {
		finding, err := i.NormalizeFinding(target, i.Finding{Collector: i.Storage, Kind: i.ExternalStorage, Identity: i.StorageIdentity{Key: "wwid:fictional-transfer"}, Facts: i.StorageFacts{Connection: "usb", Vendor: "Fictional", Model: "Transfer USB", Serial: "DEMO", SizeBytes: 32 << 30, ObservationComplete: true, Volumes: []i.StorageVolume{{Partition: 0, Mounts: []i.StorageMount{}}}}, EvidenceIDs: []string{"storage-demo"}})
		if err != nil {
			return i.CollectorResult{}, err
		}
		return i.CollectorResult{Coverage: i.Coverage{Collector: i.Storage, CollectorVersion: "fictional", Scope: "fictional-storage", Status: i.Complete, AttemptedSources: []string{"/sys/class/block"}, DiagnosticCodes: []string{}, Limitations: []string{}}, Findings: []i.Finding{finding}, Evidence: []i.Evidence{{EvidenceID: "storage-demo", SourceKind: "local_file", Location: "/sys/class/block/fictional", CollectedAt: "2026-10-25T00:00:00Z", Method: "Fictional metadata only", Values: i.FileValues{Excerpt: "No mounted filesystem"}}}}, nil
	})}}}
	output := filepath.Join(directory, "with-storage.json")
	code, err := service.CollectReport(context.Background(), Options{Input: input, Output: output})
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	file, err := reportfile.Read(output)
	if err != nil {
		t.Fatal(err)
	}
	if file.Current.ContractRevision != i.ContractRevision || file.Previous == nil || file.Previous.ContractRevision != i.LegacyContractRevision {
		t.Fatal("legacy context not preserved")
	}
	if len(file.Current.Findings) != 1 || file.Current.Findings[0].Kind != i.ExternalStorage {
		t.Fatal("storage collector not registered")
	}
}
