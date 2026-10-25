package usb

import (
	i "github.com/pirakansa/avera/internal/inventory"
	"path"
	"strings"
)

func parseInputHandlers(text string) (map[string][]string, bool) {
	out := map[string][]string{}
	valid := true
	for _, block := range strings.Split(text, "\n\n") {
		if strings.TrimSpace(block) == "" {
			continue
		}
		location := ""
		handlers := []string{}
		have := false
		sysfsCount, handlerCount := 0, 0
		for _, line := range strings.Split(block, "\n") {
			if v, ok := strings.CutPrefix(line, "S: Sysfs="); ok {
				location = "/sys" + strings.TrimSpace(v)
				sysfsCount++
			}
			if v, ok := strings.CutPrefix(line, "H: Handlers="); ok {
				have = true
				handlerCount++
				for _, h := range strings.Fields(v) {
					if strings.HasPrefix(h, "event") && decimal(strings.TrimPrefix(h, "event")) {
						h = "evdev"
					} else if (strings.HasPrefix(h, "mouse") && decimal(strings.TrimPrefix(h, "mouse"))) || (strings.HasPrefix(h, "js") && decimal(strings.TrimPrefix(h, "js"))) {
						continue
					}
					if len(h) > i.MaxUSBInputHandlers || strings.ContainsAny(h, "/\x00") {
						valid = false
						continue
					}
					handlers = append(handlers, h)
				}
			}
		}
		if !strings.HasPrefix(location, "/sys/devices/") || path.Clean(location) != location || !have || sysfsCount != 1 || handlerCount != 1 || out[location] != nil || len(handlers) > i.MaxUSBInputHandlers {
			valid = false
			continue
		}
		out[location] = i.NormalizeUSBInput(i.USBInputFacts{Handlers: handlers}).Handlers
	}
	return out, valid
}
