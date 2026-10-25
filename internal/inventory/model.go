// Package inventory defines the experimental v0 artifact contract without host I/O.
package inventory

const (
	SchemaVersion          = "0.1"
	ContractRevision       = "netfilter-2026-10-25.1"
	LegacyContractRevision = "t00-2026-10-24.1"
	IdentityVersion        = "avera.identity.v1"
	FactsVersion           = "avera.facts.v1"
)

type Collector string

const (
	Network Collector = "network"
	Sudo    Collector = "sudo"
)

type Kind string

const (
	Endpoint   Kind = "network.endpoint"
	Forwarding Kind = "network.forwarding"
	Rule       Kind = "sudo.rule"
	Unparsed   Kind = "sudo.unparsed"
)

type Status string

const (
	Complete     Status = "complete"
	Partial      Status = "partial"
	Failed       Status = "failed"
	NotCollected Status = "not_collected"
)

type Decision string

const (
	Unreviewed     Decision = "unreviewed"
	ActionRequired Decision = "action_required"
	Accepted       Decision = "accepted"
)

type Origin string

const (
	ReviewedSnapshot Origin = "reviewed_snapshot"
	Advance          Origin = "advance"
)

type Change string

const (
	New         Change = "new"
	Unchanged   Change = "unchanged"
	Changed     Change = "changed"
	NotObserved Change = "not_observed"
	Unknown     Change = "unknown"
	NotCompared Change = "not_compared"
)

type Applicability string

const (
	None        Applicability = "none"
	Applicable  Applicability = "applicable"
	NeedsReview Applicability = "needs_review"
	Conflict    Applicability = "conflict"
)

type Presence string

const (
	Current    Presence = "current"
	Historical Presence = "historical"
)

type Envelope struct {
	SchemaVersion    string `json:"schema_version"`
	ContractRevision string `json:"contract_revision"`
	Kind             string `json:"kind"`
	TargetID         string `json:"target_id"`
}

// Header returns a copy of the common artifact envelope.
func (e Envelope) Header() Envelope { return e }

type Artifact interface {
	envelope() Envelope
	Header() Envelope
}
type Snapshot struct {
	Envelope
	SnapshotID  string      `json:"snapshot_id"`
	CollectedAt string      `json:"collected_at"`
	FinishedAt  string      `json:"finished_at"`
	ToolVersion string      `json:"tool_version"`
	Environment Environment `json:"environment"`
	Coverage    []Coverage  `json:"coverage"`
	Findings    []Finding   `json:"findings"`
	Evidence    []Evidence  `json:"evidence"`
}

func (s *Snapshot) envelope() Envelope { return s.Envelope }

type Environment struct {
	OSID           string `json:"os_id"`
	OSVersion      string `json:"os_version"`
	Architecture   string `json:"architecture"`
	KernelRelease  string `json:"kernel_release"`
	ExecutionScope string `json:"execution_scope"`
	EffectiveUID   uint32 `json:"effective_uid"`
}
type Coverage struct {
	Collector        Collector `json:"collector"`
	CollectorVersion string    `json:"collector_version"`
	Scope            string    `json:"scope"`
	Status           Status    `json:"status"`
	AttemptedSources []string  `json:"attempted_sources"`
	DiagnosticCodes  []string  `json:"diagnostic_codes"`
	Limitations      []string  `json:"limitations"`
}
type Identity interface{ identity() }
type Facts interface{ facts() }
type EndpointIdentity struct {
	Transport      string `json:"transport"`
	Family         string `json:"family"`
	LocalAddress   string `json:"local_address"`
	Zone           string `json:"zone"`
	ScopeID        string `json:"scope_id"`
	LocalPort      uint16 `json:"local_port"`
	SocketCategory string `json:"socket_category"`
}

func (EndpointIdentity) identity() {}

type SourceIdentity struct {
	SourcePath string `json:"source_path"`
	Line       int64  `json:"line"`
	Ordinal    int64  `json:"ordinal"`
}

func (SourceIdentity) identity() {}

type Owner struct {
	Executable string `json:"executable"`
	UID        uint32 `json:"uid"`
}
type NetworkFacts struct {
	Owners              []Owner `json:"owners"`
	AttributionComplete bool    `json:"attribution_complete"`
}

func (NetworkFacts) facts() {}

type Principal struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
type Command struct {
	Path      string   `json:"path"`
	Arguments string   `json:"arguments"`
	Tags      []string `json:"tags"`
}
type Member struct {
	Name       string `json:"name"`
	UID        uint32 `json:"uid"`
	Membership string `json:"membership"`
}
type SudoFacts struct {
	Principal              Principal `json:"principal"`
	Host                   string    `json:"host"`
	RunAsUsers             []string  `json:"run_as_users"`
	RunAsGroups            []string  `json:"run_as_groups"`
	Commands               []Command `json:"commands"`
	LocalMembers           []Member  `json:"local_members"`
	LocalContextComplete   bool      `json:"local_context_complete"`
	PolicyContext          []string  `json:"policy_context"`
	InterpretationComplete bool      `json:"interpretation_complete"`
}

func (SudoFacts) facts() {}

type UnparsedFacts struct {
	Raw            string `json:"raw"`
	DiagnosticCode string `json:"diagnostic_code"`
}

func (UnparsedFacts) facts() {}

