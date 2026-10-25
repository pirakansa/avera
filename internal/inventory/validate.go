package inventory

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func oneOf[T ~string](v T, allowed ...T) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
func timestamp(s string) error {
	const layout = "2006-01-02T15:04:05Z"
	t, e := time.Parse(layout, s)
	if e != nil || t.Format(layout) != s || t.Year() < 1 {
		return fmt.Errorf("invalid UTC timestamp %q", s)
	}
	return nil
}
func unique[T comparable](values []T, label string) error {
	seen := map[T]bool{}
	for _, v := range values {
		if seen[v] {
			return fmt.Errorf("duplicate %s: %v", label, v)
		}
		seen[v] = true
	}
	return nil
}
func ids(values []string, label string) error {
	for _, s := range values {
		if s == "" {
			return fmt.Errorf("empty %s", label)
		}
	}
	return unique(values, label)
}
func validHash(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	for _, r := range s[7:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
func validatePayload(c Collector, k Kind, id Identity, f Facts) error {
	if !((c == Network && (k == Endpoint || k == Forwarding)) || (c == Sudo && (k == Rule || k == Unparsed)) || (c == Accounts && k == Account) || (c == Storage && k == ExternalStorage) || (c == USB && k == USBDevice)) {
		return fmt.Errorf("collector/kind mismatch")
	}
	n, e := NormalizeIdentity(k, id)
	if e != nil {
		return e
	}
	switch x := n.(type) {
	case EndpointIdentity:
		if !oneOf(x.Transport, "tcp", "udp") || x.ScopeID == "" || !((x.Transport == "tcp" && x.SocketCategory == "listener") || (x.Transport == "udp" && x.SocketCategory == "bound_unconnected")) {
			return fmt.Errorf("invalid endpoint transport/category/scope")
		}
	}
	if f == nil {
		return nil
	}
	switch k {
	case USBDevice:
		x, ok := f.(USBFacts)
		if !ok || !USBHID(x) {
			return fmt.Errorf("invalid HID finding")
		}
		if err := validUSB(x); err != nil {
			return err
		}
	case ExternalStorage:
		x, ok := f.(StorageFacts)
		if !ok || !oneOf(x.Connection, "usb", "removable") || x.SizeBytes < 0 || x.SizeBytes > MaxInteger {
			return fmt.Errorf("invalid external storage facts")
		}
		parts := map[int64]bool{}
		for _, volume := range x.Volumes {
			if volume.Partition < 0 || parts[volume.Partition] {
				return fmt.Errorf("invalid/duplicate storage partition")
			}
			parts[volume.Partition] = true
			if err := unique(volume.Mounts, "storage mount"); err != nil {
				return err
			}
			for _, mount := range volume.Mounts {
				if !strings.HasPrefix(mount.Path, "/") || mount.Filesystem == "" {
					return fmt.Errorf("invalid storage mount")
				}
			}
		}
		return nil
	case Account:
		x, ok := f.(AccountFacts)
		if !ok {
			return fmt.Errorf("account facts type mismatch")
		}
		primary := false
		for _, group := range x.Groups {
			if group.Name == "" || !oneOf(group.Membership, "primary", "supplementary") {
				return fmt.Errorf("invalid account group")
			}
		}
		for _, group := range x.Groups {
			if group.Membership == "primary" {
				if group.GID != x.PrimaryGID {
					return fmt.Errorf("primary group GID mismatch")
				}
				primary = true
			}
		}
		if x.ContextComplete && !primary {
			return fmt.Errorf("complete account requires resolved primary group")
		}
		return unique(x.Groups, "account group")
	case Endpoint:
		x, ok := f.(NetworkFacts)
		if !ok {
			return fmt.Errorf("network facts type mismatch")
		}
		return networkFacts(x)
	case Forwarding:
		x, ok := f.(ForwardingFacts)
		if !ok || x.Rule == "" || !validHash(x.ContextDigest) || x.ScopeID == "" {
			return fmt.Errorf("invalid forwarding facts")
		}
		return nil
	case Rule:
		x, ok := f.(SudoFacts)
		if !ok {
			return fmt.Errorf("sudo facts type mismatch")
		}
		if !oneOf(x.Principal.Type, "user", "group") || x.Principal.Name == "" || x.Host == "" {
			return fmt.Errorf("invalid sudo principal/host")
		}
		for _, cmd := range x.Commands {
			if cmd.Path == "" {
				return fmt.Errorf("empty command path")
			}
			for _, tag := range cmd.Tags {
				if !oneOf(tag, "PASSWD", "NOPASSWD") {
					return fmt.Errorf("invalid sudo tag %q", tag)
				}
			}
		}
		for _, m := range x.LocalMembers {
			if m.Name == "" || !oneOf(m.Membership, "primary", "supplementary") {
				return fmt.Errorf("invalid local member")
			}
		}
	case Unparsed:
		x, ok := f.(UnparsedFacts)
		if !ok {
			return fmt.Errorf("unparsed facts type mismatch")
		}
		if x.Raw == "" || x.DiagnosticCode == "" {
			return fmt.Errorf("empty unparsed raw/diagnostic")
		}
	}
	return nil
}
func networkFacts(x NetworkFacts) error {
	if x.AttributionComplete && len(x.Owners) == 0 {
		return fmt.Errorf("complete attribution requires owners")
	}
	for _, o := range x.Owners {
		if x.AttributionComplete && o.Executable == "" {
			return fmt.Errorf("complete attribution requires executable")
		}
	}
	return nil
}
func payloadSets(f Facts) error {
	switch x := f.(type) {
	case NetworkFacts:
		return unique(x.Owners, "owner")
	case SudoFacts:
		for _, a := range [][]string{x.RunAsUsers, x.RunAsGroups} {
			if e := unique(a, "run-as member"); e != nil {
				return e
			}
		}
		if e := unique(x.LocalMembers, "local member"); e != nil {
			return e
		}
		for _, c := range x.Commands {
			if e := unique(c.Tags, "tag"); e != nil {
				return e
			}
		}
	}
	return nil
}
func validate(a Artifact) error {
	env := a.envelope()
	if env.SchemaVersion != SchemaVersion || !oneOf(env.ContractRevision, ContractRevision, USBContractRevision, StorageContractRevision, AccountsContractRevision, ServicesContractRevision, ForwardingContractRevision, LegacyContractRevision) {
		return fmt.Errorf("unsupported schema_version/contract_revision; require %s / %s", SchemaVersion, ContractRevision)
	}
	if env.TargetID == "" {
		return fmt.Errorf("empty target_id")
	}
	if env.ContractRevision == LegacyContractRevision {
		switch x := a.(type) {
		case *Snapshot:
			for _, f := range x.Findings {
				if f.Kind == Forwarding {
					return fmt.Errorf("forwarding requires netfilter contract revision")
				}
			}
			for _, e := range x.Evidence {
				if e.SourceKind == "command_output" {
					return fmt.Errorf("command evidence requires netfilter contract revision")
				}
			}
		case *Assessments:
			for _, e := range x.Entries {
				if e.Selector.Kind == Forwarding {
					return fmt.Errorf("forwarding requires netfilter contract revision")
				}
			}
		}
	}
	if !oneOf(env.ContractRevision, ContractRevision, USBContractRevision, StorageContractRevision, AccountsContractRevision) {
		switch value := a.(type) {
		case *Snapshot:
			for _, f := range value.Findings {
				if f.Kind == Account {
					return fmt.Errorf("accounts require accounts contract")
				}
			}
			for _, c := range value.Coverage {
				if c.Collector == Accounts {
					return fmt.Errorf("account coverage requires accounts contract")
				}
			}
		case *Assessments:
			for _, entry := range value.Entries {
				if entry.Selector.Kind == Account {
					return fmt.Errorf("account judgments require accounts contract")
				}
			}
		case *Review:
			if _, ok := value.CoverageStatuses[Accounts]; ok {
				return fmt.Errorf("account review requires accounts contract")
			}
		}
	}
	if !oneOf(env.ContractRevision, ContractRevision, USBContractRevision, StorageContractRevision) {
		switch v := a.(type) {
		case *Snapshot:
			for _, f := range v.Findings {
				if f.Kind == ExternalStorage {
					return fmt.Errorf("storage requires storage contract")
				}
			}
			for _, c := range v.Coverage {
				if c.Collector == Storage {
					return fmt.Errorf("storage coverage requires storage contract")
				}
			}
		case *Assessments:
			for _, entry := range v.Entries {
				if entry.Selector.Kind == ExternalStorage {
					return fmt.Errorf("storage judgments require storage contract")
				}
			}
		case *Review:
			if _, ok := v.CoverageStatuses[Storage]; ok {
				return fmt.Errorf("storage review requires storage contract")
			}
		}
	}
	if !oneOf(env.ContractRevision, ContractRevision, USBContractRevision) {
		switch v := a.(type) {
		case *Snapshot:
			for _, f := range v.Findings {
				if f.Kind == USBDevice {
					return fmt.Errorf("USB requires USB contract")
				}
			}
			for _, c := range v.Coverage {
				if c.Collector == USB {
					return fmt.Errorf("USB coverage requires USB contract")
				}
			}
		case *Assessments:
			for _, e := range v.Entries {
				if e.Selector.Kind == USBDevice {
					return fmt.Errorf("USB judgment requires USB contract")
				}
			}
		case *Review:
			if _, ok := v.CoverageStatuses[USB]; ok {
				return fmt.Errorf("USB review requires USB contract")
			}
		}
	}

	switch x := a.(type) {
	case *Snapshot:
		if env.Kind != "snapshot" {
			return fmt.Errorf("kind mismatch")
		}
		return validateSnapshot(x)
	case *Assessments:
		if env.Kind != "assessments" {
			return fmt.Errorf("kind mismatch")
		}
		return validateAssessments(x)
	case *Review:
		if env.Kind != "review" {
			return fmt.Errorf("kind mismatch")
		}
		return validateReview(x)
	}
	return fmt.Errorf("unsupported artifact type")
}
func validateSnapshot(s *Snapshot) error {
	if s.SnapshotID == "" {
		return fmt.Errorf("empty snapshot_id")
	}
	if e := timestamp(s.CollectedAt); e != nil {
		return e
	}
	if e := timestamp(s.FinishedAt); e != nil {
		return e
	}
	if s.FinishedAt < s.CollectedAt {
		return fmt.Errorf("finished_at precedes collected_at")
	}
	coverage := map[Collector]bool{}
	for _, c := range s.Coverage {
		if !oneOf(c.Collector, Network, Sudo, Accounts, Storage, USB) || coverage[c.Collector] {
			return fmt.Errorf("invalid/duplicate coverage collector %q", c.Collector)
		}
		coverage[c.Collector] = true
		if !oneOf(c.Status, Complete, Partial, Failed, NotCollected) {
			return fmt.Errorf("invalid coverage status %q", c.Status)
		}
		for _, a := range [][]string{c.AttemptedSources, c.DiagnosticCodes, c.Limitations} {
			if e := ids(a, "coverage set member"); e != nil {
				return e
			}
		}
	}
	evidence := map[string]bool{}
	for _, e := range s.Evidence {
		if e.EvidenceID == "" || evidence[e.EvidenceID] {
			return fmt.Errorf("empty/duplicate evidence_id %q", e.EvidenceID)
		}
		evidence[e.EvidenceID] = true
		if err := timestamp(e.CollectedAt); err != nil {
			return err
		}
		switch v := e.Values.(type) {
		case ProcessValues:
			if e.SourceKind != "proc" {
				return fmt.Errorf("evidence source/values mismatch")
			}
			if err := networkFacts(v.Observed); err != nil {
				return err
			}
			if err := payloadSets(v.Observed); err != nil {
				return err
			}
			if v.Service != nil {
				if !oneOf(s.ContractRevision, ContractRevision, USBContractRevision, StorageContractRevision, AccountsContractRevision, ServicesContractRevision) || v.PID == nil {
					return fmt.Errorf("service evidence requires services revision and PID")
				}
				if (v.Service.EffectiveGID != nil || v.Service.SupplementaryGroups != nil) && !oneOf(s.ContractRevision, ContractRevision, USBContractRevision, StorageContractRevision, AccountsContractRevision) {
					return fmt.Errorf("runtime groups require accounts contract")
				}
				if v.Service.SupplementaryGroups != nil {
					if err := unique(*v.Service.SupplementaryGroups, "runtime group"); err != nil {
						return err
					}
				}
				if err := ids(v.Service.Diagnostics, "service diagnostic"); err != nil {
					return err
				}
				if v.Service.CgroupPath != "" && !strings.HasPrefix(v.Service.CgroupPath, "/") {
					return fmt.Errorf("service cgroup path must be absolute")
				}
				if v.Service.UnitCandidate != "" && (!strings.HasSuffix(v.Service.UnitCandidate, ".service") || strings.Contains(v.Service.UnitCandidate, "/")) {
					return fmt.Errorf("invalid service unit candidate")
				}
			}
			if v.PID != nil && *v.PID <= 0 {
				return fmt.Errorf("pid must be positive")
			}
		case AccountValues:
			if e.SourceKind != "local_identity" || !oneOf(s.ContractRevision, ContractRevision, USBContractRevision, StorageContractRevision, AccountsContractRevision) {
				return fmt.Errorf("account directory requires accounts contract")
			}
			names := map[string]bool{}
			for _, g := range v.Groups {
				if g.Name == "" || names[g.Name] {
					return fmt.Errorf("invalid/duplicate group")
				}
				names[g.Name] = true
				if err := ids(g.Members, "group member"); err != nil {
					return err
				}
			}
		case USBValues:
			if e.SourceKind != "usb_metadata" || !oneOf(s.ContractRevision, ContractRevision, USBContractRevision) {
				return fmt.Errorf("USB evidence requires USB contract")
			}
			if err := validateUSBInputEvidence(s.ContractRevision, v); err != nil {
				return err
			}
			if err := validUSBNodes(v.Nodes); err != nil {
				return err
			}
		case FileValues:
			if v.USB != nil && !oneOf(s.ContractRevision, ContractRevision, USBContractRevision) {
				return fmt.Errorf("USB link requires USB contract")
			}
			if e.SourceKind != "local_file" && e.SourceKind != "command_output" {
				return fmt.Errorf("evidence source/values mismatch")
			}
		default:
			return fmt.Errorf("invalid evidence source_kind")
		}
	}
	seen := map[string]bool{}
	for _, f := range s.Findings {
		if seen[f.FindingID] {
			return fmt.Errorf("duplicate finding_id %q", f.FindingID)
		}
		seen[f.FindingID] = true
		if !coverage[f.Collector] {
			return fmt.Errorf("finding has no coverage record")
		}
		if len(f.EvidenceIDs) == 0 {
			return fmt.Errorf("finding requires evidence_ids")
		}
		if e := ids(f.EvidenceIDs, "evidence reference"); e != nil {
			return e
		}
		for _, id := range f.EvidenceIDs {
			if !evidence[id] {
				return fmt.Errorf("dangling evidence reference %q", id)
			}
		}
		if facts, ok := f.Facts.(USBFacts); ok && facts.Input != nil && s.ContractRevision != ContractRevision {
			return fmt.Errorf("USB input requires input contract")
		}
		if e := payloadSets(f.Facts); e != nil {
			return e
		}
		id, fp, e := Hashes(s.TargetID, f.Collector, f.Kind, f.Identity, f.Facts)
		if e != nil {
			return e
		}
		if id != f.FindingID || fp != f.FactFingerprint {
			return fmt.Errorf("finding_id/fact_fingerprint mismatch for %q", f.FindingID)
		}
	}
	return validateUSBContext(s)
}
func validateExpected(k Kind, f ExpectedFacts) error {
	// The decoder already restricts fields to the selected kind. This also checks Go-created values.
	data, e := Canonical(f)
	if e != nil {
		return e
	}
	raw, e := parseJSON(data)
	if e != nil {
		return e
	}
	dst := reflect.New(reflect.TypeFor[ExpectedFacts]()).Elem()
	if e = assign(dst, raw, map[string]any{"kind": string(k)}, "expectation.facts", false); e != nil {
		return e
	}
	if f.Owners != nil {
		if e := unique(*f.Owners, "expected owner"); e != nil {
			return e
		}
		complete := false
		if f.AttributionComplete != nil {
			complete = *f.AttributionComplete
		}
		if e := networkFacts(NetworkFacts{*f.Owners, complete}); e != nil {
			return e
		}
	}
	if f.Principal != nil && (!oneOf(f.Principal.Type, "user", "group") || f.Principal.Name == "") {
		return fmt.Errorf("invalid expected principal")
	}
	for _, a := range []*[]string{f.RunAsUsers, f.RunAsGroups} {
		if a != nil {
			if e := unique(*a, "expected run-as"); e != nil {
				return e
			}
		}
	}
	if f.LocalMembers != nil {
		if e := unique(*f.LocalMembers, "expected member"); e != nil {
			return e
		}
		for _, m := range *f.LocalMembers {
			if m.Name == "" || !oneOf(m.Membership, "primary", "supplementary") {
				return fmt.Errorf("invalid expected member")
			}
		}
	}
	if f.Commands != nil {
		for _, c := range *f.Commands {
			if c.Path == "" {
				return fmt.Errorf("empty expected command")
			}
			if e := unique(c.Tags, "expected tag"); e != nil {
				return e
			}
			for _, t := range c.Tags {
				if !oneOf(t, "PASSWD", "NOPASSWD") {
					return fmt.Errorf("invalid expected tag")
				}
			}
		}
	}
	return nil
}
func validateAssessments(a *Assessments) error {
	seen := map[string]bool{}
	for _, x := range a.Entries {
		if x.AssessmentID == "" || seen[x.AssessmentID] {
			return fmt.Errorf("empty/duplicate assessment_id %q", x.AssessmentID)
		}
		seen[x.AssessmentID] = true
		if e := validatePayload(x.Selector.Collector, x.Selector.Kind, x.Selector.Identity, nil); e != nil {
			return e
		}
		if !oneOf(x.Decision, Unreviewed, ActionRequired, Accepted) {
			return fmt.Errorf("invalid decision %q", x.Decision)
		}
		if x.Decision != Unreviewed && (strings.TrimSpace(x.Actor) == "" || strings.TrimSpace(x.Rationale) == "" || x.AssessedAt == "") {
			return fmt.Errorf("decision requires actor, rationale and assessed_at")
		}
		if x.AssessedAt != "" {
			if e := timestamp(x.AssessedAt); e != nil {
				return e
			}
		}
		for _, ref := range x.EvidenceRefs {
			if ref.Provenance != "human_reported" || ref.Reference == "" {
				return fmt.Errorf("invalid evidence reference provenance/reference")
			}
		}
		if e := unique(x.EvidenceRefs, "human evidence reference"); e != nil {
			return e
		}
		switch b := x.Basis.(type) {
		case ReviewedBasis:
			if x.Origin != ReviewedSnapshot || b.SnapshotID == "" || !validHash(b.FactFingerprint) {
				return fmt.Errorf("invalid reviewed_snapshot basis")
			}
		case AdvanceBasis:
			if x.Selector.Kind == Account || x.Selector.Kind == ExternalStorage || x.Selector.Kind == USBDevice {
				return fmt.Errorf("account judgments require observed facts")
			}
			if x.Origin != Advance || b.Expectation.NormalizationVersion != FactsVersion {
				return fmt.Errorf("invalid advance origin/normalization_version")
			}
			if e := validateExpected(x.Selector.Kind, b.Expectation.Facts); e != nil {
				return e
			}
		default:
			return fmt.Errorf("invalid basis")
		}
	}
	return nil
}
func validateReview(r *Review) error {
	if r.SnapshotID == "" || (r.PreviousSnapshotID != nil && (*r.PreviousSnapshotID == "" || *r.PreviousSnapshotID == r.SnapshotID)) {
		return fmt.Errorf("invalid review snapshot IDs")
	}
	if e := ids(r.AssessmentIDs, "assessment_id"); e != nil {
		return e
	}
	if e := ids(r.UnmatchedAssessmentIDs, "unmatched assessment_id"); e != nil {
		return e
	}
	assess := map[string]bool{}
	for _, id := range r.AssessmentIDs {
		assess[id] = true
	}
	unmatched := map[string]bool{}
	for _, id := range r.UnmatchedAssessmentIDs {
		if !assess[id] {
			return fmt.Errorf("dangling unmatched assessment_id %q", id)
		}
		unmatched[id] = true
	}
	for c, s := range r.CoverageStatuses {
		if !oneOf(c, Network, Sudo, Accounts, Storage, USB) || !oneOf(s, Complete, Partial, Failed, NotCollected) {
			return fmt.Errorf("invalid coverage_statuses")
		}
	}
	summary := Summary{UnmatchedAssessments: len(unmatched)}
	seen := map[string]bool{}
	for _, x := range r.Items {
		if !validHash(x.FindingID) || seen[x.FindingID] {
			return fmt.Errorf("invalid/duplicate review finding_id")
		}
		seen[x.FindingID] = true
		if !oneOf(x.Presence, Current, Historical) || !oneOf(x.Change, New, Unchanged, Changed, NotObserved, Unknown, NotCompared) || !oneOf(x.Applicability, None, Applicable, NeedsReview, Conflict) || !oneOf(x.CurrentDecision, Unreviewed, ActionRequired, Accepted) {
			return fmt.Errorf("invalid review enum")
		}
		if e := ids(x.AssessmentIDs, "item assessment_id"); e != nil {
			return e
		}
		for _, id := range x.AssessmentIDs {
			if !assess[id] || unmatched[id] {
				return fmt.Errorf("dangling/conflicting item assessment_id %q", id)
			}
		}
		if e := ids(x.ReasonCodes, "reason code"); e != nil {
			return e
		}
		for _, code := range x.ReasonCodes {
			if !oneOf(code, "multiple_assessments", "coverage_incomplete", "facts_changed", "expectation_mismatch_or_incomplete", "not_observed") {
				return fmt.Errorf("invalid reason code %q", code)
			}
		}
		if x.Presence == Historical {
			if r.PreviousSnapshotID == nil || !oneOf(x.Change, NotObserved, Unknown) || x.Applicability != NeedsReview || x.CurrentDecision != Unreviewed || len(x.AssessmentIDs) != 0 {
				return fmt.Errorf("invalid historical item")
			}
			summary.Historical++
			continue
		}
		summary.Current++
		if x.Change == NotObserved {
			return fmt.Errorf("current item cannot be not_observed")
		}
		if r.PreviousSnapshotID == nil && x.Change != NotCompared {
			return fmt.Errorf("change requires previous snapshot")
		}
		if x.Applicability != Applicable && x.CurrentDecision != Unreviewed {
			return fmt.Errorf("decision requires applicable assessment")
		}
		switch x.Applicability {
		case None:
			if len(x.AssessmentIDs) != 0 {
				return fmt.Errorf("none requires no assessments")
			}
		case Applicable, NeedsReview:
			if len(x.AssessmentIDs) != 1 {
				return fmt.Errorf("applicability requires one assessment")
			}
		case Conflict:
			if len(x.AssessmentIDs) < 2 {
				return fmt.Errorf("conflict requires multiple assessments")
			}
		}
		switch x.CurrentDecision {
		case Accepted:
			summary.Accepted++
		case ActionRequired:
			summary.ActionRequired++
		case Unreviewed:
			summary.Unreviewed++
		}
		switch x.Applicability {
		case NeedsReview:
			summary.NeedsReview++
		case Conflict:
			summary.Conflict++
		}
	}
	if summary != r.Summary {
		return fmt.Errorf("review summary does not match items")
	}
	return nil
}

// ValidateInputs checks relationships without performing T04 matching policy.
// A supplied review requires all named immutable inputs; unrelated historical bases may be absent.
func ValidateInputs(current, previous *Snapshot, assessments *Assessments, review *Review) error {
	if current == nil {
		return fmt.Errorf("current snapshot required")
	}
	all := []Artifact{current}
	if previous != nil {
		all = append(all, previous)
	}
	if assessments != nil {
		all = append(all, assessments)
	}
	if review != nil {
		all = append(all, review)
	}
	for _, a := range all {
		if e := validate(a); e != nil {
			return e
		}
		if a.envelope().TargetID != current.TargetID {
			return fmt.Errorf("target_id mismatch")
		}
	}
	if previous != nil && previous.SnapshotID == current.SnapshotID {
		return fmt.Errorf("snapshot IDs must identify distinct runs")
	}
	if assessments != nil {
		for _, a := range assessments.Entries {
			b, ok := a.Basis.(ReviewedBasis)
			if !ok {
				continue
			}
			for _, s := range []*Snapshot{current, previous} {
				if s == nil || s.SnapshotID != b.SnapshotID {
					continue
				}
				id, err := digestIdentity(s.TargetID, a.Selector)
				if err != nil {
					return err
				}
				found := false
				for _, f := range s.Findings {
					if f.FindingID == id {
						found = true
						if f.FactFingerprint != b.FactFingerprint {
							return fmt.Errorf("assessment basis fingerprint mismatch")
						}
					}
				}
				if !found {
					return fmt.Errorf("assessment basis finding missing")
				}
			}
		}
	}
	if review == nil {
		return nil
	}
	if review.SnapshotID != current.SnapshotID {
		return fmt.Errorf("review snapshot reference mismatch")
	}
	if (review.PreviousSnapshotID == nil) != (previous == nil) {
		return fmt.Errorf("review previous snapshot reference mismatch")
	}
	if previous != nil && *review.PreviousSnapshotID != previous.SnapshotID {
		return fmt.Errorf("review previous snapshot reference mismatch")
	}
	coverage := map[Collector]Status{}
	for _, c := range current.Coverage {
		coverage[c.Collector] = c.Status
	}
	if !reflect.DeepEqual(coverage, review.CoverageStatuses) {
		return fmt.Errorf("review coverage mismatch")
	}
	supplied := map[string]bool{}
	if assessments != nil {
		for _, a := range assessments.Entries {
			supplied[a.AssessmentID] = true
		}
	}
	if len(supplied) != len(review.AssessmentIDs) {
		return fmt.Errorf("review assessment references mismatch")
	}
	for _, id := range review.AssessmentIDs {
		if !supplied[id] {
			return fmt.Errorf("dangling review assessment reference")
		}
	}
	currentIDs := map[string]bool{}
	previousIDs := map[string]bool{}
	for _, f := range current.Findings {
		currentIDs[f.FindingID] = true
	}
	if previous != nil {
		for _, f := range previous.Findings {
			previousIDs[f.FindingID] = true
		}
	}
	seenCurrent := 0
	for _, item := range review.Items {
		if item.Presence == Current {
			if !currentIDs[item.FindingID] {
				return fmt.Errorf("dangling current finding reference")
			}
			seenCurrent++
		} else if !previousIDs[item.FindingID] || currentIDs[item.FindingID] {
			return fmt.Errorf("dangling historical finding reference")
		}
	}
	if seenCurrent != len(currentIDs) {
		return fmt.Errorf("review omits current findings")
	}
	return nil
}
func digestIdentity(target string, s Selector) (string, error) {
	id, e := NormalizeIdentity(s.Kind, s.Identity)
	if e != nil {
		return "", e
	}
	return digest([]any{IdentityVersion, target, s.Collector, s.Kind, id})
}
