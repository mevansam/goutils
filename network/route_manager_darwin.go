//go:build darwin
// +build darwin

package network

import (
	"net"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
)

// List of commands to run to configure
// a tunnel interface and routes
//
// local network's gateway to the internet: 192.168.1.1
// local tunnel IP for LHS of tunnel: 192.168.111.194
// peer tunnel IP for RHS of tunnel which is also the tunnel's internet gateway: 192.168.111.1
// external IP of wireguard peer: 34.204.21.102
//
// * configure tunnel network interface
// 			/sbin/ifconfig utun6 inet 192.168.111.194/32 192.168.111.194 up
// * configure route to wireguard overlay network via tunnel network interface
// 			/sbin/route add -inet -net 192.168.111.1 -interface utun6
// * configure route to peer's public endpoint via network interface connected to the internet
// 			/sbin/route add inet -net 34.204.21.102 192.168.1.1 255.255.255.255
// * configure route to send all other traffic through the tunnel by create two routes splitting
//   the entire IPv4 range of 0.0.0.0/0. i.e. 0.0.0.0/1 and 128.0.0.0/1
// 			/sbin/route add inet -net 0.0.0.0 192.168.111.1 128.0.0.0
// 			/sbin/route add inet -net 128.0.0.0 192.168.111.1 128.0.0.0
//
// * cleanup
// 			/sbin/route delete inet -net 34.204.21.102

type routeManager struct {	
	nc *networkContext

	ifconfig,
	route run.CLI
}

type routableInterface struct {
	route run.CLI

	gatewayAddress string
}

func (c *networkContext) NewRouteManager() (RouteManager, error) {

	var (
		err error
	)

	rm := &routeManager{
		nc: c,
	}
	if rm.ifconfig, err = run.NewCLI("/sbin/ifconfig", home, nullOut, nullOut); err != nil {
		return nil, err
	}
	if rm.route, err = run.NewCLI("/sbin/route", home, &c.outputBuffer, &c.outputBuffer); err != nil {
		return nil, err
	}
	return rm, nil
}

func (m *routeManager) NewRoutableInterface(ifaceName, address string) (RoutableInterface, error) {

	var (
		err error

		ip    net.IP
		ipNet *net.IPNet
	)

	if ip, ipNet, err = net.ParseCIDR(address); err != nil {
		return nil, err
	}
	size, _ := ipNet.Mask.Size()
	if (size == 32) {
		// default to a /24 if address 
		// does not indicate network
		ipNet.Mask = net.CIDRMask(24, 32)
	}

	gatewayIP := ip.Mask(ipNet.Mask);
	IncIP(gatewayIP)
	gatewayAddress := gatewayIP.String()

	// add tunnel IP to local tunnel interface
	if err = m.ifconfig.Run([]string{ ifaceName, "inet", address, ip.String(), "up" }); err != nil {
		return nil, err
	}	
	// create route to tunnel gateway via tunnel interface
	if err = m.route.Run([]string{ "add", "-inet", "-net", gatewayAddress, "-interface", ifaceName }); err != nil {
		return nil, err
	}
	return &routableInterface{
		route:          m.route,
		gatewayAddress: gatewayAddress,
	}, nil
}

func (m *routeManager) AddExternalRouteToIPs(ips []string) error {

	var (
		err error
	)

	for _, ip := range ips {
		if err = m.route.Run([]string{ "add", "-inet", "-net", ip, defaultGateway, "255.255.255.255" }); err != nil {
			return err
		}
	}
	m.nc.routedIPs = ips
	return nil
}

func (m *routeManager) AddDefaultRoute(gateway string) error {
	return addDefaultRoute(m.route, gateway)
}

func (m *routeManager) Clear() {
	
	var (
		err error
	)

	// clear routed ips if any
	if len(m.nc.routedIPs) > 0 {
		for _, ip := range m.nc.routedIPs {
			if err = m.route.Run([]string{ "delete", "-inet", "-net", ip }); err != nil {
				logger.ErrorMessage("routeManager.Clear(): deleting route to %s: %s", ip, err.Error())
			}
		}
	}

	// clear default route if any
	_ = m.route.Run([]string{ "delete", "-inet", "-net", "0.0/1" })
	_ = m.route.Run([]string{ "delete", "-inet", "-net", "128.0/1" })	
}

func (i *routableInterface) MakeDefaultRoute() error {
	return addDefaultRoute(i.route, i.gatewayAddress)
}

func addDefaultRoute(route run.CLI, gateway string) error {

	var (
		err error
	)

	// create default route via interface's gateway
	if err = route.Run([]string{ "add", "-inet", "-net", "0.0.0.0", gateway, "128.0.0.0" }); err != nil {
		return err
	}
	if err = route.Run([]string{ "add", "-inet", "-net", "128.0.0.0", gateway, "128.0.0.0" }); err != nil {
		return err
	}
	return nil
}