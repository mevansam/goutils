//go:build linux
// +build linux

package network

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/vishvananda/netlink"
	"go4.org/netipx"

	"github.com/mevansam/goutils/logger"
)

type routeManager struct {	
	nc *networkContext
}

type routableInterface struct {
	link           netlink.Link
	gatewayAddress net.IP

	forwardRuleMap map[string]*nftables.Rule
	natRuleMap map[string]*nftables.Rule
}

type packetFilterRouter struct {
	nft nftables.Conn

	table   []*nftables.Table
	input   []*nftables.Chain
	forward []*nftables.Chain
	nat     []*nftables.Chain
}

var router *packetFilterRouter = &packetFilterRouter{}

func (c *networkContext) NewRouteManager() (RouteManager, error) {

	rm := &routeManager{
		nc: c,
	}
	return rm, nil
}

func (m *routeManager) GetDefaultInterface() (RoutableInterface, error) {

	var (
		err error
	)
	itf := routableInterface{}

	if Network.DefaultIPv4Gateway != nil {
		if itf.link, err = netlink.LinkByName(Network.DefaultIPv4Gateway.InterfaceName); err != nil {
			return nil, err
		}
		itf.gatewayAddress = Network.DefaultIPv4Gateway.GatewayIP.AsSlice()
	
	} else if Network.DefaultIPv6Gateway != nil {
		if itf.link, err = netlink.LinkByName(Network.DefaultIPv6Gateway.InterfaceName); err != nil {
			return nil, err
		}
		itf.gatewayAddress = Network.DefaultIPv6Gateway.GatewayIP.AsSlice()

	} else {
		return nil, fmt.Errorf("default interface not found")
	}
	return &itf, nil
}

func (m *routeManager) GetRoutableInterface(ifaceName string) (RoutableInterface, error) {

	var (
		err error

		link netlink.Link
	)

	if link, err = netlink.LinkByName(ifaceName); err != nil {
		return nil, err
	}

	// default interface
	if Network.DefaultIPv4Gateway != nil && 
		Network.DefaultIPv4Gateway.InterfaceName == ifaceName {

		return &routableInterface{
			gatewayAddress: Network.DefaultIPv4Gateway.GatewayIP.AsSlice(),
			link: link,
		}, nil
	}
	if Network.DefaultIPv6Gateway != nil && 
		Network.DefaultIPv6Gateway.InterfaceName == ifaceName {

		return &routableInterface{
			gatewayAddress: Network.DefaultIPv6Gateway.GatewayIP.AsSlice(),
			link: link,
		}, nil
	}

	// search static routes
	for _, r := range Network.StaticRoutes {
		if r.InterfaceName == ifaceName {
			if r.GatewayIP.IsValid() {
				return &routableInterface{
					gatewayAddress: r.GatewayIP.AsSlice(),
					link: link,
				}, nil
			}
		}
	}

	return &routableInterface{
		link: link,
	}, nil
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

	var (
		err error

		gwIP     net.IP
		routes[] netlink.Route
	)

	if gwIP = net.ParseIP(gateway); gwIP == nil {
		return fmt.Errorf("'%s' is not a valid ip", gateway)		
	}

	if routes, err = netlink.RouteGet(gwIP); err != nil {
		return err
	}
	if len(routes) > 0 {
		route := routes[0]
		if !route.Gw.Equal(gwIP) {
			return fmt.Errorf(
				"given ip '%s' is not a valid gateway as its route is via '%s'", 
				gateway, route.Gw.String(),
			)
		}
		itf := routableInterface{
			gatewayAddress: gwIP,
		}
		if itf.link, err = netlink.LinkByIndex(route.LinkIndex); err != nil {
			return err
		}
		return itf.MakeDefaultRoute()

	} else {
		return fmt.Errorf("no routes found to '%s'", gateway)
	}
}

func (m *routeManager) Clear() {

	var (
		err error
	)

	// remove all packet filter rules
	router.reset()

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
	if err = netlink.RouteReplace(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: Network.DefaultIPv4Gateway.InterfaceIndex,
		Gw:        Network.DefaultIPv4Gateway.GatewayIP.AsSlice(),
	}); err != nil {
		logger.ErrorMessage(
			"routeManager.Clear(): Unable to restore default route: %s", 
			err.Error())
	}
}

func (i *routableInterface) Address4() (string, string, error) {
	a, p, err := i.address(netlink.FAMILY_V4)
	return a.String(), p.String(), err
}

func (i *routableInterface) Address6() (string, string, error) {
	a, p, err := i.address(netlink.FAMILY_V6)
	return a.String(), p.String(), err
}

func (i *routableInterface) address(family int) (netip.Addr, netip.Prefix, error) {

	var (
		err error

		addrs []netlink.Addr
	)

	if addrs, err = netlink.AddrList(i.link, family); err != nil {
		return netip.Addr{}, netip.Prefix{}, err	
	}
	if len(addrs) == 0 {
		return netip.Addr{}, netip.Prefix{}, 
			fmt.Errorf("no addressess found for inteface %s", i.link.Attrs().Name)
	}
	addr := addrs[0]

	o, _ := addr.IPNet.Mask.Size()
	a, _ := netip.AddrFromSlice(addr.IPNet.IP)
	p := netip.PrefixFrom(a, o).Masked()
	
	return a, p, nil
}

