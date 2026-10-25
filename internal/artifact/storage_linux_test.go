//go:build linux

package artifact

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pirakansa/avera/internal/inventory"
)

func example(t *testing.T) inventory.Artifact {
	t.Helper()
	b, e := os.ReadFile("../../test/fixtures/prototype/contracts/assessments-reviewed.json")
	if e != nil {
		t.Fatal(e)
	}
	a, e := inventory.Decode(b)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func contents(t *testing.T, path string) string {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func mode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, e := os.Stat(path)
	if e != nil || info.Mode().Perm() != want {
		t.Fatalf("mode %s: %v %v", path, info, e)
	}
}
func assertClean(t *testing.T, dir string) {
	t.Helper()
	matches, e := filepath.Glob(filepath.Join(dir, ".avera-*"))
	if e != nil || len(matches) != 0 {
		t.Fatalf("staging files remain: %v %v", matches, e)
	}
}
func TestWriteAndNoClobber(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "assessments.json")
	a := example(t)
	if e := Write(ctx, path, a); e != nil {
		t.Fatal(e)
	}
	mode(t, path, FileMode)
	before := contents(t, path)
	if e := Write(ctx, path, a); !errors.Is(e, os.ErrExist) {
		t.Fatal("existing destination", e)
	}
	if contents(t, path) != before {
		t.Fatal("input overwritten")
	}
	read, e := Read(path)
	if e != nil {
		t.Fatal(e)
	}
	data, e := inventory.Encode(read)
	if e != nil || string(data) != before {
		t.Fatal("read roundtrip", e)
	}
	assertClean(t, dir)
}
func TestSymlinksAndNonRegularInputs(t *testing.T) {
	dir := t.TempDir()
	a := example(t)
	victim := filepath.Join(dir, "victim")
	if e := os.WriteFile(victim, []byte("unchanged"), 0600); e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(dir, "link")
	if e := os.Symlink(victim, link); e != nil {
		t.Fatal(e)
	}
	if e := Write(context.Background(), link, a); e == nil {
		t.Fatal("symlink destination accepted")
	}
	if _, e := Read(link); e == nil {
		t.Fatal("symlink input accepted")
	}
	if contents(t, victim) != "unchanged" {
		t.Fatal("symlink target modified")
	}
	parent := filepath.Join(dir, "parent-link")
	if e := os.Symlink(dir, parent); e != nil {
		t.Fatal(e)
	}
	if e := Write(context.Background(), filepath.Join(parent, "out"), a); e == nil {
		t.Fatal("symlink ancestor accepted")
	}
	if e := WriteRun(context.Background(), parent, map[string]inventory.Artifact{"a.json": a}); e == nil {
		t.Fatal("directory symlink accepted")
	}
	if _, e := Read(dir); e == nil {
		t.Fatal("directory accepted")
	}
	assertClean(t, dir)
}
func TestAtomicRunAndFailure(t *testing.T) {
	dir := t.TempDir()
	a := example(t)
	run := filepath.Join(dir, "run")
	artifacts := map[string]inventory.Artifact{"a.json": a, "b.json": a}
	if e := WriteRun(context.Background(), run, artifacts); e != nil {
		t.Fatal(e)
	}
	mode(t, run, DirectoryMode)
	for name := range artifacts {
		mode(t, filepath.Join(run, name), FileMode)
		if _, e := Read(filepath.Join(run, name)); e != nil {
			t.Fatal(e)
		}
	}
	before := contents(t, filepath.Join(run, "a.json"))
	if e := WriteRun(context.Background(), run, artifacts); !errors.Is(e, os.ErrExist) {
		t.Fatal("run clobber", e)
	}
	if contents(t, filepath.Join(run, "a.json")) != before {
		t.Fatal("run modified")
	}
	if e := WriteRun(context.Background(), filepath.Join(dir, "invalid"), map[string]inventory.Artifact{"../victim": a}); e == nil {
		t.Fatal("path traversal accepted")
	}
	assertClean(t, dir)
}

// cancelAfter implements deterministic interruption at a write or publication boundary.
type cancelAfter struct {
	context.Context
	calls, after int
}

