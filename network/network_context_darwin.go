//go:build darwin
// +build darwin

package network

import (
	"bufio"
	"bytes"
	"net"
	"regexp"
	"unicode"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"
	"github.com/mitchellh/go-homedir"
)

type networkContext struct { 
	outputBuffer bytes.Buffer

	ipv6Disabled bool

	origDNSServers    []string
	origSearchDomains []string

	routedIPs []string
}

var (
	defaultGatewayPattern = regexp.MustCompile(`^default\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\s+\S+\s+(\S+[0-9]+)\s*$`)

	netstat,
	networksetup run.CLI

	netServiceName,
	defaultInterface,
	defaultGateway string	
)

func init() {
	readNetworkInfo()
}

func NewNetworkContext() NetworkContext {

	readNetworkInfo()

	return &networkContext{ 
		origDNSServers:    []string{ "empty" },
		origSearchDomains: []string{ "empty" },
	}
}

func readNetworkInfo() {
	if len(netServiceName) != 0 && 
		len(defaultInterface) !=0 && 
		len(defaultGateway) != 0 {
		return
	}

	var (
		err error
		ok  bool

		outputBuffer bytes.Buffer
		result       [][]string

		line string

		ip []net.IP
	)

	home, _ := homedir.Dir()

	if netstat, err = run.NewCLI("/usr/sbin/netstat", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /usr/sbin/netstat: %s", err.Error())
		return
	}
	if networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("networkContext.init(): Error creating CLI for /usr/sbin/networksetup: %s", err.Error())
		return
	}

	// retrieve current default route by querying the current route table
	if err = netstat.Run([]string{ "-nrf", "inet" }); err != nil {
		logger.ErrorMessage("networkContext.init(): Error running \"netstat -nrf inet\": %s", err.Error())
		return
	}

	results := utils.ExtractMatches(outputBuffer.Bytes(), map[string]*regexp.Regexp{
		"gateway": defaultGatewayPattern,
	})
	outputBuffer.Reset()

	if result, ok = results["gateway"]; !ok {
		logger.ErrorMessage("networkContext.init(): Unable to determine the default gateway")
		return
	}

	defaultGateway = result[0][1]	
	if !unicode.IsDigit(rune(defaultGateway[0])) {
		if ip, err = net.LookupIP(defaultGateway); err != nil && len(ip) == 0 {
			logger.ErrorMessage("networkContext.init(): Error looking up IP of gateway \"%s\": %s", defaultGateway, err.Error())
			return
		}
		defaultGateway = ip[0].String()
	}
	defaultInterface = result[0][2]

	// determine network service name for default device
	if err = networksetup.Run([]string{ "-listallhardwareports" }); err != nil {
		logger.ErrorMessage("networkContext.init(): Error running \"networksetup -listallhardwareports\": %s", err.Error())
		return
	}

	matchDevice := "Device: " + defaultInterface
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
			defaultInterface,
		)
		return
	}
}

func (c *networkContext) DefaultDeviceName() string {
	return netServiceName
}

func (c *networkContext) DefaultInterface() string {
	return defaultInterface
}

func (c *networkContext) DefaultGateway() string {
	return defaultGateway
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
