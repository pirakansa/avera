package inventory

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

const (
	MaxUSBInputDevices   = 4096
	MaxUSBInputConsumers = 256
	MaxUSBInputHandlers  = 64
	MaxUSBInputGroups    = 4096
	MaxUSBInputPathBytes = 4096
	MaxUSBInputNameBytes = 256
)

// USBInputConsumer is a metadata projection of an observed open character-device
// descriptor. It does not establish event delivery or effective privileges.
type USBInputConsumer struct {
	ProcessName         string   `json:"process_name"`
	Executable          string   `json:"executable"`
	EffectiveUID        uint32   `json:"effective_uid"`
	EffectiveGID        uint32   `json:"effective_gid"`
	SupplementaryGroups []uint32 `json:"supplementary_groups"`
}
type USBInputFacts struct {
	Status          Status             `json:"status"`
	Handlers        []string           `json:"handlers"`
	Consumers       []USBInputConsumer `json:"consumers"`
	DiagnosticCodes []string           `json:"diagnostic_codes"`
}

// USBInputDevice retains volatile attachment names only in observation evidence.
type USBInputDevice struct {
	USBKey       string             `json:"usb_key"`
	SysfsPath    string             `json:"sysfs_path"`
	Node         string             `json:"node"`
	DeviceNumber string             `json:"device_number"`
	Handlers     []string           `json:"handlers"`
	Consumers    []USBInputConsumer `json:"consumers"`
}
type USBInputValues struct {
	Status          Status           `json:"status"`
	DiagnosticCodes []string         `json:"diagnostic_codes"`
	Devices         []USBInputDevice `json:"devices"`
}

