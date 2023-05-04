//go:build windows

package network

type routeManager struct {	
	nc *networkContext
}

type routableInterface struct {
	gatewayAddress string
}

func (c *networkContext) NewRouteManager() (RouteManager, error) {

	rm := &routeManager{
		nc: c,
	}
	return rm, nil
}

func (m *routeManager) GetDefaultInterface() (RoutableInterface, error) {
	return nil, nil
}

func (m *routeManager) GetRoutableInterface(ifaceName string) (RoutableInterface, error) {
	return nil, nil
}

func (m *routeManager) NewRoutableInterface(ifaceName, address string) (RoutableInterface, error) {
	return &routableInterface{}, nil
}

func (m *routeManager) AddExternalRouteToIPs(ips []string) error {
	return nil
}

func (m *routeManager) AddDefaultRoute(gateway string) error {
	return nil
}

func (m *routeManager) Clear() {
}

func (i *routableInterface) Address4() (string, string, error) {
	return "", "", nil
}

func (i *routableInterface) Address6() (string, string, error) {
	return "", "", nil
}

func (i *routableInterface) MakeDefaultRoute() error {
	return nil
}

func (i *routableInterface) FowardTrafficFrom(srcItf RoutableInterface, srcNetwork, destNetworks string, nat bool) error {
	// NAT packets from src to network this itf is connected
	return nil
}

func (i *routableInterface) DeleteTrafficFowarding(srcItf RoutableInterface, srcNetwork, destNetwork string) error {
	return nil
}