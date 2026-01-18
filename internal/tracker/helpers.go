package tracker

import (
	"net"
	"strconv"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func parseAddrToPeerInfo(addr string) (protocol.PeerInfo, bool) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return protocol.PeerInfo{}, false
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return protocol.PeerInfo{}, false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return protocol.PeerInfo{}, false
	}

	var ipBytes [16]byte
	copy(ipBytes[:], ip.To16())

	return protocol.PeerInfo{
		IP:   ipBytes,
		Port: uint16(port),
	}, true
}
