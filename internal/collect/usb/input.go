package usb

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	i "github.com/pirakansa/avera/internal/inventory"
)

const (
	inputDeviceLimit    = i.MaxUSBInputDevices
	inputProcessLimit   = 32768
	inputFDLimit        = 4096
	inputOperationLimit = 262144
	inputByteLimit      = 16 << 20
	inputFileLimit      = 256 << 10
	inputConsumerLimit  = i.MaxUSBInputConsumers
)

type inputScan struct {
	ctx               context.Context
	source            InputSource
	result            *i.USBInputValues
	operations, bytes int
}

func (s *inputScan) issue(code string) {
	s.result.Status = i.Partial
	if !slices.Contains(s.result.DiagnosticCodes, code) {
		s.result.DiagnosticCodes = append(s.result.DiagnosticCodes, code)
	}
}
func (s *inputScan) operation() error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if s.operations <= 0 {
		s.issue("input_operation_limit")
		return fmt.Errorf("input operation limit")
	}
	s.operations--
	return nil
}
func (s *inputScan) read(name string, limit int) (string, error) {
	if err := s.operation(); err != nil {
		return "", err
	}
	if s.bytes <= 0 {
		s.issue("input_byte_limit")
		return "", fmt.Errorf("input byte limit")
	}
	limit = min(limit, s.bytes)
	data, err := s.source.ReadFile(name, limit)
	if err != nil {
		return "", err
	}
	if len(data) > limit || !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", fmt.Errorf("invalid input metadata")
	}
	s.bytes -= len(data)
	return strings.TrimSpace(string(data)), nil
}
func (s *inputScan) directory(name string, limit int) ([]string, error) {
	if err := s.operation(); err != nil {
		return nil, err
	}
	names, more, err := s.source.ReadDir(name, limit)
	if err != nil {
		return nil, err
	}
	if len(names) > limit {
		names = names[:limit]
		more = true
	}
	if more {
		s.issue("input_directory_limit")
	}
	slices.Sort(names)
	return names, nil
}
func (s *inputScan) resolve(name string) (string, error) {
	if err := s.operation(); err != nil {
		return "", err
	}
	v, err := s.source.Resolve(name)
	if err != nil {
		return "", err
	}
	if len(v) > i.MaxUSBInputPathBytes || !utf8.ValidString(v) || path.Clean(v) != v {
		return "", fmt.Errorf("invalid input sysfs path")
	}
	return v, nil
}
func (s *inputScan) stat(name string) (string, bool, error) {
	if err := s.operation(); err != nil {
		return "", false, err
	}
	return s.source.StatFD(name)
}
func (s *inputScan) executable(name string) (string, error) {
	if err := s.operation(); err != nil {
		return "", err
	}
	v, err := s.source.Readlink(name, i.MaxUSBInputPathBytes)
	if err != nil {
		return "", err
	}
	if !utf8.ValidString(v) || !strings.HasPrefix(v, "/") || len(v) > i.MaxUSBInputPathBytes || strings.ContainsAny(v, "\x00\r\n") {
		return "", fmt.Errorf("invalid executable metadata")
	}
	return v, nil
}

