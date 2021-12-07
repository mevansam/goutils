//go:build darwin
// +build darwin

package network

import (
	"fmt"
	"strings"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
)

type dnsManager struct {
	nc *networkContext

	networksetup run.CLI
}

func (c *networkContext) NewDNSManager() (DNSManager, error) {

	var (
		err error
	)

	m := &dnsManager{
		nc: c,
	}
	if m.networksetup, err = run.NewCLI("/usr/sbin/networksetup", home, &c.outputBuffer, &c.outputBuffer); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *dnsManager) AddDNSServers(servers []string) error {
	defer func() {
		if err := recover(); err != nil {
			logger.ErrorMessage("DNSManager.AddDNSServers(): Error output: %s", m.nc.outputBuffer.String())
		}
		m.nc.outputBuffer.Reset()
	}()

	var(
		err error
	)

	// save existing configuration
	if err = m.networksetup.Run([]string{ "-getdnsservers", netServiceName }); err != nil {
		return err
	}
	origDNSServers := m.nc.outputBuffer.String()
	if origDNSServers != fmt.Sprintf("There aren't any DNS Servers set on %s.\n", netServiceName) {
		m.nc.origDNSServers = strings.Fields(origDNSServers)
	}
	m.nc.outputBuffer.Reset()
	
	args := []string{ "-setdnsservers", netServiceName }
	args = append(args, servers...)

	// set dns servers
	if err = m.networksetup.Run(args); err != nil {
		return err
	}
	// flush DNS cache
	if err = run.RunAsAdminWithArgs([]string{ "/usr/bin/dscacheutil", "-flushcache" }, nullOut, nullOut); err != nil {
		logger.ErrorMessage("Flushing DNS cache via \"dscacheutil\" failed: %s", err.Error())
	}
	if err = run.RunAsAdminWithArgs([]string{ "/usr/bin/killall", "-HUP", "mDNSResponder" }, nullOut, nullOut); err != nil {
		logger.ErrorMessage("Killing \"mDNSResponder\" failed: %s", err.Error())
	}

	return nil
}

func (m *dnsManager) AddSearchDomains(domains []string) error {
	defer func() {
		if err := recover(); err != nil {
			logger.ErrorMessage("DNSManager.AddSearchDomains(): Error output: %s", m.nc.outputBuffer.String())
		}
		m.nc.outputBuffer.Reset()
	}()

	var(
		err error
	)

	// save existing configuration
	if err = m.networksetup.Run([]string{ "-getsearchdomains", netServiceName }); err != nil {
		return err
	}
	origSearchDomains := m.nc.outputBuffer.String()
	if origSearchDomains != fmt.Sprintf("There aren't any Search Domains set on %s.\n", netServiceName) {
		m.nc.origSearchDomains = strings.Fields(origSearchDomains)
	}
	m.nc.outputBuffer.Reset()

	args := []string{ "-setsearchdomains", netServiceName }
	args = append(args, domains...)

	// set search domains
	if err = m.networksetup.Run(args); err != nil {
		return err
	}
	
	return nil
}

func (m *dnsManager) Clear() {

	var (
		err error
	)

	// clear search domains
	if err = m.networksetup.Run(
		append(
			[]string{ "-setdnsservers", netServiceName }, 
			m.nc.origDNSServers...,
		),
	); err != nil {
		logger.ErrorMessage("DNSManager.Clear(): Failed to reset dns servers: %s", m.nc.outputBuffer.String())
	}
	m.nc.outputBuffer.Reset()	

	// clear dns servers
	if err = m.networksetup.Run(
		append(
			[]string{ "-setsearchdomains", netServiceName }, 
			m.nc.origSearchDomains...,
		),
	); err != nil {
		logger.ErrorMessage("DNSManager.Clear(): Failed to reset dns search domains: %s", m.nc.outputBuffer.String())
	}
	m.nc.outputBuffer.Reset()	

	// reset saved dns server and search domains if any
	m.nc.origDNSServers = []string{ "empty" }
	m.nc.origSearchDomains = []string{ "empty" }
}