func (i *routableInterface) MakeDefaultRoute() error {

	return netlink.RouteReplace(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: i.link.Attrs().Index,
		Gw:        i.gatewayAddress,
	})
}

func (i *routableInterface) FowardTrafficFrom(srcItf RoutableInterface, srcNetwork, destNetwork string, withNat bool) error {

	var (
		err error

		srcNetworkPrefix netip.Prefix
		destNetworkPrefix netip.Prefix

		table                *nftables.Table
		forward, nat         *nftables.Chain
		forwardRule, natRule *nftables.Rule
	)
	srcRitf := srcItf.(*routableInterface)

	var getNetworkPrefix = func(itf *routableInterface, network string) (networkPrefix netip.Prefix, err error) {
		if network == LAN4 {
			_, networkPrefix, err = itf.address(netlink.FAMILY_V4)
		} else if network == LAN6 {
			_, networkPrefix, err = itf.address(netlink.FAMILY_V6)
		} else {
			networkPrefix, err = netip.ParsePrefix(network)
		}
		return 
	}
	if srcNetworkPrefix, err = getNetworkPrefix(srcRitf, srcNetwork); err != nil {
		return err
	}
	if destNetworkPrefix, err = getNetworkPrefix(i, destNetwork); err != nil {
		return err
	}
	if srcNetworkPrefix.Addr().BitLen() != destNetworkPrefix.Addr().BitLen() {
		return fmt.Errorf("attempt to create forwarding rules between incompatible network address spaces")
	}

	// Forward and NAT packets from private device & network 
	// to dest network with nat if specified via this interface

	is4 := srcNetworkPrefix.Addr().Is4()
	if table, forward, nat, err = router.getTables(is4); err != nil {
		return err
	}
	
	iifname := []byte(srcRitf.link.Attrs().Name+"\x00")
	oifname := []byte(i.link.Attrs().Name+"\x00")

	srcNetworkRange := netipx.RangeOfPrefix(srcNetworkPrefix)
	destNetworkRange := netipx.RangeOfPrefix(destNetworkPrefix)

	if is4 {
		forwardRule = &nftables.Rule{
			Table: table,
			Chain: forward,
			// iifname "srcItf" ip saddr <srcNetwork> oifname "i" ip daddr <destNetwork> accept
			Exprs: []expr.Any{
				// [ meta load iifname => reg 1 ]
				&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
				// [ cmp eq reg 1 <iifname> ]
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     iifname,
				},
				// [ payload load 4b @ network header + 12 (src addr) => reg 1 ]
				&expr.Payload{
					DestRegister: 1,
					Base:         expr.PayloadBaseNetworkHeader,
					Offset:       12,
					Len:          4,
				},
				// [ range neq reg 1 <src network min> - <src network max> ]
				&expr.Range{
					Op:       expr.CmpOpEq,
					Register: 1,
					FromData: srcNetworkRange.From().AsSlice(),
					ToData:   srcNetworkRange.To().AsSlice(),
				},
				// [ meta load iifname => reg 1 ]
				&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
				// [ cmp eq reg 1 <oifname> ]
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     oifname,
				},
			},
		}
		if destNetwork != WORLD4 {
			forwardRule.Exprs = append(forwardRule.Exprs, 
				// [ payload load 4b @ network header + 16 (dest addr) => reg 1 ]
				&expr.Payload{
					DestRegister: 1,
					Base:         expr.PayloadBaseNetworkHeader,
					Offset:       16,
					Len:          4,
				},
				// [ range neq reg 1 <src network min> - <src network max> ]
				&expr.Range{
					Op:       expr.CmpOpEq,
					Register: 1,
					FromData: destNetworkRange.From().AsSlice(),
					ToData:   destNetworkRange.To().AsSlice(),
				},			
			)	
		}
		forwardRule.Exprs = append(forwardRule.Exprs, 
			//[ immediate reg 0 accept ]
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		)
		if withNat {
			natRule = &nftables.Rule{
				Table: table,
				Chain: nat,
				// ip saddr <srcNetwork> oifname "i"
				Exprs: []expr.Any{
					// [ payload load 4b @ network header + 12 (src addr) => reg 1 ]
					&expr.Payload{
						DestRegister: 1,
						Base:         expr.PayloadBaseNetworkHeader,
						Offset:       12,
						Len:          4,
					},
					// [ range neq reg 1 <src network min> - <src network max> ]
					&expr.Range{
						Op:       expr.CmpOpEq,
						Register: 1,
						FromData: srcNetworkRange.From().AsSlice(),
						ToData:   srcNetworkRange.To().AsSlice(),
					},				
					// meta load oifname => reg 1
					&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
					// [ cmp eq reg 1 <oifname> ]
					&expr.Cmp{
						Op:       expr.CmpOpEq,
						Register: 1,
						Data:     oifname,
					},
					// masq
					&expr.Masq{},
				},
			}
		}
	}	
	router.nft.AddRule(forwardRule)
	if withNat {
		router.nft.AddRule(natRule)
	}

	return router.nft.Flush()
}