func (c *cancelAfter) Err() error {
	c.calls++
	if c.calls >= c.after {
		return context.Canceled
	}
	return nil
}
func TestInterruptedPublication(t *testing.T) {
	a := example(t)
	for _, run := range []bool{false, true} {
		for boundary := 1; boundary <= 5; boundary++ {
			dir := t.TempDir()
			path := filepath.Join(dir, "result")
			ctx := &cancelAfter{Context: context.Background(), after: boundary}
			var e error
			if run {
				e = WriteRun(ctx, path, map[string]inventory.Artifact{"a.json": a, "b.json": a})
			} else {
				e = Write(ctx, path, a)
			}
			if e != nil {
				if !errors.Is(e, context.Canceled) {
					t.Fatal(e)
				}
				if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
					t.Fatal("partial output visible", e)
				}
			} else {
				if run {
					for _, name := range []string{"a.json", "b.json"} {
						if _, e := Read(filepath.Join(path, name)); e != nil {
							t.Fatal(e)
						}
					}
				} else if _, e := Read(path); e != nil {
					t.Fatal(e)
				}
			}
			assertClean(t, dir)
		}
	}
}
func TestConcurrentPublication(t *testing.T) {
	a := example(t)
	for _, run := range []bool{false, true} {
		dir := t.TempDir()
		path := filepath.Join(dir, "result")
		results := make(chan error, 8)
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if run {
					results <- WriteRun(context.Background(), path, map[string]inventory.Artifact{"a.json": a})
				} else {
					results <- Write(context.Background(), path, a)
				}
			}()
		}
		wg.Wait()
		close(results)
		success := 0
		for e := range results {
			if e == nil {
				success++
			} else if !errors.Is(e, os.ErrExist) {
				t.Fatal(e)
			}
		}
		if success != 1 {
			t.Fatalf("%d publishers succeeded", success)
		}
		assertClean(t, dir)
	}
}
func TestInvalidOutputPreservesInputs(t *testing.T) {
	dir := t.TempDir()
	a := example(t)
	path := filepath.Join(dir, "input.json")
	if e := Write(context.Background(), path, a); e != nil {
		t.Fatal(e)
	}
	before := contents(t, path)
	a.(*inventory.Assessments).Entries[0].Decision = "safe"
	if e := Write(context.Background(), path, a); e == nil {
		t.Fatal("invalid output accepted")
	}
	if contents(t, path) != before {
		t.Fatal("input modified")
	}
	if e := WriteRun(context.Background(), filepath.Join(dir, "run"), map[string]inventory.Artifact{"invalid.json": a}); e == nil {
		t.Fatal("invalid run accepted")
	}
	if _, e := os.Stat(filepath.Join(dir, "run")); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("invalid run exposed", e)
	}
	assertClean(t, dir)
}

func TestUnsafeParentAndMixedTargets(t *testing.T) {
	dir := t.TempDir()
	a := example(t)
	unsafe := filepath.Join(dir, "unsafe")
	if e := os.Mkdir(unsafe, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(unsafe, 0777); e != nil {
		t.Fatal(e)
	}
	if e := Write(context.Background(), filepath.Join(unsafe, "out.json"), a); os.Geteuid() != 0 && e == nil {
		t.Fatal("unsafe ordinary-user parent accepted")
	} else if os.Geteuid() == 0 && e != nil {
		t.Fatal(e)
	}
	other := example(t)
	other.(*inventory.Assessments).TargetID = "other"
	if e := WriteRun(context.Background(), filepath.Join(dir, "mixed"), map[string]inventory.Artifact{"one.json": a, "two.json": other}); e == nil {
		t.Fatal("mixed targets accepted")
	}
	assertClean(t, dir)
}
func TestBoundedReadAndFIFO(t *testing.T) {
	dir := t.TempDir()
	oversized := filepath.Join(dir, "oversized")
	f, e := os.Create(oversized)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate(inventory.MaxArtifactBytes + 1); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = Read(oversized); e == nil {
		t.Fatal("oversized input accepted")
	}
	fifo := filepath.Join(dir, "fifo")
	if e = unix.Mkfifo(fifo, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Read(fifo); e == nil {
		t.Fatal("FIFO accepted")
	}
}

func TestRenderedOutputPublication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.md")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := WriteBytes(ctx, path, []byte("incomplete")); err == nil {
		t.Fatal("cancelled write succeeded")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cancelled output exists: %v", err)
	}
	if err := CheckDestination(path); err != nil {
		t.Fatal(err)
	}
	if err := WriteBytes(context.Background(), path, []byte("complete")); err != nil {
		t.Fatal(err)
	}
	if err := CheckDestination(path); err == nil {
		t.Fatal("existing destination accepted")
	}
	if err := WriteBytes(context.Background(), path, []byte("replace")); err == nil {
		t.Fatal("output replaced")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "complete" {
		t.Fatalf("output changed: %q %v", b, err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Mode().Perm() != FileMode {
		t.Fatalf("permissions: %v", err)
	}
}

func TestDestinationPreflightRequiresWritableParent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory write permission")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Error(err)
		}
	})
	if err := CheckDestination(filepath.Join(dir, "run")); err == nil || !strings.Contains(err.Error(), "cannot create a private temporary directory") || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unwritable parent diagnostic: %v", err)
	}
}

