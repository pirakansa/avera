package usb

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const inputPath = "/sys/class/input"
const inputHandlersPath = "/proc/bus/input/devices"

// InputSource exposes metadata operations only. StatFD never opens the target.
type InputSource interface {
	Source
	StatFD(string) (string, bool, error)
	Readlink(string, int) (string, error)
}

type LocalInputSource struct{ LocalSource }

func decimal(v string) bool { _, err := strconv.ParseUint(v, 10, 32); return err == nil && v != "" }
func procMetadata(name string) bool {
	if name == "/proc" || name == inputHandlersPath {
		return true
	}
	parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
	if len(parts) < 2 || parts[0] != "proc" || !decimal(parts[1]) {
		return false
	}
	if len(parts) == 2 {
		return true
	}
	if len(parts) == 3 {
		return parts[2] == "status" || parts[2] == "stat" || parts[2] == "fd" || parts[2] == "exe"
	}
	return len(parts) == 4 && parts[2] == "fd" && decimal(parts[3])
}
func inputMetadata(name string) bool {
	return name == inputPath || strings.HasPrefix(name, inputPath+"/") || procMetadata(name)
}
func fdPath(name string) bool { return procMetadata(name) && strings.Contains(name, "/fd/") }

func (s LocalInputSource) ReadFile(name string, limit int) ([]byte, error) {
	// Do not open executable or descriptor targets, even to inspect their type.
	if fdPath(name) || path.Base(name) == "exe" {
		return nil, fmt.Errorf("not a readable metadata file")
	}
	return s.LocalSource.ReadFile(name, limit)
}
func (s LocalInputSource) StatFD(name string) (string, bool, error) {
	if !fdPath(name) {
		return "", false, fmt.Errorf("not a process descriptor")
	}
	parent, err := s.resolved(path.Dir(name))
	if err != nil {
		return "", false, err
	}
	var stat unix.Stat_t
	if err := unix.Stat(filepath.Join(parent, path.Base(name)), &stat); err != nil {
		return "", false, err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFCHR {
		return "", false, nil
	}
	return fmt.Sprintf("%d:%d", unix.Major(uint64(stat.Rdev)), unix.Minor(uint64(stat.Rdev))), true, nil
}
func (s LocalInputSource) Readlink(name string, limit int) (string, error) {
	if !procMetadata(name) || path.Base(name) != "exe" || s.Root == "" || filepath.Clean(name) != name {
		return "", fmt.Errorf("not executable metadata")
	}
	// Resolve the parent only; the executable is never opened or traversed.
	parent, err := s.resolved(path.Dir(name))
	if err != nil {
		return "", err
	}
	value, err := os.Readlink(filepath.Join(parent, "exe"))
	if err != nil {
		return "", err
	}
	if len(value) > limit {
		return "", fmt.Errorf("executable path limit")
	}
	return value, nil
}
