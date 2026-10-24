// Package network observes TCP listeners and bound, unconnected UDP sockets.
package network

import (
	"io"
	"os"
	"strconv"
)

// Source provides bounded, injectable access to a proc filesystem.
// ReadDir returns at most limit entries and reports whether entries were omitted.
// Implementations must bound Readlink results to limit bytes.
type Source interface {
	Open(path string) (io.ReadCloser, error)
	ReadDir(path string, limit int) ([]string, bool, error)
	Readlink(path string, limit int) (string, error)
}

// ProcSource reads the current process's observable proc filesystem.
type ProcSource struct{}

func (ProcSource) Open(path string) (io.ReadCloser, error) { return os.Open(path) }

func (ProcSource) ReadDir(path string, limit int) ([]string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	omit := ""
	if path == "/proc/self/fd" || path == "/proc/"+strconv.Itoa(os.Getpid())+"/fd" {
		// Enumeration itself opens a descriptor that disappears on Close.
		// Exclude only this known descriptor, preserving genuine race diagnostics.
		omit = strconv.FormatUint(uint64(f.Fd()), 10)
	}
	names, more, err := readDirNames(f, limit, omit)
	closeErr := f.Close()
	if err != nil {
		return nil, false, err
	}
	if closeErr != nil {
		return nil, false, closeErr
	}
	return names, more, nil
}

func readDirNames(f *os.File, limit int, omit string) ([]string, bool, error) {
	extra := 1
	if omit != "" {
		extra++
	}
	entries, err := f.ReadDir(limit + extra)
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() != omit {
			names = append(names, entry.Name())
		}
	}
	more := len(names) > limit
	if more {
		names = names[:limit]
	}
	return names, more, nil
}

func (ProcSource) Readlink(path string, limit int) (string, error) {
	// os.Readlink uses the kernel's bounded proc symlink targets. The collector
	// rejects oversized results instead of silently fingerprinting truncated paths.
	s, err := os.Readlink(path)
	if err == nil && len(s) > limit {
		return "", errLimit
	}
	return s, err
}
