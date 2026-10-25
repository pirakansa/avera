package network

import (
	"context"

	"github.com/pirakansa/avera/internal/collect/forwarding"
	i "github.com/pirakansa/avera/internal/inventory"
)

// WithForwarding collects socket and translation configuration in the same
// namespace scope. Ordinary Collector remains usable for socket-only callers.
type WithForwarding struct {
	Sockets    Collector
	Forwarding forwarding.Collector
}

func (c WithForwarding) Collect(ctx context.Context, target string) (i.CollectorResult, error) {
	r, err := c.Sockets.Collect(ctx, target)
	if err != nil {
		return r, err
	}
	c.Forwarding.ScopeID = r.Coverage.Scope
	f, err := c.Forwarding.Collect(ctx, target)
	if err != nil {
		return r, err
	}
	r.Findings = append(r.Findings, f.Findings...)
	r.Evidence = append(r.Evidence, f.Evidence...)
	r.Coverage.CollectorVersion += "+" + f.Coverage.CollectorVersion
	r.Coverage.AttemptedSources = append(r.Coverage.AttemptedSources, f.Coverage.AttemptedSources...)
	r.Coverage.DiagnosticCodes = append(r.Coverage.DiagnosticCodes, f.Coverage.DiagnosticCodes...)
	r.Coverage.Limitations = append(r.Coverage.Limitations, f.Coverage.Limitations...)
	bothFailed := r.Coverage.Status == i.Failed && f.Coverage.Status == i.Failed
	if r.Coverage.Status != i.Complete || f.Coverage.Status != i.Complete {
		r.Coverage.Status = i.Partial
	}
	if bothFailed {
		r.Coverage.Status = i.Failed
	}
	return r, nil
}
