package inventory

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/netip"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Canonical encodes the hash preimage without HTML or Unicode escaping.
func Canonical(value any) ([]byte, error) {
	if err := validStrings(reflect.ValueOf(value)); err != nil {
		return nil, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	v, err := parseJSON(data)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	canonicalValue(&b, v)
	return b.Bytes(), nil
}
func canonicalString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 32 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
func canonicalValue(b *bytes.Buffer, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		canonicalString(b, x)
	case json.Number:
		n, _ := strconv.ParseInt(string(x), 10, 64)
		b.WriteString(strconv.FormatInt(n, 10))
	case []any:
		b.WriteByte('[')
		for i, v := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			canonicalValue(b, v)
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			canonicalString(b, k)
			b.WriteByte(':')
			canonicalValue(b, x[k])
		}
		b.WriteByte('}')
	}
}
func digest(v any) (string, error) {
	b, e := Canonical(v)
	if e != nil {
		return "", e
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b)), nil
}
func sortedSet[T comparable](a []T, cmp func(T, T) int) []T {
	out := append([]T{}, a...)
	slices.SortFunc(out, cmp)
	return slices.Compact(out)
}
func stringsSet(a []string) []string { return sortedSet(a, strings.Compare) }
func ownerCompare(a, b Owner) int {
	if a.UID < b.UID {
		return -1
	}
	if a.UID > b.UID {
		return 1
	}
	return strings.Compare(a.Executable, b.Executable)
}
func memberCompare(a, b Member) int {
	if a.UID < b.UID {
		return -1
	}
	if a.UID > b.UID {
		return 1
	}
	if c := strings.Compare(a.Name, b.Name); c != 0 {
		return c
	}
	return strings.Compare(a.Membership, b.Membership)
}
func normalizeAddress(family, address string) (string, error) {
	if family == "ipv4" {
		parts := strings.Split(address, ".")
		if len(parts) != 4 {
			return "", fmt.Errorf("invalid IPv4 address")
		}
		for i, p := range parts {
			if p == "" {
				return "", fmt.Errorf("invalid IPv4 octet")
			}
			for _, r := range p {
				if r < '0' || r > '9' {
					return "", fmt.Errorf("invalid IPv4 octet")
				}
			}
			n, e := strconv.ParseUint(p, 10, 8)
			if e != nil {
				return "", fmt.Errorf("invalid IPv4 octet")
			}
			parts[i] = strconv.FormatUint(n, 10)
		}
		return strings.Join(parts, "."), nil
	}
	if family != "ipv6" {
		return "", fmt.Errorf("invalid address family")
	}
	a, e := netip.ParseAddr(address)
	if e != nil || !a.Is6() || a.Zone() != "" {
		return "", fmt.Errorf("invalid IPv6 address (zone must be separate)")
	}
	raw := a.As16()
	parts := make([]string, 8)
	bestStart, bestLen := -1, 0
	for i := 0; i < 8; i++ {
		parts[i] = strconv.FormatUint(uint64(raw[2*i])<<8|uint64(raw[2*i+1]), 16)
	}
	for i := 0; i < 8; {
		if parts[i] != "0" {
			i++
			continue
		}
		j := i
		for j < 8 && parts[j] == "0" {
			j++
		}
		if j-i > bestLen && j-i >= 2 {
			bestStart, bestLen = i, j-i
		}
		i = j
	}
	if bestStart < 0 {
		return strings.Join(parts, ":"), nil
	}
	return strings.Join(parts[:bestStart], ":") + "::" + strings.Join(parts[bestStart+bestLen:], ":"), nil
}

