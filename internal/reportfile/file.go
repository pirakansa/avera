// Package reportfile owns the portable report container and explicit human edits.
package reportfile

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

const Version = "avera.report.v2"
const legacyVersion = "avera.report.v1"
const MaxBytes = i.MaxArtifactBytes

type File struct {
	Binding              string
	Current, Previous    *i.Snapshot
	Assessments, History *i.Assessments
	Routes, RouteHistory []Route
}
type wire struct {
	Format       string          `json:"format"`
	Binding      string          `json:"machine_binding"`
	Current      json.RawMessage `json:"current"`
	Previous     json.RawMessage `json:"previous"`
	Assessments  json.RawMessage `json:"assessments"`
	History      json.RawMessage `json:"assessment_history"`
	Routes       []Route         `json:"routes,omitempty"`
	RouteHistory []Route         `json:"route_history,omitempty"`
}

func empty(s *i.Snapshot) *i.Assessments {
	e := s.Envelope
	e.Kind = "assessments"
	return &i.Assessments{Envelope: e, Entries: []i.Assessment{}}
}

// New adds editable starters and retains unmatched context, without rebasing old judgments.
func New(s, p *i.Snapshot, a, h *i.Assessments, binding string) (*File, error) {
	r, err := assessment.Evaluate(s, p, a)
	if err != nil {
		return nil, err
	}
	active := *r.Editable
	active.Entries = append(append([]i.Assessment{}, r.Editable.Entries...), r.Unmatched.Entries...)
	if h == nil {
		h = empty(s)
	}
	f := &File{Binding: binding, Current: s, Previous: p, Assessments: &active, History: h}
	if _, err = f.Encode(); err != nil {
		return nil, err
	}
	return f, nil
}
func (f *File) Validate() error {
	if f == nil || f.Current == nil || f.Assessments == nil || f.History == nil {
		return fmt.Errorf("incomplete report")
	}
	if f.Binding != "" {
		if len(f.Binding) != 71 || !strings.HasPrefix(f.Binding, "sha256:") {
			return fmt.Errorf("invalid machine binding")
		}
		for _, c := range f.Binding[7:] {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return fmt.Errorf("invalid machine binding")
			}
		}
	}
	if _, err := assessment.Evaluate(f.Current, f.Previous, f.Assessments); err != nil {
		return err
	}
	if f.History.TargetID != f.Current.TargetID {
		return fmt.Errorf("history target mismatch")
	}
	if _, err := i.Encode(f.History); err != nil {
		return err
	}
	archived := map[string]bool{}
	for _, a := range f.History.Entries {
		archived[a.AssessmentID] = true
	}
	for _, a := range f.Assessments.Entries {
		if archived[a.AssessmentID] {
			return fmt.Errorf("active and archived assessment IDs must be distinct")
		}
	}
	pairs := map[string]bool{}
	for _, r := range f.Routes {
		key := r.EntranceID + "/" + r.InternalID
		if pairs[key] {
			return fmt.Errorf("duplicate active route")
		}
		pairs[key] = true
	}
	if err := validateRoutes(f.Routes); err != nil {
		return err
	}
	if err := validateRoutes(f.RouteHistory); err != nil {
		return err
	}
	return nil
}
func (f *File) Encode() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	w := wire{Format: Version, Binding: f.Binding, Routes: f.Routes, RouteHistory: f.RouteHistory}
	var err error
	if w.Current, err = i.Encode(f.Current); err != nil {
		return nil, err
	}
	if f.Previous != nil {
		if w.Previous, err = i.Encode(f.Previous); err != nil {
			return nil, err
		}
	} else {
		w.Previous = json.RawMessage("null")
	}
	if w.Assessments, err = i.Encode(f.Assessments); err != nil {
		return nil, err
	}
	if w.History, err = i.Encode(f.History); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err = i.ValidateJSON(data); err != nil {
		return nil, fmt.Errorf("report exceeds JSON limits or contains invalid data")
	}
	return data, nil
}
func Decode(data []byte) (*File, error) {
	if err := i.ValidateJSON(data); err != nil {
		return nil, fmt.Errorf("invalid or oversized report JSON")
	}
	var w wire
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil || (w.Format != Version && w.Format != legacyVersion) {
		return nil, fmt.Errorf("unsupported report format or fields; require %s", Version)
	}
	if w.Format == legacyVersion && (len(w.Routes) > 0 || len(w.RouteHistory) > 0) {
		return nil, fmt.Errorf("route metadata requires report v2")
	}
	f := &File{Binding: w.Binding, Routes: w.Routes, RouteHistory: w.RouteHistory}
	decode := func(data []byte, kind string) (i.Artifact, error) {
		a, err := i.Decode(data)
		if err != nil || a.Header().Kind != kind {
			return nil, fmt.Errorf("invalid nested %s", kind)
		}
		return a, nil
	}
	a, err := decode(w.Current, "snapshot")
	if err != nil {
		return nil, err
	}
	f.Current = a.(*i.Snapshot)
	if string(w.Previous) != "null" {
		a, err = decode(w.Previous, "snapshot")
		if err != nil {
			return nil, err
		}
		f.Previous = a.(*i.Snapshot)
	}
	a, err = decode(w.Assessments, "assessments")
	if err != nil {
		return nil, err
	}
	f.Assessments = a.(*i.Assessments)
	a, err = decode(w.History, "assessments")
	if err != nil {
		return nil, err
	}
	f.History = a.(*i.Assessments)
	if err = f.Validate(); err != nil {
		return nil, fmt.Errorf("incompatible report contents: check targets and assessment bases")
	}
	return f, nil
}
func Read(path string) (*File, error) {
	data, err := artifact.ReadBytes(path, MaxBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot read report; check path, permissions and size")
	}
	return Decode(data)
}
func (f *File) Write(ctx context.Context, path string) error {
	data, err := f.Encode()
	if err != nil {
		return err
	}
	return artifact.WriteBytes(ctx, path, data)
}

