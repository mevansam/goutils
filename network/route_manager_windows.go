//go:build windows
// +build windows

package network

func (c *networkContext) NewRouteManager() (RouteManager, error) {
	return nil, nil
}
