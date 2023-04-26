// +build linux

package network

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/vishvananda/netlink"

	"github.com/mevansam/goutils/logger"
)

type networkContext struct {
	routedItfs []routableInterface
	routedIPs  []netlink.Route
}

func NewNetworkContext() (NetworkContext, err) {

	if err := waitForInit(); err != nil {
		return nil, err
	}

	return &networkContext{}, nil
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
	} else {
		dnsManager.Clear()
	}

	if routeManager, err = c.NewRouteManager(); err != nil {
		logger.ErrorMessage(
			"networkContext.Clear(): Error creating DNS manager to use to clear network context: %s", 
			err.Error(),
		)
	} else {
		routeManager.Clear()
	}
}

func init() {
	go readNetworkInfo()
}

func readNetworkInfo() {

	var (
		err error

		routes []netlink.Route
	)

	Network.ScopedDefaults = nil
	Network.StaticRoutes = nil

	if routes, err = netlink.RouteList(nil, netlink.FAMILY_V4); err != nil {
		logger.ErrorMessage("networkContext.init(): Error looking up ipv4 routes: %s", err.Error())
		initErr <- err
		return
	}
	if err = readRoutes(
		netip.MustParseAddr("0.0.0.0"), 
		netip.MustParsePrefix("0.0.0.0/0"), 
		routes,
	); err != nil {
		initErr <- err
		return
	}

	if routes, err = netlink.RouteList(nil, netlink.FAMILY_V6); err != nil {
		logger.ErrorMessage("networkContext.init(): Error looking up ipv6 routes: %s", err.Error())
		initErr <- err
		return
	}
	if err = readRoutes(
		netip.MustParseAddr("::"),
		netip.MustParsePrefix("::/0"),
		routes,
	); err != nil {
		initErr <- err
		return
	}

	if Network.DefaultIPv4Gateway == nil {
		initErr <- fmt.Errorf(
			"unable to determine default network interface and gateway", 
			Network.DefaultIPv4Gateway.InterfaceName,
		)
		return
	}

	initErr <- nil
}

func readRoutes(
	defaultRouteIP netip.Addr, 
	defaultRouteCIDR netip.Prefix,  
	routes []netlink.Route,
) error {

	var (
		err error
		ok  bool

		iface  *net.Interface
	)

	for _, route := range routes {
		if iface, err = net.InterfaceByIndex(route.LinkIndex); err != nil {
			logger.ErrorMessage(
				"networkContext.readRoutes(): Error looking up interface for index %d: %s",
				route.LinkIndex,
				err.Error(),
			)
			return err
		}
		r := &Route{
			InterfaceIndex:    route.LinkIndex,
			InterfaceName:     iface.Name,
			IsInterfaceScoped: route.Scope == netlink.SCOPE_LINK,
		}
		if route.Gw != nil {
			if r.GatewayIP, ok = netip.AddrFromSlice(route.Gw); !ok {
				logger.ErrorMessage("networkContext.readRoutes(): Error invalid gateway IP: %s", err.Error())
				continue
			}
		}
		if route.Src != nil {
			if r.SrcIP, ok = netip.AddrFromSlice(route.Src); !ok {
				logger.ErrorMessage("networkContext.readRoutes(): Error invalid source IP: %s", err.Error())
				continue
			}
		}
		if route.Dst == nil {
			r.DestIP = defaultRouteIP
			r.DestCIDR = defaultRouteCIDR
			Network.DefaultIPv4Gateway = r

		} else {
			if r.DestIP, ok = netip.AddrFromSlice(route.Dst.IP); !ok {
				logger.ErrorMessage("networkContext.readRoutes(): Error invalid destination CIDR: %s", err.Error())
				continue
			}
			ones, _ := route.Dst.Mask.Size()
			r.DestCIDR = netip.PrefixFrom(r.DestIP, ones)
			Network.StaticRoutes = append(Network.StaticRoutes, r)
		}
	}

	return nil
}