// NormalizeIdentity preserves namespace and source anchors exactly.
func NormalizeIdentity(kind Kind, id Identity) (Identity, error) {
	switch kind {
	case USBDevice:
		x, ok := id.(USBIdentity)
		if !ok || x.Key == "" {
			return nil, fmt.Errorf("invalid USB identity")
		}
		return x, nil
	case ExternalStorage:
		x, ok := id.(StorageIdentity)
		if !ok || x.Key == "" {
			return nil, fmt.Errorf("invalid storage identity")
		}
		return x, nil
	case Account:
		x, ok := id.(AccountIdentity)
		if !ok || x.Name == "" || strings.ContainsAny(x.Name, "/:\x00\r\n\t ") {
			return nil, fmt.Errorf("invalid account identity")
		}
		return x, nil
	case Endpoint:
		x, ok := id.(EndpointIdentity)
		if !ok {
			return nil, fmt.Errorf("endpoint identity type mismatch")
		}
		s, e := normalizeAddress(x.Family, x.LocalAddress)
		if e != nil {
			return nil, e
		}
		x.LocalAddress = s
		return x, nil
	case Rule, Unparsed, Forwarding:
		x, ok := id.(SourceIdentity)
		if !ok {
			return nil, fmt.Errorf("source identity type mismatch")
		}
		if !strings.HasPrefix(x.SourcePath, "/") || x.Line <= 0 || x.Ordinal < 0 {
			return nil, fmt.Errorf("invalid source anchor")
		}
		for _, p := range strings.Split(x.SourcePath, "/") {
			if p == "." || p == ".." {
				return nil, fmt.Errorf("unresolved source path component")
			}
		}
		return x, nil
	}
	return nil, fmt.Errorf("unknown finding kind %q", kind)
}
func stripTerminator(s string) string {
	s = strings.TrimSuffix(s, "\n")
	return strings.TrimSuffix(s, "\r")
}
func normalizeFacts(f Facts) Facts {
	switch x := f.(type) {
	case USBFacts:
		if x.Input != nil {
			v := NormalizeUSBInput(*x.Input)
			x.Input = &v
		}
		x.StorageFingerprints = append([]string{}, x.StorageFingerprints...)
		slices.Sort(x.StorageFingerprints)
		x.Interfaces = append([]USBInterface{}, x.Interfaces...)
		slices.SortFunc(x.Interfaces, func(a, b USBInterface) int { return strings.Compare(a.Number, b.Number) })
		return x
	case StorageFacts:
		x.Volumes = append([]StorageVolume{}, x.Volumes...)
		for n := range x.Volumes {
			x.Volumes[n].Mounts = sortedSet(x.Volumes[n].Mounts, func(a, b StorageMount) int {
				if c := strings.Compare(a.Path, b.Path); c != 0 {
					return c
				}
				if c := strings.Compare(a.Filesystem, b.Filesystem); c != 0 {
					return c
				}
				if a.ReadOnly == b.ReadOnly {
					return 0
				}
				if a.ReadOnly {
					return -1
				}
				return 1
			})
		}
		slices.SortFunc(x.Volumes, func(a, b StorageVolume) int {
			if a.Partition < b.Partition {
				return -1
			}
			if a.Partition > b.Partition {
				return 1
			}
			return 0
		})
		return x
	case AccountFacts:
		x.Groups = sortedSet(x.Groups, func(a, b AccountGroup) int {
			if c := strings.Compare(a.Name, b.Name); c != 0 {
				return c
			}
			if a.GID < b.GID {
				return -1
			}
			if a.GID > b.GID {
				return 1
			}
			return strings.Compare(a.Membership, b.Membership)
		})
		return x
	case NetworkFacts:
		x.Owners = sortedSet(x.Owners, ownerCompare)
		return x
	case SudoFacts:
		x.RunAsUsers = stringsSet(x.RunAsUsers)
		x.RunAsGroups = stringsSet(x.RunAsGroups)
		x.LocalMembers = sortedSet(x.LocalMembers, memberCompare)
		x.Commands = append([]Command{}, x.Commands...)
		for i := range x.Commands {
			x.Commands[i].Tags = stringsSet(x.Commands[i].Tags)
		}
		x.PolicyContext = append([]string{}, x.PolicyContext...)
		for i := range x.PolicyContext {
			x.PolicyContext[i] = stripTerminator(x.PolicyContext[i])
		}
		return x
	case UnparsedFacts:
		x.Raw = stripTerminator(x.Raw)
		return x
	}
	return f
}

// Hashes centralizes the versioned, target-scoped identity and target-independent facts hash.
func Hashes(target string, collector Collector, kind Kind, id Identity, facts Facts) (string, string, error) {
	if target == "" || facts == nil {
		return "", "", fmt.Errorf("target and typed facts are required")
	}
	if err := validatePayload(collector, kind, id, facts); err != nil {
		return "", "", err
	}
	id, err := NormalizeIdentity(kind, id)
	if err != nil {
		return "", "", err
	}
	facts = normalizeFacts(facts)
	finding, err := digest([]any{IdentityVersion, target, collector, kind, id})
	if err != nil {
		return "", "", err
	}
	fingerprint, err := digest([]any{FactsVersion, kind, id, facts})
	return finding, fingerprint, err
}

// NormalizeFinding copies mutable slices and computes both hashes. It never edits caller data.
func NormalizeFinding(target string, f Finding) (Finding, error) {
	id, fp, err := Hashes(target, f.Collector, f.Kind, f.Identity, f.Facts)
	if err != nil {
		return Finding{}, err
	}
	f.Identity, err = NormalizeIdentity(f.Kind, f.Identity)
	if err != nil {
		return Finding{}, err
	}
	f.Facts = normalizeFacts(f.Facts)
	f.EvidenceIDs = stringsSet(f.EvidenceIDs)
	f.FindingID = id
	f.FactFingerprint = fp
	return f, nil
}

// MergeEndpoints unions duplicate endpoint observations and conservatively combines completeness.
func MergeEndpoints(target string, findings []Finding) ([]Finding, error) {
	out := []Finding{}
	byID := map[string]int{}
	for _, f := range findings {
		if f.Kind != Endpoint {
			return nil, fmt.Errorf("MergeEndpoints requires network endpoints")
		}
		n, err := NormalizeFinding(target, f)
		if err != nil {
			return nil, err
		}
		if i, ok := byID[n.FindingID]; ok {
			old := out[i]
			a := old.Facts.(NetworkFacts)
			b := n.Facts.(NetworkFacts)
			a.Owners = append(a.Owners, b.Owners...)
			a.AttributionComplete = a.AttributionComplete && b.AttributionComplete
			old.Facts = a
			old.EvidenceIDs = append(old.EvidenceIDs, n.EvidenceIDs...)
			out[i], err = NormalizeFinding(target, old)
			if err != nil {
				return nil, err
			}
		} else {
			byID[n.FindingID] = len(out)
			out = append(out, n)
		}
	}
	slices.SortFunc(out, func(a, b Finding) int { return strings.Compare(a.FindingID, b.FindingID) })
	return out, nil
}
