// +build linux

package network

import (
	"net"

	"github.com/vishvananda/netlink"
	"inet.af/netaddr"

	"github.com/mevansam/goutils/logger"
)

type networkContext struct {
	routedItfs []routableInterface
	routedIPs  []netlink.Route
}

func NewNetworkContext() NetworkContext {
	return &networkContext{}
}

func (c *networkContext) DefaultDeviceName() string {
	return Network.DefaultIPv4Gateway.InterfaceName
}

func (c *networkContext) DefaultInterface() string {
	return Network.DefaultIPv4Gateway.InterfaceName
}

func (c *networkContext) DefaultGateway() string {
	return Network.DefaultIPv4Gateway.GatewayIP.String()
}

func (c *networkContext) DisableIPv6() error {
	return nil
}

func (c *networkContext) Clear() {

	var (
		err error

		dnsManager   DNSManager
		routeManager RouteManager
	)

	if dnsManager, err = c.NewDNSManager(); err != nil {
		logger.ErrorMessage(
			"networkContext.Clear(): Error creating DNS manager to use to clear network context: %s", 
			err.Error(),
		)
	}
	dnsManager.Clear()

	if routeManager, err = c.NewRouteManager(); err != nil {
		logger.ErrorMessage(
			"networkContext.Clear(): Error creating DNS manager to use to clear network context: %s", 
			err.Error(),
		)
	}
	routeManager.Clear()
}

func init() {

	var (
		err error
		ok  bool

		routes []netlink.Route
		iface  *net.Interface
	)

	defaultIPv4 := netaddr.MustParseIP("0.0.0.0")
	defaultIPv4Route := netaddr.MustParseIPPrefix("0.0.0.0/0")
	if routes, err = netlink.RouteList(nil, netlink.FAMILY_V4); err != nil {
		logger.ErrorMessage("networkContext.init(): Error looking up ipv4 routes: %s", err.Error())
		return
	}
	for _, route := range routes {		
		if iface, err = net.InterfaceByIndex(route.LinkIndex); err != nil {
			logger.ErrorMessage(
				"networkContext.init(): Error looking up interface for index %d: %s",
				route.LinkIndex,
				err.Error(),
			)
			return
		}
		r := &Route{
			InterfaceIndex:    route.LinkIndex,
			InterfaceName:     iface.Name,
			IsInterfaceScoped: route.Scope == netlink.SCOPE_LINK,
		}
		if route.Gw != nil {
			if r.GatewayIP, ok = netaddr.FromStdIP(route.Gw); !ok {
				logger.ErrorMessage("networkContext.init(): Error invalid gateway IP: %s", err.Error())
				continue
			}
		} else {
			continue
		}
		if route.Src != nil {
			if r.SrcIP, ok = netaddr.FromStdIP(route.Src); !ok {
				logger.ErrorMessage("networkContext.init(): Error invalid source IP: %s", err.Error())
				continue
			}
		}
		if route.Dst == nil {
			r.DestCIDR = defaultIPv4Route
			r.DestIP = defaultIPv4
			Network.DefaultIPv4Gateway = r

		} else {
			if r.DestCIDR, ok = netaddr.FromStdIPNet(route.Dst); !ok {
				logger.ErrorMessage("networkContext.init(): Error invalid destination CIDR: %s", err.Error())
				continue
			}
			r.DestIP = r.DestCIDR.IP()
			Network.StaticRoutes = append(Network.StaticRoutes, r)
		}
	}

	defaultIPv6 := netaddr.MustParseIP("::")
	defaultIPv6Route := netaddr.MustParseIPPrefix("::/0")
	if routes, err = netlink.RouteList(nil, netlink.FAMILY_V6); err != nil {
		logger.ErrorMessage("networkContext.init(): Error looking up ipv6 routes: %s", err.Error())
		return
	}
	for _, route := range routes {
		if iface, err = net.InterfaceByIndex(route.LinkIndex); err != nil {
			logger.ErrorMessage(
				"networkContext.init(): Error looking up interface for index %d: %s",
				route.LinkIndex,
				err.Error(),
			)
			return
		}
		r := &Route{
			InterfaceIndex:    route.LinkIndex,
			InterfaceName:     iface.Name,
			IsInterfaceScoped: route.Scope == netlink.SCOPE_LINK ,
		}
		if route.Gw != nil {
			if r.GatewayIP, ok = netaddr.FromStdIP(route.Gw); !ok {
				logger.ErrorMessage("networkContext.init(): Error invalid gateway IP: %s", err.Error())
				continue
			}
		} else {
			continue
		}
		if route.Src != nil {
			if r.SrcIP, ok = netaddr.FromStdIP(route.Src); !ok {
				logger.ErrorMessage("networkContext.init(): Error invalid source IP: %s", err.Error())
				continue
			}
		}
		if route.Dst == nil {
			r.DestCIDR = defaultIPv6Route
			r.DestIP = defaultIPv6
			Network.DefaultIPv6Gateway = r

		} else {
			if r.DestCIDR, ok = netaddr.FromStdIPNet(route.Dst); !ok {
				logger.ErrorMessage("networkContext.init(): Error invalid destination CIDR: %s", err.Error())
				continue
			}
			r.DestIP = r.DestCIDR.IP()
			Network.StaticRoutes = append(Network.StaticRoutes, r)
		}
	}
}
