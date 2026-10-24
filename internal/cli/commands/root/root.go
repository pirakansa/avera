// Package root wires the experimental commands to application use cases.
package root

import (
	"fmt"

	"github.com/pirakansa/avera/internal/app"
	"github.com/pirakansa/avera/internal/cli/shared"
	"github.com/spf13/cobra"
)

func New(svc app.Service) *cobra.Command {
	root := &cobra.Command{Use: "avera", Short: "Collect facts, review explanations, and render offline reports", SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	for _, name := range []string{"collect", "review", "report"} {
		var o app.Options
		cmd := &cobra.Command{Use: name, Args: cobra.NoArgs}
		cmd.Flags().StringVar(&o.Output, "output", "", "New output path (parent must exist)")
		cmd.Flags().StringVar(&o.Previous, "previous", "", "Previous snapshot JSON for comparison")
		cmd.Flags().StringVar(&o.Assessments, "assessments", "", "Human assessment JSON, including advance context")
		if name == "collect" {
			cmd.Short = "Collect the current network namespace as root"
			cmd.Flags().StringVar(&o.TargetID, "target-id", "", "Explicit logical target ID")
			cmd.RunE = func(cmd *cobra.Command, _ []string) error {
				code, err := svc.Collect(cmd.Context(), o)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Saved complete artifact set to %q; review assessments.json, then run avera report with the same prior context.\n", o.Output)
				if code == 2 {
					fmt.Fprintln(cmd.ErrOrStderr(), "Collection has incomplete sources; inspect snapshot.json coverage and diagnostics.")
					return shared.ExitError{Code: 2}
				}
				return nil
			}
		} else {
			renderReport := name == "report"
			cmd.Short = "Recompute review from explicit saved files without host access"
			if renderReport {
				cmd.Short = "Render guidance and a Markdown or HTML report from explicit saved files"
				cmd.Flags().StringVar(&o.Format, "format", "markdown", "Output format: markdown or html")
				cmd.Flags().StringVar(&o.Language, "lang", "en", "Report language: en or ja")
			}
			cmd.Flags().StringVar(&o.Snapshot, "snapshot", "", "Current snapshot JSON")
			cmd.RunE = func(cmd *cobra.Command, _ []string) error {
				if err := svc.Offline(cmd.Context(), o, renderReport); err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Saved output to %q; keep the same --previous and --assessments context when regenerating.\n", o.Output)
				return nil
			}
		}
		root.AddCommand(cmd)
	}
	return root
}
