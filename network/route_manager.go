package network

// route manager type common functions

func (i *routableInterface) FowardTrafficTo(dstItf RoutableInterface, srcNetwork, dstNetwork string, nat bool) error {
	return dstItf.FowardTrafficFrom(i, srcNetwork, dstNetwork, nat)
}

func (i *routableInterface) DeleteTrafficForwardedTo(dstItf RoutableInterface, srcNetwork, dstNetwork string, nat bool) error {
	return dstItf.DeleteTrafficForwardedFrom(i, srcNetwork, dstNetwork)
}
