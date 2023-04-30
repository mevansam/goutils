package network

import (
	"fmt"
	"net/netip"
	"time"
)

type Route struct {
	InterfaceIndex int
	InterfaceName  string

	GatewayIP,
	SrcIP,
	DestIP   netip.Addr
	DestCIDR netip.Prefix
	
	IsIPv6            bool
	IsInterfaceScoped bool
}

// global network properties
var Network = struct {

	// default interface and gateway for 
	// all WAN traffic (i.e. 0.0.0.0/0)
	DefaultIPv4Gateway *Route
	DefaultIPv6Gateway *Route

	// additional default routes scoped
	// to specific interfaces
  ScopedDefaults []*Route

	// interface and gateway for LAN
	// traffic that is routed to the
	// internet
	StaticRoutes []*Route
}{}

var (
	initErr     chan error
	initialized bool
)

const (
	WORLD4 = "0.0.0.0/0"
	WORLD6 = "::/0"
	LAN4   = "4"
	LAN6   = "6"
)

// route type functions

func (r *Route) String() string {
	if r.GatewayIP.IsValid() {
		return fmt.Sprintf(
			"%s via %s on interface %s (ip: %s, scoped: %t)",
			r.DestCIDR, r.GatewayIP, r.InterfaceName, r.SrcIP, r.IsInterfaceScoped,
		)	
	} else {
		return fmt.Sprintf(
			"%s on interface %s (ip: %s, scoped: %t)",
			r.DestCIDR, r.InterfaceName, r.SrcIP, r.IsInterfaceScoped,
		)
	}
}

// network context type common functions

func (c *networkContext) DefaultInterface() string {
	return Network.DefaultIPv4Gateway.InterfaceName
}

func (c *networkContext) DefaultGateway() string {
	return Network.DefaultIPv4Gateway.GatewayIP.String()
}

func (c *networkContext) DefaultIP() string {
	return Network.DefaultIPv4Gateway.SrcIP.String()
}

// commong network context initialization functions

func waitForInit() error {

	var (
		err error
	)

	if !initialized {
		select {
		case err = <-initErr:
			initialized = true
		case <-time.After(time.Second * 30):
			err = fmt.Errorf("timedout waiting for network to complete initialization")
		}		
	}
	return err
}

func init() {
	initErr = make(chan error)
	initialized = false
}