type Finding struct {
	FindingID       string    `json:"finding_id"`
	Collector       Collector `json:"collector"`
	Kind            Kind      `json:"kind"`
	Identity        Identity  `json:"identity"`
	Facts           Facts     `json:"facts"`
	FactFingerprint string    `json:"fact_fingerprint"`
	EvidenceIDs     []string  `json:"evidence_ids"`
}
type EvidenceValues interface{ evidenceValues() }
type ProcessValues struct {
	Observed NetworkFacts `json:"observed"`
	PID      *int64       `json:"pid,omitempty"`
}

func (ProcessValues) evidenceValues() {}

type FileValues struct {
	Excerpt string `json:"excerpt"`
}

func (FileValues) evidenceValues() {}

type Evidence struct {
	EvidenceID  string         `json:"evidence_id"`
	SourceKind  string         `json:"source_kind"`
	Location    string         `json:"location"`
	CollectedAt string         `json:"collected_at"`
	Method      string         `json:"method"`
	Values      EvidenceValues `json:"values"`
}
type Assessments struct {
	Envelope
	Entries []Assessment `json:"entries"`
}

func (s *Assessments) envelope() Envelope { return s.Envelope }

type Selector struct {
	Collector Collector `json:"collector"`
	Kind      Kind      `json:"kind"`
	Identity  Identity  `json:"identity"`
}
type Basis interface{ basis() }
type ReviewedBasis struct {
	SnapshotID      string `json:"snapshot_id"`
	FactFingerprint string `json:"fact_fingerprint"`
}

func (ReviewedBasis) basis() {}

type AdvanceBasis struct {
	Expectation Expectation `json:"expectation"`
}

func (AdvanceBasis) basis() {}

// ExpectedFacts uses pointers to preserve absent fields in partial expectations.
type ExpectedFacts struct {
	Owners                 *[]Owner   `json:"owners,omitempty"`
	AttributionComplete    *bool      `json:"attribution_complete,omitempty"`
	Principal              *Principal `json:"principal,omitempty"`
	Host                   *string    `json:"host,omitempty"`
	RunAsUsers             *[]string  `json:"run_as_users,omitempty"`
	RunAsGroups            *[]string  `json:"run_as_groups,omitempty"`
	Commands               *[]Command `json:"commands,omitempty"`
	LocalMembers           *[]Member  `json:"local_members,omitempty"`
	LocalContextComplete   *bool      `json:"local_context_complete,omitempty"`
	PolicyContext          *[]string  `json:"policy_context,omitempty"`
	InterpretationComplete *bool      `json:"interpretation_complete,omitempty"`
	Raw                    *string    `json:"raw,omitempty"`
	DiagnosticCode         *string    `json:"diagnostic_code,omitempty"`
}
type Expectation struct {
	NormalizationVersion string        `json:"normalization_version"`
	Facts                ExpectedFacts `json:"facts"`
}
type EvidenceReference struct {
	Reference  string `json:"reference"`
	Provenance string `json:"provenance"`
}
type Assessment struct {
	AssessmentID string              `json:"assessment_id"`
	Origin       Origin              `json:"origin"`
	Selector     Selector            `json:"selector"`
	Basis        Basis               `json:"basis"`
	Purpose      string              `json:"purpose"`
	Exposure     string              `json:"exposure"`
	Safeguards   string              `json:"safeguards"`
	Rationale    string              `json:"rationale"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
	Decision     Decision            `json:"decision"`
	Actor        string              `json:"actor"`
	AssessedAt   string              `json:"assessed_at"`
}
type Review struct {
	Envelope
	SnapshotID             string               `json:"snapshot_id"`
	PreviousSnapshotID     *string              `json:"previous_snapshot_id"`
	AssessmentIDs          []string             `json:"assessment_ids"`
	CoverageStatuses       map[Collector]Status `json:"coverage_statuses"`
	Items                  []ReviewItem         `json:"items"`
	UnmatchedAssessmentIDs []string             `json:"unmatched_assessment_ids"`
	Summary                Summary              `json:"summary"`
}

func (s *Review) envelope() Envelope { return s.Envelope }

type ReviewItem struct {
	FindingID       string        `json:"finding_id"`
	Presence        Presence      `json:"presence"`
	Change          Change        `json:"change"`
	Applicability   Applicability `json:"applicability"`
	AssessmentIDs   []string      `json:"assessment_ids"`
	CurrentDecision Decision      `json:"current_decision"`
	ReasonCodes     []string      `json:"reason_codes"`
}
type Summary struct {
	Current              int `json:"current"`
	Historical           int `json:"historical"`
	Accepted             int `json:"accepted"`
	ActionRequired       int `json:"action_required"`
	Unreviewed           int `json:"unreviewed"`
	NeedsReview          int `json:"needs_review"`
	Conflict             int `json:"conflict"`
	UnmatchedAssessments int `json:"unmatched_assessments"`
}

// CollectorResult reports usable observations, including partial/failed sources in Coverage.
// A collector returns a separate error only for fatal application failures (e.g. cancellation).
type CollectorResult struct {
	Findings []Finding
	Evidence []Evidence
	Coverage Coverage
}

// ForwardingFacts records configuration text and its complete source context hash.
// It does not establish packet reachability or effective forwarding.
type ForwardingFacts struct {
	Rule          string `json:"rule"`
	ContextDigest string `json:"context_digest"`
	ScopeID       string `json:"scope_id"`
}

func (ForwardingFacts) facts() {}
