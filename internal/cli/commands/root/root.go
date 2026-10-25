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
		var legacyOutput, singleReport bool
		cmd := &cobra.Command{Use: name, Args: cobra.NoArgs}
		cmd.Flags().StringVar(&o.Output, "output", "", "New output path (parent must exist)")
		cmd.Flags().StringVar(&o.Previous, "previous", "", "Previous snapshot JSON for comparison")
		cmd.Flags().StringVar(&o.Assessments, "assessments", "", "Human assessment JSON, including advance context")
		if name == "collect" {
			cmd.Short = "Collect facts into one portable report JSON as root"
			cmd.Flags().StringVar(&o.Input, "input", "", "Previous portable report JSON (optional on first collection)")
			cmd.Flags().BoolVar(&o.AdoptTarget, "adopt-target", false, "Bind an imported unbound report to this installation")
			cmd.Flags().BoolVar(&legacyOutput, "legacy-output", false, "Use the older split-artifact run directory")
			cmd.Flags().StringVar(&o.TargetID, "target-id", "", "Explicit logical target ID")
			cmd.RunE = func(cmd *cobra.Command, _ []string) error {
				var code int
				var err error
				if legacyOutput {
					if o.Input != "" || o.AdoptTarget {
						return fmt.Errorf("--legacy-output cannot be combined with --input or --adopt-target")
					}
					code, err = svc.Collect(cmd.Context(), o)
				} else {
					code, err = svc.CollectReport(cmd.Context(), o)
				}
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Saved results to %q; open the report with avera serve --input REPORT --output-dir SAVES.\n", o.Output)
				if code == 2 {
					fmt.Fprintln(cmd.ErrOrStderr(), "Collection has incomplete sources; inspect coverage and diagnostics in the saved output.")
					return shared.ExitError{Code: 2}
				}
				return nil
			}
		} else {
			renderReport := name == "report"
			cmd.Short = "Recompute review from explicit saved files without host access"
			if renderReport {
				cmd.Short = "Render guidance and a Markdown or HTML report from explicit saved files"
				cmd.Flags().StringVar(&o.Input, "input", "", "Portable report JSON; replaces split context inputs")
				cmd.Flags().StringVar(&o.Format, "format", "markdown", "Output format: markdown or html")
				cmd.Flags().StringVar(&o.Language, "lang", "en", "Report language: en or ja")
			}
			if !renderReport {
				cmd.Flags().BoolVar(&singleReport, "single-report", false, "Import legacy inputs into one portable report JSON")
			}
			cmd.Flags().StringVar(&o.Snapshot, "snapshot", "", "Current snapshot JSON")
			cmd.RunE = func(cmd *cobra.Command, _ []string) error {
				var err error
				if singleReport {
					err = svc.ImportReport(cmd.Context(), o)
				} else {
					err = svc.Offline(cmd.Context(), o, renderReport)
				}
				if err != nil {
					return err
				}
				if singleReport {
					fmt.Fprintf(cmd.ErrOrStderr(), "Saved portable report to %q; open it with avera serve --input REPORT --output-dir SAVES.\n", o.Output)
				} else if o.Input != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Saved output to %q from the portable report.\n", o.Output)
				} else {
					fmt.Fprintf(cmd.ErrOrStderr(), "Saved output to %q; keep the same --previous and --assessments context when regenerating.\n", o.Output)
				}
				return nil
			}
		}
		root.AddCommand(cmd)
	}
	var input, outputDir, listen, language string
	serve := &cobra.Command{Use: "serve", Short: "Edit judgments in a local browser and export JSON/HTML", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if input == "" || outputDir == "" {
			return fmt.Errorf("--input and --output-dir are required")
		}
		return svc.ServeReport(cmd.Context(), input, outputDir, listen, language, cmd.ErrOrStderr())
	}}
	serve.Flags().StringVar(&input, "input", "", "Portable report JSON")
	serve.Flags().StringVar(&outputDir, "output-dir", "", "Existing private directory for new saved reports")
	serve.Flags().StringVar(&listen, "listen", "127.0.0.1:0", "Literal loopback IP and port (0 chooses a free port)")
	serve.Flags().StringVar(&language, "lang", "ja", "Report content language: ja or en; editor controls are Japanese")
	root.AddCommand(serve)
	return root
}
