//go:build linux
// +build linux

package network

func (c *networkContext) NewRouteManager() (RouteManager, error) {
	return nil, nil
}
