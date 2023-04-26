// +build windows

package network

type networkContext struct {
}

func NewNetworkContext() (NetworkContext, error) {
	return &networkContext{}, nil
}

func (c *networkContext) DefaultDeviceName() string {
	return Network.DefaultIPv4Gateway.InterfaceName
}

func (c *networkContext) DefaultInterface() string {
	return Network.DefaultIPv4Gateway.InterfaceName
}

func (c *networkContext) DefaultGateway() string {
	return Network.DefaultIPv4Gateway.GatewayIP.String()
}

func (c *networkContext) DisableIPv6() error {
	return nil
}

func (c *networkContext) Clear() {
}
