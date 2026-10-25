// Package app orchestrates explicit-file processing and privileged collection.
package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"runtime"
	"time"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/report"
	"github.com/pirakansa/avera/internal/reportfile"
)

// Collector is the registration boundary for future collection domains.
type Collector interface {
	Collect(context.Context, string) (i.CollectorResult, error)
}
type Registration struct {
	Name      i.Collector
	Collector Collector
}

// Service injects all live operations; offline operations never call them.
type Service struct {
	EffectiveUID func() int
	Now          func() time.Time
	Collectors   []Registration
	// HostIdentity is retained for compatibility and is no longer consulted.
	HostIdentity func() (string, error)
}
type Options struct {
	TargetID, Snapshot, Previous, Assessments, Output, Format, Language, Input string
	AdoptTarget                                                                bool
}

func load(path, kind, flag string) (i.Artifact, error) {
	a, err := artifact.Read(path)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot read valid artifact; check file, schema, contract revision and JSON fields", flag)
	}
	if a.Header().Kind != kind {
		return nil, fmt.Errorf("%s: expected %s artifact", flag, kind)
	}
	return a, nil
}
func inputs(o Options) (*i.Snapshot, *i.Assessments, error) {
	var p *i.Snapshot
	var a *i.Assessments
	if o.Previous != "" {
		v, e := load(o.Previous, "snapshot", "--previous")
		if e != nil {
			return nil, nil, e
		}
		p = v.(*i.Snapshot)
	}
	if o.Assessments != "" {
		v, e := load(o.Assessments, "assessments", "--assessments")
		if e != nil {
			return nil, nil, e
		}
		a = v.(*i.Assessments)
	}
	return p, a, nil
}
func destination(path string) error {
	if path == "" {
		return fmt.Errorf("--output is required; choose a new path with an existing parent")
	}
	if err := artifact.CheckDestination(path); err != nil {
		return fmt.Errorf("--output: %w", err)
	}
	return nil
}
func fileDestination(path string) error {
	if path == "" {
		return fmt.Errorf("--output is required; specify a new filename in an existing private directory")
	}
	if err := artifact.CheckFileDestination(path); err != nil {
		return fmt.Errorf("--output: %w", err)
	}
	return nil
}

func evaluate(s, p *i.Snapshot, a *i.Assessments) (*assessment.Result, error) {
	r, err := assessment.Evaluate(s, p, a)
	if err != nil {
		return nil, fmt.Errorf("incompatible inputs: check target_id, distinct snapshot IDs and assessment basis")
	}
	return r, nil
}

// Offline reads only the named artifacts and recomputes every derived decision/count.
func (svc Service) Offline(ctx context.Context, o Options, renderReport bool) error {
	if renderReport {
		if err := report.ValidateOptions(report.Options{Format: o.Format, Language: o.Language}); err != nil {
			return err
		}
	}
	if o.Input != "" {
		if !renderReport || o.Snapshot != "" || o.Previous != "" || o.Assessments != "" {
			return fmt.Errorf("--input cannot be combined with legacy context flags")
		}
		if err := fileDestination(o.Output); err != nil {
			return err
		}
		f, err := reportfile.Read(o.Input)
		if err != nil {
			return err
		}
		text, err := report.RenderFile(f, report.Options{Format: o.Format, Language: o.Language})
		if err != nil {
			return err
		}
		return artifact.WriteBytes(ctx, o.Output, []byte(text))
	}
	if o.Snapshot == "" {
		return fmt.Errorf("--snapshot is required")
	}
	if err := destination(o.Output); err != nil {
		return err
	}
	v, err := load(o.Snapshot, "snapshot", "--snapshot")
	if err != nil {
		return err
	}
	s := v.(*i.Snapshot)
	p, a, err := inputs(o)
	if err != nil {
		return err
	}
	r, err := evaluate(s, p, a)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if renderReport {
		text, err := report.RenderWithOptions(s, p, a, report.Options{Format: o.Format, Language: o.Language})
		if err != nil {
			return fmt.Errorf("cannot render validated report")
		}
		if err = artifact.WriteBytes(ctx, o.Output, []byte(text)); err != nil {
			return fmt.Errorf("cannot publish report; choose a new output path and check permissions")
		}
		return nil
	}
	return publish(ctx, o.Output, s, r)
}
func publish(ctx context.Context, path string, s *i.Snapshot, r *assessment.Result) error {
	// Preserve unmatched context in the editable file so subsequent runs cannot lose it.
	a := *r.Editable
	a.Entries = append(append([]i.Assessment{}, r.Editable.Entries...), r.Unmatched.Entries...)
	if err := artifact.WriteRun(ctx, path, map[string]i.Artifact{"snapshot.json": s, "review.json": r.Review, "assessments.json": &a}); err != nil {
		return fmt.Errorf("cannot publish complete run; choose a new output path and check permissions")
	}
	return nil
}

