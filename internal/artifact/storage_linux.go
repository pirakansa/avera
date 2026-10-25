//go:build linux

// Package artifact provides bounded reads and private, no-clobber publication.
package artifact

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pirakansa/avera/internal/inventory"
	"golang.org/x/sys/unix"
)

const (
	FileMode      = 0600
	DirectoryMode = 0700
)

// openParent pins each parent component and enforces the publication trust policy.
func openParent(name string) (int, string, error) {
	dir, base, err := walkParent(name)
	if err != nil {
		return -1, "", err
	}
	if err := checkParent(dir, filepath.Dir(name)); err != nil {
		_ = unix.Close(dir)
		return -1, "", err
	}
	return dir, base, nil
}

// walkParent never follows symlinks. It only opens directories; it does not
// create entries or inspect file contents before the parent policy is checked.
func walkParent(name string) (int, string, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return -1, "", fmt.Errorf("cannot resolve destination path: %w", err)
	}
	base := filepath.Base(abs)
	if base == "/" || base == "." || base == ".." {
		return -1, "", fmt.Errorf("destination must name a new file or run directory")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, "", fmt.Errorf("cannot open destination root: %w", err)
	}
	component := "/"
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Dir(abs), "/"), "/") {
		if part == "" {
			continue
		}
		component = filepath.Join(component, part)
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			var st unix.Stat_t
			statErr := unix.Fstatat(fd, part, &st, unix.AT_SYMLINK_NOFOLLOW)
			_ = unix.Close(fd)
			switch {
			case statErr == nil && st.Mode&unix.S_IFMT == unix.S_IFLNK:
				return -1, "", fmt.Errorf("parent component %q is a symbolic link; use a path without symbolic-link components: %w", component, openErr)
			case errors.Is(openErr, unix.ENOENT):
				return -1, "", fmt.Errorf("parent directory %q does not exist; create it before choosing a new output filename: %w", component, openErr)
			case errors.Is(openErr, unix.ENOTDIR):
				return -1, "", fmt.Errorf("parent component %q is not a directory; choose an existing directory: %w", component, openErr)
			case errors.Is(openErr, unix.EACCES) || errors.Is(openErr, unix.EPERM):
				return -1, "", fmt.Errorf("cannot access parent directory %q; check directory access permissions: %w", component, openErr)
			default:
				return -1, "", fmt.Errorf("cannot open parent directory %q: %w", component, openErr)
			}
		}
		if err := unix.Close(fd); err != nil {
			_ = unix.Close(next)
			return -1, "", err
		}
		fd = next
	}
	return fd, base, nil
}

func checkParent(dir int, name string) error {
	var st unix.Stat_t
	if err := unix.Fstat(dir, &st); err != nil {
		return fmt.Errorf("cannot inspect parent directory %q: %w", name, err)
	}
	return checkParentMetadata(st, name, os.Geteuid())
}

func checkParentMetadata(st unix.Stat_t, name string, effectiveUID int) error {
	// Root may explicitly select an ordinary user's working directory. Protect
	// the staging directory itself; parent ownership is not an authorization rule
	// for privileged callers. Ordinary callers retain the existing private-parent policy.
	if effectiveUID == 0 {
		return nil
	}
	if st.Uid != uint32(effectiveUID) && st.Uid != 0 {
		return fmt.Errorf("parent directory %q is owned by UID %d; expected effective UID %d or root (UID 0). Choose a private directory owned by the user running AVERA: %w", name, st.Uid, effectiveUID, unix.EPERM)
	}

	if st.Mode&0022 != 0 && st.Mode&unix.S_ISVTX == 0 {
		return fmt.Errorf("parent directory %q allows group/other writes without the sticky bit; choose a private directory with mode 0700: %w", name, unix.EPERM)
	}
	return nil
}

// Read rejects symlinks, non-regular files, and oversized data before domain decoding.
func Read(path string) (inventory.Artifact, error) {
	data, err := ReadBytes(path, inventory.MaxArtifactBytes)
	if err != nil {
		return nil, err
	}
	return inventory.Decode(data)
}

