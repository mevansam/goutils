// +build windows

package network

type networkContext struct {
	defaultInterface string
	defaultGateway   string	
}

func NewNetworkContext() NetworkContext {
	return &networkContext{}
}

func (c *networkContext) DefaultDeviceName() string {
	return c.defaultInterface
}

func (c *networkContext) DefaultInterface() string {
	return c.defaultInterface
}

func (c *networkContext) DefaultGateway() string {
	return c.defaultGateway
}

func (c *networkContext) Clear() {
}
