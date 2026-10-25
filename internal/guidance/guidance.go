// Package guidance supplies bundled review questions, not acceptance rules.
package guidance

import (
	i "github.com/pirakansa/avera/internal/inventory"
	"strings"
)

// Content helps a person explain an observation without certifying safety.
type Content struct {
	Observed  string
	Why       string
	Questions []string
}

// For returns fresh content for every implemented finding kind.
func For(kind i.Kind) Content {
	switch kind {
	case i.Forwarding:
		return Content{"A destination translation or redirect rule was observed; effective reachability is not verified.", "Review the entrance and its internal destination. Upstream safeguards do not remove the observed configuration.", []string{"What traffic, interface and source conditions apply, and where does it go?", "Does this lead to a loopback service? Select the observed service and explain the relationship.", "Which upstream router or cloud restrictions protect this entrance? Record references, remaining risk and the reason for your decision."}}
	case i.Endpoint:
		return Content{"A local network endpoint was observed. This does not establish external reachability.", "Review who needs this service and how access is protected.", []string{
			"What function does the service provide? Is it public static content or does it accept sensitive data?",
			"Who is the intended audience, and which networks or users should be allowed access?",
			"Which safeguards apply? Authentication may be inapplicable to intentionally public content; explain why and describe other protections.",
			"Is access restricted by an external firewall? Record this as a human statement and provide a configuration or design reference.",
			"What evidence supports the explanation, what risk remains, and who should decide the next action?",
		}}
	case i.Rule:
		return Content{"A sudo configuration rule was observed; it is not proof of every effective privilege.", "Review whether the configured privilege is needed and appropriately limited.", []string{
			"Who uses this configured privilege, including locally resolved group members?",
			"What required function needs this privilege?",
			"Which target users, commands, and arguments are allowed, and can they be limited?",
			"Which safeguards apply, and what configuration or operational evidence supports them?",
			"What interpretation or identity limitations remain, and who can resolve them before deciding?",
		}}
	case i.Unparsed:
		return Content{"Sudo configuration text could not be fully interpreted.", "Unsupported text needs human interpretation; no effective privilege conclusion follows.", []string{
			"Who can interpret this source in its full sudo policy context?",
			"What users, target users, commands, required function, and limits does it describe?",
			"What safeguards and supporting evidence can the reviewer provide?",
		}}
	default:
		return Content{"An observation needs review.", "Ask the collector owner to explain its meaning.", []string{"Who can interpret this finding and provide supporting evidence?"}}
	}
}

// Missing reports blank explanatory fields only. It does not judge adequacy,
// change a decision, or identify a tool-verified defect.
func Missing(a i.Assessment) []string {
	var out []string
	for _, f := range []struct{ name, value string }{{"purpose", a.Purpose}, {"exposure", a.Exposure}, {"safeguards", a.Safeguards}, {"rationale", a.Rationale}} {
		if strings.TrimSpace(f.value) == "" {
			out = append(out, f.name)
		}
	}
	if len(a.EvidenceRefs) == 0 {
		out = append(out, "supporting evidence references")
	}
	return out
}
