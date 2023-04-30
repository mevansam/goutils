package network

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
	
	Clear()
}

type RoutableInterface interface {
	Address4() (string, string, error)
	Address6() (string, string, error)
	MakeDefaultRoute() error
	FowardTrafficFrom(srcItf RoutableInterface, srcNetwork, destNetwork string, nat bool) error
}