// ReadBytes applies the same bounded regular-file and no-symlink rules to portable reports.
func ReadBytes(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid byte limit")
	}
	fd, base, e := openParent(path)
	if e != nil {
		return nil, e
	}
	defer unix.Close(fd)
	input, e := unix.Openat(fd, base, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(input), base)
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input must be a regular file")
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("artifact byte limit exceeded")
	}
	data, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file byte limit exceeded")
	}
	return data, nil
}
func writeFile(ctx context.Context, dir int, name string, data []byte) (created bool, err error) {
	if e := ctx.Err(); e != nil {
		return created, e
	}
	fd, e := unix.Openat(dir, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, FileMode)
	if e != nil {
		return created, e
	}
	created = true
	f := os.NewFile(uintptr(fd), name)
	defer func() { err = errors.Join(err, f.Close()) }()
	if e = f.Chmod(FileMode); e != nil {
		return created, e
	}
	if _, e = f.Write(data); e != nil {
		return created, e
	}
	if e = ctx.Err(); e != nil {
		return created, e
	}
	return created, f.Sync()
}

// Write publishes one complete artifact atomically using a hard link that cannot replace a name.
// A post-publication directory-sync error may mean the complete output exists; retry never overwrites it.
func Write(ctx context.Context, path string, a inventory.Artifact) (err error) {
	data, e := inventory.Encode(a)
	if e != nil {
		return e
	}
	return WriteBytes(ctx, path, data)
}

// WriteBytes publishes rendered output with the same private no-clobber protocol.
func WriteBytes(ctx context.Context, path string, data []byte) (err error) {
	dir, base, e := openParent(path)
	if e != nil {
		return e
	}
	defer unix.Close(dir)
	temp, stage, e := createStage(dir)
	if e != nil {
		return e
	}
	defer unix.Close(stage)
	created := false
	defer func() {
		names := []string{}
		if created {
			names = append(names, "artifact")
		}
		err = errors.Join(err, removeStage(dir, stage, temp, names))
	}()
	if created, e = writeFile(ctx, stage, "artifact", data); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = unix.Linkat(stage, "artifact", dir, base, 0); e != nil {
		return e
	}
	return unix.Fsync(dir)
}

func createStage(dir int) (string, int, error) {
	name := ".avera-" + rand.Text()
	if err := unix.Mkdirat(dir, name, DirectoryMode); err != nil {
		return "", -1, err
	}
	stage, err := openPrivateStage(dir, name)
	if err != nil {
		return "", -1, errors.Join(err, unix.Unlinkat(dir, name, unix.AT_REMOVEDIR))
	}
	return name, stage, nil
}

// openPrivateStage checks the opened inode, not the replaceable entry in the
// output parent. Never chmod or write into a substituted untrusted directory.
func openPrivateStage(parent int, name string) (int, error) {
	stage, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	var st unix.Stat_t
	if err = unix.Fstat(stage, &st); err == nil {
		err = checkStageMetadata(st, os.Geteuid())
	}
	if err == nil {
		err = unix.Fchmod(stage, DirectoryMode)
	}
	if err != nil {
		_ = unix.Close(stage)
		return -1, err
	}
	return stage, nil
}
func checkStageMetadata(st unix.Stat_t, effectiveUID int) error {
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uint32(effectiveUID) || st.Mode&0077 != 0 {
		return fmt.Errorf("private staging directory was replaced or is not owned exclusively by the effective user: %w", unix.EPERM)
	}
	return nil
}

func removeStage(dir, stage int, temp string, names []string) (err error) {
	for _, name := range names {
		e := unix.Unlinkat(stage, name, 0)
		if e != nil && !errors.Is(e, unix.ENOENT) {
			err = errors.Join(err, e)
		}
	}
	return errors.Join(err, unix.Unlinkat(dir, temp, unix.AT_REMOVEDIR))
}

