package storage

import (
	"fmt"
	"strconv"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
)

type mountTable struct {
	root           string
	rootFilesystem string
	byDevice       map[string][]i.StorageMount
}

func deviceNumber(value string) bool {
	fields := strings.Split(value, ":")
	if len(fields) != 2 {
		return false
	}
	for _, field := range fields {
		if _, err := strconv.ParseUint(field, 10, 32); err != nil {
			return false
		}
	}
	return true
}
func mountPath(raw string) (string, error) {
	var out strings.Builder
	for n := 0; n < len(raw); n++ {
		if raw[n] != '\\' {
			out.WriteByte(raw[n])
			continue
		}
		if n+3 >= len(raw) {
			return "", fmt.Errorf("invalid mount escape")
		}
		switch raw[n+1 : n+4] {
		case "040":
			out.WriteByte(' ')
		case "011":
			out.WriteByte('\t')
		case "012":
			out.WriteByte('\n')
		case "134":
			out.WriteByte('\\')
		default:
			return "", fmt.Errorf("invalid mount escape")
		}
		n += 3
	}
	value := out.String()
	if !strings.HasPrefix(value, "/") || len(value) > attributeBytes {
		return "", fmt.Errorf("nonabsolute mount")
	}
	return value, nil
}
func parseMounts(text string, limit int) (mountTable, error) {
	table := mountTable{byDevice: map[string][]i.StorageMount{}}
	rows := 0
	roots := 0
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		rows++
		if rows > limit {
			return table, fmt.Errorf("mount row limit")
		}
		fields := strings.Fields(line)
		separator := -1
		for n, value := range fields {
			if value == "-" {
				separator = n
				break
			}
		}
		if len(fields) < 10 || separator < 6 || len(fields) < separator+4 || !deviceNumber(fields[2]) {
			return table, fmt.Errorf("invalid mount row")
		}
		point, err := mountPath(fields[4])
		if err != nil {
			return table, err
		}
		readOnly := false
		modeFound := false
		for _, option := range strings.Split(fields[5], ",") {
			if option == "ro" {
				readOnly = true
				modeFound = true
			}
			if option == "rw" {
				modeFound = true
			}
		}
		if !modeFound {
			return table, fmt.Errorf("missing mount mode")
		}
		table.byDevice[fields[2]] = append(table.byDevice[fields[2]], i.StorageMount{Path: point, Filesystem: fields[separator+1], ReadOnly: readOnly})
		if point == "/" {
			roots++
			table.root = fields[2]
			table.rootFilesystem = fields[separator+1]
		}
	}
	if roots != 1 {
		return table, fmt.Errorf("root mount is unavailable or ambiguous")
	}
	return table, nil
}
