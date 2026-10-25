// Package storage inventories connected USB/removable non-root storage metadata.
package storage

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	i "github.com/pirakansa/avera/internal/inventory"
)

const Version = "sysfs-storage-v1"
const (
	blockPath          = "/sys/class/block"
	mountInfoPath      = "/proc/self/mountinfo"
	attributeBytes     = 4096
	mountInfoBytes     = 4 << 20
	defaultDevices     = 4096
	defaultMountRows   = 16384
	defaultSourceBytes = 8 << 20
	defaultOperations  = 65536
	maximumVolumes     = 128
	maximumMounts      = 256
	metadataLabelBytes = 256
	sectorBytes        = 512
)

type Collector struct {
	Source Source
	Now    func() time.Time
	// ScopeID asserts a fixed test incarnation. Live reads use a fresh random scope.
	ScopeID                               string
	Devices, MountRows, Bytes, Operations int
}
type scan struct {
	ctx               context.Context
	source            Source
	bytes, operations int
	diagnose          func(string)
	directories       map[string][]string
}

func (s *scan) read(name string, limit int) (string, error) {
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	if s.bytes <= 0 || s.operations <= 0 {
		s.diagnose("storage_source_limit")
		return "", fmt.Errorf("metadata limit")
	}
	s.operations--
	limit = min(limit, s.bytes)
	data, err := s.source.ReadFile(name, limit)
	if err != nil {
		return "", err
	}
	s.bytes -= len(data)
	if len(data) > limit || !utf8.Valid(data) {
		s.diagnose("storage_source_limit")
		return "", fmt.Errorf("invalid bounded metadata")
	}
	return strings.TrimSpace(string(data)), nil
}
func (s *scan) dir(name string, limit int) ([]string, error) {
	if values, ok := s.directories[name]; ok {
		return values, nil
	}
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	if s.operations <= 0 {
		s.diagnose("storage_source_limit")
		return nil, fmt.Errorf("metadata limit")
	}
	s.operations--
	names, more, err := s.source.ReadDir(name, limit)
	if more {
		s.diagnose("storage_device_limit")
	}
	slices.Sort(names)
	if err == nil {
		if s.directories == nil {
			s.directories = map[string][]string{}
		}
		s.directories[name] = names
	}
	return names, err
}
func (s *scan) resolve(name string) (string, error) {
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	if s.operations <= 0 {
		s.diagnose("storage_source_limit")
		return "", fmt.Errorf("metadata limit")
	}
	s.operations--
	value, err := s.source.Resolve(name)
	if err == nil && (!strings.HasPrefix(value, "/") || len(value) > attributeBytes) {
		return "", fmt.Errorf("invalid sysfs path")
	}
	return value, err
}
func safeName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\x00\r\n\t ")
}
func usbAncestor(location string) string {
	parts := strings.Split(location, "/")
	bus := -1
	for n, part := range parts {
		if strings.HasPrefix(part, "usb") {
			if _, err := strconv.ParseUint(strings.TrimPrefix(part, "usb"), 10, 32); err == nil {
				bus = n
				break
			}
		}
	}
	if bus < 0 {
		return ""
	}
	for n := bus + 1; n < len(parts); n++ {
		part := parts[n]
		if strings.Contains(part, ":") {
			return strings.Join(parts[:n], "/")
		}
	}
	return ""
}

type node struct {
	name, location, number, parent string
	partition                      int64
	removable                      bool
	valid                          bool
}

