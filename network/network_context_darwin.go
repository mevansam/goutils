//go:build darwin
// +build darwin

package network

import (
	"bufio"
	"bytes"
	"net"
	"net/netip"
	"regexp"
	"syscall"
	"time"

	"github.com/mitchellh/go-homedir"
	netroute "golang.org/x/net/route"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"
)

type networkContext struct { 
	ipv6Disabled bool

	origDNSServers    []string
	origSearchDomains []string

	routedIPs []string
}

var (
	ifconfig, 
	route,
	pfctl,
	arp,
	networksetup run.CLI

	netServiceName string	
)

func NewNetworkContext() NetworkContext {

	readNetworkInfo()

	return &networkContext{ 
		origDNSServers:    []string{ "empty" },
		origSearchDomains: []string{ "empty" },
	}
}

func (c *networkContext) DefaultDeviceName() string {
	return netServiceName
}

func (c *networkContext) DefaultInterface() string {
	return Network.DefaultIPv4Gateway.InterfaceName
}

func (c *networkContext) DefaultGateway() string {
	return Network.DefaultIPv4Gateway.GatewayIP.String()
}

func (c *networkContext) DisableIPv6() error {
	if err := networksetup.Run([]string{ "-setv6off", netServiceName }); err != nil {
		logger.ErrorMessage("networkContext.DisableIPv6(): Error running \"networksetup -setv6off %s\": %s", netServiceName, err.Error())
		return err
	}
	c.ipv6Disabled = true
	return nil
}

func (c *networkContext) Clear() {
	
	var (
		err error

		dnsManager   DNSManager
		routeManager RouteManager
	)

	if c.ipv6Disabled {
		if err := networksetup.Run([]string{ "-setv6automatic", netServiceName }); err != nil {
			logger.ErrorMessage("networkContext.DisableIPv6(): Error running \"networksetup -setv6automatic %s\": %s", netServiceName, err.Error())
		}	else {
			c.ipv6Disabled = false
		}
	}

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
	)

	// initialize networking CLIs
	home, _ := homedir.Dir()
	if ifconfig, err = run.NewCLI("/sbin/ifconfig", home, nullOut, nullOut); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /sbin/ifconfig: %s", err.Error())
		return
	}
	if route, err = run.NewCLI("/sbin/route", home, nullOut, nullOut); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /sbin/route: %s", err.Error())
		return
	}
	if pfctl, err = run.NewCLI("/sbin/pfctl", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /sbin/pfctl: %s", err.Error())
		return
	}
	if arp, err = run.NewCLI("/usr/sbin/arp", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /usr/sbin/arp: %s", err.Error())
		return
	}
	if networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /usr/sbin/networksetup: %s", err.Error())
		return
	}

	readNetworkInfo()
}

func readNetworkInfo() {
	if len(netServiceName) != 0 {
		return
	}

	var (
		err error

		results map[string][][]string		
		line    string
	)

	Network.ScopedDefaults = nil
	Network.StaticRoutes = nil

	// read network routing details
	readRouteTable()
	if Network.DefaultIPv4Gateway == nil {
		// enumerate all network service interfaces
		if err = networksetup.Run([]string{ "-listnetworkserviceorder" }); err != nil {
			logger.ErrorMessage("networkContext.init(): Error running \"networksetup -listnetworkserviceorder\": %s", err.Error())
			return
		}
		results = utils.ExtractMatches(outputBuffer.Bytes(), map[string]*regexp.Regexp{
			"ports": regexp.MustCompile(`^\(Hardware Port: .* Device: ([a-z]+[0-9]*)\)$`),
		})
		outputBuffer.Reset()

		// restart each network interface
		for _, p := range results["ports"] {
			if len(p) == 2 {
				if err = run.RunAsAdminWithArgs([]string{"ifconfig", p[1], "down"}, &outputBuffer, &outputBuffer); err == nil {
					_ = run.RunAsAdminWithArgs([]string{"ifconfig", p[1], "up"}, &outputBuffer, &outputBuffer)
				}
			}
		}
		// allow network service to re-initialize route table
		time.Sleep(5 * time.Second)

		readRouteTable()
		if Network.DefaultIPv4Gateway == nil {
			logger.ErrorMessage("networkContext.init(): Unable to determine the default gateway. Please restart you systems network services.")
			return
		}
	}

	// determine network service name for default device
	if err = arp.Run([]string{ "-a" }); err != nil {
		logger.ErrorMessage("networkContext.init(): Error running \"arp -a\": %s", err.Error())
		return
	}
	results = utils.ExtractMatches(outputBuffer.Bytes(), map[string]*regexp.Regexp{
		"interfaces": regexp.MustCompile(`^(.*) \(([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})\) at ([0-9a-f]{1,2}:[0-9a-f]{1,2}:[0-9a-f]{1,2}:[0-9a-f]{1,2}:[0-9a-f]{1,2}:[0-9a-f]{1,2}) on ([a-z]+[0-9]+) ifscope \[ethernet\]$`),
	})
	outputBuffer.Reset()

	lanGatewayItf := Network.DefaultIPv4Gateway.InterfaceName
	lanItfs := results["interfaces"]	
	if len(lanItfs) > 0 && len(lanItfs[0]) == 5 {
		lanGatewayItf = lanItfs[0][4]
	}
	
	if err = networksetup.Run([]string{ "-listallhardwareports" }); err != nil {
		logger.ErrorMessage("networkContext.init(): Error running \"networksetup -listallhardwareports\": %s", err.Error())
		return
	}
	matchDevice := "Device: " + lanGatewayItf
	prevLine := ""
	scanner := bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))
	for scanner.Scan() {
		line = scanner.Text()
		if line == matchDevice && len(prevLine) > 0 {
			netServiceName = prevLine[15:]
			break;
		}
		prevLine = line
	}
	outputBuffer.Reset()

	if len(netServiceName) == 0 {
		logger.ErrorMessage(
			"networkContext.init(): Unable to determine default network service name for interface \"%s\"", 
			Network.DefaultIPv4Gateway.InterfaceName,
		)
		return
	}
}

