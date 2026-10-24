package assessment

import i "github.com/pirakansa/avera/internal/inventory"

// Partial expectations are valid context, but never an applicability match.
func expectationMatches(target string, f i.Finding, e i.Expectation) bool {
	if e.NormalizationVersion != i.FactsVersion {
		return false
	}
	x := e.Facts
	var facts i.Facts
	switch f.Kind {
	case i.Endpoint:
		if x.Owners == nil || x.AttributionComplete == nil || !*x.AttributionComplete {
			return false
		}
		facts = i.NetworkFacts{Owners: *x.Owners, AttributionComplete: *x.AttributionComplete}
	case i.Rule:
		if x.Principal == nil || x.Host == nil || x.RunAsUsers == nil || x.RunAsGroups == nil || x.Commands == nil || x.LocalMembers == nil || x.LocalContextComplete == nil || x.PolicyContext == nil || x.InterpretationComplete == nil || !*x.LocalContextComplete || !*x.InterpretationComplete {
			return false
		}
		facts = i.SudoFacts{Principal: *x.Principal, Host: *x.Host, RunAsUsers: *x.RunAsUsers, RunAsGroups: *x.RunAsGroups, Commands: *x.Commands, LocalMembers: *x.LocalMembers, LocalContextComplete: *x.LocalContextComplete, PolicyContext: *x.PolicyContext, InterpretationComplete: *x.InterpretationComplete}
	default:
		return false
	}
	_, fp, err := i.Hashes(target, f.Collector, f.Kind, f.Identity, facts)
	return err == nil && fp == f.FactFingerprint
}