// Edit contains only human-owned fields. Identity, provenance and timestamp are server-owned.
type Edit struct {
	FindingID        string                `json:"finding_id"`
	Decision         i.Decision            `json:"decision"`
	Purpose          string                `json:"purpose"`
	Exposure         string                `json:"exposure"`
	Safeguards       string                `json:"safeguards"`
	Rationale        string                `json:"rationale"`
	Actor            string                `json:"actor"`
	Evidence         []i.EvidenceReference `json:"evidence"`
	RelatedIDs       []string              `json:"related_ids"`
	RouteExplanation string                `json:"route_explanation"`
}

// Apply explicitly replaces judgments for a current finding and archives them verbatim.
func (f *File) Apply(edit Edit, now time.Time) (*File, error) {
	data, err := f.Encode()
	if err != nil {
		return nil, err
	}
	out, err := Decode(data)
	if err != nil {
		return nil, err
	}
	var found *i.Finding
	for n := range out.Current.Findings {
		if out.Current.Findings[n].FindingID == edit.FindingID {
			found = &out.Current.Findings[n]
			break
		}
	}
	if found == nil || assessment.IsReference(*found) {
		return nil, fmt.Errorf("only current findings can be edited")
	}
	entries := []i.Assessment{}
	for _, a := range out.Assessments.Entries {
		id, _, err := i.Hashes(out.Current.TargetID, a.Selector.Collector, a.Selector.Kind, a.Selector.Identity, found.Facts)
		// Hashes may reject a different finding kind; it cannot match this current item.
		if err == nil && id == found.FindingID {
			if a.Actor != "" || a.Purpose != "" || a.Rationale != "" || a.Exposure != "" || a.Safeguards != "" || len(a.EvidenceRefs) > 0 || a.Decision != i.Unreviewed {
				out.History.Entries = append(out.History.Entries, a)
			}
		} else {
			entries = append(entries, a)
		}
	}
	refs := edit.Evidence
	if refs == nil {
		refs = []i.EvidenceReference{}
	}
	entries = append(entries, i.Assessment{AssessmentID: "human-" + rand.Text(), Origin: i.ReviewedSnapshot, Selector: i.Selector{Collector: found.Collector, Kind: found.Kind, Identity: found.Identity}, Basis: i.ReviewedBasis{SnapshotID: out.Current.SnapshotID, FactFingerprint: found.FactFingerprint}, Decision: edit.Decision, Purpose: edit.Purpose, Exposure: edit.Exposure, Safeguards: edit.Safeguards, Rationale: edit.Rationale, Actor: edit.Actor, AssessedAt: now.UTC().Format(time.RFC3339), EvidenceRefs: refs})
	out.Assessments.Entries = entries
	if err = out.applyRoutes(edit, *found, now); err != nil {
		return nil, err
	}
	if err = out.Validate(); err != nil {
		return nil, fmt.Errorf("invalid judgment: check decision, actor, rationale, references and field limits")
	}
	return out, nil
}