func readRouteTable() {

	var (
		err error
		ok  bool

		rib     []byte
		msgs    []netroute.Message
		rm      *netroute.RouteMessage
		iface   *net.Interface
		netMask netip.Addr
	)

	// retrieve network route information by querying the system's routing table
	// ref syscall constants: https://github.com/apple/darwin-xnu/blob/main/bsd/net/route.h

	if rib, err = netroute.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_DUMP2, 0); err != nil {
		logger.ErrorMessage("networkContext.init(): Error fetching system route table: %s", err.Error())
		return
	}
	if msgs, err = netroute.ParseRIB(syscall.NET_RT_IFLIST2, rib); err != nil {
		logger.ErrorMessage("networkContext.init(): Error parsing fetched route table data: %s", err.Error())
		return
	}

	defaultIPv4Route := netip.MustParsePrefix("0.0.0.0/0")
	defaultIPv6Route := netip.MustParsePrefix("::/0")

	var getAddr = func(addr netroute.Addr) (netip.Addr, bool, bool) {
		if addr != nil {
			switch addr.Family() {
			case syscall.AF_INET:
				return netip.AddrFrom4(addr.(*netroute.Inet4Addr).IP), false, true
			case syscall.AF_INET6:
				return netip.AddrFrom16(addr.(*netroute.Inet6Addr).IP), true, true
			}	
		}
		return netip.Addr{}, false, false
	}

	for _, m := range msgs {
		if rm, ok = m.(*netroute.RouteMessage); !ok {
			continue
		}
		if rm.Flags & syscall.RTF_UP == 0 || 
			rm.Flags & syscall.RTF_GATEWAY == 0 || 
			rm.Flags & syscall.RTF_WASCLONED != 0 || 
			len(rm.Addrs) == 0 {
			continue
		}
		if iface, err = net.InterfaceByIndex(rm.Index); err != nil {
			logger.ErrorMessage(
				"networkContext.init(): Error looking up interface for index %d: %s",
				rm.Index,
				err.Error(),
			)
			continue
		}
		r := &Route{
			InterfaceIndex: rm.Index,
			InterfaceName:  iface.Name,
		}
		if r.GatewayIP, r.IsIPv6, ok = getAddr(rm.Addrs[syscall.RTAX_GATEWAY]); !ok {
			logger.ErrorMessage("networkContext.init(): Gateway address not present for route message: %# v", rm)
			continue
		}
		if r.SrcIP, _, ok = getAddr(rm.Addrs[syscall.RTAX_IFA]); !ok {
			logger.ErrorMessage("networkContext.init(): Source address not present for route message: %# v", rm)
			continue
		}
		if r.DestIP, _, ok = getAddr(rm.Addrs[syscall.RTAX_DST]); !ok {
			logger.ErrorMessage("networkContext.init(): Destination address not present for route message: %# v", rm)
			continue
		}
		if netMask, _, ok = getAddr(rm.Addrs[syscall.RTAX_NETMASK]); !ok {
			logger.DebugMessage("networkContext.init(): Broadcast address not present for route message: %# v", rm)
		}
		ones, _ := net.IPMask(netMask.AsSlice()).Size()
		r.DestCIDR = netip.PrefixFrom(r.DestIP, ones)
		
		r.IsInterfaceScoped = rm.Flags & syscall.RTF_IFSCOPE != 0
		if r.IsIPv6 {
			if r.DestCIDR == defaultIPv6Route {
				if !r.IsInterfaceScoped {
					if Network.DefaultIPv6Gateway == nil {
						Network.DefaultIPv6Gateway = r
					}	else {
						logger.ErrorMessage("networkContext.init(): Duplicate default ip v6 route will be ignored: %# v", r)
					}		
				} else {
					Network.ScopedDefaults = append(Network.ScopedDefaults, r)
				}
			} else {
				Network.StaticRoutes = append(Network.StaticRoutes, r)
			}
		} else {
			if r.DestCIDR == defaultIPv4Route {
				if !r.IsInterfaceScoped {
					if Network.DefaultIPv4Gateway == nil {
						Network.DefaultIPv4Gateway = r
					}	else {
						logger.ErrorMessage("networkContext.init(): Duplicate default ip v4 route will be ignored: %# v", r)
					}
				} else {
					Network.ScopedDefaults = append(Network.ScopedDefaults, r)
				}
			} else {
				Network.StaticRoutes = append(Network.StaticRoutes, r)
			}
		}
	}
}
