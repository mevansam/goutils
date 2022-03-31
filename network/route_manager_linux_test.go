//go:build linux
// +build linux

package network_test

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"

	"github.com/mevansam/goutils/network"
	"github.com/mevansam/goutils/run"

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
		if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "link", "add", "wg99", "type", "wireguard" }, &outputBuffer, &outputBuffer); err != nil {			
			Fail(fmt.Sprintf("exec \"/usr/sbin/ip link add wg1 type wireguard\" failed: \n\n%s\n", outputBuffer.String()))
		}

		nc = network.NewNetworkContext()
	})

	AfterEach(func() {
		nc.Clear()

		if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "link", "delete", "dev", "wg99" }, &outputBuffer, &outputBuffer); err != nil {			
			fmt.Printf("exec \"/usr/sbin/ip link delete dev wg99\" failed: \n\n%s\n", outputBuffer.String())
		}
	})

	FIt("creates a new default gateway with routes that bypass it", func() {

		isAdmin, err := run.IsAdmin()
		Expect(err).NotTo(HaveOccurred())
		if !isAdmin {
			Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
		}

		routeManager, err := nc.NewRouteManager()
		Expect(err).NotTo(HaveOccurred())
		err = routeManager.AddExternalRouteToIPs([]string{ "34.204.21.102" })
		Expect(err).NotTo(HaveOccurred())
		routableInterface, err := routeManager.NewRoutableInterface("wg99", "192.168.111.2/32")
		Expect(err).NotTo(HaveOccurred())
		err = routableInterface.MakeDefaultRoute()
		Expect(err).NotTo(HaveOccurred())

		outputBuffer.Reset()
		err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "route", "show" }, &outputBuffer, &outputBuffer)
		Expect(err).NotTo(HaveOccurred())

		fmt.Printf("\n%s\n", outputBuffer.String())

		counter := 0
		scanner := bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))

		var matchRoutes = func(line string) {
			matched, _ := regexp.MatchString(`^default via 192\.168\.111\.1 dev wg99 $`, line)
			if matched { counter++; return }
			matched, _ = regexp.MatchString(`^34.204.21.102 via [0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3} dev en[a-z0-9]+ $`, line)
			if matched { counter++; return }
			matched, _ = regexp.MatchString(`^192.168.111.0/24 dev wg99 .* link src 192.168.111.2 $`, line)
			if matched { counter++; return }
		}

		for scanner.Scan() {
			line := scanner.Text()
			matchRoutes(line)
			fmt.Printf("Test route: '%s' <= %d\n", line, counter)
		}
		Expect(counter).To(Equal(3))
	})
})