package network

import "net/netip"

type NetworkContext interface {	
	DefaultDeviceName() string
	DefaultInterface() string
	DefaultGateway() string
	DefaultIP() string

	DisableIPv6() error

	NewDNSManager() (DNSManager, error)
	NewRouteManager() (RouteManager, error)

	Clear()
}

type DNSManager interface {
	AddDNSServers(servers []string) error
	AddSearchDomains(domains []string) error

	Clear()
}

type RouteManager interface {
	GetDefaultInterface() (RoutableInterface, error)
	GetRoutableInterface(ifaceName string) (RoutableInterface, error)
	NewRoutableInterface(ifaceName, tunAddress string) (RoutableInterface, error)
	AddExternalRouteToIPs(ips []string) error
	AddDefaultRoute(gateway string) error

	// BlackListIPs(ips []netip.Addr) error
	// DeleteBlackListedIPs(ips []netip.Addr) error
	// WhiteListIPs(ips []netip.Addr) error
	// DeleteWhiteListedIPs(ips []netip.Addr) error
	
	Clear()
}

type RoutableInterface interface {
	Address4() (string, string, error)
	Address6() (string, string, error)
	MakeDefaultRoute() error

	// SetSecurityGroups() error
	// DeleteSecurityGroups() error

	ForwardPortTo(srcPort int, dstPort int, dstIP netip.Addr) error
	DeletePortForwardedTo(srcPort int, dstPort int, dstIP netip.Addr) error

	FowardTrafficTo(dstItf RoutableInterface, srcNetwork, dstNetwork string, nat bool) error
	DeleteTrafficForwardedTo(dstItf RoutableInterface, srcNetwork, dstNetwork string, nat bool) error
	FowardTrafficFrom(srcItf RoutableInterface, srcNetwork, dstNetwork string, nat bool) error
	DeleteTrafficForwardedFrom(srcItf RoutableInterface, srcNetwork, destNetwork string) error
}