func collectInput(ctx context.Context, source InputSource, nodes []i.USBNode) *i.USBInputValues {
	result := &i.USBInputValues{Status: i.Complete, DiagnosticCodes: []string{}, Devices: []i.USBInputDevice{}}
	hasHID := false
	for _, n := range nodes {
		hasHID = hasHID || i.USBHID(n.Facts)
	}
	if !hasHID {
		return result
	}
	if source == nil {
		result.Status = i.Failed
		result.DiagnosticCodes = []string{"input_source_unavailable"}
		return result
	}
	s := inputScan{ctx: ctx, source: source, result: result, operations: inputOperationLimit, bytes: inputByteLimit}
	names, err := s.directory(inputPath, inputDeviceLimit)
	if err != nil {
		result.Status = i.Failed
		result.DiagnosticCodes = []string{"input_directory_unreadable"}
		return result
	}
	handlersText, err := s.read(inputHandlersPath, 1<<20)
	handlers := map[string][]string{}
	if err != nil {
		s.issue("input_handlers_unreadable")
	} else {
		var valid bool
		handlers, valid = parseInputHandlers(handlersText)
		if !valid {
			s.issue("input_handlers_malformed")
		}
	}
	for _, name := range names {
		if !strings.HasPrefix(name, "event") || !decimal(strings.TrimPrefix(name, "event")) {
			continue
		}
		base := inputPath + "/" + name
		location, err := s.resolve(base)
		if err != nil {
			s.issue("input_device_unreadable")
			continue
		}
		owner := ""
		prefix := ""
		for _, n := range nodes {
			if i.USBHID(n.Facts) && strings.HasPrefix(location, n.SysfsPath+"/") && len(n.SysfsPath) > len(prefix) {
				owner = n.Key
				prefix = n.SysfsPath
			}
		}
		if owner == "" {
			continue
		}
		number, err := s.read(base+"/dev", 128)
		if err != nil || !validDeviceNumber(number) {
			s.issue("input_device_number_invalid")
			continue
		}
		inputLocation := path.Dir(location)
		h, known := handlers[inputLocation]
		if !known {
			s.issue("input_handlers_unmatched")
		}
		result.Devices = append(result.Devices, i.USBInputDevice{USBKey: owner, SysfsPath: location, Node: "/dev/input/" + name, DeviceNumber: number, Handlers: append([]string{}, h...), Consumers: []i.USBInputConsumer{}})
	}
	// A kernel handler can exist without an evdev node (for example CONFIG_EVDEV=n).
	covered := map[string]bool{}
	for _, d := range result.Devices {
		covered[path.Dir(d.SysfsPath)] = true
	}
	for _, name := range names {
		if !strings.HasPrefix(name, "input") || !decimal(strings.TrimPrefix(name, "input")) {
			continue
		}
		location, err := s.resolve(inputPath + "/" + name)
		if err != nil {
			s.issue("input_device_unreadable")
			continue
		}
		if covered[location] {
			continue
		}
		owner := ""
		prefix := ""
		for _, n := range nodes {
			if i.USBHID(n.Facts) && strings.HasPrefix(location, n.SysfsPath+"/") && len(n.SysfsPath) > len(prefix) {
				owner = n.Key
				prefix = n.SysfsPath
			}
		}
		if owner == "" {
			continue
		}
		h, ok := handlers[location]
		if !ok {
			s.issue("input_handlers_unmatched")
		}
		result.Devices = append(result.Devices, i.USBInputDevice{USBKey: owner, SysfsPath: location, Node: "", DeviceNumber: "", Handlers: append([]string{}, h...), Consumers: []i.USBInputConsumer{}})
	}

	hasEvent := false
	for _, d := range result.Devices {
		hasEvent = hasEvent || d.Node != ""
	}
	if hasEvent {
		s.processes()
	}
	// A separate metadata pass rejects event-node or USB attachment replacement.
	stable := []i.USBInputDevice{}
	for _, d := range result.Devices {
		base := inputPath + "/" + path.Base(d.Node)
		if d.Node == "" {
			base = inputPath + "/" + path.Base(d.SysfsPath)
		}
		location, err := s.resolve(base)
		number := d.DeviceNumber
		var numberErr error
		if d.Node != "" {
			number, numberErr = s.read(base+"/dev", 128)
		}
		if err != nil || numberErr != nil || location != d.SysfsPath || number != d.DeviceNumber {
			s.issue("input_device_changed")
			continue
		}
		stable = append(stable, d)
	}
	result.Devices = stable
	for _, n := range nodes {
		if !i.USBHID(n.Facts) {
			continue
		}
		base := busPath + "/" + n.PortPath
		location, err := s.resolve(base)
		bus, e1 := s.read(base+"/busnum", 128)
		device, e2 := s.read(base+"/devnum", 128)
		config, e3 := s.read(base+"/bConfigurationValue", 128)
		vendor, e4 := s.read(base+"/idVendor", 128)
		product, e5 := s.read(base+"/idProduct", 128)
		if err != nil || e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || location != n.SysfsPath || bus+"/"+device+"/"+config != n.Instance || vendor != n.Facts.VendorID || product != n.Facts.ProductID {
			s.issue("input_usb_attachment_changed")
			result.Devices = slices.DeleteFunc(result.Devices, func(d i.USBInputDevice) bool { return d.USBKey == n.Key })
		}
	}
	for n := range result.Devices {
		v := i.NormalizeUSBInput(i.USBInputFacts{Status: i.Complete, Handlers: result.Devices[n].Handlers, Consumers: result.Devices[n].Consumers, DiagnosticCodes: []string{}})
		result.Devices[n].Handlers = v.Handlers
		result.Devices[n].Consumers = v.Consumers
	}
	// Keep the union for each USB judgment bounded, across all its input interfaces.
	consumerSets := map[string]map[string]bool{}
	handlerSets := map[string]map[string]bool{}
	for n := range result.Devices {
		d := &result.Devices[n]
		if consumerSets[d.USBKey] == nil {
			consumerSets[d.USBKey] = map[string]bool{}
			handlerSets[d.USBKey] = map[string]bool{}
		}
		consumers := []i.USBInputConsumer{}
		for _, c := range d.Consumers {
			data, _ := i.Canonical(c)
			key := string(data)
			if !consumerSets[d.USBKey][key] && len(consumerSets[d.USBKey]) >= inputConsumerLimit {
				s.issue("input_consumer_limit")
				continue
			}
			consumerSets[d.USBKey][key] = true
			consumers = append(consumers, c)
		}
		d.Consumers = consumers
		handlers := []string{}
		for _, h := range d.Handlers {
			if !handlerSets[d.USBKey][h] && len(handlerSets[d.USBKey]) >= i.MaxUSBInputHandlers {
				s.issue("input_handler_limit")
				continue
			}
			handlerSets[d.USBKey][h] = true
			handlers = append(handlers, h)
		}
		d.Handlers = handlers
	}

	slices.Sort(result.DiagnosticCodes)
	return result
}
func validDeviceNumber(v string) bool {
	a, b, ok := strings.Cut(v, ":")
	return ok && decimal(a) && decimal(b)
}
