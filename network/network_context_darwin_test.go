//go:build darwin
// +build darwin

package network_test

import (
	"github.com/mevansam/goutils/network"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network Context", func() {

	It("initialize the network context", func() {

		nc := network.NewNetworkContext()

		Expect(len(nc.DefaultInterface())).To(BeNumerically(">", 0))
		Expect(nc.DefaultGateway()).To(MatchRegexp(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`))
		Expect(nc.DefaultInterface()).To(MatchRegexp(`^en[0-9]+$`))
		Expect(len(nc.DefaultDeviceName())).To(BeNumerically(">", 0))
	})
})
