//go:build linux
// +build linux

package network

func (c *networkContext) NewDNSManager() (DNSManager, error) {
	return nil, nil
}
