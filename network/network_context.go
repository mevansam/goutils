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

var initErr chan error

const (
	WORLD4 = "0.0.0.0/0"
	WORLD6 = "::/0"
	LAN4   = "4"
	LAN6   = "6"
)

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

func waitForInit() error {

	var (
		err error
	)

	select {
	case err = <-initErr:
	case <-time.After(time.Second * 30):
		err = fmt.Errorf("timedout waiting for network to complete initialization")
	}	
	return err
}

func init() {
	initErr = make(chan error)
}