// packetFilterRouter functions

func (r *packetFilterRouter) getTables(isIPv4 bool) (*nftables.Table, *nftables.Chain, *nftables.Chain, error) {

	var (
		err error
	)

	if r.table == nil {
		r.table = make([]*nftables.Table, 2)
		r.table[0] = router.nft.AddTable(&nftables.Table{
			Family: nftables.TableFamilyIPv4,
			Name:   "mycs_router_ipv4",
		})
		r.table[1] = router.nft.AddTable(&nftables.Table{
			Family: nftables.TableFamilyIPv6,
			Name:   "mycs_router_ipv6",
		})

		r.forward = make([]*nftables.Chain, 2)
		r.nat = make([]*nftables.Chain, 2)

		// NOTE: policy ref for policy constants missing in nftables package
		var chainPolicyRef = func (p nftables.ChainPolicy) *nftables.ChainPolicy {
			return &p
		}

		for i, table := range r.table {
			r.forward[i] = router.nft.AddChain(&nftables.Chain{
				Name:     "input",
				Table:    table,
				Hooknum:  nftables.ChainHookInput,
				Priority: nftables.ChainPriorityFilter,
				Type:     nftables.ChainTypeFilter,
				Policy:   chainPolicyRef(nftables.ChainPolicyAccept),
			})
			r.forward[i] = router.nft.AddChain(&nftables.Chain{
				Name:     "forward",
				Table:    table,
				Hooknum:  nftables.ChainHookForward,
				Priority: nftables.ChainPriorityFilter,
				Type:     nftables.ChainTypeFilter,
				Policy:   chainPolicyRef(nftables.ChainPolicyDrop),
			})
			r.nat[i] = router.nft.AddChain(&nftables.Chain{
				Name:     "nat",
				Table:    table,
				Hooknum:  nftables.ChainHookPostrouting,
				Priority: nftables.ChainPriorityNATSource,
				Type:     nftables.ChainTypeNAT,
				Policy:   chainPolicyRef(nftables.ChainPolicyAccept),
			})
			router.nft.AddRule(&nftables.Rule{
				Table: table,
				Chain: r.forward[i],
				// ct state invalid drop
				Exprs: []expr.Any{
					// [ ct load status => reg 1 ]
					&expr.Ct{
						Register:       1,
						SourceRegister: false,
						Key:            expr.CtKeySTATE,
					},
					// [ bitwise reg 1 = (reg=1 & invalid ) ^ 0x00000000 ]
					&expr.Bitwise{
						SourceRegister: 1,
						DestRegister:   1,
						Len:            4,
						Mask:           binaryutil.NativeEndian.PutUint32(expr.CtStateBitINVALID),
						Xor:            binaryutil.NativeEndian.PutUint32(0),
					},
					// [ cmp neq reg 1 0x00000000 ]
					&expr.Cmp{
						Op:       expr.CmpOpNeq,
						Register: 1,
						Data:     binaryutil.NativeEndian.PutUint32(0),
					},
					//[ immediate reg 0 accept ]
					&expr.Verdict{
						Kind: expr.VerdictDrop,
					},
				},
			})
			router.nft.AddRule(&nftables.Rule{
				Table: table,
				Chain: r.forward[i],
				// ct state established,related accept
				Exprs: []expr.Any{
					// [ ct load status => reg 1 ]
					&expr.Ct{
						Register:       1,
						SourceRegister: false,
						Key:            expr.CtKeySTATE,
					},
					// [ bitwise reg 1 = (reg=1 & (established | related) ) ^ 0x00000000 ]
					&expr.Bitwise{
						SourceRegister: 1,
						DestRegister:   1,
						Len:            4,
						Mask:           binaryutil.NativeEndian.PutUint32(
															expr.CtStateBitESTABLISHED | expr.CtStateBitRELATED,
														),
						Xor:            binaryutil.NativeEndian.PutUint32(0),
					},
					// [ cmp neq reg 1 0x00000000 ]
					&expr.Cmp{
						Op:       expr.CmpOpNeq,
						Register: 1,
						Data:     binaryutil.NativeEndian.PutUint32(0),
					},
					//[ immediate reg 0 accept ]
					&expr.Verdict{
						Kind: expr.VerdictAccept,
					},
				},
			})
		}
		err = router.nft.Flush()
	}
	if isIPv4 {
		return r.table[0], r.forward[0], r.nat[0], err
	} else {
		return r.table[1], r.forward[1], r.nat[1], err
	}
}

func (r *packetFilterRouter) reset() {

	if len(r.table) > 0 {
		for _, t := range r.table {
			r.nft.DelTable(t)
		}		
		if err := router.nft.Flush(); err != nil {
			logger.ErrorMessage(
				"packetFilterRouter.reset(): Error commiting deletion of mycs nftables: %s", 
				err.Error(),
			)
		}

		r.table = nil
		r.forward = nil
		r.nat = nil
	}
}
