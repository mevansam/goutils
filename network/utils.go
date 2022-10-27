package network

import "net/netip"

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
