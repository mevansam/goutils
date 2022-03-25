package network

import (
	"fmt"

	"inet.af/netaddr"
)

type Route struct {
	Interface string

	GatewayIP, DestIP netaddr.IP
	DestCIDR          netaddr.IPPrefix
	
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

func (r *Route) String() string {
	return fmt.Sprintf(
		"%s via %s on interface %s (scoped: %t)",
		r.DestCIDR, r.GatewayIP, r.Interface, r.IsInterfaceScoped,
	)
}
