package usb

import (
	"fmt"
	i "github.com/pirakansa/avera/internal/inventory"
	"slices"
	"strconv"
	"strings"
)

func processStart(text string) (string, error) {
	end := strings.LastIndex(text, ")")
	if end < 0 {
		return "", fmt.Errorf("invalid process stat")
	}
	fields := strings.Fields(text[end+1:])
	if len(fields) < 20 {
		return "", fmt.Errorf("invalid process stat")
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", err
	}
	return fields[19], nil
}
func parseInputConsumer(text, exe string) (i.USBInputConsumer, error) {
	c := i.USBInputConsumer{Executable: exe, SupplementaryGroups: []uint32{}}
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || !slices.Contains([]string{"Name", "Uid", "Gid", "Groups"}, key) {
			continue
		}
		if seen[key] {
			return c, fmt.Errorf("duplicate process metadata")
		}
		seen[key] = true
		fields := strings.Fields(value)
		if key == "Name" {
			c.ProcessName = strings.TrimSpace(value)
			continue
		}
		if key != "Groups" && len(fields) != 4 {
			return c, fmt.Errorf("invalid process identity")
		}
		if key == "Groups" && len(fields) > i.MaxUSBInputGroups {
			return c, fmt.Errorf("group limit")
		}
		for n, f := range fields {
			v, err := strconv.ParseUint(f, 10, 32)
			if err != nil {
				return c, err
			}
			switch {
			case key == "Uid" && n == 1:
				c.EffectiveUID = uint32(v)
			case key == "Gid" && n == 1:
				c.EffectiveGID = uint32(v)
			case key == "Groups":
				c.SupplementaryGroups = append(c.SupplementaryGroups, uint32(v))
			}
		}
	}
	if len(seen) != 4 || c.ProcessName == "" || len(c.ProcessName) > i.MaxUSBInputNameBytes {
		return c, fmt.Errorf("incomplete process metadata")
	}
	c = i.NormalizeUSBInput(i.USBInputFacts{Consumers: []i.USBInputConsumer{c}}).Consumers[0]
	return c, nil
}
func (s *inputScan) processes() {
	byNumber := map[string][]int{}
	for n, d := range s.result.Devices {
		if d.DeviceNumber != "" {
			byNumber[d.DeviceNumber] = append(byNumber[d.DeviceNumber], n)
		}
	}
	pids, err := s.directory("/proc", inputProcessLimit)
	if err != nil {
		s.issue("input_processes_unreadable")
		return
	}
	for _, pid := range pids {
		if !decimal(pid) {
			continue
		}
		if s.operations <= 0 || s.ctx.Err() != nil {
			s.issue("input_process_scan_incomplete")
			break
		}
		base := "/proc/" + pid
		stat, err := s.read(base+"/stat", inputFileLimit)
		if err != nil {
			s.issue("input_process_unreadable")
			continue
		}
		start, err := processStart(stat)
		if err != nil {
			s.issue("input_process_malformed")
			continue
		}
		fds, err := s.directory(base+"/fd", inputFDLimit)
		if err != nil {
			s.issue("input_descriptors_unreadable")
			continue
		}
		matches := map[string]string{}
		for _, fd := range fds {
			if !decimal(fd) {
				s.issue("input_descriptor_malformed")
				continue
			}
			number, char, err := s.stat(base + "/fd/" + fd)
			if err != nil {
				s.issue("input_descriptor_unreadable")
				continue
			}
			if char && len(byNumber[number]) > 0 {
				matches[fd] = number
			}
		}
		if len(matches) == 0 {
			continue
		}
		status, err := s.read(base+"/status", inputFileLimit)
		if err != nil {
			s.issue("input_consumer_unreadable")
			continue
		}
		exe, err := s.executable(base + "/exe")
		if err != nil {
			s.issue("input_executable_unreadable")
			continue
		}
		consumer, err := parseInputConsumer(status, exe)
		if err != nil {
			s.issue("input_consumer_malformed")
			continue
		}
		after, err := s.read(base+"/stat", inputFileLimit)
		checked, parseErr := processStart(after)
		statusAfter, statusErr := s.read(base+"/status", inputFileLimit)
		exeAfter, exeErr := s.executable(base + "/exe")
		consumerAfter, consumerErr := parseInputConsumer(statusAfter, exeAfter)
		left, _ := i.Canonical(consumer)
		right, _ := i.Canonical(consumerAfter)
		if err != nil || parseErr != nil || checked != start || statusErr != nil || exeErr != nil || consumerErr != nil || string(left) != string(right) {
			s.issue("input_consumer_changed")
			continue
		}
		for fd, number := range matches {
			again, char, err := s.stat(base + "/fd/" + fd)
			if err != nil || !char || again != number {
				s.issue("input_descriptor_changed")
				continue
			}
			for _, index := range byNumber[number] {
				d := &s.result.Devices[index]
				duplicate := false
				for _, prior := range d.Consumers {
					a, _ := i.Canonical(prior)
					b, _ := i.Canonical(consumer)
					if string(a) == string(b) {
						duplicate = true
						break
					}
				}
				if duplicate {
					continue
				}
				if len(d.Consumers) >= inputConsumerLimit {
					s.issue("input_consumer_limit")
					continue
				}
				d.Consumers = append(d.Consumers, consumer)
			}
		}
	}
}
