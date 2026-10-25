package app

import (
	"context"
	"crypto/rand"
	"fmt"

	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

// CollectReport inherits target/context from a single report, then publishes a new one.
func (svc Service) CollectReport(ctx context.Context, o Options) (int, error) {
	if err := svc.requireRoot(); err != nil {
		return 1, err
	}
	if o.Snapshot != "" || o.Previous != "" || o.Assessments != "" {
		return 1, fmt.Errorf("single-report collection uses --input; use --legacy-output for split inputs")
	}
	if err := fileDestination(o.Output); err != nil {
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
			return 1, fmt.Errorf("logical group label differs; omit --target-id to reuse the input group")
		}
		o.TargetID = prior.Current.TargetID
		p = prior.Current
		a = prior.Assessments
		h = prior.History

	} else {
		if o.AdoptTarget {
			return 1, fmt.Errorf("--adopt-target requires --input (the flag is no longer needed)")
		}
		if o.TargetID == "" {
			o.TargetID = "target-" + rand.Text()
		}
	}
	if err := ctx.Err(); err != nil {
		return 1, err
	}
	s, _, code, err := svc.observe(ctx, o, p, a)
	if err != nil {
		return 1, err
	}
	f, err := reportfile.New(s, p, a, h, "")
	if err != nil {
		return 1, fmt.Errorf("cannot construct report; check context and report size limits")
	}
	if prior != nil {
		f.Policies = prior.Policies
		f.Routes = prior.Routes
		f.RouteHistory = prior.RouteHistory
	}
	if err = f.Write(ctx, o.Output); err != nil {
		return 1, fmt.Errorf("cannot publish report to %q; input remains unchanged: %w", o.Output, err)
	}
	return code, nil
}

// ImportReport converts explicit legacy files for the single-file browser workflow.
func (svc Service) ImportReport(ctx context.Context, o Options) error {
	if o.Input != "" || o.Snapshot == "" {
		return fmt.Errorf("import requires --snapshot and optional --previous/--assessments")
	}
	if err := fileDestination(o.Output); err != nil {
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
