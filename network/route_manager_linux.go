//go:build linux
// +build linux

package network

import (
	"net"

	"github.com/vishvananda/netlink"

	"github.com/mevansam/goutils/logger"
)

type routeManager struct {	
	nc *networkContext
}

type routableInterface struct {
	link           netlink.Link
	gatewayAddress net.IP
}

func (c *networkContext) NewRouteManager() (RouteManager, error) {

	rm := &routeManager{
		nc: c,
	}
	return rm, nil
}

func (m *routeManager) GetRoutableInterface(ifaceName string) (RoutableInterface, error) {

	var (
		err error
	)
	itf := routableInterface{}

	if itf.link, err = netlink.LinkByName(ifaceName); err != nil {
		return nil, err
	}
	return &itf, nil
}

func (m *routeManager) NewRoutableInterface(ifaceName, address string) (RoutableInterface, error) {

	var (
		err error

		ip    net.IP
		ipNet *net.IPNet
	)
	itf := routableInterface{}

	if ip, ipNet, err = net.ParseCIDR(address); err != nil {
		return nil, err
	}
	size, _ := ipNet.Mask.Size()
	if (size == 32) {
		// default to a /24 if address 
		// does not indicate network
		ipNet.Mask = net.CIDRMask(24, 32)
	}

	if itf.link, err = netlink.LinkByName(ifaceName); err != nil {
		return nil, err
	}
	ipConfig := &netlink.Addr{IPNet: &net.IPNet{
		IP: ip,
		Mask: ipNet.Mask,
	}}
	if err = netlink.AddrAdd(itf.link, ipConfig); err != nil {
		return nil, err
	}
	if err = netlink.LinkSetUp(itf.link); err != nil {
		return nil, err
	}

	// determine gateway from interface's subnet
	itf.gatewayAddress = ip.Mask(ipNet.Mask);
	IncIP(itf.gatewayAddress)

	m.nc.routedItfs = append(m.nc.routedItfs, itf)
	return &itf, nil
}

func (m *routeManager) AddExternalRouteToIPs(ips []string) error {

	var (
		err error

		destIP net.IP
	)
	gatewayIP := Network.DefaultIPv4Gateway.GatewayIP.AsSlice()

	for _, ip := range ips {
		if destIP = net.ParseIP(ip); destIP != nil {
			route := netlink.Route{
				Scope:     netlink.SCOPE_UNIVERSE,
				LinkIndex: Network.DefaultIPv4Gateway.InterfaceIndex,
				Dst:       &net.IPNet{IP: destIP, Mask: net.CIDRMask(32, 32)},
				Gw:        gatewayIP,
			}
			if err = netlink.RouteAdd(&route); err != nil {
				logger.ErrorMessage(
					"routeManager.AddExternalRouteToIPs(): Unable to add static route %s via gateway %s: %s", 
					route.Dst, Network.DefaultIPv4Gateway.GatewayIP.String(), err.Error())
			}	else {
				m.nc.routedIPs = append(m.nc.routedIPs, route)
			}
		}
	}
	return nil
}

func (m *routeManager) AddDefaultRoute(gateway string) error {
	return nil
}

func (m *routeManager) Clear() {

	var (
		err error
	)

	// down all added interfaces. this will 
	// clear any routes via the interface
	if len(m.nc.routedItfs) > 0 {
		for _, itf := range m.nc.routedItfs {
			if err = netlink.LinkSetDown(itf.link); err != nil {
				logger.DebugMessage(
					"routeManager.Clear(): Interface %s down returned message: %s", 
					itf.link.Attrs().Name, err.Error())
			}
		}
	}

	// clear routed ips if any
	if len(m.nc.routedIPs) > 0 {
		for _, route := range m.nc.routedIPs {
			if err = netlink.RouteDel(&route); err != nil {
				logger.ErrorMessage(
					"routeManager.Clear(): Unable to delete static route to IP %s: %s", 
					route.Dst, err.Error())
			}
		}
		m.nc.routedIPs = nil
	}

	// restore default lan route
	if err = netlink.RouteAdd(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: Network.DefaultIPv4Gateway.InterfaceIndex,
		Gw:        Network.DefaultIPv4Gateway.GatewayIP.AsSlice(),
	}); err != nil {
		logger.ErrorMessage(
			"routeManager.Clear(): Unable to restore default route: %s", 
			err.Error())
	}
}

func (i *routableInterface) MakeDefaultRoute() error {

	var (
		err error
	)

	if err = netlink.RouteDel(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: Network.DefaultIPv4Gateway.InterfaceIndex,
		Gw:        Network.DefaultIPv4Gateway.GatewayIP.AsSlice(),
	}); err != nil {
		return err
	}
	// create a default route override via
	// the routable interface's gateway
	return netlink.RouteAdd(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: i.link.Attrs().Index,
		Gw:        i.gatewayAddress,
	})
}

func (i *routableInterface) AddStaticRouteFrom(srcItf, srcNetwork string) error {
	// Route packets from src to network this itf is connected
	return nil
}

func (i *routableInterface) FowardTrafficFrom(srcItf, srcNetwork string) error {
	// NAT packets from src to network this itf is connected
	return nil
}
