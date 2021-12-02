package network

type NetworkContext interface {	
	DefaultDeviceName() string
	DefaultInterface() string
	DefaultGateway() string

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
	NewRoutableInterface(ifaceName, tunAddress string) (RoutableInterface, error)
	AddExternalRouteToIPs(ips []string) error
	
	Clear()
}

type RoutableInterface interface {
	MakeDefaultRoute() error
}