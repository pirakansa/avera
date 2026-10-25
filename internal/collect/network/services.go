package network

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pirakansa/avera/internal/inventory"
)

const (
	serviceSourceBytes = 64 << 10
	serviceNameBytes   = 256
	serviceVersion     = "services-v1"
)

// readServiceSource never retains command lines, environments, or whole proc files.
func readServiceSource(ctx context.Context, source Source, path string, limit int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f, err := source.Open(path)
	if err != nil {
		return "", err
	}
	data, readErr := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	closeErr := f.Close()
	if readErr != nil {
		return "", readErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if len(data) > limit || !utf8.Valid(data) {
		return "", errLimit
	}
	return string(data), ctx.Err()
}

func processStart(text string) (string, error) {
	end := strings.LastIndex(text, ")")
	if end < 0 {
		return "", fmt.Errorf("invalid stat")
	}
	fields := strings.Fields(text[end+1:])
	if len(fields) < 20 {
		return "", fmt.Errorf("invalid stat")
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", err
	}
	return fields[19], nil
}

func parseServiceStatus(text string) (string, *uint32) {
	var name string
	var uid *uint32
	names, uids := 0, 0
	for _, line := range strings.Split(text, "\n") {
		if value, ok := strings.CutPrefix(line, "Name:"); ok {
			names++
			name = strings.TrimSpace(value)
		}
		if value, ok := strings.CutPrefix(line, "Uid:"); ok {
			uids++
			fields := strings.Fields(value)
			if len(fields) != 4 {
				continue
			}
			valid := true
			for n, field := range fields {
				parsed, err := strconv.ParseUint(field, 10, 32)
				if err != nil {
					valid = false
					break
				}
				if n == 1 {
					v := uint32(parsed)
					uid = &v
				}
			}
			if !valid {
				uid = nil
			}
		}
	}
	if names != 1 || len(name) > serviceNameBytes {
		name = ""
	}
	if uids != 1 {
		uid = nil
	}
	return name, uid
}

// serviceCgroup accepts the unified hierarchy or explicit systemd v1 hierarchy.
// A cgroup name is provenance for a candidate, never verification by systemd.
func serviceCgroup(text string) (string, string, error) {
	path := ""
	rows := 0
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, ":", 3)
		if len(fields) != 3 || !strings.HasPrefix(fields[2], "/") {
			return "", "", fmt.Errorf("invalid cgroup")
		}
		hierarchy, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil || (hierarchy == 0 && fields[1] != "") || (hierarchy != 0 && fields[1] == "") {
			return "", "", fmt.Errorf("invalid cgroup hierarchy")
		}
		rows++
		if fields[0] != "0" || fields[1] != "" {
			if fields[1] != "name=systemd" {
				continue
			}
		}
		if path != "" && path != fields[2] {
			return "", "", fmt.Errorf("ambiguous service cgroup")
		}
		path = fields[2]
	}
	if rows == 0 {
		return "", "", fmt.Errorf("empty cgroup")
	}
	unit := ""
	for _, part := range strings.Split(path, "/") {
		if strings.HasSuffix(part, ".service") && len(part) > len(".service") {
			unit = part
		}
	}
	return unit, path, nil
}

func observeService(ctx context.Context, source Source, base, executable string, limits Limits, diagnose func(string)) *inventory.ServiceObservation {
	observation := &inventory.ServiceObservation{Diagnostics: []string{}}
	fail := func(code string) { observation.Diagnostics = append(observation.Diagnostics, code); diagnose(code) }
	limit := min(limits.SourceBytes, serviceSourceBytes)
	before, beforeErr := readServiceSource(ctx, source, base+"/stat", limit)
	start, startErr := processStart(before)
	status, err := readServiceSource(ctx, source, base+"/status", limit)
	if err != nil {
		fail("service_status_unreadable")
	} else {
		observation.ProcessName, observation.EffectiveUID = parseServiceStatus(status)
		if observation.ProcessName == "" || observation.EffectiveUID == nil {
			fail("service_status_malformed")
		}
	}
	cgroup, err := readServiceSource(ctx, source, base+"/cgroup", limit)
	if err != nil {
		fail("service_cgroup_unreadable")
	} else {
		observation.UnitCandidate, observation.CgroupPath, err = serviceCgroup(cgroup)
		if err != nil || len(observation.CgroupPath) > limits.PathBytes {
			observation.UnitCandidate = ""
			observation.CgroupPath = ""
			fail("service_cgroup_malformed")
		}
	}
	after, afterErr := readServiceSource(ctx, source, base+"/stat", limit)
	end, endErr := processStart(after)
	exe, exeErr := source.Readlink(base+"/exe", limits.PathBytes)
	if beforeErr != nil || startErr != nil || afterErr != nil || endErr != nil || exeErr != nil || start != end || exe != executable || executable == "" {
		observation.ProcessName = ""
		observation.EffectiveUID = nil
		observation.UnitCandidate = ""
		observation.CgroupPath = ""
		fail("service_process_changed")
	}
	return observation
}
