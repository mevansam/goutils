//go:build linux

package network_test

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"regexp"
	"time"

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

	Context("creates routes on a new interface", func() {

		BeforeEach(func() {
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "link", "add", "wg99", "type", "wireguard" }, &outputBuffer, &outputBuffer); err != nil {			
				Fail(fmt.Sprintf("exec \"/usr/sbin/ip link add wg99 type wireguard\" failed: \n\n%s\n", outputBuffer.String()))
			}
	
			nc, err = network.NewNetworkContext()
			Expect(err).ToNot(HaveOccurred())
		})
	
		AfterEach(func() {
			nc.Clear()
	
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "link", "delete", "dev", "wg99" }, &outputBuffer, &outputBuffer); err != nil {			
				fmt.Printf("exec \"/usr/sbin/ip link delete dev wg99\" failed: \n\n%s\n", outputBuffer.String())
			}
		})
	
		It("creates a new default gateway with routes that bypass it", func() {
	
			isAdmin, err := run.IsAdmin()
			Expect(err).ToNot(HaveOccurred())
			if !isAdmin {
				Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
			}
	
			routeManager, err := nc.NewRouteManager()
			Expect(err).ToNot(HaveOccurred())
			err = routeManager.AddExternalRouteToIPs([]string{ "34.204.21.102" })
			Expect(err).ToNot(HaveOccurred())
			routableInterface, err := routeManager.NewRoutableInterface("wg99", "192.168.111.2/32")
			Expect(err).ToNot(HaveOccurred())
			err = routableInterface.MakeDefaultRoute()
			Expect(err).ToNot(HaveOccurred())
	
			outputBuffer.Reset()
			err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "route", "show" }, &outputBuffer, &outputBuffer)
			Expect(err).ToNot(HaveOccurred())
	
			fmt.Printf("\n%s\n", outputBuffer.String())
	
			counter := 0
			scanner := bufio.NewScanner(bytes.NewReader(outputBuffer.Bytes()))
	
			var matchRoutes = func(line string) {
				matched, _ := regexp.MatchString(`^default via 192\.168\.111\.1 dev wg99 $`, line)
				if matched { counter++; return }
				matched, _ = regexp.MatchString(`^34.204.21.102 via [0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3} dev e[a-z0-9]+ $`, line)
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

	Context("creates routes and manages routes", func() {

		var (
			skipTests bool

			itf2 net.Interface
			itf3 net.Interface
		)

		BeforeEach(func() {
			itfs, err := net.Interfaces()
			Expect(err).ToNot(HaveOccurred())
			if len(itfs) < 4 {
				fmt.Println("To test packet forwarding and nat the test environment needs 2 additional interfaces connected to flat networks without DHCP or a gateway.")

				skipTests = true
				return
			}
			itf2 = itfs[2]
			itf3 = itfs[3]

			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "addr", "flush", itf2.Name }, &outputBuffer, &outputBuffer); err != nil {
				Fail(fmt.Sprintf("exec \"/usr/sbin/ip addr flush dev eth1\" failed: \n\n%s\n", outputBuffer.String()))
			}
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "addr", "add", "192.168.10.1/24", "dev", itf2.Name }, &outputBuffer, &outputBuffer); err != nil {			
				Fail(fmt.Sprintf("exec \"/usr/sbin/ip addr add 192.168.10.1/24 dev eth1\" failed: \n\n%s\n", outputBuffer.String()))
			}
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "addr", "flush", itf3.Name }, &outputBuffer, &outputBuffer); err != nil {
				Fail(fmt.Sprintf("exec \"/usr/sbin/ip addr flush dev eth2\" failed: \n\n%s\n", outputBuffer.String()))
			}
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "addr", "add", "192.168.11.1/24", "dev", itf3.Name }, &outputBuffer, &outputBuffer); err != nil {			
				Fail(fmt.Sprintf("exec \"/usr/sbin/ip addr add 192.168.11.1/24 dev eth2\" failed: \n\n%s\n", outputBuffer.String()))
			}

			nc, err = network.NewNetworkContext()
			Expect(err).ToNot(HaveOccurred())

			fmt.Printf("\n>> default gateway : %+v\n", network.Network.DefaultIPv4Route)
			for _, r := range network.Network.StaticRoutes {
				fmt.Printf(">> static route : %+v\n", r)
			}
			fmt.Println()
		})
	
		AfterEach(func() {
			if skipTests {
				return
			}
	
			nc.Clear()
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "addr", "flush", itf2.Name }, &outputBuffer, &outputBuffer); err != nil {			
				fmt.Printf("exec \"/usr/sbin/ip addr flush dev eth1\" failed: \n\n%s\n", outputBuffer.String())
			}
			if err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "addr", "flush", itf3.Name }, &outputBuffer, &outputBuffer); err != nil {			
				fmt.Printf("exec \"/usr/sbin/ip addr flush dev eth2\" failed: \n\n%s\n", outputBuffer.String())
			}
		})

		FIt("creates a NAT route on an interface", func() {
			if skipTests {
				fmt.Println("No second interface so skipping test \"creates a NAT route on an interface\"...")
			}

			isAdmin, err := run.IsAdmin()
			Expect(err).ToNot(HaveOccurred())
			if !isAdmin {
				Fail("This test needs to be run with root privileges. i.e. sudo -E go test -v ./...")
			}

			routeManager, err := nc.NewRouteManager()
			Expect(err).ToNot(HaveOccurred())
			ritf1, err := routeManager.GetDefaultInterface()            // interface to world 
			Expect(err).ToNot(HaveOccurred())
			ritf2, err := routeManager.GetRoutableInterface(itf2.Name)  // interface to lan1
			Expect(err).ToNot(HaveOccurred())
			ritf3, err := routeManager.GetRoutableInterface(itf3.Name)  // interface to lan2
			Expect(err).ToNot(HaveOccurred())

			ritf2IP, ritf2NW, err := ritf2.Address4()
			Expect(err).ToNot(HaveOccurred())
			Expect(ritf2IP).To(Equal("192.168.10.1"))
			Expect(ritf2NW).To(Equal("192.168.10.0/24"))
			ritf3IP, ritf3NW, err := ritf3.Address4()
			Expect(err).ToNot(HaveOccurred())
			Expect(ritf3IP).To(Equal("192.168.11.1"))
			Expect(ritf3NW).To(Equal("192.168.11.0/24"))

			// forward packets from lan1 to world (ip v4)
			err = ritf1.FowardTrafficFrom(ritf2, network.LAN4, network.WORLD4, true)
			Expect(err).ToNot(HaveOccurred())
			// forward packets from lan1 to lan2 (ip v4)
			err = ritf3.FowardTrafficFrom(ritf2, network.LAN4, network.LAN4, false)
			Expect(err).ToNot(HaveOccurred())
			// forward packets from lan2 to external network 8.8.8.8/32 only (ip v4)
			err = ritf1.FowardTrafficFrom(ritf3, network.LAN4, "8.8.8.8/32", true)
			Expect(err).ToNot(HaveOccurred())
			// forward packets from lan2 to lan1 (ip v4)
			err = ritf2.FowardTrafficFrom(ritf3, network.LAN4, network.LAN4, false)
			Expect(err).ToNot(HaveOccurred())

			outputBuffer.Reset()
			err = run.RunAsAdminWithArgs([]string{ "/usr/sbin/ip", "route", "show" }, &outputBuffer, &outputBuffer)
			Expect(err).ToNot(HaveOccurred())
	
			fmt.Printf("\n# ip route show\n===\n%s===\n", outputBuffer.String())

			outputBuffer.Reset()
			err = run.RunAsAdminWithArgs([]string{ 
				"/bin/sh", "-c", 
				"nft list ruleset | sed -n '/^table ip mycs_router_ipv4 {/,/^}/p' | sed -n '/chain forward {/,/}/p'",
			}, &outputBuffer, &outputBuffer)
			Expect(err).ToNot(HaveOccurred())
	
			fmt.Printf("\n# nft list ruleset - forward rules after config\n")
			matched, unmatched := outputMatcher(outputBuffer, forwardRuleMatches)
			Expect(matched).To(Equal(6))
			Expect(unmatched).To(Equal(0))

			outputBuffer.Reset()
			err = run.RunAsAdminWithArgs([]string{ 
				"/bin/sh", "-c", 
				"nft list ruleset | sed -n '/^table ip mycs_router_ipv4 {/,/^}/p' | sed -n '/chain nat {/,/}/p'",
			}, &outputBuffer, &outputBuffer)
			Expect(err).ToNot(HaveOccurred())

			fmt.Printf("\n# nft list ruleset - nat rules after config\n")
			matched, unmatched = outputMatcher(outputBuffer, natRuleMatches)
			Expect(matched).To(Equal(2))
			Expect(unmatched).To(Equal(0))

			time.Sleep(time.Second * 15) // increase to pause for manual validation

			// delete forwarding rules
			err = ritf1.DeleteTrafficFowarding(ritf3, network.LAN4, "8.8.8.8/32")
			Expect(err).ToNot(HaveOccurred())

			outputBuffer.Reset()
			err = run.RunAsAdminWithArgs([]string{ 
				"/bin/sh", "-c", 
				"nft list ruleset | sed -n '/^table ip mycs_router_ipv4 {/,/^}/p' | sed -n '/chain forward {/,/}/p'",
			}, &outputBuffer, &outputBuffer)
			Expect(err).ToNot(HaveOccurred())

			fmt.Printf("\n# nft list ruleset - forward rules after deletion of rule\n")
			matched, unmatched = outputMatcher(outputBuffer, forwardRuleMatches)
			Expect(matched).To(Equal(5))
			Expect(unmatched).To(Equal(1))

			outputBuffer.Reset()
			err = run.RunAsAdminWithArgs([]string{ 
				"/bin/sh", "-c", 
				"nft list ruleset | sed -n '/^table ip mycs_router_ipv4 {/,/^}/p' | sed -n '/chain nat {/,/}/p'",
			}, &outputBuffer, &outputBuffer)
			Expect(err).ToNot(HaveOccurred())

			fmt.Printf("\n# nft list ruleset - nat rules after deletion of rule\n")
			matched, unmatched = outputMatcher(outputBuffer, natRuleMatches)
			Expect(matched).To(Equal(1))
			Expect(unmatched).To(Equal(1))

			time.Sleep(time.Second * 15) // increase to pause for manual validation
		})
	})
})

