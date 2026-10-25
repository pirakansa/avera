package inventory

import (
	"testing"
)

func usbNode() USBNode {
	return USBNode{Key: "test", PortPath: "1-2", SysfsPath: "/sys/devices/fictional/usb1/1-2", Instance: "1/2/1", Role: "device", Facts: USBFacts{VendorID: "1234", ProductID: "5678", DeviceClass: "00", Interfaces: []USBInterface{{Number: "00", Class: "03", Subclass: "01", Protocol: "02"}}, StorageFingerprints: []string{}, ObservationComplete: true}}
}
func TestUSBTopologyRejectsCyclesAndUnknownParents(t *testing.T) {
	node := usbNode()
	node.ParentKey = node.Key
	if err := validUSBNodes([]USBNode{node}); err == nil {
		t.Fatal("cycle accepted")
	}
	node.ParentKey = "missing"
	if err := validUSBNodes([]USBNode{node}); err == nil {
		t.Fatal("unknown parent accepted")
	}
	node.ParentKey = ""
	if err := validUSBNodes([]USBNode{node, node}); err == nil {
		t.Fatal("duplicate key accepted")
	}
	node.Facts.Interfaces = append(node.Facts.Interfaces, node.Facts.Interfaces[0])
	if err := validUSBNodes([]USBNode{node}); err == nil {
		t.Fatal("duplicate interface accepted")
	}
}
func TestUSBAttachmentNeedsAllInstanceFields(t *testing.T) {
	node := usbNode()
	finding, err := NormalizeFinding("test", Finding{Collector: USB, Kind: USBDevice, Identity: USBIdentity{Key: node.Key}, Facts: node.Facts, EvidenceIDs: []string{"usb"}})
	if err != nil {
		t.Fatal(err)
	}
	storage := Finding{FindingID: "storage", Kind: ExternalStorage, EvidenceIDs: []string{"block"}}
	link := &USBLink{SysfsPath: node.SysfsPath, Instance: node.Instance, VendorID: node.Facts.VendorID, ProductID: node.Facts.ProductID}
	s := &Snapshot{Findings: []Finding{finding, storage}, Evidence: []Evidence{{EvidenceID: "usb", Values: USBValues{Nodes: []USBNode{node}}}, {EvidenceID: "block", Values: FileValues{USB: link}}}}
	if USBStorageOwners(s)["storage"] != finding.FindingID {
		t.Fatal("matching attachment not grouped")
	}
	link.Instance = "1/3/1"
	if len(USBStorageOwners(s)) != 0 {
		t.Fatal("different instance grouped")
	}
	link.Instance = node.Instance
	link.VendorID = "ffff"
	if len(USBStorageOwners(s)) != 0 {
		t.Fatal("different descriptors grouped")
	}
}
