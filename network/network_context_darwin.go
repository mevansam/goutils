//go:build darwin
// +build darwin

package network

import (
	"bufio"
	"bytes"
	"fmt"
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

	origDNSServers    []string
	origSearchDomains []string

	routedIPs []string
}

var (
	defaultGatewayPattern = regexp.MustCompile(`^default\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\s+\S+\s+(\S+[0-9]+)\s*$`)

	netServiceName,
	defaultInterface,
	defaultGateway string	
)

func init() {

	var (
		err error
		ok  bool

		netstat,
		networksetup run.CLI

		outputBuffer bytes.Buffer
		result       [][]string

		line string

		ip []net.IP
	)

	home, _ := homedir.Dir()

	if netstat, err = run.NewCLI("/usr/sbin/netstat", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("network_context.init(): Error creating CLI for /usr/sbin/netstat: %s", err.Error())
		panic(err)
	}
	if networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, &outputBuffer, &outputBuffer); err != nil {
		logger.ErrorMessage("network_context.init(): Error creating CLI for /usr/sbin/networksetup: %s", err.Error())
		panic(err)
	}

	// retrieve current default route by querying the current route table
	if err = netstat.Run([]string{ "-nrf", "inet" }); err != nil {
		logger.ErrorMessage("network_context.init(): Error running \"netstat -nrf inet\": %s", err.Error())
		panic(err)
	}

	results := utils.ExtractMatches(outputBuffer.Bytes(), map[string]*regexp.Regexp{
		"gateway": defaultGatewayPattern,
	})
	outputBuffer.Reset()

	if result, ok = results["gateway"]; !ok {
		panic(fmt.Errorf("unable to determine the default gateway"))
	}

	defaultGateway = result[0][1]	
	if !unicode.IsDigit(rune(defaultGateway[0])) {
		if ip, err = net.LookupIP(defaultGateway); err != nil && len(ip) == 0 {
			logger.ErrorMessage("network_context.init(): Error looking up IP of gateway \"%s\": %s", defaultGateway, err.Error())
			panic(err)
		}
		defaultGateway = ip[0].String()
	}
	defaultInterface = result[0][2]

	// determine network service name for default device
	if err = networksetup.Run([]string{ "-listallhardwareports" }); err != nil {
		logger.ErrorMessage("network_context.init(): Error running \"networksetup -listallhardwareports\": %s", err.Error())
		panic(err)
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
		panic(fmt.Errorf(
			"unable to determine default network service name for interface \"%s\"", 
			defaultInterface,
		))
	}
}

func NewNetworkContext() NetworkContext {

	return &networkContext{ 
		origDNSServers:    []string{ "empty" },
		origSearchDomains: []string{ "empty" },
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