var forwardRuleMatches = []*regexp.Regexp{
	regexp.MustCompile(`^\s+ct state invalid drop\s*$`),
	regexp.MustCompile(`^\s+ct state established,related accept\s*$`),
	// routing between lan1 to lan2
	regexp.MustCompile(`^\s+iifname "eth1" oifname "eth2" ip saddr 192.168.10.0-192.168.10.255 ip daddr 192.168.11.0-192.168.11.255 accept\s*$`),
	regexp.MustCompile(`^\s+iifname "eth2" oifname "eth1" ip saddr 192.168.11.0-192.168.11.255 ip daddr 192.168.10.0-192.168.10.255 accept\s*$`),
	// allow lan1 access to internet
	regexp.MustCompile(`^\s+iifname "eth1" oifname "eth0" ip saddr 192.168.10.0-192.168.10.255 accept\s*$`),
	// allow lan2 access to only 8.8.8.8 externally
	regexp.MustCompile(`^\s+iifname "eth2" oifname "eth0" ip saddr 192.168.11.0-192.168.11.255 ip daddr 8.8.8.8-8.8.8.8 accept\s*$`),
}

var natRuleMatches = []*regexp.Regexp{
	// masq lan1 to world
	regexp.MustCompile(`^\s+oifname "eth0" ip saddr 192.168.10.0-192.168.10.255 masquerade\s*$`),
	// masq lan2 to only 8.8.8.8 externally
	regexp.MustCompile(`^\s+oifname "eth0" ip saddr 192.168.11.0-192.168.11.255 ip daddr 8.8.8.8-8.8.8.8 masquerade\s*$`),
}