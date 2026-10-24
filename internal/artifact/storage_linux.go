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

// openParent walks every directory component using pinned descriptors and O_NOFOLLOW.
// No ancestor may be a symlink, including paths used by a root caller.
func openParent(path string) (int, string, error) {
	abs, e := filepath.Abs(path)
	if e != nil {
		return -1, "", e
	}
	base := filepath.Base(abs)
	if base == "/" || base == "." || base == ".." {
		return -1, "", fmt.Errorf("invalid destination")
	}
	fd, e := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if e != nil {
		return -1, "", e
	}
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Dir(abs), "/"), "/") {
		if part == "" {
			continue
		}
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		closeErr := unix.Close(fd)
		if e != nil {
			return -1, "", e
		}
		if closeErr != nil {
			_ = unix.Close(next)
			return -1, "", closeErr
		}
		fd = next
	}
	var st unix.Stat_t
	if e := unix.Fstat(fd, &st); e != nil {
		_ = unix.Close(fd)
		return -1, "", e
	}
	// A pinned descriptor prevents ancestor substitution. Reject directories in
	// which another principal could replace our private staging entry.
	if (st.Uid != uint32(os.Geteuid()) && st.Uid != 0) || (st.Mode&0022 != 0 && st.Mode&unix.S_ISVTX == 0) {
		_ = unix.Close(fd)
		return -1, "", fmt.Errorf("parent must be owned by the effective user or root and must not allow unprotected group/other writes")
	}
	return fd, base, nil
}

// Read rejects symlinks, non-regular files, and oversized data before domain decoding.
func Read(path string) (inventory.Artifact, error) {
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
	if info.Size() > inventory.MaxArtifactBytes {
		return nil, fmt.Errorf("artifact byte limit exceeded")
	}
	data, e := io.ReadAll(io.LimitReader(f, inventory.MaxArtifactBytes+1))
	if e != nil {
		return nil, e
	}
	return inventory.Decode(data)
}
func writeFile(ctx context.Context, dir int, name string, data []byte) (err error) {
	if e := ctx.Err(); e != nil {
		return e
	}
	fd, e := unix.Openat(dir, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, FileMode)
	if e != nil {
		return e
	}
	f := os.NewFile(uintptr(fd), name)
	defer func() { err = errors.Join(err, f.Close()) }()
	if e = f.Chmod(FileMode); e != nil {
		return e
	}
	if _, e = f.Write(data); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return f.Sync()
}

// Write publishes one complete artifact atomically using a hard link that cannot replace a name.
// A post-publication directory-sync error may mean the complete output exists; retry never overwrites it.
func Write(ctx context.Context, path string, a inventory.Artifact) (err error) {
	data, e := inventory.Encode(a)
	if e != nil {
		return e
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
	defer func() { err = errors.Join(err, removeStage(dir, stage, temp, []string{"artifact"})) }()
	if e = writeFile(ctx, stage, "artifact", data); e != nil {
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
	temp := ".avera-" + rand.Text()
	if e := unix.Mkdirat(dir, temp, DirectoryMode); e != nil {
		return "", -1, e
	}
	stage, e := unix.Openat(dir, temp, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return "", -1, errors.Join(e, unix.Unlinkat(dir, temp, unix.AT_REMOVEDIR))
	}
	if e = unix.Fchmod(stage, DirectoryMode); e != nil {
		_ = unix.Close(stage)
		return "", -1, errors.Join(e, unix.Unlinkat(dir, temp, unix.AT_REMOVEDIR))
	}
	return temp, stage, nil
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
	published := false
	defer func() {
		if published {
			return
		}
		names := make([]string, 0, len(encoded))
		for name := range encoded {
			names = append(names, name)
		}
		err = errors.Join(err, removeStage(dir, stage, temp, names))
	}()
	for name, data := range encoded {
		if e = writeFile(ctx, stage, name, data); e != nil {
			return e
		}
	}
	if e = unix.Fsync(stage); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = unix.Renameat2(dir, temp, dir, base, unix.RENAME_NOREPLACE); e != nil {
		return e
	}
	published = true
	return unix.Fsync(dir)
}