func (s *scan) nodes(names []string) map[string]node {
	out := map[string]node{}
	for _, name := range names {
		if !safeName(name) {
			s.diagnose("storage_metadata_malformed")
			continue
		}
		base := blockPath + "/" + name
		item := node{name: name, valid: true}
		var err error
		item.location, err = s.resolve(base)
		if err != nil {
			s.diagnose("storage_device_unreadable")
			continue
		}
		item.number, err = s.read(base+"/dev", attributeBytes)
		if err != nil || !deviceNumber(item.number) {
			s.diagnose("storage_device_unreadable")
			continue
		}
		partition, partitionErr := s.read(base+"/partition", attributeBytes)
		if partitionErr == nil {
			item.partition, err = strconv.ParseInt(partition, 10, 32)
			if err != nil || item.partition <= 0 {
				item.valid = false
				s.diagnose("storage_metadata_malformed")
			}
			item.parent = path.Base(path.Dir(item.location))
		} else if !errors.Is(partitionErr, os.ErrNotExist) {
			item.valid = false
			s.diagnose("storage_device_unreadable")
		}
		if item.partition == 0 {
			value, readErr := s.read(base+"/removable", attributeBytes)
			if readErr != nil || (value != "0" && value != "1") {
				item.valid = false
				s.diagnose("storage_device_unreadable")
			}
			item.removable = value == "1"
		}
		out[name] = item
	}
	return out
}
func (s *scan) backing(nodes map[string]node, number string) (map[string]bool, map[string]bool, error) {
	roots := map[string]bool{}
	start := ""
	for name, item := range nodes {
		if item.number == number {
			if start != "" {
				return nil, nil, fmt.Errorf("ambiguous root device")
			}
			start = name
		}
	}
	if start == "" {
		return nil, nil, fmt.Errorf("root backing device unavailable")
	}
	visited, active := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(name string) error {
		if active[name] {
			return fmt.Errorf("root backing cycle")
		}
		if visited[name] {
			return nil
		}
		visited[name] = true
		active[name] = true
		defer delete(active, name)
		item, ok := nodes[name]
		if !ok || !item.valid {
			return fmt.Errorf("root metadata unavailable")
		}
		if item.partition > 0 {
			return visit(item.parent)
		}
		children, err := s.dir(blockPath+"/"+name+"/slaves", len(nodes))
		if err != nil {
			return err
		}
		if len(children) == 0 {
			if strings.Contains(item.location, "/virtual/") {
				return fmt.Errorf("unresolved virtual root")
			}
			roots[name] = true
			return nil
		}
		for _, child := range children {
			if !safeName(child) {
				return fmt.Errorf("invalid root child")
			}
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(start); err != nil {
		return nil, nil, err
	}
	return roots, visited, nil
}
func (s *scan) optional(name string) string {
	value, err := s.read(name, attributeBytes)
	if err != nil {
		return ""
	}
	if len(value) > metadataLabelBytes {
		s.diagnose("storage_metadata_limit")
		return ""
	}
	return value
}
func (c Collector) Collect(ctx context.Context, target string) (i.CollectorResult, error) {
	r := i.CollectorResult{Findings: []i.Finding{}, Evidence: []i.Evidence{}, Coverage: i.Coverage{Collector: i.Storage, CollectorVersion: Version, Status: i.Complete, Scope: "USB/removable non-root backing block devices in current mount namespace", AttemptedSources: []string{blockPath, mountInfoPath}, DiagnosticCodes: []string{}, Limitations: []string{"USB ancestry/removable flags identify candidates, not guaranteed physical externality. Other fixed external transports and USB devices without a block node are outside this backend.", "Root filesystem backing disks and every partition on those disks are excluded. Root support is ext2/ext3/ext4/xfs with a visible block graph; unsupported or unresolved roots suppress candidate judgments, not an absence claim.", "Metadata only: no mounting, content reads, file-permission audit, malware check, device disabling or connection-history collection. Read-only flags are observations, not access guarantees.", "Non-atomic reads; device identity metadata is not authenticated. Missing/duplicate persistent identifiers use fresh run-scoped identities."}}}
	if target == "" {
		return r, fmt.Errorf("target is required")
	}
	if c.Source == nil {
		c.Source = LocalSource{Root: "/"}
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	for _, pair := range []struct {
		value    *int
		fallback int
	}{{&c.Devices, defaultDevices}, {&c.MountRows, defaultMountRows}, {&c.Bytes, defaultSourceBytes}, {&c.Operations, defaultOperations}} {
		if *pair.value < 0 {
			return r, fmt.Errorf("negative storage limit")
		}
		if *pair.value == 0 {
			*pair.value = pair.fallback
		}
	}
	if c.ScopeID == "" {
		c.ScopeID = rand.Text()
	}
	r.Coverage.Scope = "storage-namespace:" + c.ScopeID
	r.Coverage.Limitations = append(r.Coverage.Limitations, "Live namespace incarnation is unverified; fresh scope avoids claiming a prior device disappeared across hosts/namespaces. Stable observed identifiers can still match human judgments.")
	diagnose := func(code string) {
		r.Coverage.Status = i.Partial
		if !slices.Contains(r.Coverage.DiagnosticCodes, code) {
			r.Coverage.DiagnosticCodes = append(r.Coverage.DiagnosticCodes, code)
		}
	}
	s := scan{ctx: ctx, source: c.Source, bytes: c.Bytes, operations: c.Operations, diagnose: diagnose}
	raw, err := s.read(mountInfoPath, mountInfoBytes)
	if err != nil {
		diagnose("storage_mounts_unreadable")
		r.Coverage.Status = i.Failed
		return r, ctx.Err()
	}
	mounts, err := parseMounts(raw, c.MountRows)
	if err != nil {
		diagnose("storage_mounts_malformed")
		return r, ctx.Err()
	}
	switch mounts.rootFilesystem {
	case "ext2", "ext3", "ext4", "xfs":
	default:
		diagnose("storage_root_unresolved")
		return r, ctx.Err()
	}
	names, err := s.dir(blockPath, c.Devices)
	if err != nil {
		diagnose("storage_devices_unreadable")
		r.Coverage.Status = i.Failed
		return r, ctx.Err()
	}
	nodes := s.nodes(names)
	roots, _, err := s.backing(nodes, mounts.root)
	if err != nil {
		diagnose("storage_root_unresolved")
		return r, ctx.Err()
	}
	mountedNodes := map[string][]i.StorageMount{}
	numbers := make([]string, 0, len(mounts.byDevice))
	for number := range mounts.byDevice {
		numbers = append(numbers, number)
	}
	slices.Sort(numbers)
	for _, number := range numbers {
		known := false
		for _, item := range nodes {
			if item.number == number {
				known = true
				break
			}
		}
		if !known {
			continue
		}
		disks, trace, err := s.backing(nodes, number)
		if err != nil {
			diagnose("storage_mount_mapping_incomplete")
			continue
		}
		for disk := range disks {
			partitions := []string{}
			for name := range trace {
				item := nodes[name]
				if item.partition > 0 && item.parent == disk {
					partitions = append(partitions, name)
				}
			}
			if len(partitions) == 0 {
				partitions = []string{disk}
			}
			for _, name := range partitions {
				mountedNodes[name] = append(mountedNodes[name], mounts.byDevice[number]...)
			}
		}
	}
	type candidate struct {
		item     node
		facts    i.StorageFacts
		key      string
		nodes    []string
		sequence string
	}
	candidates := []candidate{}
	keys := map[string]int{}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		item, ok := nodes[name]
		if !ok || item.partition > 0 || !item.valid || roots[name] {
			continue
		}
		ancestor := usbAncestor(item.location)
		if ancestor == "" && !item.removable {
			continue
		}
		base := blockPath + "/" + name
		facts := i.StorageFacts{Connection: "removable", Vendor: s.optional(base + "/device/vendor"), Model: s.optional(base + "/device/model"), Serial: s.optional(base + "/device/serial"), Volumes: []i.StorageVolume{}, ObservationComplete: true}
		if ancestor != "" {
			facts.Connection = "usb"
			if facts.Serial == "" {
				facts.Serial = s.optional(ancestor + "/serial")
			}
		}
		sequence := s.optional(base + "/diskseq")
		size, sizeErr := s.read(base+"/size", attributeBytes)
		sectors, parseErr := strconv.ParseInt(size, 10, 64)
		ro, roErr := s.read(base+"/ro", attributeBytes)
		if sizeErr != nil || parseErr != nil || sectors < 0 || sectors > i.MaxInteger/sectorBytes || roErr != nil || (ro != "0" && ro != "1") {
			diagnose("storage_device_unreadable")
			continue
		}
		facts.SizeBytes = sectors * sectorBytes
		facts.ReadOnly = ro == "1"
		key := ""
		wwid := s.optional(base + "/device/wwid")
		if wwid != "" {
			key = "wwid:" + wwid
		} else if ancestor != "" && facts.Serial != "" {
			vendor := s.optional(ancestor + "/idVendor")
			product := s.optional(ancestor + "/idProduct")
			if vendor != "" && product != "" {
				key = "usb:" + vendor + ":" + product + ":" + facts.Serial
			}
		}
		deviceNodes := []string{}
		mountCount := 0
		seenPartitions := map[int64]bool{}
		for _, volumeName := range names {
			volume, exists := nodes[volumeName]
			if !exists || !volume.valid || (volume.name != name && volume.parent != name) {
				continue
			}
			if len(facts.Volumes) >= maximumVolumes {
				diagnose("storage_volume_limit")
				facts.ObservationComplete = false
				break
			}
			if seenPartitions[volume.partition] {
				diagnose("storage_metadata_malformed")
				facts.ObservationComplete = false
				continue
			}
			seenPartitions[volume.partition] = true
			observed := slices.Clone(mountedNodes[volume.name])
			uniqueMounts := []i.StorageMount{}
			seenMounts := map[i.StorageMount]bool{}
			for _, mount := range observed {
				if !seenMounts[mount] {
					uniqueMounts = append(uniqueMounts, mount)
					seenMounts[mount] = true
				}
			}
			observed = uniqueMounts
			if observed == nil {
				observed = []i.StorageMount{}
			}
			if len(observed) > maximumMounts-mountCount {
				observed = observed[:maximumMounts-mountCount]
				diagnose("storage_mount_limit")
				facts.ObservationComplete = false
			}
			mountCount += len(observed)
			facts.Volumes = append(facts.Volumes, i.StorageVolume{Partition: volume.partition, Mounts: observed})
			deviceNodes = append(deviceNodes, "/dev/"+volume.name)
		}
		// Recheck the device instance when the kernel exports a disk sequence.
		number, numberErr := s.read(base+"/dev", attributeBytes)
		location, locationErr := s.resolve(base)
		after := s.optional(base + "/diskseq")
		if numberErr != nil || number != item.number || locationErr != nil || location != item.location || (sequence != "" && sequence != after) {
			diagnose("storage_device_changed")
			continue
		}
		candidates = append(candidates, candidate{item: item, facts: facts, key: key, nodes: deviceNodes, sequence: sequence})
		if key != "" {
			keys[key]++
		}
	}
	timestamp := c.Now().UTC().Format(time.RFC3339)
	for n, candidate := range candidates {
		key := candidate.key
		if key == "" || keys[key] > 1 {
			key = "unverified-device:" + c.ScopeID + "/" + candidate.item.name
		}
		eid := fmt.Sprintf("storage-%d", n+1)
		excerpt, err := json.Marshal(struct {
			DeviceNodes  []string       `json:"device_nodes"`
			DeviceNumber string         `json:"device_number"`
			DiskSequence string         `json:"disk_sequence"`
			Observed     i.StorageFacts `json:"observed"`
		}{candidate.nodes, candidate.item.number, candidate.sequence, candidate.facts})
		if err != nil {
			return r, err
		}
		r.Evidence = append(r.Evidence, i.Evidence{EvidenceID: eid, SourceKind: "local_file", Location: blockPath + "/" + candidate.item.name, CollectedAt: timestamp, Method: "bounded sysfs block metadata and sanitized current-namespace mountinfo projection; no device contents read", Values: i.FileValues{Excerpt: string(excerpt)}})
		finding, err := i.NormalizeFinding(target, i.Finding{Collector: i.Storage, Kind: i.ExternalStorage, Identity: i.StorageIdentity{Key: key}, Facts: candidate.facts, EvidenceIDs: []string{eid}})
		if err != nil {
			return r, err
		}
		r.Findings = append(r.Findings, finding)
	}
	slices.Sort(r.Coverage.DiagnosticCodes)
	return r, ctx.Err()
}
