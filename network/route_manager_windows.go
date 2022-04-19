//go:build windows
// +build windows

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

func (i *routableInterface) MakeDefaultRoute() error {
	return nil
}

func (i *routableInterface) AddStaticRouteFrom(srcItf, srcNetwork string) error {
	// Route packets from src to network this itf is connected
	return nil
}

func (i *routableInterface) FowardTrafficFrom(srcItf, srcNetwork string) error {
	// NAT packets from src to network this itf is connected
	return nil
}
