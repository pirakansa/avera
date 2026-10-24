package network

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pirakansa/avera/internal/inventory"
)

func TestProcSourceBoundedAccess(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"one", "two", "three"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := ProcSource{}
	names, more, err := s.ReadDir(dir, 2)
	if err != nil || !more || len(names) != 2 {
		t.Fatalf("%v %v %v", names, more, err)
	}
	names, more, err = s.ReadDir(dir, 5)
	if err != nil || more || len(names) != 3 {
		t.Fatalf("%v %v %v", names, more, err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink("fictional-executable", link); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Readlink(link, 4); !errors.Is(err, errLimit) {
		t.Fatal(err)
	}
	if got, err := s.Readlink(link, 100); err != nil || got != "fictional-executable" {
		t.Fatalf("%s %v", got, err)
	}
	f, err := s.Open(filepath.Join(dir, "one"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if string(b) != "data" {
		t.Fatal(string(b))
	}
	if _, _, err := s.ReadDir(filepath.Join(dir, "missing"), 1); err == nil {
		t.Fatal("missing directory succeeded")
	}
}

func TestDescriptorEnumerationOmitsOnlyItsOwnTemporaryDescriptor(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"10", "20", "30"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		limit int
		omit  string
		want  int
		more  bool
	}{
		{1, "20", 1, true}, {2, "20", 2, false}, {3, "20", 2, false},
		{2, "missing", 2, true}, {3, "", 3, false},
	} {
		f, err := os.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		names, more, err := readDirNames(f, tc.limit, tc.omit)
		if closeErr := f.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err != nil || more != tc.more || len(names) != tc.want || (tc.omit != "" && slices.Contains(names, tc.omit)) {
			t.Fatalf("limit=%d omit=%s: %v %v %v", tc.limit, tc.omit, names, more, err)
		}
	}
}

type failingSource struct {
	Source
	closeFailure bool
}
type failingReader struct {
	io.Reader
	closeFailure bool
}

func (r failingReader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	if err == io.EOF && !r.closeFailure {
		return n, errors.New("synthetic read failure")
	}
	return n, err
}
func (r failingReader) Close() error {
	if r.closeFailure {
		return errors.New("synthetic close failure")
	}
	return nil
}
func (s failingSource) Open(path string) (io.ReadCloser, error) {
	return failingReader{strings.NewReader(header), s.closeFailure}, nil
}

func TestReadAndCloseFailures(t *testing.T) {
	for _, closeFailure := range []bool{false, true} {
		r, err := (Collector{Source: failingSource{fixture(t), closeFailure}, ScopeID: "fixture"}).Collect(context.Background(), "target")
		if err != nil {
			t.Fatal(err)
		}
		if r.Coverage.Status != inventory.Partial || !slices.Contains(r.Coverage.DiagnosticCodes, "source_read_failed") {
			t.Fatalf("%+v", r.Coverage)
		}
	}
}
