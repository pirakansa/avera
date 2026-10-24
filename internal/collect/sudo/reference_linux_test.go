package sudo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/inventory"
)

// This optional check reads only a standalone fictional tree. It never asks
// sudo for policy decisions or edits the installed machine's configuration.
func TestDisposableVisudoReference(t *testing.T) {
	if os.Getenv("AVERA_SUDO_REFERENCE") != "1" {
		t.Skip("explicit disposable syntax comparison only")
	}
	executable, err := exec.LookPath("visudo")
	if err != nil {
		t.Skip("optional reference checker unavailable")
	}
	root := fixture(t)
	// visudo resolves absolute include paths against its own filesystem. Relative
	// paths keep both reference checker and collector inside this temporary tree.
	write(t, root, "etc/sudoers", "alice ALL=(root) NOPASSWD: /usr/bin/id, /usr/bin/whoami, PASSWD: /usr/bin/true\n#include sudoers.local\n#includedir sudoers.d\n")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-c", "-f", filepath.Join(root, "etc/sudoers"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fictional syntax check: %v: %s", err, output)
	}
	r := collect(t, root, Limits{})
	if r.Coverage.Status != inventory.Complete || len(r.Findings) != 4 {
		t.Fatalf("reference tree not understood: %+v", r.Coverage)
	}
	write(t, root, "etc/sudoers.local", "Defaults authenticate\n%operators ALL=(root:root) /usr/bin/id\n")
	cmd = exec.CommandContext(ctx, executable, "-c", "-f", filepath.Join(root, "etc/sudoers"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("unsupported valid syntax check: %v: %s", err, output)
	}
	r = collect(t, root, Limits{})
	if !slices.Contains(r.Coverage.DiagnosticCodes, "unsupported_construct") {
		t.Fatal("reference-valid unsupported construct not visible")
	}
}
