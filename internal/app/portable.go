package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pirakansa/avera/internal/artifact"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

// MachineBinding identifies the OS installation without storing its raw machine ID.
func MachineBinding() (string, error) {
	data, err := artifact.ReadBytes("/etc/machine-id", 128)
	if err != nil {
		return "", fmt.Errorf("cannot read local machine identity")
	}
	id := strings.TrimSpace(string(data))
	b, err := hex.DecodeString(id)
	if err != nil || len(b) != 16 {
		return "", fmt.Errorf("invalid local machine identity")
	}
	sum := sha256.Sum256([]byte("avera-machine-v1:" + id))
	return fmt.Sprintf("sha256:%x", sum), nil
}

// CollectReport inherits target/context from a single report, then publishes a new one.
func (svc Service) CollectReport(ctx context.Context, o Options) (int, error) {
	if err := svc.requireRoot(); err != nil {
		return 1, err
	}
	if o.Snapshot != "" || o.Previous != "" || o.Assessments != "" {
		return 1, fmt.Errorf("single-report collection uses --input; use --legacy-output for split inputs")
	}
	if err := destination(o.Output); err != nil {
		return 1, err
	}
	var prior *reportfile.File
	var p *i.Snapshot
	var a, h *i.Assessments
	if o.Input != "" {
		var err error
		prior, err = reportfile.Read(o.Input)
		if err != nil {
			return 1, err
		}
		if o.TargetID != "" && o.TargetID != prior.Current.TargetID {
			return 1, fmt.Errorf("target mismatch; use the report belonging to this target")
		}
		o.TargetID = prior.Current.TargetID
		p = prior.Current
		a = prior.Assessments
		h = prior.History
		if prior.Binding != "" && o.AdoptTarget {
			return 1, fmt.Errorf("--adopt-target is only valid for an unbound imported report")
		}
		if prior.Binding == "" && !o.AdoptTarget {
			return 1, fmt.Errorf("unbound imported report; verify its target and pass --adopt-target once")
		}
	} else {
		if o.AdoptTarget {
			return 1, fmt.Errorf("--adopt-target requires an unbound --input report")
		}
		if o.TargetID == "" {
			o.TargetID = "target-" + rand.Text()
		}
	}
	if err := ctx.Err(); err != nil {
		return 1, err
	}
	identity := svc.HostIdentity
	if identity == nil {
		identity = MachineBinding
	}
	binding, err := identity()
	if err != nil {
		return 1, fmt.Errorf("cannot identify target installation; check local machine-id")
	}
	if prior != nil && prior.Binding != "" && prior.Binding != binding {
		return 1, fmt.Errorf("report belongs to a different target installation; start a new report without --input")
	}
	s, _, code, err := svc.observe(ctx, o, p, a)
	if err != nil {
		return 1, err
	}
	f, err := reportfile.New(s, p, a, h, binding)
	if err != nil {
		return 1, fmt.Errorf("cannot construct report; check context and report size limits")
	}
	if err = f.Write(ctx, o.Output); err != nil {
		return 1, fmt.Errorf("cannot publish report; input remains unchanged, choose a new output path")
	}
	return code, nil
}

// ImportReport converts explicit legacy files for the single-file browser workflow.
func (svc Service) ImportReport(ctx context.Context, o Options) error {
	if o.Input != "" || o.Snapshot == "" {
		return fmt.Errorf("import requires --snapshot and optional --previous/--assessments")
	}
	if err := destination(o.Output); err != nil {
		return err
	}
	v, err := load(o.Snapshot, "snapshot", "--snapshot")
	if err != nil {
		return err
	}
	p, a, err := inputs(o)
	if err != nil {
		return err
	}
	f, err := reportfile.New(v.(*i.Snapshot), p, a, nil, "")
	if err != nil {
		return fmt.Errorf("incompatible import context")
	}
	return f.Write(ctx, o.Output)
}
