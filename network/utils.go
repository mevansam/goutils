package network

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/mevansam/goutils/logger"
)

var privateNetworks = []netip.Prefix{
	// Private or RFC 1918 address space
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	// Unique Local Addresses (RFC 4193)
	netip.MustParsePrefix("fc00::/7"),
	// Link-Local Addresses
	netip.MustParsePrefix("fe80::/10"),
}

func IsPrivateAddr(addr netip.Addr) bool {
	for _, privateNetwork := range privateNetworks {
		if privateNetwork.Contains(addr) {
			return true
		}
	}
	return false
}

// test tcp connection
func CanConnect(host string, port int) bool {

	endpoint := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", endpoint, time.Second)
	if err != nil {
		logger.TraceMessage(
			"Connectivity test to '%s' failed: %s",
			endpoint, err.Error(),
		)
		return false
	}

	defer conn.Close()
	return true
}
