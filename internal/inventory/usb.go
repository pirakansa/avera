package inventory

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

type USBIdentity struct {
	Key string `json:"key"`
}

func (USBIdentity) identity() {}

type USBInterface struct {
	Number   string `json:"number"`
	Class    string `json:"class"`
	Subclass string `json:"subclass"`
	Protocol string `json:"protocol"`
}
type USBFacts struct {
	Input               *USBInputFacts `json:"input,omitempty"`
	VendorID            string         `json:"vendor_id"`
	ProductID           string         `json:"product_id"`
	Manufacturer        string         `json:"manufacturer"`
	Product             string         `json:"product"`
	Serial              string         `json:"serial"`
	DeviceClass         string         `json:"device_class"`
	Interfaces          []USBInterface `json:"interfaces"`
	StorageFingerprints []string       `json:"storage_fingerprints"`
	ObservationComplete bool           `json:"observation_complete"`
}

func (USBFacts) facts() {}

type USBNode struct {
	Key       string   `json:"key"`
	ParentKey string   `json:"parent_key"`
	PortPath  string   `json:"port_path"`
	SysfsPath string   `json:"sysfs_path"`
	Instance  string   `json:"instance"`
	Role      string   `json:"role"`
	Facts     USBFacts `json:"facts"`
}
type USBValues struct {
	Input *USBInputValues `json:"input,omitempty"`
	Nodes []USBNode       `json:"nodes"`
}

func (USBValues) evidenceValues() {}

func USBHID(f USBFacts) bool {
	if f.DeviceClass == "03" {
		return true
	}
	for _, v := range f.Interfaces {
		if v.Class == "03" {
			return true
		}
	}
	return false
}
func validUSB(f USBFacts) error {
	if f.Input != nil {
		if err := validUSBInputFacts(*f.Input); err != nil {
			return err
		}
	}
	hex := func(v string, n int) bool {
		return len(v) == n && strings.IndexFunc(v, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) < 0
	}
	if !hex(f.VendorID, 4) || !hex(f.ProductID, 4) || !hex(f.DeviceClass, 2) || len(f.Interfaces) > 256 {
		return fmt.Errorf("invalid USB metadata")
	}
	for _, v := range []string{f.Manufacturer, f.Product, f.Serial} {
		if len(v) > 256 {
			return fmt.Errorf("USB label limit")
		}
	}
	if err := ids(f.StorageFingerprints, "USB grouped storage fingerprint"); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, v := range f.Interfaces {
		if !hex(v.Number, 2) || !hex(v.Class, 2) || !hex(v.Subclass, 2) || !hex(v.Protocol, 2) || seen[v.Number] {
			return fmt.Errorf("invalid USB interface")
		}
		seen[v.Number] = true
	}
	return nil
}
func validUSBNodes(nodes []USBNode) error {
	if len(nodes) > 1024 {
		return fmt.Errorf("USB topology limit")
	}
	byKey := map[string]USBNode{}
	for _, n := range nodes {
		if n.Key == "" || byKey[n.Key].Key != "" || !oneOf(n.Role, "root_hub", "hub", "device") || !strings.HasPrefix(n.SysfsPath, "/sys/devices/") || path.Clean(n.SysfsPath) != n.SysfsPath || n.PortPath == "" || strings.ContainsAny(n.PortPath, "/\\") {
			return fmt.Errorf("invalid USB topology node")
		}
		if err := validUSB(n.Facts); err != nil {
			return err
		}
		byKey[n.Key] = n
	}
	for _, n := range nodes {
		seen := map[string]bool{}
		for k := n.Key; k != ""; {
			if seen[k] {
				return fmt.Errorf("USB topology cycle")
			}
			seen[k] = true
			v, ok := byKey[k]
			if !ok {
				return fmt.Errorf("unknown USB parent")
			}
			k = v.ParentKey
		}
	}
	return nil
}

