package usb

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	i "github.com/pirakansa/avera/internal/inventory"
)

type inputFixtureSource struct {
	LocalInputSource
	descriptors    map[string]string
	denied         map[string]bool
	changedFD      bool
	changedProcess bool
	reads          map[string]int
	calls          []string
}

func (s *inputFixtureSource) ReadFile(name string, limit int) ([]byte, error) {
	s.calls = append(s.calls, "read:"+name)
	s.reads[name]++
	if s.denied[name] {
		return nil, os.ErrPermission
	}
	if s.changedProcess && name == "/proc/100/stat" && s.reads[name] > 1 {
		return []byte(statText("999")), nil
	}
	return s.LocalInputSource.ReadFile(name, limit)
}
func (s *inputFixtureSource) ReadDir(name string, limit int) ([]string, bool, error) {
	s.calls = append(s.calls, "dir:"+name)
	if s.denied[name] {
		return nil, false, os.ErrPermission
	}
	return s.LocalInputSource.ReadDir(name, limit)
}
func (s *inputFixtureSource) StatFD(name string) (string, bool, error) {
	s.calls = append(s.calls, "stat:"+name)
	s.reads[name]++
	if s.denied[name] {
		return "", false, os.ErrPermission
	}
	number := s.descriptors[name]
	if s.changedFD && name == "/proc/100/fd/3" && s.reads[name] > 1 {
		number = "13:99"
	}
	return number, number != "", nil
}
func statText(start string) string {
	fields := append([]string{"S"}, slices.Repeat([]string{"0"}, 18)...)
	return "100 (fictional reader) " + strings.Join(append(fields, start), " ") + "\n"
}
func inputFixture(t *testing.T) (string, Collector, *inputFixtureSource) {
	t.Helper()
	root, c := fixture(t)
	deviceBase := "/sys/devices/pci/usb1/1-2/1-2.1/1-2.1:1.0/input/input8"
	write(t, root, deviceBase+"/event5/dev", "13:69\n")
	for name, target := range map[string]string{"event5": deviceBase + "/event5", "input8": deviceBase} {
		file := filepath.Join(root, inputPath, name)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, target), file); err != nil {
			t.Fatal(err)
		}
	}
	write(t, root, inputHandlersPath, "I: Bus=0003 Vendor=1234 Product=0001 Version=0100\nS: Sysfs="+strings.TrimPrefix(deviceBase, "/sys")+"\nH: Handlers=kbd event5\n\n")
	write(t, root, "/proc/100/stat", statText("123"))
	write(t, root, "/proc/100/status", "Name:\tfictional-Xorg\nUid:\t1000 0 1000 1000\nGid:\t1000 44 1000 1000\nGroups:\t44 1000\nUnrelated: DO-NOT-RETAIN\n")
	write(t, root, "/proc/100/fd/3", "not input data")
	write(t, root, "/proc/100/fd/4", "not input data")
	if err := os.Symlink("/usr/bin/fictional-Xorg", filepath.Join(root, "proc/100/exe")); err != nil {
		t.Fatal(err)
	}
	source := &inputFixtureSource{LocalInputSource: LocalInputSource{LocalSource{Root: root}}, descriptors: map[string]string{"/proc/100/fd/3": "13:69", "/proc/100/fd/4": "13:69"}, denied: map[string]bool{}, reads: map[string]int{}}
	c.InputSource = source
	return root, c, source
}
func TestInputRouteAndRuntimePrivileges(t *testing.T) {
	_, c, source := inputFixture(t)
	r := collect(t, c)
	if r.Coverage.Status != i.Complete {
		t.Fatal(r.Coverage)
	}
	input := r.Evidence[0].Values.(i.USBValues).Input
	if input == nil || input.Status != i.Complete || len(input.Devices) != 1 {
		t.Fatal(input)
	}
	device := input.Devices[0]
	if device.Node != "/dev/input/event5" || !slices.Contains(device.Handlers, "kbd") || !slices.Contains(device.Handlers, "evdev") || len(device.Consumers) != 1 {
		t.Fatal(device)
	}
	consumer := device.Consumers[0]
	if consumer.EffectiveUID != 0 || consumer.EffectiveGID != 44 || consumer.ProcessName != "fictional-Xorg" || consumer.Executable != "/usr/bin/fictional-Xorg" || !slices.Equal(consumer.SupplementaryGroups, []uint32{44, 1000}) {
		t.Fatal(consumer)
	}
	for _, call := range source.calls {
		if strings.HasPrefix(call, "read:") && (strings.Contains(call, "/fd/") || strings.HasSuffix(call, "/exe")) {
			t.Fatal("opened input/executable content", call)
		}
	}
	value, err := i.Canonical(input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(value), "DO-NOT-RETAIN") || strings.Contains(string(value), "\"pid\"") {
		t.Fatal("unrelated process data retained")
	}
}
func TestInputFailuresAndRacesAreExplicit(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*inputFixtureSource)
		code   string
	}{
		{"permission", func(s *inputFixtureSource) { s.denied["/proc/100/fd"] = true }, "input_descriptors_unreadable"},
		{"descriptor change", func(s *inputFixtureSource) { s.changedFD = true; delete(s.descriptors, "/proc/100/fd/4") }, "input_descriptor_changed"},
		{"pid reuse", func(s *inputFixtureSource) { s.changedProcess = true }, "input_consumer_changed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, c, source := inputFixture(t)
			test.mutate(source)
			r := collect(t, c)
			input := r.Evidence[0].Values.(i.USBValues).Input
			if r.Coverage.Status != i.Partial || input.Status != i.Partial || !slices.Contains(input.DiagnosticCodes, test.code) {
				t.Fatal(r.Coverage, input)
			}
			for _, device := range input.Devices {
				if len(device.Consumers) != 0 {
					t.Fatal("unstable consumer retained", device)
				}
			}
		})
	}
}
func TestKernelOnlyAndNoOpenReaders(t *testing.T) {
	root, c, source := inputFixture(t)
	source.descriptors = map[string]string{}
	r := collect(t, c)
	input := r.Evidence[0].Values.(i.USBValues).Input
	if input.Status != i.Complete || len(input.Devices[0].Consumers) != 0 {
		t.Fatal(input)
	}
	for _, name := range []string{inputPath + "/event5", "/sys/devices/pci/usb1/1-2/1-2.1/1-2.1:1.0/input/input8/event5"} {
		if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	write(t, root, inputHandlersPath, "S: Sysfs=/devices/pci/usb1/1-2/1-2.1/1-2.1:1.0/input/input8\nH: Handlers=kbd\n\n")
	r = collect(t, c)
	input = r.Evidence[0].Values.(i.USBValues).Input
	if input.Status != i.Complete || len(input.Devices) != 1 || input.Devices[0].Node != "" || !slices.Equal(input.Devices[0].Handlers, []string{"kbd"}) {
		t.Fatal(input)
	}
}
func TestInputSourceNeverOpensDescriptors(t *testing.T) {
	root, c, _ := inputFixture(t)
	source := LocalInputSource{LocalSource{Root: root}}
	for _, name := range []string{"/dev/input/event5", "/proc/100/fd/3", "/proc/100/exe", "/proc/100/environ", "/proc/100/mem"} {
		if _, err := source.ReadFile(name, 100); err == nil {
			t.Fatal("content read allowed", name)
		}
	}
	if _, _, err := source.StatFD("/dev/input/event5"); err == nil {
		t.Fatal("non-proc descriptor accepted")
	}
	if _, char, err := source.StatFD("/proc/100/fd/3"); err != nil || char {
		t.Fatal("regular file misclassified", char, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Collect(ctx, "fictional"); err == nil {
		t.Fatal("cancel ignored")
	}
}
func TestInputHandlerParsingRejectsAmbiguity(t *testing.T) {
	for _, text := range []string{"S: Sysfs=/devices/test\nH: Handlers=kbd\nH: Handlers=event5", "S: Sysfs=/devices/../outside\nH: Handlers=kbd"} {
		if _, ok := parseInputHandlers(text); ok {
			t.Fatal("ambiguous handler source accepted", text)
		}
	}
	if _, err := parseInputConsumer("Name: reader\nUid: 0 0 0 0\nGid: 0 0 0 0\nGroups: bad", "/usr/bin/reader"); err == nil {
		t.Fatal("malformed privileges accepted")
	}
}

func TestInputRenumberingPreservesJudgmentBasis(t *testing.T) {
	root, c, source := inputFixture(t)
	before := collect(t, c)
	base := "/sys/devices/pci/usb1/1-2/1-2.1/1-2.1:1.0/input/input8"
	if err := os.Remove(filepath.Join(root, inputPath, "event5")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, base, "event5"), filepath.Join(root, base, "event9")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, base, "event9"), filepath.Join(root, inputPath, "event9")); err != nil {
		t.Fatal(err)
	}
	write(t, root, inputHandlersPath, "S: Sysfs="+strings.TrimPrefix(base, "/sys")+"\nH: Handlers=kbd event9\n\n")
	if err := os.Rename(filepath.Join(root, "proc/100"), filepath.Join(root, "proc/200")); err != nil {
		t.Fatal(err)
	}
	source.descriptors = map[string]string{"/proc/200/fd/3": "13:69", "/proc/200/fd/4": "13:69"}
	after := collect(t, c)
	if after.Coverage.Status != i.Complete {
		t.Fatal(after.Coverage)
	}
	for _, a := range before.Findings {
		for _, b := range after.Findings {
			if a.FindingID == b.FindingID && a.FactFingerprint != b.FactFingerprint {
				t.Fatal("volatile names changed the judgment basis")
			}
		}
	}
	write(t, root, "/proc/200/status", "Name: fictional-Xorg\nUid: 1000 1001 1000 1000\nGid: 1000 44 1000 1000\nGroups: 44 1000\n")
	changed := collect(t, c)
	found := false
	for _, a := range before.Findings {
		for _, b := range changed.Findings {
			if a.FindingID == b.FindingID && a.FactFingerprint != b.FactFingerprint {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("changed effective UID did not change judgment basis")
	}
}

func TestInputMetadataBudgetsAreExplicit(t *testing.T) {
	_, _, source := inputFixture(t)
	result := &i.USBInputValues{Status: i.Complete, DiagnosticCodes: []string{}, Devices: []i.USBInputDevice{}}
	scan := inputScan{ctx: context.Background(), source: source, result: result, operations: 0, bytes: 1024}
	if _, err := scan.read("/proc/100/status", 100); err == nil || !slices.Contains(result.DiagnosticCodes, "input_operation_limit") {
		t.Fatal("operation budget not reported")
	}
	scan.operations = 1
	scan.bytes = 0
	if _, err := scan.read("/proc/100/status", 100); err == nil || !slices.Contains(result.DiagnosticCodes, "input_byte_limit") {
		t.Fatal("byte budget not reported")
	}
}
