package app

import (
	"context"
	"github.com/pirakansa/avera/internal/viewer"
	"io"
)

// ServeReport edits only explicit report data using a local browser session.
func (svc Service) ServeReport(ctx context.Context, input, outputDir, listen, language string, diagnostics io.Writer) error {
	return viewer.Serve(ctx, input, outputDir, listen, language, diagnostics)
}
