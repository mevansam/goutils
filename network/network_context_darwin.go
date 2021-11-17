//go:build darwin
// +build darwin

package network

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"regexp"
	"unicode"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"
	"github.com/mitchellh/go-homedir"
)

type networkContext struct { 
	ifconfig,
	route,
	networksetup run.CLI

	nullOut *os.File

	netServiceName,
	defaultInterface,
	defaultGateway string
	
	outputBuffer bytes.Buffer

	origDNSServers    []string
	origSearchDomains []string

	routedIPs []string
}

var defaultDestinationPattern = regexp.MustCompile(`^\s*destination:\s+(.*)$`)
var defaultGatewayPattern = regexp.MustCompile(`^\s*gateway:\s+(.*)$`)
var defaultInterfacePattern = regexp.MustCompile(`^\s*interface:\s+(.*)$`)

func NewNetworkContext() (NetworkContext, error) {

	var (
		err    error
		ok     bool
		result []string

		line string

		ip []net.IP
	)

	home, _ := homedir.Dir()
	null, _ := os.Open(os.DevNull)

	c := &networkContext{ 
		nullOut: null,

		origDNSServers:    []string{ "empty" },
		origSearchDomains: []string{ "empty" },
	}
	
	if c.ifconfig, err = run.NewCLI("/sbin/ifconfig", home, null, null); err != nil {
		return nil, err
	}
	if c.route, err = run.NewCLI("/sbin/route", home, &c.outputBuffer, &c.outputBuffer); err != nil {
		return nil, err
	}
	if c.networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, &c.outputBuffer, &c.outputBuffer); err != nil {
		return nil, err
	}

	// retrieve current default route. this will return the 
	// default gateway and interface that will be used to 
	// retrieve a public internet resource
	if err = c.route.Run([]string{ "get", "1.1.1.1" }); err != nil {
		return nil, err
	}

	results := utils.ExtractMatches(c.outputBuffer.Bytes(), map[string]*regexp.Regexp{
		"destination": defaultDestinationPattern,
		"gateway": defaultGatewayPattern,
		"interface": defaultInterfacePattern,
	})
	c.outputBuffer.Reset()

	if result, ok = results["destination"]; !ok || result[1] != "default" {
		return nil, fmt.Errorf("unable to determine the default network device")
	}
	if result, ok = results["gateway"]; !ok {
		return nil, fmt.Errorf("unable to determine the default gateway")
	}
	if unicode.IsDigit(rune(result[1][0])) {
		c.defaultGateway = result[1]
	} else {
		if ip, err = net.LookupIP(result[1]); err != nil && len(ip) == 0 {
			return nil, err
		}
		c.defaultGateway = ip[0].String()
	}
	if result, ok = results["interface"]; !ok {
		return nil, fmt.Errorf("unable to determine the default interface")
	}
	c.defaultInterface = result[1]
	
	// determine network service name for default device
	if err = c.networksetup.Run([]string{ "-listallhardwareports" }); err != nil {
		return nil, err
	}

	matchDevice := "Device: " + c.defaultInterface
	prevLine := ""
	scanner := bufio.NewScanner(bytes.NewReader(c.outputBuffer.Bytes()))
	for scanner.Scan() {
		line = scanner.Text()
		if line == matchDevice && len(prevLine) > 0 {
			c.netServiceName = prevLine[15:]
			break;
		}
		prevLine = line
	}
	c.outputBuffer.Reset()

	if len(c.netServiceName) == 0 {
		return nil, fmt.Errorf(
			"unable to determine default network service name for interface \"%s\"", 
			c.defaultInterface,
		)
	}

	return c, nil
}

func (c *networkContext) DefaultDeviceName() string {
	return c.netServiceName
}

func (c *networkContext) DefaultInterface() string {
	return c.defaultInterface
}

func (c *networkContext) DefaultGateway() string {
	return c.defaultGateway
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
