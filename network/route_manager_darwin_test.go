//go:build darwin

package network_test

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"

	"github.com/mevansam/goutils/network"
	"github.com/mevansam/goutils/run"
	"github.com/mitchellh/go-homedir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route Manager", func() {

	var (
		err error

		outputBuffer bytes.Buffer

		nc network.NetworkContext
	)

	BeforeEach(func() {
		if err = run.RunAsAdminWithArgs([]string{ "/sbin/ifconfig", "feth99", "create" }, &outputBuffer, &outputBuffer); err != nil {			
			Fail(fmt.Sprintf("exec \"/sbin/ifconfig feth99 create\" failed: \n\n%s\n", outputBuffer.String()))
		}

		nc, err = network.NewNetworkContext()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		nc.Clear()

		if err = run.RunAsAdminWithArgs([]string{ "/sbin/ifconfig", "feth99", "destroy" }, &outputBuffer, &outputBuffer); err != nil {			
			fmt.Printf("exec \"/sbin/ifconfig feth99 destroy\" failed: \n\n%s\n", outputBuffer.String())
		}
	})

	It("retrieves the default interface", func() {

		isAdmin, err := run.IsAdmin()
		Expect(err).NotTo(HaveOccurred())
		if !isAdmin {
			Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
		}

		routeManager, err := nc.NewRouteManager()
		Expect(err).NotTo(HaveOccurred())
		routableInterface, err := routeManager.GetDefaultInterface()
		Expect(err).NotTo(HaveOccurred())
		Expect(routableInterface).ToNot(BeNil())
	})

	It("creates a new default gateway with routes that bypass it", func() {

		isAdmin, err := run.IsAdmin()
		Expect(err).NotTo(HaveOccurred())
		if !isAdmin {
			Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
		}

		routeManager, err := nc.NewRouteManager()
		Expect(err).NotTo(HaveOccurred())
		err = routeManager.AddExternalRouteToIPs([]string{ "34.204.21.102" })
		Expect(err).NotTo(HaveOccurred())
		routableInterface, err := routeManager.NewRoutableInterface("feth99", "192.168.111.2/32")
		Expect(err).NotTo(HaveOccurred())
		err = routableInterface.MakeDefaultRoute()
		Expect(err).NotTo(HaveOccurred())

		home, _ := homedir.Dir()
		netstat, err := run.NewCLI("/usr/sbin/netstat", home, &outputBuffer, &outputBuffer)
		Expect(err).NotTo(HaveOccurred())
		err = netstat.Run([]string{ "-nrf", "inet" })
		Expect(err).NotTo(HaveOccurred())

		fmt.Printf("\n%s\n", outputBuffer.String())

		counter := 0
		scanner := bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))

		var matchRoutes = func(line string) {
			matched, _ := regexp.MatchString(`^default\s+192.168.111.1\s+UGScg?\s+feth99\s+$`, line)
			if matched { counter++; return }
			matched, _ = regexp.MatchString(`^34.204.21.102/32\s+([0-9]+\.?)+\s+UGSc\s+en[0-9]\s+$`, line)
			if matched { counter++; return }
			matched, _ = regexp.MatchString(`^192.168.111.1/32\s+\S+\s+\S+\s+feth99\s+\!?$`, line)
			if matched { counter++; return }
			matched, _ = regexp.MatchString(`^192.168.111.2/32\s+\S+\s+\S+\s+feth99\s+\!?$`, line)
			if matched { counter++ }
		}

		for scanner.Scan() {
			line := scanner.Text()
			matchRoutes(line)
			fmt.Printf("Test route: %s <= %d\n", line, counter)
		}
		Expect(counter).To(Equal(4))		
	})
})