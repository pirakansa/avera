package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	i "github.com/pirakansa/avera/internal/inventory"
)

func write(t *testing.T, root, name, text string) {
	t.Helper()
	target := filepath.Join(root, strings.TrimPrefix(name, "/"))
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func link(t *testing.T, root, name, target string) {
	t.Helper()
	name = filepath.Join(root, strings.TrimPrefix(name, "/"))
	target = filepath.Join(root, strings.TrimPrefix(target, "/"))
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(filepath.Dir(name), target)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(relative, name); err != nil {
		t.Fatal(err)
	}
}
func block(t *testing.T, root, name, number, location string, partition int64, removable bool) {
	t.Helper()
	write(t, root, location+"/dev", number+"\n")
	if partition > 0 {
		write(t, root, location+"/partition", fmt.Sprint(partition))
	} else {
		value := "0"
		if removable {
			value = "1"
		}
		write(t, root, location+"/removable", value)
		write(t, root, location+"/size", "67108864\n")
		write(t, root, location+"/ro", "0\n")
		if err := os.MkdirAll(filepath.Join(root, strings.TrimPrefix(location+"/slaves", "/")), 0700); err != nil {
			t.Fatal(err)
		}
	}
	link(t, root, blockPath+"/"+name, location)
}

const rootLocation = "/sys/devices/pci0000:00/ata1/host0/target0:0:0/0:0:0:0/block/sda"
const usbLocation = "/sys/devices/pci0000:00/usb1/1-2/1-2:1.0/host2/target2:0:0/2:0:0:0/block/sdb"
const secondUSB = "/sys/devices/pci0000:00/usb1/1-3/1-3:1.0/host3/target3:0:0/3:0:0:0/block/sdc"

func fixture(t *testing.T) (string, Collector) {
	t.Helper()
	root := t.TempDir()
	block(t, root, "sda", "8:0", rootLocation, 0, false)
	block(t, root, "sda1", "8:1", rootLocation+"/sda1", 1, false)
	block(t, root, "sda2", "8:2", rootLocation+"/sda2", 2, false)
	block(t, root, "sdb", "8:16", usbLocation, 0, false)
	block(t, root, "sdb1", "8:17", usbLocation+"/sdb1", 1, false)
	block(t, root, "sdc", "8:32", secondUSB, 0, true)
	for _, entry := range []struct{ base, model, serial string }{{usbLocation, "Fictional backup SSD", "DEMO-SSD-A"}, {secondUSB, "Fictional transfer USB", "DEMO-USB-B"}} {
		write(t, root, entry.base+"/device/vendor", "Fictional")
		write(t, root, entry.base+"/device/model", entry.model)
		write(t, root, entry.base+"/device/serial", entry.serial)
		write(t, root, entry.base+"/device/wwid", "wwid-"+entry.serial)
	}
	write(t, root, mountInfoPath, "10 1 8:1 / / rw,relatime - ext4 /dev/sda1 rw\n11 10 8:2 / /os-data rw - ext4 /dev/sda2 rw\n12 10 8:17 / /srv/app\\040backup rw - ext4 /dev/sdb1 rw,password=DO-NOT-RETAIN\n")
	return root, Collector{Source: LocalSource{Root: root}, ScopeID: "fictional-run", Now: func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }}
}
func collect(t *testing.T, c Collector) i.CollectorResult {
	t.Helper()
	result, err := c.Collect(context.Background(), "fictional")
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func snapshot(r i.CollectorResult) *i.Snapshot {
	return &i.Snapshot{Envelope: i.Envelope{SchemaVersion: i.SchemaVersion, ContractRevision: i.ContractRevision, Kind: "snapshot", TargetID: "fictional"}, SnapshotID: "storage-first", CollectedAt: "2026-10-25T00:00:00Z", FinishedAt: "2026-10-25T00:00:00Z", ToolVersion: Version, Environment: i.Environment{}, Coverage: []i.Coverage{r.Coverage}, Findings: r.Findings, Evidence: r.Evidence}
}
func TestMountedAndUnmountedStorageWithoutRootDisk(t *testing.T) {
	_, c := fixture(t)
	result := collect(t, c)
	if result.Coverage.Status != i.Complete || len(result.Findings) != 2 {
		t.Fatal(result.Coverage, len(result.Findings))
	}
	mounted, unmounted := false, false
	for _, finding := range result.Findings {
		facts := finding.Facts.(i.StorageFacts)
		if facts.Serial == "DEMO-SSD-A" {
			for _, volume := range facts.Volumes {
				for _, mount := range volume.Mounts {
					if mount.Path == "/srv/app backup" {
						mounted = true
					}
				}
			}
		}
		if facts.Serial == "DEMO-USB-B" {
			unmounted = true
			for _, volume := range facts.Volumes {
				if len(volume.Mounts) > 0 {
					t.Fatal("invented mount")
				}
			}
		}
		if facts.SizeBytes != 32<<30 {
			t.Fatal("sector size not decoded", facts.SizeBytes)
		}
	}
	if !mounted || !unmounted {
		t.Fatal("missing mounted or unmounted storage")
	}
	data, err := i.Encode(snapshot(result))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "DO-NOT-RETAIN") {
		t.Fatal("raw mount options persisted")
	}
	decoded, err := i.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := i.Encode(decoded)
	if err != nil || string(data) != string(again) {
		t.Fatal("storage roundtrip", err)
	}
	old := snapshot(result)
	old.ContractRevision = i.AccountsContractRevision
	if _, err = i.Encode(old); err == nil {
		t.Fatal("storage accepted under old contract")
	}
}
func TestRootBackingThroughStackAndUSB(t *testing.T) {
	root, c := fixture(t)
	// A root filesystem on a mapping excludes every backing disk, including USB.
	dm := "/sys/devices/virtual/block/dm-0"
	block(t, root, "dm-0", "253:0", dm, 0, false)
	link(t, root, dm+"/slaves/sda1", rootLocation+"/sda1")
	link(t, root, dm+"/slaves/sdb1", usbLocation+"/sdb1")
	write(t, root, mountInfoPath, "10 1 253:0 / / rw - ext4 /dev/mapper/fictional-root rw\n")
	result := collect(t, c)
	if result.Coverage.Status != i.Complete || len(result.Findings) != 1 || result.Findings[0].Facts.(i.StorageFacts).Serial != "DEMO-USB-B" {
		t.Fatal(result.Coverage, result.Findings)
	}
}
func TestMountedEncryptedExternalPartition(t *testing.T) {
	root, c := fixture(t)
	dm := "/sys/devices/virtual/block/dm-1"
	block(t, root, "dm-1", "253:1", dm, 0, false)
	link(t, root, dm+"/slaves/sdb1", usbLocation+"/sdb1")
	write(t, root, mountInfoPath, "10 1 8:1 / / rw - ext4 /dev/sda1 rw\n12 10 253:1 / /srv/encrypted ro - ext4 /dev/mapper/fictional-backup ro\n")
	result := collect(t, c)
	found := false
	for _, finding := range result.Findings {
		for _, volume := range finding.Facts.(i.StorageFacts).Volumes {
			for _, mount := range volume.Mounts {
				if mount.Path == "/srv/encrypted" && volume.Partition == 1 && mount.ReadOnly {
					found = true
				}
			}
		}
	}
	if !found || result.Coverage.Status != i.Complete {
		t.Fatal("mapping reported as unmounted", result.Coverage)
	}
}
func TestInternalDataDiskIsNotExternal(t *testing.T) {
	root, c := fixture(t)
	block(t, root, "sdd", "8:48", "/sys/devices/pci0000:00/ata4/block/sdd", 0, false)
	result := collect(t, c)
	if len(result.Findings) != 2 || result.Coverage.Status != i.Complete {
		t.Fatal(result.Coverage, result.Findings)
	}
}
func TestIncompleteSourcesAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, string, *Collector)
		code   string
	}{
		{"unsupported multidevice root", func(t *testing.T, r string, c *Collector) {
			write(t, r, mountInfoPath, "10 1 8:1 / / rw - btrfs /dev/sda1 rw\n")
		}, "storage_root_unresolved"},
		{"unknown root", func(t *testing.T, r string, c *Collector) {
			write(t, r, mountInfoPath, "10 1 0:99 / / rw - overlay overlay rw\n")
		}, "storage_root_unresolved"},
		{"bad mount", func(t *testing.T, r string, c *Collector) { write(t, r, mountInfoPath, "broken") }, "storage_mounts_malformed"},
		{"missing mount", func(t *testing.T, r string, c *Collector) {
			if err := os.Remove(filepath.Join(r, strings.TrimPrefix(mountInfoPath, "/"))); err != nil {
				t.Fatal(err)
			}
		}, "storage_mounts_unreadable"},
		{"bad size", func(t *testing.T, r string, c *Collector) { write(t, r, usbLocation+"/size", "invalid") }, "storage_device_unreadable"},
		{"device bound", func(t *testing.T, r string, c *Collector) { c.Devices = 2 }, "storage_device_limit"},
		{"operation bound", func(t *testing.T, r string, c *Collector) { c.Operations = 3 }, "storage_source_limit"},
		{"byte bound", func(t *testing.T, r string, c *Collector) { c.Bytes = 20 }, "storage_mounts_unreadable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, c := fixture(t)
			tc.mutate(t, root, &c)
			result := collect(t, c)
			if result.Coverage.Status == i.Complete || !slices.Contains(result.Coverage.DiagnosticCodes, tc.code) {
				t.Fatal(result.Coverage)
			}
			if tc.code == "storage_root_unresolved" && len(result.Findings) != 0 {
				t.Fatal("unknown root generated judgments")
			}
		})
	}
}
func TestMissingOrDuplicatePersistentIdentifiersUseRunScope(t *testing.T) {
	root, c := fixture(t)
	for _, base := range []string{usbLocation, secondUSB} {
		write(t, root, base+"/device/wwid", "duplicated-id")
	}
	c.ScopeID = ""
	first := collect(t, c)
	second := collect(t, c)
	for n, finding := range first.Findings {
		if !strings.HasPrefix(finding.Identity.(i.StorageIdentity).Key, "unverified-device:") || finding.FindingID == second.Findings[n].FindingID {
			t.Fatal("ambiguous identifier reused")
		}
	}
}
func TestStableIdentityIgnoresTransientDeviceNodes(t *testing.T) {
	root, c := fixture(t)
	first := collect(t, c)
	// Observed device numbers/instance counters remain evidence, not facts.
	write(t, root, usbLocation+"/dev", "8:80")
	write(t, root, usbLocation+"/diskseq", "99")
	second := collect(t, c)
	for n, finding := range first.Findings {
		if finding.FindingID != second.Findings[n].FindingID || finding.FactFingerprint != second.Findings[n].FactFingerprint {
			t.Fatal("transient evidence changed matching")
		}
	}
	if reflect.DeepEqual(first.Evidence, second.Evidence) {
		t.Fatal("changed evidence not retained")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Collect(ctx, "fictional"); err != context.Canceled {
		t.Fatal(err)
	}
	c.Devices = -1
	if _, err := c.Collect(context.Background(), "fictional"); err == nil {
		t.Fatal("negative limit accepted")
	}
}
func TestLocalSourceRefusesDeviceContentReads(t *testing.T) {
	source := LocalSource{Root: t.TempDir()}
	if _, err := source.ReadFile("/dev/sda", 1024); err == nil {
		t.Fatal("source accepted a block-device path")
	}
}

func TestNoExternalDevicesIsCompleteEmpty(t *testing.T) {
	root, c := fixture(t)
	for _, name := range []string{"sdb", "sdb1", "sdc"} {
		if err := os.Remove(filepath.Join(root, "sys/class/block", name)); err != nil {
			t.Fatal(err)
		}
	}
	write(t, root, mountInfoPath, "10 1 8:1 / / rw - ext4 /dev/sda1 rw\n")
	result := collect(t, c)
	if result.Coverage.Status != i.Complete || len(result.Findings) != 0 {
		t.Fatal("successful empty mistaken for failure", result.Coverage)
	}
}