// USBNodes returns only bounded, validated metadata from the current observation.
func USBNodes(s *Snapshot) []USBNode {
	for _, e := range s.Evidence {
		if v, ok := e.Values.(USBValues); ok && validUSBNodes(v.Nodes) == nil {
			return slices.Clone(v.Nodes)
		}
	}
	return nil
}

// USBStorageOwners groups storage only when independent reads agree on the current
// sysfs attachment, instance and descriptors. Product names never establish a link.
func USBStorageNodes(s *Snapshot) map[string]string {
	owners := map[string]string{}
	nodes := USBNodes(s)
	evidence := map[string]FileValues{}
	for _, e := range s.Evidence {
		if v, ok := e.Values.(FileValues); ok {
			evidence[e.EvidenceID] = v
		}
	}
	for _, f := range s.Findings {
		if f.Kind != ExternalStorage {
			continue
		}
		for _, eid := range f.EvidenceIDs {
			link := evidence[eid].USB
			if link == nil || link.Instance == "" {
				continue
			}
			for _, n := range nodes {
				if n.SysfsPath == link.SysfsPath && n.Instance == link.Instance && n.Facts.VendorID == link.VendorID && n.Facts.ProductID == link.ProductID && n.Facts.Serial == link.Serial {
					owners[f.FindingID] = n.Key
				}
			}
		}
	}
	return owners
}

// USBStorageOwners selects only attachments with an HID judgment owner.
func USBStorageOwners(s *Snapshot) map[string]string {
	nodes := USBStorageNodes(s)
	hid := map[string]string{}
	for _, f := range s.Findings {
		if id, ok := f.Identity.(USBIdentity); ok {
			hid[id.Key] = f.FindingID
		}
	}
	out := map[string]string{}
	for storage, key := range nodes {
		if owner := hid[key]; owner != "" {
			out[storage] = owner
		}
	}
	return out
}

// EnrichUSBStorage includes grouped storage changes in the USB judgment basis.
func EnrichUSBStorage(s *Snapshot) error {
	owners := USBStorageOwners(s)
	fingerprints := map[string][]string{}
	for _, f := range s.Findings {
		if owner := owners[f.FindingID]; owner != "" {
			fingerprints[owner] = append(fingerprints[owner], f.FactFingerprint)
		}
	}
	for n, f := range s.Findings {
		if facts, ok := f.Facts.(USBFacts); ok {
			facts.StorageFingerprints = append([]string{}, fingerprints[f.FindingID]...)
			slices.Sort(facts.StorageFingerprints)
			f.Facts = facts
			v, err := NormalizeFinding(s.TargetID, f)
			if err != nil {
				return err
			}
			s.Findings[n] = v
		}
	}
	return nil
}

func validateUSBContext(s *Snapshot) error {
	owners := USBStorageOwners(s)
	fps := map[string][]string{}
	nodes := map[string]USBNode{}
	for _, n := range USBNodes(s) {
		nodes[n.Key] = n
	}
	for _, f := range s.Findings {
		if owner := owners[f.FindingID]; owner != "" {
			fps[owner] = append(fps[owner], f.FactFingerprint)
		}
	}
	for _, f := range s.Findings {
		facts, ok := f.Facts.(USBFacts)
		if !ok {
			continue
		}
		node, found := nodes[f.Identity.(USBIdentity).Key]
		if !found || !USBHID(node.Facts) {
			return fmt.Errorf("USB finding lacks HID topology evidence")
		}
		expected := append([]string{}, fps[f.FindingID]...)
		slices.Sort(expected)
		actual := slices.Clone(facts.StorageFingerprints)
		slices.Sort(actual)
		if !slices.Equal(actual, expected) {
			return fmt.Errorf("USB grouped storage basis mismatch")
		}
		observed := node.Facts
		observed.StorageFingerprints = facts.StorageFingerprints
		_, fingerprint, err := Hashes(s.TargetID, USB, USBDevice, f.Identity, observed)
		if err != nil || fingerprint != f.FactFingerprint {
			return fmt.Errorf("USB finding/topology facts mismatch")
		}
	}
	return nil
}
