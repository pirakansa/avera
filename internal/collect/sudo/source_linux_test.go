package sudo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/pirakansa/avera/internal/inventory"
)

func TestLocalSourceConfinementAndTypes(t *testing.T) {
	root := fixture(t)
	outside := t.TempDir()
	write(t, outside, "policy", "alice ALL=(root) ALL\n")
	for _, name := range []string{"link", "parent"} {
		target := filepath.Join(outside, "policy")
		if name == "parent" {
			target = outside
		}
		if err := os.Symlink(target, filepath.Join(root, "etc", name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := syscall.Mkfifo(filepath.Join(root, "etc/fifo"), 0600); err != nil {
		t.Fatal(err)
	}
	source := LocalSource{Root: root}
	for _, name := range []string{"/etc/link", "/etc/parent/policy", "/etc/fifo", "/etc", "/etc/../policy", "relative"} {
		if _, err := source.ReadFile(name, 100); err == nil {
			t.Fatalf("unsafe file accepted: %s", name)
		}
	}
	if _, err := source.ReadDir("/etc/sudoers", 100); err == nil {
		t.Fatal("regular file treated as directory")
	}
	if _, err := source.ReadFile("/etc/sudoers", 1); !errors.Is(err, errLimit) {
		t.Fatalf("file limit: %v", err)
	}
	if _, err := source.ReadDir("/etc/sudoers.d", 1); !errors.Is(err, errLimit) {
		t.Fatalf("directory limit: %v", err)
	}
	write(t, root, "etc/sudoers", "@include link\n")
	r := collect(t, root, Limits{})
	if !slices.Contains(r.Coverage.DiagnosticCodes, "source_unreadable") {
		t.Fatal("symlink omission hidden")
	}
}

type cancelSource struct {
	Source
	cancel context.CancelFunc
}

func (s cancelSource) ReadFile(name string, n int) ([]byte, error) {
	s.cancel()
	return s.Source.ReadFile(name, n)
}
func TestCancellationDuringReads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := Collector{Source: cancelSource{Source: LocalSource{Root: fixture(t)}, cancel: cancel}}
	if _, err := c.Collect(ctx, "host"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

type deniedSource struct{ Source }

func (s deniedSource) ReadFile(name string, limit int) ([]byte, error) {
	if name == "/etc/sudoers.local" {
		return nil, errors.New("fictional confidential error detail")
	}
	return s.Source.ReadFile(name, limit)
}
func TestUnreadableSourceHasSafeVisibleDiagnostic(t *testing.T) {
	root := fixture(t)
	c := Collector{Source: deniedSource{Source: LocalSource{Root: root}}}
	r, err := c.Collect(context.Background(), "host")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(r.Coverage.DiagnosticCodes, "source_unreadable") {
		t.Fatal("read failure hidden")
	}
	for _, e := range r.Evidence {
		if strings.Contains(e.Values.(inventory.FileValues).Excerpt, "confidential") {
			t.Fatal("raw operating-system error exposed")
		}
	}
}
