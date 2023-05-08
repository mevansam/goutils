//go:build linux

package network

import (
	"bytes"
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
}

type packetFilterRouter struct {
	nft nftables.Conn

	table   []*nftables.Table
	input   []*nftables.Chain
	forward []*nftables.Chain
	nat     []*nftables.Chain

	forwardRuleMap map[string]*nftables.Rule
	natRuleMap map[string]*nftables.Rule
}

var router *packetFilterRouter = &packetFilterRouter{
	forwardRuleMap: make(map[string]*nftables.Rule),
	natRuleMap:     make(map[string]*nftables.Rule),
}

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

	if Network.DefaultIPv4Route != nil {
		if itf.link, err = netlink.LinkByName(Network.DefaultIPv4Route.InterfaceName); err != nil {
			return nil, err
		}
		itf.gatewayAddress = Network.DefaultIPv4Route.GatewayIP.AsSlice()
	
	} else if Network.DefaultIPv6Route != nil {
		if itf.link, err = netlink.LinkByName(Network.DefaultIPv6Route.InterfaceName); err != nil {
			return nil, err
		}
		itf.gatewayAddress = Network.DefaultIPv6Route.GatewayIP.AsSlice()

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
	if Network.DefaultIPv4Route != nil && 
		Network.DefaultIPv4Route.InterfaceName == ifaceName {

		return &routableInterface{
			gatewayAddress: Network.DefaultIPv4Route.GatewayIP.AsSlice(),
			link: link,
		}, nil
	}
	if Network.DefaultIPv6Route != nil && 
		Network.DefaultIPv6Route.InterfaceName == ifaceName {

		return &routableInterface{
			gatewayAddress: Network.DefaultIPv6Route.GatewayIP.AsSlice(),
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
	gatewayIP := Network.DefaultIPv4Route.GatewayIP.AsSlice()

	for _, ip := range ips {
		if destIP = net.ParseIP(ip); destIP != nil {
			route := netlink.Route{
				Scope:     netlink.SCOPE_UNIVERSE,
				LinkIndex: Network.DefaultIPv4Route.InterfaceIndex,
				Dst:       &net.IPNet{IP: destIP, Mask: net.CIDRMask(32, 32)},
				Gw:        gatewayIP,
			}
			if err = netlink.RouteAdd(&route); err != nil {
				logger.ErrorMessage(
					"routeManager.AddExternalRouteToIPs(): Unable to add static route %s via gateway %s: %s", 
					route.Dst, Network.DefaultIPv4Route.GatewayIP.String(), err.Error())
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
		LinkIndex: Network.DefaultIPv4Route.InterfaceIndex,
		Gw:        Network.DefaultIPv4Route.GatewayIP.AsSlice(),
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

func (i *routableInterface) ForwardPortTo(srcPort int, dstPort int, dstIP netip.Addr) error {
	return nil
}

func (i *routableInterface) DeletePortForwardedTo(srcPort int, dstPort int, dstIP netip.Addr) error {
	return nil
}

func (i *routableInterface) FowardTrafficFrom(srcItf RoutableInterface, srcNetwork, dstNetwork string, withNat bool) error {

	var (
		err error

		srcNetworkPrefix netip.Prefix
		dstNetworkPrefix netip.Prefix

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
	if dstNetworkPrefix, err = getNetworkPrefix(i, dstNetwork); err != nil {
		return err
	}
	if srcNetworkPrefix.Addr().BitLen() != dstNetworkPrefix.Addr().BitLen() {
		return fmt.Errorf("attempt to create forwarding rules between incompatible network address spaces")
	}

	is4 := srcNetworkPrefix.Addr().Is4()
	addrLen, srcOffset, destOffset := ipHeaderOffsets(is4)

	if table, forward, nat, err = router.getTables(is4); err != nil {
		return err
	}

	iritfName := srcRitf.link.Attrs().Name
	oritfName := i.link.Attrs().Name
	ruleKey := iritfName+">"+oritfName+":"+srcNetwork+":"+dstNetwork
	
	iifname := []byte(iritfName+"\x00")
	oifname := []byte(oritfName+"\x00")

	srcNetworkRange := netipx.RangeOfPrefix(srcNetworkPrefix)
	dstNetworkRange := netipx.RangeOfPrefix(dstNetworkPrefix)

	// ip saddr <srcNetwork> ip daddr <dstNetwork>
	ipSrcDstExprs := []expr.Any{		
		// [ payload load 4b @ network header + 12 (src addr) => reg 1 ]
		&expr.Payload{
			DestRegister: 1,
			Base:         expr.PayloadBaseNetworkHeader,
			Offset:       srcOffset,
			Len:          addrLen,
		},
		// [ range neq reg 1 <src network min> - <src network max> ]
		&expr.Range{
			Op:       expr.CmpOpEq,
			Register: 1,
			FromData: srcNetworkRange.From().AsSlice(),
			ToData:   srcNetworkRange.To().AsSlice(),
		},
	}
	if dstNetwork != WORLD4 && dstNetwork != WORLD6 {
		ipSrcDstExprs = append(ipSrcDstExprs, 
			// [ payload load 4b @ network header + 16 (dest addr) => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       destOffset,
				Len:          addrLen,
			},
			// [ range neq reg 1 <src network min> - <src network max> ]
			&expr.Range{
				Op:       expr.CmpOpEq,
				Register: 1,
				FromData: dstNetworkRange.From().AsSlice(),
				ToData:   dstNetworkRange.To().AsSlice(),
			},			
		)	
	}

	forwardRule = &nftables.Rule{
		Table: table,
		Chain: forward,
	}
	forwardRule.Exprs = append(
		// iifname "srcItf" oifname "i" ip saddr <srcNetwork>ip daddr <dstNetwork> accept
		[]expr.Any{
			// [ meta load iifname => reg 1 ]
			&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
			// [ cmp eq reg 1 <iifname> ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     iifname,
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
		append(
			ipSrcDstExprs,
			//[ immediate reg 0 accept ]
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		)...
	)

	if withNat {
		natRule = &nftables.Rule{
			Table: table,
			Chain: nat,
		}
		natRule.Exprs = append(
			// oifname "i" ip saddr <srcNetwork> ip daddr <dstNetwork> masquerade
			[]expr.Any{	
				// meta load oifname => reg 1
				&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
				// [ cmp eq reg 1 <oifname> ]
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     oifname,
				},					
			},
			append(
				ipSrcDstExprs, 
				// masq
				&expr.Masq{},
			)...
		)
	}

	router.nft.AddRule(forwardRule)
	if withNat {
		router.nft.AddRule(natRule)
	}

	if err = router.nft.Flush(); err == nil {
		if forwardRule, err = loadSavedRule(forwardRule, is4); err != nil {
			logger.ErrorMessage(
				"routableInterface.FowardTrafficFrom(): NFT forward rule flushed but rule with handle could not be loaded: %s",
				err.Error(),
			)
		} else {
			router.forwardRuleMap[ruleKey] = forwardRule
		}
		if withNat {
			if natRule, err = loadSavedRule(natRule, is4); err != nil {
				logger.ErrorMessage(
					"routableInterface.FowardTrafficFrom(): NFT nat rule flushed but rule with handle could not be loaded: %s",
					err.Error(),
				)
			} else {
				router.natRuleMap[ruleKey] = natRule
			}
		}
		return nil
	}
	return err
}

func (i *routableInterface) DeleteTrafficForwardedFrom(srcItf RoutableInterface, srcNetwork, dstNetwork string) error {
	
	srcRitf := srcItf.(*routableInterface)
	return deleteRule(
		srcRitf.link.Attrs().Name + 
		">" + i.link.Attrs().Name + 
		":" + srcNetwork +
		":" + dstNetwork)
}

// helper functions

// returns ip addrLen and srcOffset, destOffset in ip header
func ipHeaderOffsets(is4 bool) (uint32, uint32, uint32) {
	if (is4) {
		return 4, 12, 16
	} else {
		return 16, 64, 192
	}
}

func loadSavedRule(rule *nftables.Rule, is4 bool) (*nftables.Rule, error) {

	var (
		err error

		rules []*nftables.Rule

		ruleExprData [][]byte
		exprData     []byte
	)

	for _, e := range rule.Exprs {
		if is4 {
			exprData, err = expr.Marshal(byte(nftables.TableFamilyIPv4), e)
		} else {
			exprData, err = expr.Marshal(byte(nftables.TableFamilyIPv6), e)
		}
		if err != nil {
			return nil, err
		}
		ruleExprData = append(ruleExprData, exprData)
	}

	// asynchronously match saved rules data with given
	// rule data to get saved instance with handle

	if rules, err = router.nft.GetRules(rule.Table, rule.Chain); err != nil {
		return nil, err
	}

	foundRule := make(chan *nftables.Rule, len(rules))
	checkRuleMatch := func(savedRule, rule *nftables.Rule, ruleExprData [][]byte) {
		if savedRule.Flags == rule.Flags && bytes.Equal(savedRule.UserData, rule.UserData) {
			for i, e := range savedRule.Exprs {
				if is4 {
					exprData, err = expr.Marshal(byte(nftables.TableFamilyIPv4), e)
				} else {
					exprData, err = expr.Marshal(byte(nftables.TableFamilyIPv6), e)
				}
				if err != nil {
					logger.ErrorMessage("loadSavedRule(): Rule marshal error: %s", err.Error())
					foundRule <-nil
					return
				}
				if !bytes.Equal(exprData, ruleExprData[i]) {
					foundRule <-nil
					return
				}
			}
			foundRule <-savedRule
		}
	}
	for _, r := range rules {
		go checkRuleMatch(r, rule, ruleExprData)
	}
	for i := 0; i < len(rules); i++ {
		if rule = <-foundRule; rule != nil {
			break
		}
	}

	if rule == nil {
		return nil, fmt.Errorf("saved rule instance not found")
	} else {
		return rule, nil
	}	
}

func deleteRule(ruleKey string) error {

	var (
		err       error
		ok, flush bool

		forwardRule, natRule *nftables.Rule
	)

	if forwardRule, ok = router.forwardRuleMap[ruleKey]; ok {
		if err = router.nft.DelRule(forwardRule); err != nil {
			return err
		}
		delete(router.forwardRuleMap, ruleKey)
		flush = true
	}
	if natRule, ok = router.natRuleMap[ruleKey]; ok {
		if err = router.nft.DelRule(natRule); err != nil {
			return err
		}
		delete(router.natRuleMap, ruleKey)
		flush = true
	}
	if flush {
		return router.nft.Flush()
	} else {
		return nil
	}
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

		r.input = make([]*nftables.Chain, 2)
		r.forward = make([]*nftables.Chain, 2)
		r.nat = make([]*nftables.Chain, 2)

		// NOTE: policy ref for policy constants missing in nftables package
		var chainPolicyRef = func (p nftables.ChainPolicy) *nftables.ChainPolicy {
			return &p
		}

		for i, table := range r.table {

			// map ctstate { 
			//   type ct_state : verdict
			//   elements = { invalid : drop, established : accept, related : accept }
			// }
			ctstateVmap := &nftables.Set{
				Name:     "ctstate",
				Table:    table,
				IsMap:    true,
				KeyType:  nftables.TypeCTState,
				DataType: nftables.TypeVerdict,
			}
			if err = router.nft.AddSet(ctstateVmap,
				[]nftables.SetElement{
					{
						Key:         binaryutil.NativeEndian.PutUint32(expr.CtStateBitINVALID),
						VerdictData: &expr.Verdict{ Kind: expr.VerdictDrop },	
					},
					{
						Key:         binaryutil.NativeEndian.PutUint32(expr.CtStateBitESTABLISHED),
						VerdictData: &expr.Verdict{ Kind: expr.VerdictAccept },	
					},
					{
						Key:         binaryutil.NativeEndian.PutUint32(expr.CtStateBitRELATED),
						VerdictData: &expr.Verdict{ Kind: expr.VerdictAccept },	
					},
				},
			); err != nil {
				return nil, nil, nil, err
			}
			ctstateExpr := []expr.Any{
				// [ ct load status => reg 1 ]
				&expr.Ct{
					Register:       1,
					SourceRegister: false,
					Key:            expr.CtKeySTATE,
				},
				// [ lookup reg 1 map ctstateVmap ]
				&expr.Lookup{
					SourceRegister: 1,
					SetName:        ctstateVmap.Name,
					SetID:          ctstateVmap.ID,
					DestRegister:   0,
					IsDestRegSet:   true,
				},			
			}
			
			// chains
			r.input[i] = router.nft.AddChain(&nftables.Chain{
				Name:     "input",
				Table:    table,
				Hooknum:  nftables.ChainHookInput,
				Priority: nftables.ChainPriorityFilter,
				Type:     nftables.ChainTypeFilter,
				Policy:   chainPolicyRef(nftables.ChainPolicyDrop),
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

			// chain input
			//
			// ct state vmap @ctstate
			router.nft.AddRule(&nftables.Rule{
				Table: table,
				Chain: r.input[i],
				Exprs: ctstateExpr,
			})
			// iifname "lo" accept
			router.nft.AddRule(&nftables.Rule{
				Table: table,
				Chain: r.input[i],
				Exprs: []expr.Any{
					// [ meta load iifname => reg 1 ]
					&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
					// [ cmp eq reg 1 <iifname> ]
					&expr.Cmp{
						Op:       expr.CmpOpEq,
						Register: 1,
						Data:     []byte("lo"+"\x00"),
					},
					//[ immediate reg 0 accept ]
					&expr.Verdict{
						Kind: expr.VerdictAccept,
					},
				},
			})

			// chain forward
			//
			// ct state vmap @ctstate
			router.nft.AddRule(&nftables.Rule{
				Table: table,
				Chain: r.forward[i],
				Exprs: ctstateExpr,
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
