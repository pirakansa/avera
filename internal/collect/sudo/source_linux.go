package sudo

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"golang.org/x/sys/unix"
)

var errLimit = errors.New("source limit exceeded")
var errSource = errors.New("unsupported source type")

// Source supplies bounded local files, never NSS or external identity services.
// Implementations must reject symlinks, special files and paths outside their root.
type Source interface {
	ReadFile(string, int) ([]byte, error)
	ReadDir(string, int) ([]string, error)
}

// LocalSource maps absolute policy paths into Root ("/" for an explicit live scan).
// Each operation pins directory descriptors and refuses symlinks at every step;
// tests use disposable fictional roots.
type LocalSource struct{ Root string }

func (s LocalSource) open(name string) (*os.File, error) {
	if s.Root == "" || !path.IsAbs(name) || path.Clean(name) != name {
		return nil, errSource
	}
	fd, err := unix.Open(s.Root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	// The closure follows the current pinned directory descriptor as we descend.
	defer func() { _ = unix.Close(fd) }()
	parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
	for _, part := range parts[:len(parts)-1] {
		next, err := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return nil, err
		}
		_ = unix.Close(fd)
		fd = next
	}
	// Nonblocking open also prevents a replaced FIFO from hanging collection.
	child, err := unix.Openat(fd, parts[len(parts)-1], unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(child), name)
	info, err := f.Stat()
	if err != nil || (!info.Mode().IsRegular() && !info.IsDir()) {
		_ = f.Close()
		return nil, errSource
	}
	return f, nil
}

func (s LocalSource) ReadFile(name string, limit int) ([]byte, error) {
	f, err := s.open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errSource
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, errLimit
	}
	return data, nil
}

func (s LocalSource) ReadDir(name string, limit int) ([]string, error) {
	f, err := s.open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(limit + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(names) > limit {
		return nil, errLimit
	}
	return names, nil
}
