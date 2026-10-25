package usb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Source reads sysfs metadata only; no operation opens a device node.
type Source interface {
	ReadFile(string, int) ([]byte, error)
	ReadDir(string, int) ([]string, bool, error)
	Resolve(string) (string, error)
}

// LocalSource confines reads to Root. Tests use a private synthetic sysfs/proc tree.
// Kernel metadata reads remain non-atomic; Root must be trusted.
type LocalSource struct{ Root string }

func (s LocalSource) resolved(name string) (string, error) {
	if name != busPath && !strings.HasPrefix(name, busPath+"/") && !strings.HasPrefix(name, "/sys/devices/") && !inputMetadata(name) {
		return "", fmt.Errorf("not a USB metadata source")
	}
	if s.Root == "" || !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return "", fmt.Errorf("invalid metadata path")
	}
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(root, strings.TrimPrefix(name, "/")))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("metadata path leaves root")
	}
	return target, nil
}
func (s LocalSource) Resolve(name string) (string, error) {
	target, err := s.resolved(name)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	return "/" + filepath.ToSlash(relative), nil
}
func (s LocalSource) ReadFile(name string, limit int) ([]byte, error) {
	if fdPath(name) || filepath.Base(name) == "exe" {
		return nil, fmt.Errorf("not a readable metadata file")
	}
	target, err := s.resolved(name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(target)
	if err != nil {
		return nil, err
	}
	info, statErr := f.Stat()
	if statErr != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, fmt.Errorf("metadata is not a regular file")
	}
	data, readErr := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	closeErr := f.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(data) > limit {
		return nil, fmt.Errorf("metadata byte limit")
	}
	return data, nil
}
func (s LocalSource) ReadDir(name string, limit int) ([]string, bool, error) {
	target, err := s.resolved(name)
	if err != nil {
		return nil, false, err
	}
	f, err := os.Open(target)
	if err != nil {
		return nil, false, err
	}
	entries, readErr := f.Readdirnames(limit + 1)
	closeErr := f.Close()
	if readErr != nil && readErr != io.EOF {
		return nil, false, readErr
	}
	if closeErr != nil {
		return nil, false, closeErr
	}
	more := len(entries) > limit
	if more {
		entries = entries[:limit]
	}
	return entries, more, nil
}
