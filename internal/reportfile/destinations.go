package reportfile

import (
	"encoding/json"
	"net/netip"
	"strings"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

// DetectedTargets matches only explicit DNAT address/port and protocol literals.
// It describes configured destinations, never effective reachability. Redirects,
// maps, ranges, missing protocols and ambiguous syntax remain for human review.
func (f *File) DetectedTargets(finding i.Finding) []i.Finding {
	facts, ok := finding.Facts.(i.ForwardingFacts)
	if !ok {
		return nil
	}
	proto, address, port := "", "", uint16(0)
	if strings.HasPrefix(strings.TrimSpace(facts.Rule), "{") {

		var wrapper struct {
			Rule struct {
				Expr []map[string]json.RawMessage `json:"expr"`
			} `json:"rule"`
		}
		if json.Unmarshal([]byte(facts.Rule), &wrapper) != nil {
			return nil
		}
		translations := 0
		for _, expr := range wrapper.Rule.Expr {
			if raw, ok := expr["dnat"]; ok {
				translations++
				var dest struct {
					Address string `json:"addr"`
					Port    uint16 `json:"port"`
				}
				if json.Unmarshal(raw, &dest) != nil {
					return nil
				}
				address = dest.Address
				port = dest.Port
			}
			if raw, ok := expr["match"]; ok {
				var match struct {
					Left struct {
						Payload struct {
							Protocol string `json:"protocol"`
							Field    string `json:"field"`
						} `json:"payload"`
					} `json:"left"`
				}
				if json.Unmarshal(raw, &match) != nil {
					return nil
				}
				payload := match.Left.Payload
				if payload.Field == "dport" && (payload.Protocol == "tcp" || payload.Protocol == "udp") {
					if proto != "" && proto != payload.Protocol {
						return nil
					}
					proto = payload.Protocol
				}
			}
		}
		if translations != 1 {
			return nil
		}

	} else {
		// Quoted/escaped text requires a full tokenizer; never guess from comments.
		if strings.ContainsAny(facts.Rule, "\"'\\") {
			return nil
		}
		fields := strings.Fields(facts.Rule)
		dnat := false
		for n := 0; n+1 < len(fields); n++ {
			switch fields[n] {
			case "-p", "--protocol":
				proto = fields[n+1]
			case "-j", "--jump":
				dnat = fields[n+1] == "DNAT"
			case "--to-destination":
				dest, err := netip.ParseAddrPort(fields[n+1])
				if err == nil {
					address = dest.Addr().String()
					port = dest.Port()
				}
			}
		}
		if !dnat {
			return nil
		}
	}
	addr, err := netip.ParseAddr(address)
	if err != nil || !addr.Unmap().IsLoopback() || port == 0 || (proto != "tcp" && proto != "udp") {
		return nil
	}
	out := []i.Finding{}
	for _, v := range f.Current.Findings {
		if !assessment.IsReference(v) {
			continue
		}
		id := v.Identity.(i.EndpointIdentity)
		a, err := netip.ParseAddr(id.LocalAddress)
		if err == nil && a.Unmap() == addr.Unmap() && id.Transport == proto && id.LocalPort == port {
			out = append(out, v)
		}
	}
	return out
}