// WriteRun stages all artifacts privately and publishes a directory using RENAME_NOREPLACE.
// Filesystem support for renameat2 is required; unsupported systems fail without a fallback rename.
func WriteRun(ctx context.Context, path string, artifacts map[string]inventory.Artifact) (err error) {
	if len(artifacts) == 0 {
		return fmt.Errorf("empty run")
	}
	encoded := map[string][]byte{}
	target := ""
	for name, a := range artifacts {
		if filepath.Base(name) != name || name == "." || name == ".." || name == "" {
			return fmt.Errorf("invalid artifact filename %q", name)
		}
		data, e := inventory.Encode(a)
		if e != nil {
			return e
		}
		if target == "" {
			target = a.Header().TargetID
		} else if a.Header().TargetID != target {
			return fmt.Errorf("run target_id mismatch")
		}
		encoded[name] = data
	}
	dir, base, e := openParent(path)
	if e != nil {
		return e
	}
	defer unix.Close(dir)
	temp, stage, e := createStage(dir)
	if e != nil {
		return e
	}
	defer unix.Close(stage)
	const runName = "run"
	if e = unix.Mkdirat(stage, runName, DirectoryMode); e != nil {
		return errors.Join(e, removeStage(dir, stage, temp, nil))
	}
	run, e := openPrivateStage(stage, runName)
	if e != nil {
		return errors.Join(e, unix.Unlinkat(stage, runName, unix.AT_REMOVEDIR), removeStage(dir, stage, temp, nil))
	}
	defer unix.Close(run)
	published := false
	defer func() {
		if !published {
			for name := range encoded {
				if e := unix.Unlinkat(run, name, 0); e != nil && !errors.Is(e, unix.ENOENT) {
					err = errors.Join(err, e)
				}
			}
			err = errors.Join(err, unix.Unlinkat(stage, runName, unix.AT_REMOVEDIR))
		}
		err = errors.Join(err, removeStage(dir, stage, temp, nil))
	}()
	for name, data := range encoded {
		if _, e = writeFile(ctx, run, name, data); e != nil {
			return e
		}
	}
	if e = unix.Fsync(run); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	// The parent owner may move/replace temp, but cannot substitute run inside
	// our verified private stage. Publish from that pinned descriptor.
	if e = unix.Renameat2(stage, runName, dir, base, unix.RENAME_NOREPLACE); e != nil {
		return e
	}
	published = true

	return unix.Fsync(dir)
}

// CheckDestination validates a new file or legacy run-directory destination.
// Publication independently enforces no replacement to handle concurrent creators.
func CheckDestination(name string) error { return checkDestination(name, false) }

// CheckFileDestination also rejects directory syntax and identifies existing
// directories before evaluating the output parent's ownership policy.
func CheckFileDestination(name string) error { return checkDestination(name, true) }

func checkDestination(name string, fileOnly bool) error {
	if fileOnly && (strings.HasSuffix(name, string(filepath.Separator)) || filepath.Base(name) == "." || filepath.Base(name) == "..") {
		return fmt.Errorf("output must name a new file, not a directory; include a filename such as report.json")
	}
	dir, base, err := walkParent(name)
	if err != nil {
		return err
	}
	defer unix.Close(dir)
	var st unix.Stat_t
	err = unix.Fstatat(dir, base, &st, unix.AT_SYMLINK_NOFOLLOW)
	if err == nil {
		if st.Mode&unix.S_IFMT == unix.S_IFLNK {
			return fmt.Errorf("destination %q is a symbolic link; choose a new path without symbolic links: %w", name, os.ErrExist)
		}
		if fileOnly && st.Mode&unix.S_IFMT == unix.S_IFDIR {
			return fmt.Errorf("output %q is a directory; include a new filename such as report.json inside it: %w", name, os.ErrExist)
		}
		return fmt.Errorf("destination %q already exists; choose a new output name (existing files are never overwritten): %w", name, os.ErrExist)
	}
	if !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("cannot inspect destination %q: %w", name, err)
	}
	if err := checkParent(dir, filepath.Dir(name)); err != nil {
		return err
	}
	// Verify staging before any privileged source reads. This empty directory is
	// removed immediately and contains no collected data.
	temp, stage, err := createStage(dir)
	if err != nil {
		return fmt.Errorf("cannot create a private temporary directory in %q; check write permissions, available space and whether the filesystem is read-only: %w", filepath.Dir(name), err)
	}
	return errors.Join(unix.Close(stage), unix.Unlinkat(dir, temp, unix.AT_REMOVEDIR))
}