func TestFileDestinationDiagnostics(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.json")
	if err := os.WriteFile(existing, []byte("keep this file"), 0600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "link")
	if err := os.Symlink(dir, symlink); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, path, message string
		cause               error
	}{
		{"directory syntax", dir + "/", "include a filename", nil},
		{"existing directory", dir, "is a directory", os.ErrExist},
		{"existing file", existing, "already exists", os.ErrExist},
		{"missing parent", filepath.Join(dir, "missing", "report.json"), "does not exist", os.ErrNotExist},
		{"symlink parent", filepath.Join(symlink, "report.json"), "symbolic link", nil},
		{"symlink destination", symlink, "symbolic link", os.ErrExist},
		{"file as parent", filepath.Join(existing, "report.json"), "not a directory", unix.ENOTDIR},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := CheckFileDestination(test.path)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q, got %v", test.message, err)
			}
			if test.cause != nil && !errors.Is(err, test.cause) {
				t.Fatalf("lost original error: %v", err)
			}
		})
	}
	if contents(t, existing) != "keep this file" {
		t.Fatal("preflight changed existing file")
	}
	assertClean(t, dir)
}

func TestParentPolicyDiagnostics(t *testing.T) {
	const ordinaryUID = 1000
	for _, test := range []struct {
		name        string
		owner       uint32
		effective   int
		permissions uint32
		message     string
	}{
		{"sudo user-owned directory", ordinaryUID, 0, 0700, ""},
		{"different ordinary owner", ordinaryUID + 1, ordinaryUID, 0700, "expected effective UID 1000"},
		{"writable by other users", ordinaryUID, ordinaryUID, 0777, "allows group/other writes"},
		{"root private directory", 0, 0, 0700, ""},
		{"sudo group-writable user directory", ordinaryUID, 0, 0775, ""},
		{"sudo world-writable parent", ordinaryUID, 0, 0777, ""},
		{"own private directory", ordinaryUID, ordinaryUID, 0700, ""},
		{"root-owned sticky tmp", 0, ordinaryUID, unix.S_ISVTX | 0777, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkParentMetadata(unix.Stat_t{Uid: test.owner, Mode: test.permissions}, "./fuga", test.effective)
			if test.message == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.message) || !strings.Contains(err.Error(), "./fuga") || !errors.Is(err, unix.EPERM) {
				t.Fatalf("policy diagnostic: %v", err)
			}
		})
	}
}

func TestStagingDirectoryMustBelongToWriter(t *testing.T) {
	for _, test := range []struct {
		name  string
		st    unix.Stat_t
		uid   int
		valid bool
	}{
		{"root private stage", unix.Stat_t{Uid: 0, Mode: unix.S_IFDIR | 0700}, 0, true},
		{"ordinary private stage", unix.Stat_t{Uid: 1000, Mode: unix.S_IFDIR | 0700}, 1000, true},
		{"substituted user-owned stage under root", unix.Stat_t{Uid: 1000, Mode: unix.S_IFDIR | 0700}, 0, false},
		{"public stage", unix.Stat_t{Uid: 0, Mode: unix.S_IFDIR | 0777}, 0, false},
		{"non-directory", unix.Stat_t{Uid: 0, Mode: unix.S_IFREG | 0600}, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkStageMetadata(test.st, test.uid)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t error=%v", test.valid, err)
			}
		})
	}
}

type replaceStageName struct {
	context.Context
	t             *testing.T
	parent, moved string
	calls         int
}

func (c *replaceStageName) Err() error {
	c.calls++
	// One artifact: writeFile checks before/after the write, then publication checks.
	if c.calls == 3 {
		matches, err := filepath.Glob(filepath.Join(c.parent, ".avera-*"))
		if err != nil || len(matches) != 1 {
			c.t.Fatalf("find stage: %v %v", matches, err)
		}
		if err := os.Rename(matches[0], c.moved); err != nil {
			c.t.Fatal(err)
		}
		if err := os.Mkdir(matches[0], 0700); err != nil {
			c.t.Fatal(err)
		}
		// The replacement has attacker-chosen contents at the original source name.
		if err := os.WriteFile(filepath.Join(matches[0], "a.json"), []byte("substituted"), 0600); err != nil {
			c.t.Fatal(err)
		}
	}
	return nil
}
func TestParentNameReplacementCannotSubstitutePublishedRun(t *testing.T) {
	dir := t.TempDir()
	ctx := &replaceStageName{Context: context.Background(), t: t, parent: dir, moved: filepath.Join(dir, "moved-stage")}
	output := filepath.Join(dir, "run")
	a := example(t)
	err := WriteRun(ctx, output, map[string]inventory.Artifact{"a.json": a})
	// Cleanup can report the substituted, nonempty parent's name. Publication must
	// nevertheless have come from the pinned private source, never its replacement.
	if err != nil && !errors.Is(err, unix.ENOTEMPTY) {
		t.Fatal(err)
	}
	data, encodeErr := inventory.Encode(a)
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if got := contents(t, filepath.Join(output, "a.json")); got != string(data) {
		t.Fatalf("published substituted contents: %q", got)
	}
	mode(t, output, DirectoryMode)
	mode(t, filepath.Join(output, "a.json"), FileMode)
}

