package reportfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

// Policy is an explicit human judgment for a service class, independent of host
// identity, process ownership and namespace incarnation. It is not an observation.
type Policy struct {
	Transport  string       `json:"transport"`
	Family     string       `json:"family"`
	Port       uint16       `json:"port"`
	Binding    string       `json:"binding"`
	Assessment i.Assessment `json:"assessment"`
}

const SharedJudgment = "shared_service_judgment"

func servicePolicy(f i.Finding) (Policy, bool) {
	if f.Kind != i.Endpoint || assessment.IsReference(f) {
		return Policy{}, false
	}
	id, ok := f.Identity.(i.EndpointIdentity)
	if !ok {
		return Policy{}, false
	}
	addr, err := netip.ParseAddr(id.LocalAddress)
	if err != nil {
		return Policy{}, false
	}
	binding := "specific_address"
	if addr.IsUnspecified() {
		binding = "all_interfaces"
	}
	return Policy{Transport: id.Transport, Family: id.Family, Port: id.LocalPort, Binding: binding}, true
}

type serviceKey struct {
	Transport, Family, Binding string
	Port                       uint16
}

func (p Policy) key() serviceKey       { return serviceKey{p.Transport, p.Family, p.Binding, p.Port} }
func (p Policy) matches(q Policy) bool { return p.key() == q.key() }
func (f *File) validatePolicies() error {
	if len(f.Policies) > i.MaxEntries {
		return fmt.Errorf("too many shared policies")
	}
	seen := map[serviceKey]bool{}
	byID := map[string]i.Assessment{}
	for _, a := range f.Assessments.Entries {
		byID[a.AssessmentID] = a
	}
	for _, p := range f.Policies {
		if p.Port == 0 || (p.Transport != "tcp" && p.Transport != "udp") || (p.Family != "ipv4" && p.Family != "ipv6") || (p.Binding != "all_interfaces" && p.Binding != "specific_address") || p.Assessment.Selector.Kind != i.Endpoint {
			return fmt.Errorf("invalid shared service policy")
		}
		if seen[p.key()] {
			return fmt.Errorf("duplicate shared service policy")
		}
		seen[p.key()] = true
		key, ok := servicePolicy(i.Finding{Collector: i.Network, Kind: i.Endpoint, Identity: p.Assessment.Selector.Identity})
		if !ok || !p.matches(key) {
			return fmt.Errorf("policy selector does not match its service class")
		}
		if a, ok := byID[p.Assessment.AssessmentID]; ok {
			left, err := i.Canonical(a)
			if err != nil {
				return err
			}
			right, err := i.Canonical(p.Assessment)
			if err != nil || !bytes.Equal(left, right) {
				return fmt.Errorf("shared assessment ID has inconsistent content")
			}
		}
		a := &i.Assessments{Envelope: f.Assessments.Envelope, Entries: []i.Assessment{p.Assessment}}
		if _, err := i.Encode(a); err != nil {
			return fmt.Errorf("invalid policy judgment: %w", err)
		}
	}
	return nil
}
func (f *File) applyPolicy(edit Edit, found i.Finding, a i.Assessment) error {
	if edit.Reuse == nil {
		return nil
	}
	p, ok := servicePolicy(found)
	if !ok {
		if *edit.Reuse {
			return fmt.Errorf("shared policies currently support network endpoints only")
		}
		return nil
	}
	kept := []Policy{}
	for _, old := range f.Policies {
		if !p.matches(old) {
			kept = append(kept, old)
		}
	}
	if *edit.Reuse {
		p.Assessment = a
		kept = append(kept, p)
	}
	f.Policies = kept
	return nil
}

// PolicyJudgments includes shared explanations for presentation without changing
// their original selector, basis, author or timestamp.
func (f *File) PolicyJudgments() *i.Assessments {
	a := *f.Assessments
	a.Entries = append([]i.Assessment{}, a.Entries...)
	seen := map[string]bool{}
	for _, v := range a.Entries {
		seen[v.AssessmentID] = true
	}
	for _, p := range f.Policies {
		if !seen[p.Assessment.AssessmentID] {
			a.Entries = append(a.Entries, p.Assessment)
			seen[p.Assessment.AssessmentID] = true
		}
	}
	return &a
}
func (f *File) applyPolicies(result *assessment.Result) {
	sharedIDs := map[string]bool{}
	for _, p := range f.Policies {
		sharedIDs[p.Assessment.AssessmentID] = true
	}
	unmatched := []string{}
	for _, id := range result.Review.UnmatchedAssessmentIDs {
		if !sharedIDs[id] {
			unmatched = append(unmatched, id)
		}
	}
	result.Review.UnmatchedAssessmentIDs = unmatched
	entries := []i.Assessment{}
	for _, a := range result.Unmatched.Entries {
		if !sharedIDs[a.AssessmentID] {
			entries = append(entries, a)
		}
	}
	result.Unmatched.Entries = entries
	findings := map[string]i.Finding{}
	for _, v := range f.Current.Findings {
		findings[v.FindingID] = v
	}

	policies := map[serviceKey]Policy{}
	for _, p := range f.Policies {
		policies[p.key()] = p
	}
	byID := map[string]i.Assessment{}
	for _, a := range f.Assessments.Entries {
		byID[a.AssessmentID] = a
	}
	for n := range result.Review.Items {
		item := &result.Review.Items[n]
		if item.Presence != i.Current || item.Applicability == i.Conflict {
			continue
		}
		key, ok := servicePolicy(findings[item.FindingID])
		if !ok {
			continue
		}
		p, ok := policies[key.key()]
		if !ok {
			continue
		}
		separate := false
		for _, id := range item.AssessmentIDs {
			a := byID[id]
			if a.AssessmentID != p.Assessment.AssessmentID && a.Actor != "" {
				separate = true
				break
			}
		}
		if separate {
			continue
		}
		item.CurrentDecision = p.Assessment.Decision
		item.Applicability = i.Applicable
		item.AssessmentIDs = []string{p.Assessment.AssessmentID}
		item.ReasonCodes = []string{SharedJudgment}
		for _, c := range f.Current.Coverage {
			if c.Collector == i.Network && c.Status != i.Complete {
				item.ReasonCodes = append(item.ReasonCodes, assessment.CoverageIncomplete)
				break
			}
		}
	}

}

// UnmarshalJSON uses the strict inventory decoder for interface-valued assessments.
func (p *Policy) UnmarshalJSON(data []byte) error {
	var wire struct {
		Transport  string          `json:"transport"`
		Family     string          `json:"family"`
		Port       uint16          `json:"port"`
		Binding    string          `json:"binding"`
		Assessment json.RawMessage `json:"assessment"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		return err
	}
	wrapped, err := json.Marshal(map[string]any{"schema_version": i.SchemaVersion, "contract_revision": i.ContractRevision, "kind": "assessments", "target_id": "shared-policy", "entries": []json.RawMessage{wire.Assessment}})
	if err != nil {
		return err
	}
	a, err := i.Decode(wrapped)
	if err != nil {
		return err
	}
	*p = Policy{Transport: wire.Transport, Family: wire.Family, Port: wire.Port, Binding: wire.Binding, Assessment: a.(*i.Assessments).Entries[0]}
	return nil
}
