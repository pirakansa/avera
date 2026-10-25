// Package usb inventories current USB metadata without reading input or storage data.
package usb

import (
	"context"
	"crypto/rand"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	i "github.com/pirakansa/avera/internal/inventory"
)

const (
	Version           = "sysfs-usb-input-v2"
	busPath           = "/sys/bus/usb/devices"
	maxEntries        = 8192
	maxDevices        = 1024
	maxAttributeBytes = 4096
	maxSourceBytes    = 4 << 20
)

type Collector struct {
	Source       Source
	InputSource  InputSource
	IncludeInput bool
	Now          func() time.Time
	// ScopeID is for deterministic synthetic tests; live collections use a fresh scope.
	ScopeID string
}

func (c Collector) Collect(ctx context.Context, target string) (i.CollectorResult, error) {
	if c.Source == nil {
		c.Source = LocalSource{Root: "/"}
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.ScopeID == "" {
		var scope [16]byte
		if _, err := rand.Read(scope[:]); err != nil {
			return i.CollectorResult{}, err
		}
		c.ScopeID = fmt.Sprintf("%x", scope)
	}
	r := i.CollectorResult{Findings: []i.Finding{}, Evidence: []i.Evidence{}, Coverage: i.Coverage{Collector: i.USB, CollectorVersion: Version, Scope: "current USB sysfs devices and active interface metadata", Status: i.Complete, AttemptedSources: []string{busPath}, DiagnosticCodes: []string{}, Limitations: []string{
		"Current metadata only; no input events, HID reports, storage contents, device disabling or USB authorization writes.",
		"HID declarations are candidates, not authenticated device capabilities. Hubs and other USB devices are reference topology only; unoccupied ports and allowed-device enforcement are not inventoried.",
		"Purpose, permitted use, manuals and physical operating safeguards are human statements, not verified controls. Metadata reads are non-atomic.",
		"Missing or duplicate persistent identifiers use fresh collection-scoped keys; an unchanged serial does not authenticate a device.",
	}}}
	issues := 0
	diagnose := func(code string) {
		issues++
		r.Coverage.Status = i.Partial
		if !slices.Contains(r.Coverage.DiagnosticCodes, code) {
			r.Coverage.DiagnosticCodes = append(r.Coverage.DiagnosticCodes, code)
		}
	}
	remaining := maxSourceBytes
	read := func(base, name string, required bool) string {
		if ctx.Err() != nil || remaining <= 0 {
			diagnose("usb_source_limit")
			return ""
		}
		data, err := c.Source.ReadFile(base+"/"+name, min(maxAttributeBytes, remaining))
		if err != nil {
			if required {
				diagnose("usb_metadata_unreadable")
			}
			return ""
		}
		if len(data) > min(maxAttributeBytes, remaining) {
			diagnose("usb_source_limit")
			return ""
		}
		remaining -= len(data)
		v := strings.TrimSpace(string(data))
		if len(v) > 256 || !utf8.ValidString(v) || strings.ContainsAny(v, "\x00\r\n") {
			diagnose("usb_metadata_malformed")
			return ""
		}
		return v
	}
	names, more, err := c.Source.ReadDir(busPath, maxEntries)
	if err != nil {
		r.Coverage.Status = i.Failed
		r.Coverage.DiagnosticCodes = []string{"usb_directory_unreadable"}
		return r, ctx.Err()
	}
	if len(names) > maxEntries {
		names = names[:maxEntries]
		more = true
	}
	if more {
		diagnose("usb_entry_limit")
	}
	slices.Sort(names)
	nodes := []i.USBNode{}
	keys := map[string]int{}
	portKeys := map[string]string{}
	instances := map[string]string{}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		if strings.Contains(name, ":") {
			continue
		}
		if name == "" || path.Base(name) != name || name == "." || name == ".." {
			diagnose("usb_metadata_malformed")
			continue
		}
		if len(nodes) >= maxDevices {
			diagnose("usb_device_limit")
			break
		}
		base := busPath + "/" + name
		location, e := c.Source.Resolve(base)
		if e != nil || !strings.HasPrefix(location, "/sys/devices/") {
			diagnose("usb_metadata_unreadable")
			continue
		}
		before := issues
		f := i.USBFacts{VendorID: read(base, "idVendor", true), ProductID: read(base, "idProduct", true), Manufacturer: read(base, "manufacturer", false), Product: read(base, "product", false), Serial: read(base, "serial", false), DeviceClass: read(base, "bDeviceClass", true), StorageFingerprints: []string{}, Interfaces: []i.USBInterface{}, ObservationComplete: true}
		instance := read(base, "busnum", true) + "/" + read(base, "devnum", true) + "/" + read(base, "bConfigurationValue", true)
		expected, expectedErr := strconv.Atoi(read(base, "bNumInterfaces", true))
		if expectedErr != nil || expected < 0 || expected > 256 {
			diagnose("usb_interface_count_invalid")
			f.ObservationComplete = false
		}
		prefix := name + ":"
		role := "device"
		if strings.HasPrefix(name, "usb") {
			role = "root_hub"
			prefix = strings.TrimPrefix(name, "usb") + "-0:"
		} else if f.DeviceClass == "09" {
			role = "hub"
		}
		for _, entry := range names {
			if !strings.HasPrefix(entry, prefix) {
				continue
			}
			ib := busPath + "/" + entry
			if len(f.Interfaces) >= 256 {
				diagnose("usb_interface_limit")
				break
			}
			f.Interfaces = append(f.Interfaces, i.USBInterface{Number: read(ib, "bInterfaceNumber", true), Class: read(ib, "bInterfaceClass", true), Subclass: read(ib, "bInterfaceSubClass", true), Protocol: read(ib, "bInterfaceProtocol", true)})
		}
		if len(f.Interfaces) != expected || issues != before {
			f.ObservationComplete = false
			diagnose("usb_interfaces_incomplete")
		}
		if f.DeviceClass == "" || f.VendorID == "" || f.ProductID == "" {
			continue
		}
		key := ""
		if f.Serial != "" && role == "device" {
			key = "usb-serial:" + f.VendorID + ":" + f.ProductID + ":" + f.Serial
			keys[key]++
		}
		nodes = append(nodes, i.USBNode{Key: key, PortPath: name, SysfsPath: location, Instance: instance, Role: role, Facts: f})
		instances[name] = instance
	}
	for n := range nodes {
		v := &nodes[n]
		if v.Key == "" || keys[v.Key] > 1 {
			v.Key = "unverified-usb:" + c.ScopeID + "/" + v.PortPath
		}
		portKeys[v.PortPath] = v.Key
	}
	timestamp := c.Now().UTC().Format(time.RFC3339)
	// Recheck current device instance; attachment addresses are evidence, never stable identities.
	stable := []i.USBNode{}
	for _, v := range nodes {
		base := busPath + "/" + v.PortPath
		instance := read(base, "busnum", true) + "/" + read(base, "devnum", true) + "/" + read(base, "bConfigurationValue", true)
		location, e := c.Source.Resolve(base)
		if e != nil || location != v.SysfsPath || instance != instances[v.PortPath] || read(base, "idVendor", true) != v.Facts.VendorID || read(base, "idProduct", true) != v.Facts.ProductID {
			diagnose("usb_device_changed")
			delete(portKeys, v.PortPath)
			continue
		}
		stable = append(stable, v)
	}
	for n := range stable {
		v := &stable[n]
		if v.Role != "root_hub" {
			parent := ""
			if dot := strings.LastIndex(v.PortPath, "."); dot >= 0 {
				parent = v.PortPath[:dot]
			} else if dash := strings.Index(v.PortPath, "-"); dash > 0 {
				parent = "usb" + v.PortPath[:dash]
			}
			v.ParentKey = portKeys[parent]
			if v.ParentKey == "" {
				diagnose("usb_parent_unknown")
			}
		}
		if !i.USBHID(v.Facts) {
			continue
		}
		f, e := i.NormalizeFinding(target, i.Finding{Collector: i.USB, Kind: i.USBDevice, Identity: i.USBIdentity{Key: v.Key}, Facts: v.Facts, EvidenceIDs: []string{"usb-topology"}})
		if e != nil {
			diagnose("usb_metadata_malformed")
			v.Facts.ObservationComplete = false
			continue
		}
		r.Findings = append(r.Findings, f)
	}
	// Discard malformed nodes rather than publish an invalid artifact.
	clean := []i.USBNode{}
	for _, v := range stable {
		test := v
		test.ParentKey = ""
		s := &i.Snapshot{Evidence: []i.Evidence{{Values: i.USBValues{Nodes: []i.USBNode{test}}}}}
		if len(i.USBNodes(s)) == 1 {
			clean = append(clean, v)
		} else {
			diagnose("usb_metadata_malformed")
		}
	}
	present := map[string]bool{}
	for _, v := range clean {
		present[v.Key] = true
	}
	for n := range clean {
		if !present[clean[n].ParentKey] {
			clean[n].ParentKey = ""
		}
	}
	var input *i.USBInputValues
	if c.IncludeInput || c.InputSource != nil {
		source := c.InputSource
		if source == nil {
			if local, ok := c.Source.(LocalSource); ok {
				source = LocalInputSource{local}
			}
		}
		input = collectInput(ctx, source, clean)
		r.Coverage.AttemptedSources = append(r.Coverage.AttemptedSources, inputPath, inputHandlersPath, "/proc/*/fd", "/proc/*/status", "/proc/*/stat", "/proc/*/exe")
		r.Coverage.Limitations = append(r.Coverage.Limitations, "Input relationships are sysfs ancestry and open character-device descriptors only, not delivered events, focused applications or tty login access. hidraw, aggregate mice devices, device brokers and kernel internals beyond registered handlers are outside this backend. Process UID/GID/groups are observations in the reader namespace, not effective privilege certification.")
		if input.Status != i.Complete {
			diagnose("usb_input_incomplete")
		}
		for n := range clean {
			if !i.USBHID(clean[n].Facts) {
				continue
			}
			summary := i.USBInputFacts{Status: input.Status, Handlers: []string{}, Consumers: []i.USBInputConsumer{}, DiagnosticCodes: input.DiagnosticCodes}
			for _, d := range input.Devices {
				if d.USBKey == clean[n].Key {
					summary.Handlers = append(summary.Handlers, d.Handlers...)
					summary.Consumers = append(summary.Consumers, d.Consumers...)
				}
			}
			summary = i.NormalizeUSBInput(summary)
			clean[n].Facts.Input = &summary
		}
		r.Findings = []i.Finding{}
		for _, node := range clean {
			if !i.USBHID(node.Facts) {
				continue
			}
			finding, err := i.NormalizeFinding(target, i.Finding{Collector: i.USB, Kind: i.USBDevice, Identity: i.USBIdentity{Key: node.Key}, Facts: node.Facts, EvidenceIDs: []string{"usb-topology"}})
			if err != nil {
				return r, err
			}
			r.Findings = append(r.Findings, finding)
		}
	}
	r.Evidence = append(r.Evidence, i.Evidence{EvidenceID: "usb-topology", SourceKind: "usb_metadata", Location: busPath, CollectedAt: timestamp, Method: "bounded USB device and active interface sysfs metadata; no input data or device writes", Values: i.USBValues{Nodes: clean, Input: input}})
	slices.Sort(r.Coverage.DiagnosticCodes)
	return r, ctx.Err()
}