// NormalizeUSBInput removes process/descriptor multiplicity and ordering noise.
// PID, descriptor numbers and event-node numbers never enter this stable basis.
func NormalizeUSBInput(v USBInputFacts) USBInputFacts {
	v.Handlers = sortedSet(v.Handlers, strings.Compare)
	v.DiagnosticCodes = sortedSet(v.DiagnosticCodes, strings.Compare)
	v.Consumers = append([]USBInputConsumer{}, v.Consumers...)
	for n := range v.Consumers {
		v.Consumers[n].SupplementaryGroups = sortedSet(v.Consumers[n].SupplementaryGroups, func(a, b uint32) int {
			if a < b {
				return -1
			}
			if a > b {
				return 1
			}
			return 0
		})
	}
	slices.SortFunc(v.Consumers, func(a, b USBInputConsumer) int {
		left, _ := Canonical(a)
		right, _ := Canonical(b)
		return strings.Compare(string(left), string(right))
	})
	v.Consumers = slices.CompactFunc(v.Consumers, func(a, b USBInputConsumer) bool {
		left, _ := Canonical(a)
		right, _ := Canonical(b)
		return string(left) == string(right)
	})
	return v
}
func validUSBInputFacts(v USBInputFacts) error {
	if !oneOf(v.Status, Complete, Partial, Failed) || len(v.Handlers) > MaxUSBInputHandlers || len(v.Consumers) > MaxUSBInputConsumers || len(v.DiagnosticCodes) > MaxUSBInputHandlers {
		return fmt.Errorf("invalid USB input scope")
	}
	if err := ids(v.Handlers, "input handler"); err != nil {
		return err
	}
	if err := ids(v.DiagnosticCodes, "input diagnostic"); err != nil {
		return err
	}
	for _, h := range v.Handlers {
		if len(h) > MaxUSBInputHandlers || strings.ContainsAny(h, "/\x00\r\n\t ") {
			return fmt.Errorf("invalid input handler")
		}
	}
	for _, c := range v.Consumers {
		if c.ProcessName == "" || len(c.ProcessName) > MaxUSBInputNameBytes || !strings.HasPrefix(c.Executable, "/") || len(c.Executable) > MaxUSBInputPathBytes || len(c.SupplementaryGroups) > MaxUSBInputGroups {
			return fmt.Errorf("invalid USB input consumer")
		}
		if err := unique(c.SupplementaryGroups, "input runtime group"); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	for _, c := range v.Consumers {
		data, _ := Canonical(c)
		if seen[string(data)] {
			return fmt.Errorf("duplicate input consumer")
		}
		seen[string(data)] = true
	}
	return nil
}
func validateUSBInputEvidence(revision string, v USBValues) error {
	hasInput := v.Input != nil
	for _, n := range v.Nodes {
		hasInput = hasInput || n.Facts.Input != nil
	}
	if !hasInput {
		return nil
	}
	if revision != ContractRevision || v.Input == nil {
		return fmt.Errorf("USB input requires input evidence and contract")
	}
	if !oneOf(v.Input.Status, Complete, Partial, Failed) || len(v.Input.Devices) > MaxUSBInputDevices || len(v.Input.DiagnosticCodes) > MaxUSBInputHandlers {
		return fmt.Errorf("invalid input evidence scope")
	}
	if err := ids(v.Input.DiagnosticCodes, "input diagnostics"); err != nil {
		return err
	}
	keys := map[string]USBNode{}
	for _, n := range v.Nodes {
		keys[n.Key] = n
	}
	seen := map[string]bool{}
	for _, d := range v.Input.Devices {
		n, ok := keys[d.USBKey]
		if !ok || !USBHID(n.Facts) || !strings.HasPrefix(d.SysfsPath, n.SysfsPath+"/") || path.Clean(d.SysfsPath) != d.SysfsPath || (d.Node != "" && (!strings.HasPrefix(d.Node, "/dev/input/event") || path.Clean(d.Node) != d.Node || !inputNumber(d.DeviceNumber))) || (d.Node == "" && (d.DeviceNumber != "" || len(d.Consumers) > 0)) || seen[d.SysfsPath] || len(d.SysfsPath) > MaxUSBInputPathBytes || (d.Node != "" && (!inputSuffix(d.Node, "/dev/input/event") || path.Base(d.Node) != path.Base(d.SysfsPath))) || (d.Node == "" && !inputSuffix(path.Base(d.SysfsPath), "input")) {
			return fmt.Errorf("invalid input attachment evidence")
		}
		seen[d.SysfsPath] = true
		if err := validUSBInputFacts(USBInputFacts{Status: Complete, Handlers: d.Handlers, Consumers: d.Consumers, DiagnosticCodes: []string{}}); err != nil {
			return err
		}
	}
	for _, n := range v.Nodes {
		if !USBHID(n.Facts) {
			continue
		}
		if n.Facts.Input == nil {
			return fmt.Errorf("missing USB input summary")
		}
		derived := USBInputFacts{Status: v.Input.Status, Handlers: []string{}, Consumers: []USBInputConsumer{}, DiagnosticCodes: v.Input.DiagnosticCodes}
		for _, d := range v.Input.Devices {
			if d.USBKey == n.Key {
				derived.Handlers = append(derived.Handlers, d.Handlers...)
				derived.Consumers = append(derived.Consumers, d.Consumers...)
			}
		}
		a, _ := Canonical(NormalizeUSBInput(derived))
		b, _ := Canonical(NormalizeUSBInput(*n.Facts.Input))
		if string(a) != string(b) {
			return fmt.Errorf("USB input evidence summary mismatch")
		}
	}
	return nil
}
func inputNumber(v string) bool {
	a, b, ok := strings.Cut(v, ":")
	if !ok || a == "" || b == "" {
		return false
	}
	return strings.IndexFunc(a+b, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// USBInputFor returns event-node evidence for one currently observed USB device.
func USBInputFor(s *Snapshot, key string) []USBInputDevice {
	out := []USBInputDevice{}
	for _, e := range s.Evidence {
		if v, ok := e.Values.(USBValues); ok && v.Input != nil {
			for _, d := range v.Input.Devices {
				if d.USBKey == key {
					out = append(out, d)
				}
			}
		}
	}
	return slices.Clone(out)
}

func inputSuffix(v, prefix string) bool {
	tail, ok := strings.CutPrefix(v, prefix)
	return ok && tail != "" && len(tail) <= 10 && strings.IndexFunc(tail, func(r rune) bool { return r < '0' || r > '9' }) < 0
}
