//go:build networklive && linux && amd64

package network

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/inventory"
	"golang.org/x/sys/unix"
)

// TestLiveUbuntuEndpoints is an explicit opt-in test for a disposable Ubuntu
// container. Never run it with host networking or on a product host. It creates
// sockets; the production collector remains a read-only observer.
func TestLiveUbuntuEndpoints(t *testing.T) {
	if os.Getenv("AVERA_NETWORK_LIVE") != "disposable-ubuntu" {
		t.Skip("requires explicit disposable Ubuntu acknowledgement")
	}
	// The documented container command runs this binary as PID 1 with only a
	// loopback interface. Refuse ordinary host execution before opening sockets.
	if os.Getpid() != 1 || os.Geteuid() != 0 {
		t.Fatal("run as root PID 1 in the documented disposable container")
	}
	release, err := os.ReadFile("/etc/os-release")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(release), "\nID=ubuntu\n") {
		t.Fatal("the validation target must be Ubuntu")
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(interfaces) != 1 || interfaces[0].Flags&net.FlagLoopback == 0 {
		t.Fatal("requires a dedicated network namespace with only loopback (--network none)")
	}
	// Docker's none driver may leave ::1 unconfigured. Configure only the
	// disposable namespace's loopback; NET_ADMIN is needed only for this step.
	if err := ensureLiveIPv6Loopback(interfaces[0]); err != nil {
		t.Fatal(err)
	}
	kernel, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("OS: %s; kernel: %s; architecture: %s; Go: %s", strings.TrimSpace(string(release)), strings.TrimSpace(string(kernel)), runtime.GOARCH, runtime.Version())

	closeAfter := func(c io.Closer) {
		t.Cleanup(func() {
			if err := c.Close(); err != nil {
				t.Errorf("close test socket: %v", err)
			}
		})
	}
	key := func(transport, family, address string, port int) inventory.EndpointIdentity {
		category := "listener"
		if transport == "udp" {
			category = "bound_unconnected"
		}
		return inventory.EndpointIdentity{Transport: transport, Family: family, LocalAddress: address, LocalPort: uint16(port), ScopeID: "disposable-live-test", SocketCategory: category}
	}
	expected := map[inventory.EndpointIdentity]bool{}
	for _, sample := range []struct{ family, suffix, local, peer string }{
		{"ipv4", "4", "0.0.0.0", "127.0.0.1"},
		{"ipv4", "4", "127.0.0.1", "127.0.0.1"},
		{"ipv6", "6", "::", "::1"},
		{"ipv6", "6", "::1", "::1"},
	} {
		listener, err := net.ListenTCP("tcp"+sample.suffix, &net.TCPAddr{IP: net.ParseIP(sample.local)})
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(listener)
		port := listener.Addr().(*net.TCPAddr).Port
		expected[key("tcp", sample.family, sample.local, port)] = true
		if err := listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		client, err := net.DialTimeout("tcp"+sample.suffix, net.JoinHostPort(sample.peer, strconv.Itoa(port)), 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(client)
		accepted, err := listener.AcceptTCP()
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(accepted)

		udp, err := net.ListenUDP("udp"+sample.suffix, &net.UDPAddr{IP: net.ParseIP(sample.local)})
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(udp)
		udpPort := udp.LocalAddr().(*net.UDPAddr).Port
		expected[key("udp", sample.family, sample.local, udpPort)] = true
		connected, err := net.DialUDP("udp"+sample.suffix, nil, &net.UDPAddr{IP: net.ParseIP(sample.peer), Port: udpPort})
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(connected)
		// No UDP datagram is sent. UDP connect only sets the local peer state.
	}
	for _, family := range []int{unix.AF_INET, unix.AF_INET6} {
		fd, err := unix.Socket(family, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := unix.Close(fd); err != nil {
				t.Errorf("close unbound socket: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := (Collector{ScopeID: "disposable-live-test"}).Collect(ctx, "disposable-ubuntu")
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage.Status != inventory.Complete {
		t.Fatalf("incomplete coverage: %+v", result.Coverage)
	}
	if len(result.Findings) != len(expected) {
		t.Fatalf("got %d endpoints, want %d; findings: %+v", len(result.Findings), len(expected), result.Findings)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Findings {
		identity := f.Identity.(inventory.EndpointIdentity)
		if !expected[identity] {
			t.Errorf("unexpected endpoint: %+v", identity)
		}
		delete(expected, identity)
		facts := f.Facts.(inventory.NetworkFacts)
		if !facts.AttributionComplete || len(facts.Owners) != 1 || facts.Owners[0].Executable != executable || facts.Owners[0].UID != 0 {
			t.Errorf("unexpected attribution for %+v: %+v", identity, facts)
		}
	}
	if len(expected) != 0 {
		t.Errorf("missing endpoints: %+v", expected)
	}
	t.Log("matched all 8 wildcard/loopback TCP/UDP endpoints and their owners; established TCP, connected UDP and unbound UDP did not add findings")
}

func ensureLiveIPv6Loopback(loopback net.Interface) (resultErr error) {
	addresses, err := loopback.Addrs()
	if err != nil {
		return err
	}
	for _, a := range addresses {
		if a.String() == "::1/128" {
			return nil
		}
	}
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_ROUTE)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, unix.Close(fd)) }()
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &unix.Timeval{Sec: 5}); err != nil {
		return err
	}
	// RTM_NEWADDR: netlink header + ifaddrmsg + IFA_ADDRESS (::1).
	const headerBytes, addressBytes, attributeBytes = 16, 8, 20
	message := make([]byte, headerBytes+addressBytes+attributeBytes)
	binary.NativeEndian.PutUint32(message[0:4], uint32(len(message)))
	binary.NativeEndian.PutUint16(message[4:6], unix.RTM_NEWADDR)
	binary.NativeEndian.PutUint16(message[6:8], unix.NLM_F_REQUEST|unix.NLM_F_ACK|unix.NLM_F_CREATE|unix.NLM_F_EXCL)
	binary.NativeEndian.PutUint32(message[8:12], 1)
	message[16], message[17], message[19] = unix.AF_INET6, 128, unix.RT_SCOPE_HOST
	binary.NativeEndian.PutUint32(message[20:24], uint32(loopback.Index))
	binary.NativeEndian.PutUint16(message[24:26], attributeBytes)
	binary.NativeEndian.PutUint16(message[26:28], unix.IFA_ADDRESS)
	copy(message[28:], net.ParseIP("::1").To16())
	if err := unix.Sendto(fd, message, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return err
	}
	ack := make([]byte, 4096)
	n, _, err := unix.Recvfrom(fd, ack, 0)
	if err != nil {
		return err
	}
	if n < headerBytes+4 || binary.NativeEndian.Uint16(ack[4:6]) != unix.NLMSG_ERROR || binary.NativeEndian.Uint32(ack[8:12]) != 1 {
		return fmt.Errorf("invalid loopback address acknowledgement")
	}
	code := int32(binary.NativeEndian.Uint32(ack[16:20]))
	if code == 0 || code == -int32(unix.EEXIST) {
		return nil
	}
	return fmt.Errorf("configure disposable IPv6 loopback (requires container NET_ADMIN and enabled IPv6): %w", unix.Errno(-code))
}
