package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pirakansa/avera/internal/app"
	"github.com/pirakansa/avera/internal/cli/shared"
	i "github.com/pirakansa/avera/internal/inventory"
)

type fakeCollector struct {
	status i.Status
	called *bool
}

func (f fakeCollector) Collect(context.Context, string) (i.CollectorResult, error) {
	*f.called = true
	return i.CollectorResult{Coverage: i.Coverage{Collector: i.Network, Status: f.status, Scope: "injected", AttemptedSources: []string{}, DiagnosticCodes: []string{"injected"}, Limitations: []string{}}, Findings: []i.Finding{}, Evidence: []i.Evidence{}}, nil
}
func execute(t *testing.T, svc app.Service, args ...string) (int, string, string) {
	t.Helper()
	cmd := New(svc)
	var out, diag bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diag)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	if err != nil && shared.Code(err) != 2 {
		diag.WriteString(err.Error())
	}
	return shared.Code(err), out.String(), diag.String()
}
func TestCommandContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		uid    int
		status i.Status
		want   int
		called bool
	}{
		{name: "help", args: []string{"--help"}},
		{name: "unsupported flag", args: []string{"collect", "--bogus"}, want: 1},
		{name: "unsupported command", args: []string{"unknown"}, want: 1},
		{name: "positional", args: []string{"review", "extra"}, want: 1},
		{name: "wrong command flag", args: []string{"report", "--target-id", "x"}, want: 1},
		{name: "missing snapshot", args: []string{"report"}, want: 1},
		{name: "missing target", args: []string{"collect"}, want: 1},
		{name: "nonroot", args: []string{"collect", "--target-id", "fictional"}, uid: 1000, want: 1},
		{name: "collect", args: []string{"collect", "--target-id", "fictional"}, status: i.Complete, called: true},
		{name: "partial", args: []string{"collect", "--target-id", "fictional"}, status: i.Partial, want: 2, called: true},
		{name: "failed", args: []string{"collect", "--target-id", "fictional"}, status: i.Failed, want: 2, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := app.Service{EffectiveUID: func() int { return tc.uid }, Collectors: []app.Registration{{Name: i.Network, Collector: fakeCollector{tc.status, &called}}}}
			args := append([]string{}, tc.args...)
			output := filepath.Join(t.TempDir(), "run")
			if args[0] == "collect" {
				args = append(args, "--legacy-output", "--output", output)
			}
			code, out, diag := execute(t, svc, args...)
			if code != tc.want || called != tc.called {
				t.Fatalf("code=%d called=%v diag=%s", code, called, diag)
			}
			if called {
				if out != "" || !strings.Contains(diag, output) {
					t.Fatalf("output paths missing: %s / %s", out, diag)
				}
				if _, err := os.Stat(filepath.Join(output, "review.json")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestOfflineCommands(t *testing.T) {
	current, err := filepath.Abs("../../../../test/fixtures/prototype/contracts/snapshot-first.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"review", "report"} {
		t.Run(name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "result")
			code, out, diag := execute(t, app.Service{EffectiveUID: func() int { t.Fatal("offline privilege access"); return 0 }}, name, "--snapshot", current, "--output", output)
			if code != 0 || out != "" || !strings.Contains(diag, output) {
				t.Fatalf("%d %s %s", code, out, diag)
			}
			if code, _, _ := execute(t, app.Service{}, name, "--snapshot", current, "--output", output); code != 1 {
				t.Fatal("overwrite accepted")
			}
		})
	}
}

func TestReportPresentationFlags(t *testing.T) {
	current, err := filepath.Abs("../../../../test/fixtures/prototype/contracts/snapshot-first.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"html", "markdown"} {
		output := filepath.Join(t.TempDir(), "report")
		code, _, diag := execute(t, app.Service{}, "report", "--snapshot", current, "--format", format, "--lang", "ja", "--output", output)
		if code != 0 {
			t.Fatalf("code=%d %s", code, diag)
		}
		b, err := os.ReadFile(output)
		if err != nil || !strings.Contains(string(b), "確認レポート") {
			t.Fatalf("missing Japanese report: %v", err)
		}
	}
	for _, args := range [][]string{{"--format", "pdf"}, {"--lang", "fr"}} {
		output := filepath.Join(t.TempDir(), "not-created")
		options := append([]string{"report", "--snapshot", current, "--output", output}, args...)
		if code, _, _ := execute(t, app.Service{}, options...); code != 1 {
			t.Fatal("invalid presentation accepted")
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatal("invalid output published")
		}
	}
	if code, _, _ := execute(t, app.Service{}, "review", "--lang", "ja"); code != 1 {
		t.Fatal("report-only flag accepted by review")
	}
}

func TestSingleReportImportExportFlags(t *testing.T) {
	current, err := filepath.Abs("../../../../test/fixtures/prototype/contracts/snapshot-first.json")
	if err != nil {
		t.Fatal(err)
	}
	portable := filepath.Join(t.TempDir(), "portable.json")
	if code, _, diag := execute(t, app.Service{}, "review", "--single-report", "--snapshot", current, "--output", portable); code != 0 {
		t.Fatal(diag)
	}
	output := filepath.Join(t.TempDir(), "out.html")
	if code, _, diag := execute(t, app.Service{}, "report", "--input", portable, "--format", "html", "--lang", "ja", "--output", output); code != 0 {
		t.Fatal(diag)
	}
	for _, args := range [][]string{{"serve"}, {"collect", "--legacy-output", "--input", portable}, {"report", "--input", portable, "--snapshot", current, "--output", filepath.Join(t.TempDir(), "bad")}} {
		if code, _, _ := execute(t, app.Service{}, args...); code != 1 {
			t.Fatalf("invalid flags accepted: %v", args)
		}
	}
}