// This optional integration test is executed separately under sudo with a
// disposable directory created by the ordinary user. Default tests need no root.
func TestSudoPublicationInUserOwnedDirectory(t *testing.T) {
	parent := os.Getenv("AVERA_TEST_USER_DIRECTORY")
	if os.Geteuid() != 0 || parent == "" {
		t.Skip("requires explicit sudo integration fixture")
	}
	var st unix.Stat_t
	if err := unix.Stat(parent, &st); err != nil {
		t.Fatal(err)
	}
	if st.Uid == 0 {
		t.Fatal("integration parent must belong to an ordinary user")
	}
	a := example(t)
	file := filepath.Join(parent, "root-report.json")
	run := filepath.Join(parent, "root-run")
	for _, name := range []string{file, run} {
		if _, err := os.Lstat(name); !os.IsNotExist(err) {
			t.Fatalf("test output must be absent: %s (%v)", name, err)
		}
	}
	t.Cleanup(func() {
		for _, name := range []string{file, run} {
			if err := os.RemoveAll(name); err != nil {
				t.Error(err)
			}
		}
	})
	if err := CheckFileDestination(file); err != nil {
		t.Fatal(err)
	}
	if err := Write(context.Background(), file, a); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(file); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(filepath.Join(parent, "user-input.json")); err != nil {
		t.Fatal("root reading user-owned input", err)
	}
	if err := WriteRun(context.Background(), run, map[string]inventory.Artifact{"a.json": a}); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(filepath.Join(run, "a.json")); err != nil {
		t.Fatal(err)
	}
	if err := Write(context.Background(), file, a); !errors.Is(err, os.ErrExist) {
		t.Fatal("overwrite allowed", err)
	}
	mode(t, file, FileMode)
	mode(t, run, DirectoryMode)
	assertClean(t, parent)
	var after unix.Stat_t
	if err := unix.Stat(parent, &after); err != nil || after.Uid != st.Uid || after.Mode != st.Mode {
		t.Fatal("parent ownership or permissions changed", err)
	}
	// A directory inserted by the parent owner must not be chmodded or used by root.
	substituted := filepath.Join(parent, "foreign-stage")
	if err := os.Mkdir(substituted, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(substituted); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chown(substituted, int(st.Uid), int(st.Gid)); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if stage, err := openPrivateStage(fd, "foreign-stage"); err == nil {
		_ = unix.Close(stage)
		t.Fatal("root accepted substituted ordinary-user staging directory")
	}
	mode(t, substituted, DirectoryMode)
}

func TestParentNameReplacementCannotSubstitutePublishedFile(t *testing.T) {
	dir := t.TempDir()
	ctx := &replaceStageName{Context: context.Background(), t: t, parent: dir, moved: filepath.Join(dir, "moved-stage")}
	output := filepath.Join(dir, "report.json")
	err := WriteBytes(ctx, output, []byte("original complete report"))
	if err != nil && !errors.Is(err, unix.ENOTEMPTY) {
		t.Fatal(err)
	}
	if got := contents(t, output); got != "original complete report" {
		t.Fatalf("substituted file: %q", got)
	}
	mode(t, output, FileMode)
}

type insertExistingStageFile struct {
	context.Context
	t        *testing.T
	parent   string
	inserted string
}

func (c *insertExistingStageFile) Err() error {
	if c.inserted != "" {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(c.parent, ".avera-*"))
	if err != nil || len(matches) != 1 {
		c.t.Fatal("stage not found", matches, err)
	}
	c.inserted = filepath.Join(matches[0], "artifact")
	if err := os.WriteFile(c.inserted, []byte("pre-existing private data"), 0600); err != nil {
		c.t.Fatal(err)
	}
	return nil
}
func TestFailedExclusiveCreateNeverDeletesPreexistingStageContents(t *testing.T) {
	dir := t.TempDir()
	ctx := &insertExistingStageFile{Context: context.Background(), t: t, parent: dir}
	output := filepath.Join(dir, "report.json")
	err := WriteBytes(ctx, output, []byte("new output"))
	if !errors.Is(err, os.ErrExist) {
		t.Fatal("exclusive create was bypassed", err)
	}
	if got := contents(t, ctx.inserted); got != "pre-existing private data" {
		t.Fatal("unowned stage content changed", got)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("failed publication exposed output", err)
	}
}
