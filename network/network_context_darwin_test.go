//go:build darwin
// +build darwin

package network_test

import (
	"github.com/mevansam/goutils/network"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network Context", func() {
	
	var (
		err error

		nc network.NetworkContext
	)

	It("initialize the network context", func() {

		nc, err = network.NewNetworkContext()
		Expect(err).NotTo(HaveOccurred())

		Expect(len(nc.DefaultInterface())).To(BeNumerically(">", 0))
		Expect(nc.DefaultGateway()).To(MatchRegexp(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`))
	})
})
