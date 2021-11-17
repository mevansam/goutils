//go:build windows
// +build windows

package network

func (c *networkContext) NewDNSManager() (DNSManager, error) {
	return nil, nil
}