// Collect validates supplied context and destination before any collector is invoked.
func (svc Service) Collect(ctx context.Context, o Options) (int, error) {
	if err := svc.requireRoot(); err != nil {
		return 1, err
	}
	if o.TargetID == "" {
		return 1, fmt.Errorf("--target-id is required")
	}
	if err := destination(o.Output); err != nil {
		return 1, err
	}
	p, a, err := inputs(o)
	if err != nil {
		return 1, err
	}
	s, r, code, err := svc.observe(ctx, o, p, a)
	if err != nil {
		return 1, err
	}
	if err = publish(ctx, o.Output, s, r); err != nil {
		return 1, err
	}
	return code, nil
}
func (svc Service) requireRoot() error {
	if svc.EffectiveUID == nil || svc.EffectiveUID() != 0 {
		return fmt.Errorf("collection requires root; run explicitly as root on the selected target")
	}
	return nil
}
func (svc Service) observe(ctx context.Context, o Options, p *i.Snapshot, a *i.Assessments) (*i.Snapshot, *assessment.Result, int, error) {
	var err error
	now := svc.Now
	if now == nil {
		now = time.Now
	}
	timestamp := now().UTC().Format(time.RFC3339)
	s := &i.Snapshot{Envelope: i.Envelope{SchemaVersion: i.SchemaVersion, ContractRevision: i.ContractRevision, Kind: "snapshot", TargetID: o.TargetID}, SnapshotID: "run-" + rand.Text(), CollectedAt: timestamp, FinishedAt: timestamp, ToolVersion: "prototype-t06", Environment: i.Environment{OSID: "unknown", OSVersion: "unknown", Architecture: runtime.GOARCH, KernelRelease: "unknown", ExecutionScope: "current observable namespace; distribution and kernel release not inspected", EffectiveUID: 0}, Coverage: []i.Coverage{}, Findings: []i.Finding{}, Evidence: []i.Evidence{}}
	if _, err = evaluate(s, p, a); err != nil {
		return nil, nil, 1, err
	}
	if len(svc.Collectors) == 0 {
		return nil, nil, 1, fmt.Errorf("no collectors registered")
	}
	seen := map[i.Collector]bool{}
	for _, reg := range svc.Collectors {
		if reg.Collector == nil || (reg.Name != i.Network && reg.Name != i.Sudo && reg.Name != i.Accounts && reg.Name != i.Storage && reg.Name != i.USB) || seen[reg.Name] {
			return nil, nil, 1, fmt.Errorf("invalid collector registration")
		}
		seen[reg.Name] = true
	}
	code := 0
	for _, reg := range svc.Collectors {
		if err = ctx.Err(); err != nil {
			return nil, nil, 1, err
		}
		result, e := reg.Collector.Collect(ctx, o.TargetID)
		if e != nil {
			return nil, nil, 1, fmt.Errorf("collection interrupted or failed fatally; no complete output published")
		}
		if result.Coverage.Collector != reg.Name || result.Coverage.Status == i.NotCollected {
			return nil, nil, 1, fmt.Errorf("invalid collector result")
		}
		if result.Coverage.Status == i.Partial || result.Coverage.Status == i.Failed {
			code = 2
		}
		s.Coverage = append(s.Coverage, result.Coverage)
		s.Findings = append(s.Findings, result.Findings...)
		s.Evidence = append(s.Evidence, result.Evidence...)
	}
	for _, name := range []i.Collector{i.Network, i.Sudo} {
		if !seen[name] {
			s.Coverage = append(s.Coverage, i.Coverage{Collector: name, Status: i.NotCollected, AttemptedSources: []string{}, DiagnosticCodes: []string{"not_enabled_in_increment"}, Limitations: []string{"Not enabled in this collection increment; no absence claim."}})
		}
	}
	if err := i.EnrichUSBStorage(s); err != nil {
		return nil, nil, 1, err
	}
	s.FinishedAt = now().UTC().Format(time.RFC3339)
	r, err := evaluate(s, p, a)
	if err != nil {
		return nil, nil, 1, err
	}
	return s, r, code, nil
}
