package network

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/pirakansa/avera/internal/inventory"
)

const (
	tcpListen      = "0A"
	udpUnconnected = "07"
)

type socket struct {
	identity inventory.EndpointIdentity
	uid      uint32
	inode    string
	location string
}

// proc addresses are printed as native-endian 32-bit words, including IPv6.
// The initial runtime backend supports linux/amd64 (little endian) only.
// Verified against upstream Linux v6.8 TCP/UDP formatters; references and the
// outstanding Ubuntu execution comparison are recorded in network-collector.md.
func address(raw string, family string) (string, uint16, error) {
	parts := strings.Split(raw, ":")
	width := 8
	if family == "ipv6" {
		width = 32
	}
	if len(parts) != 2 || len(parts[0]) != width || len(parts[1]) != 4 {
		return "", 0, fmt.Errorf("invalid endpoint")
	}
	b, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", 0, err
	}
	for i := 0; i < len(b); i += 4 {
		v := binary.BigEndian.Uint32(b[i : i+4])
		binary.LittleEndian.PutUint32(b[i:i+4], v)
	}
	a, ok := netip.AddrFromSlice(b)
	if !ok {
		return "", 0, fmt.Errorf("invalid address")
	}
	p, err := strconv.ParseUint(parts[1], 16, 16)
	return a.String(), uint16(p), err
}

func parseRow(line, transport, family, scope, location string) (socket, bool, error) {
	f := strings.Fields(line)
	bad := func() (socket, bool, error) { return socket{}, false, fmt.Errorf("malformed socket row") }
	if len(f) < 10 || !strings.HasSuffix(f[0], ":") {
		return bad()
	}
	if _, err := strconv.ParseUint(strings.TrimSuffix(f[0], ":"), 10, 32); err != nil {
		return bad()
	}
	local, port, err := address(f[1], family)
	if err != nil {
		return bad()
	}
	remote, remotePort, err := address(f[2], family)
	if err != nil {
		return bad()
	}
	if len(f[3]) != 2 {
		return bad()
	}
	if _, err := strconv.ParseUint(f[3], 16, 8); err != nil {
		return bad()
	}
	uid, err := strconv.ParseUint(f[7], 10, 32)
	if err != nil {
		return bad()
	}
	inode, err := strconv.ParseUint(f[9], 10, 64)
	if err != nil {
		return bad()
	}
	category := "listener"
	include := f[3] == tcpListen
	if transport == "udp" {
		category = "bound_unconnected"
		r, _ := netip.ParseAddr(remote)
		include = f[3] == udpUnconnected && r.IsUnspecified() && remotePort == 0 && port != 0
	}
	return socket{inventory.EndpointIdentity{Transport: transport, Family: family, LocalAddress: local, LocalPort: port, ScopeID: scope, SocketCategory: category}, uint32(uid), strconv.FormatUint(inode, 10), location}, include, nil
}
