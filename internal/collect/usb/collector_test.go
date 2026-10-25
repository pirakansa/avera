package usb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	i "github.com/pirakansa/avera/internal/inventory"
)

func write(t *testing.T, root, name, value string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}
func device(t *testing.T, root, port, location, class, serial string, interfaces []i.USBInterface) {
	t.Helper()
	attributes := map[string]string{"idVendor": "1234", "idProduct": "0001", "manufacturer": "Fictional", "product": "Fictional device", "serial": serial, "bDeviceClass": class, "busnum": "1", "devnum": "2", "bConfigurationValue": "1", "bNumInterfaces": fmt.Sprint(len(interfaces))}
	for name, value := range attributes {
		write(t, root, location+"/"+name, value+"\n")
	}
	target := filepath.Join(root, busPath, port)
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, location), target); err != nil {
		t.Fatal(err)
	}
	for n, v := range interfaces {
		name := port
		if strings.HasPrefix(port, "usb") {
			name = strings.TrimPrefix(port, "usb") + "-0"
		}
		name += fmt.Sprintf(":1.%d", n)
		for attr, value := range map[string]string{"bInterfaceNumber": v.Number, "bInterfaceClass": v.Class, "bInterfaceSubClass": v.Subclass, "bInterfaceProtocol": v.Protocol} {
			write(t, root, busPath+"/"+name+"/"+attr, value+"\n")
		}
	}
}
func fixture(t *testing.T) (string, Collector) {
	t.Helper()
	root := t.TempDir()
	hub := []i.USBInterface{{Number: "00", Class: "09", Subclass: "00", Protocol: "00"}}
	device(t, root, "usb1", "/sys/devices/pci/usb1", "09", "", hub)
	device(t, root, "1-2", "/sys/devices/pci/usb1/1-2", "09", "", hub)
	device(t, root, "1-2.1", "/sys/devices/pci/usb1/1-2/1-2.1", "00", "mouse", []i.USBInterface{{Number: "00", Class: "03", Subclass: "01", Protocol: "02"}})
	device(t, root, "1-2.2", "/sys/devices/pci/usb1/1-2/1-2.2", "00", "composite", []i.USBInterface{{Number: "00", Class: "03", Subclass: "00", Protocol: "00"}, {Number: "01", Class: "08", Subclass: "06", Protocol: "50"}})
	device(t, root, "1-3", "/sys/devices/pci/usb1/1-3", "00", "storage", []i.USBInterface{{Number: "00", Class: "08", Subclass: "06", Protocol: "50"}})
	return root, Collector{Source: LocalSource{Root: root}, ScopeID: "test", Now: func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }}
}
func collect(t *testing.T, c Collector) i.CollectorResult {
	t.Helper()
	r, err := c.Collect(context.Background(), "fictional")
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestHIDAndTopology(t *testing.T) {
	_, c := fixture(t)
	r := collect(t, c)
	if r.Coverage.Status != i.Complete || len(r.Findings) != 2 {
		t.Fatalf("%+v findings %d", r.Coverage, len(r.Findings))
	}
	nodes := r.Evidence[0].Values.(i.USBValues).Nodes
	if len(nodes) != 5 {
		t.Fatal(nodes)
	}
	byPort := map[string]i.USBNode{}
	for _, n := range nodes {
		byPort[n.PortPath] = n
	}
	if byPort["1-2.1"].ParentKey != byPort["1-2"].Key || byPort["1-2"].ParentKey != byPort["usb1"].Key {
		t.Fatal("attachment parents lost")
	}
	for _, f := range r.Findings {
		if !i.USBHID(f.Facts.(i.USBFacts)) || !f.Facts.(i.USBFacts).ObservationComplete {
			t.Fatal(f)
		}
	}
}
func TestIncompleteAndUnavailable(t *testing.T) {
	root, c := fixture(t)
	if err := os.Remove(filepath.Join(root, busPath, "1-2.2:1.1/bInterfaceClass")); err != nil {
		t.Fatal(err)
	}
	r := collect(t, c)
	if r.Coverage.Status != i.Partial {
		t.Fatal(r.Coverage)
	}
	c.Source = LocalSource{Root: t.TempDir()}
	if r = collect(t, c); r.Coverage.Status != i.Failed {
		t.Fatal(r.Coverage)
	}
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, busPath), 0700); err != nil {
		t.Fatal(err)
	}
	c.Source = LocalSource{Root: root}
	r = collect(t, c)
	if r.Coverage.Status != i.Complete || len(r.Findings) != 0 {
		t.Fatal(r.Coverage)
	}
}
func TestMissingSerialDoesNotReuseIdentity(t *testing.T) {
	root, c := fixture(t)
	write(t, root, "/sys/devices/pci/usb1/1-2/1-2.1/serial", "")
	first := collect(t, c)
	c.ScopeID = "next"
	second := collect(t, c)
	for _, a := range first.Findings {
		if strings.HasPrefix(a.Identity.(i.USBIdentity).Key, "unverified-usb:") {
			for _, b := range second.Findings {
				if a.FindingID == b.FindingID {
					t.Fatal("transient identifier reused")
				}
			}
		}
	}
}
func TestLocalSourceBoundariesAndCancellation(t *testing.T) {
	root, c := fixture(t)
	s := LocalSource{Root: root}
	if _, err := s.ReadFile("/dev/hidraw0", 10); err == nil {
		t.Fatal("device read allowed")
	}
	if _, err := s.ReadFile(busPath+"/1-2.1/product", 2); err == nil {
		t.Fatal("oversized read allowed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Collect(ctx, "fictional"); err == nil {
		t.Fatal("cancellation lost")
	}
	outside := t.TempDir()
	p := filepath.Join(root, busPath, "escaped")
	if err := os.Symlink(outside, p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(busPath + "/escaped"); err == nil {
		t.Fatal("root escape allowed")
	}
}

type changedSource struct {
	Source
	reads int
}

func (s *changedSource) ReadFile(name string, limit int) ([]byte, error) {
	if strings.HasSuffix(name, "/1-2.1/devnum") {
		s.reads++
		if s.reads > 1 {
			return []byte("9\n"), nil
		}
	}
	return s.Source.ReadFile(name, limit)
}
func TestAttachmentChangeIsNotPublished(t *testing.T) {
	_, c := fixture(t)
	c.Source = &changedSource{Source: c.Source}
	r := collect(t, c)
	if r.Coverage.Status != i.Partial || len(r.Findings) != 1 {
		t.Fatal(r.Coverage, len(r.Findings))
	}
	for _, n := range r.Evidence[0].Values.(i.USBValues).Nodes {
		if n.PortPath == "1-2.1" {
			t.Fatal("changed attachment published")
		}
	}
}
