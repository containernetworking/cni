// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/testutils"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/current"

	"github.com/containernetworking/cni/pkg/utils/hwaddr"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	netConfStr = `
	"cniVersion": "%s",
	"name": "mynet",
	"type": "bridge",
	"bridge": "%s",
	"isDefaultGateway": true,
	"ipMasq": false`

	ipamConfStr4 = `,
	"ipam": {
		"version": "4",
		"type": "host-local",
		"subnet": "%s"
	}`
	ipamConfStr6 = `,
	"ipam6": {
		"version": "6",
		"type": "host-local",
		"subnet": "%s"
	}`
)

func processIPAMConf(format string, ipCIDR string) (string, net.IPNet) {
	conf := fmt.Sprintf(format, ipCIDR)
	netIP, subnet, err := net.ParseCIDR(ipCIDR)
	Expect(err).NotTo(HaveOccurred())
	gwAddr := ip.NextIP(netIP)
	return conf, net.IPNet{IP: gwAddr, Mask: subnet.Mask}
}

func checkBridgeConfig03x(version string, originalNS ns.NetNS) {
	const BRNAME = "cni0"
	const IFNAME = "eth0"

	testCases := []struct {
		ipCIDR4 string
		ipCIDR6 string
	}{
		{
			// IPv4 only
			ipCIDR4: "10.1.2.0/24",
		},
		// Enable the following IPv6 test cases after PR #279 and
		// PR #394 are committed.
		//{
		//	// IPv6 only
		//	ipCIDR6: "2001:db8::0/64",
		//},
		//{
		//	// Dual-Stack
		//	ipCIDR4: "192.168.0.0/24",
		//	ipCIDR6: "fd00::0/64",
		//},
	}

	for _, tc := range testCases {
		var gateways []net.IPNet
		var expMACPrefix string
		var expAddrs4, expAddrs6 = 0, 1 // IPv6 link local is automatic

		// Generate network config and calculate gateway addresses
		conf := fmt.Sprintf(netConfStr, version, BRNAME)
		if tc.ipCIDR4 != "" {
			ipamConf, gw := processIPAMConf(ipamConfStr4, tc.ipCIDR4)
			conf += ipamConf
			gateways = append(gateways, gw)
			expAddrs4++
			expMACPrefix = hwaddr.PrivateMACPrefixString
		}
		if tc.ipCIDR6 != "" {
			ipamConf, gw := processIPAMConf(ipamConfStr6, tc.ipCIDR6)
			conf += ipamConf
			gateways = append(gateways, gw)
			expAddrs6++
			// Uncomment the following lines when PR #394 is committed.
			//if expMACPrefix == "" {
			//	expMACPrefix = hwaddr.PrivateMACPrefixString6
			//}
		}
		conf = "{" + conf + "\n}"

		targetNs, err := ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
		defer targetNs.Close()

		args := &skel.CmdArgs{
			ContainerID: "dummy",
			Netns:       targetNs.Path(),
			IfName:      IFNAME,
			StdinData:   []byte(conf),
		}

		var result *current.Result
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			r, raw, err := testutils.CmdAddWithResult(targetNs.Path(), IFNAME, []byte(conf), func() error {
				return cmdAdd(args)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Index(string(raw), "\"interfaces\":")).Should(BeNumerically(">", 0))

			result, err = current.GetResult(r)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Interfaces)).To(Equal(3))
			Expect(result.Interfaces[0].Name).To(Equal(BRNAME))
			Expect(result.Interfaces[2].Name).To(Equal(IFNAME))

			// Make sure bridge link exists
			link, err := netlink.LinkByName(result.Interfaces[0].Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(BRNAME))
			Expect(link).To(BeAssignableToTypeOf(&netlink.Bridge{}))
			Expect(link.Attrs().HardwareAddr.String()).To(Equal(result.Interfaces[0].Mac))
			hwAddr := fmt.Sprintf("%s", link.Attrs().HardwareAddr)
			Expect(hwAddr).To(HavePrefix(expMACPrefix))

			// Ensure bridge has gateway address(es)
			addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(addrs)).To(BeNumerically(">", 0))
			for _, gw := range gateways {
				found := false
				subnetPrefix, subnetBits := gw.Mask.Size()
				for _, a := range addrs {
					aPrefix, aBits := a.IPNet.Mask.Size()
					if a.IPNet.IP.Equal(gw.IP) && aPrefix == subnetPrefix && aBits == subnetBits {
						found = true
						break
					}
				}
				Expect(found).To(Equal(true))
			}

			// Check for the veth link in the main namespace
			links, err := netlink.LinkList()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(links)).To(Equal(3)) // Bridge, veth, and loopback

			link, err = netlink.LinkByName(result.Interfaces[1].Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(BeAssignableToTypeOf(&netlink.Veth{}))
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Find the veth peer in the container namespace and the default route
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netlink.LinkByName(IFNAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(IFNAME))
			Expect(link).To(BeAssignableToTypeOf(&netlink.Veth{}))

			addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(addrs)).To(Equal(expAddrs4))
			addrs, err = netlink.AddrList(link, netlink.FAMILY_V6)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(addrs)).To(Equal(expAddrs6))

			hwAddr := fmt.Sprintf("%s", link.Attrs().HardwareAddr)
			Expect(hwAddr).To(HavePrefix(expMACPrefix))

			// Ensure the default route(s)
			routes, err := netlink.RouteList(link, 0)
			Expect(err).NotTo(HaveOccurred())

			var defaultRouteFound bool
			for _, gw := range gateways {
				for _, route := range routes {
					defaultRouteFound = (route.Dst == nil && route.Src == nil && route.Gw.Equal(gw.IP))
					if defaultRouteFound {
						break
					}
				}
				Expect(defaultRouteFound).To(Equal(true))
			}

			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err := testutils.CmdDelWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdDel(args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure the host veth has been deleted
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netlink.LinkByName(IFNAME)
			Expect(err).To(HaveOccurred())
			Expect(link).To(BeNil())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure the container veth has been deleted
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netlink.LinkByName(result.Interfaces[1].Name)
			Expect(err).To(HaveOccurred())
			Expect(link).To(BeNil())
			return nil
		})

		// Clean up bridge addresses for next test case
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			br, err := netlink.LinkByName(BRNAME)
			Expect(err).NotTo(HaveOccurred())
			for _, gw := range gateways {
				addr := netlink.Addr{IPNet: &gw}
				err = netlink.AddrDel(br, &addr)
				Expect(err).NotTo(HaveOccurred())
			}
			return nil
		})
	}
}

var _ = Describe("bridge Operations", func() {
	var originalNS ns.NetNS

	BeforeEach(func() {
		// Create a new NetNS so we don't modify the host
		var err error
		originalNS, err = ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(originalNS.Close()).To(Succeed())
	})

	It("creates a bridge", func() {
		const IFNAME = "bridge0"

		conf := &NetConf{
			NetConf: types.NetConf{
				CNIVersion: "0.3.1",
				Name:       "testConfig",
				Type:       "bridge",
			},
			BrName: IFNAME,
			IsGW:   false,
			IPMasq: false,
			MTU:    5000,
		}

		err := originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			bridge, _, err := setupBridge(conf)
			Expect(err).NotTo(HaveOccurred())
			Expect(bridge.Attrs().Name).To(Equal(IFNAME))

			// Double check that the link was added
			link, err := netlink.LinkByName(IFNAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(IFNAME))
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("handles an existing bridge", func() {
		const IFNAME = "bridge0"

		err := originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err := netlink.LinkAdd(&netlink.Bridge{
				LinkAttrs: netlink.LinkAttrs{
					Name: IFNAME,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			link, err := netlink.LinkByName(IFNAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(IFNAME))
			ifindex := link.Attrs().Index

			conf := &NetConf{
				NetConf: types.NetConf{
					CNIVersion: "0.3.1",
					Name:       "testConfig",
					Type:       "bridge",
				},
				BrName: IFNAME,
				IsGW:   false,
				IPMasq: false,
			}

			bridge, _, err := setupBridge(conf)
			Expect(err).NotTo(HaveOccurred())
			Expect(bridge.Attrs().Name).To(Equal(IFNAME))
			Expect(bridge.Attrs().Index).To(Equal(ifindex))

			// Double check that the link has the same ifindex
			link, err = netlink.LinkByName(IFNAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(IFNAME))
			Expect(link.Attrs().Index).To(Equal(ifindex))
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("configures and deconfigures a bridge and veth with default route with ADD/DEL for 0.3.0 config", func() {
		checkBridgeConfig03x("0.3.0", originalNS)
	})

	It("configures and deconfigures a bridge and veth with default route with ADD/DEL for 0.3.1 config", func() {
		checkBridgeConfig03x("0.3.1", originalNS)
	})

	It("deconfigures an unconfigured bridge with DEL", func() {
		const BRNAME = "cni0"
		const IFNAME = "eth0"

		_, subnet, err := net.ParseCIDR("10.1.2.1/24")
		Expect(err).NotTo(HaveOccurred())

		conf := fmt.Sprintf(`{
    "cniVersion": "0.3.0",
    "name": "mynet",
    "type": "bridge",
    "bridge": "%s",
    "isDefaultGateway": true,
    "ipMasq": false,
    "ipam": {
        "type": "host-local",
        "subnet": "%s"
    }
}`, BRNAME, subnet.String())

		targetNs, err := ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
		defer targetNs.Close()

		args := &skel.CmdArgs{
			ContainerID: "dummy",
			Netns:       targetNs.Path(),
			IfName:      IFNAME,
			StdinData:   []byte(conf),
		}

		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err := testutils.CmdDelWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdDel(args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("configures and deconfigures a bridge and veth with default route with ADD/DEL for 0.1.0 config", func() {
		const BRNAME = "cni0"
		const IFNAME = "eth0"

		testCases := []struct {
			ipCIDR4 string
			ipCIDR6 string
		}{
			{
				// IPv4 only
				ipCIDR4: "10.1.2.0/24",
			},
			// Enable the following IPv6 test case after PR #279 and
			// PR #394 are committed.
			//{
			//	// IPv6 only
			//	ipCIDR6: "2001:db8::0/64",
			//},
			// Dual-stack is not supported for CNI versions 0.2.0
			// or earlier since those versions do not support
			// multiple IP addresses.
		}
		for _, tc := range testCases {
			var gateways []net.IPNet
			var expMACPrefix string
			var expAddrs4, expAddrs6 = 0, 1 // IPv6 link local is automatic

			// Generate network config and calculate gateway addresses
			conf := fmt.Sprintf(netConfStr, "0.1.0", BRNAME)
			if tc.ipCIDR4 != "" {
				ipamConf, gw := processIPAMConf(ipamConfStr4, tc.ipCIDR4)
				conf += ipamConf
				gateways = append(gateways, gw)
				expAddrs4++
				expMACPrefix = hwaddr.PrivateMACPrefixString
			}
			if tc.ipCIDR6 != "" {
				ipamConf, gw := processIPAMConf(ipamConfStr6, tc.ipCIDR6)
				conf += ipamConf
				gateways = append(gateways, gw)
				expAddrs6++
				// Uncomment the following lines when PR #394 is committed.
				//if expMACPrefix == "" {
				//	expMACPrefix = hwaddr.PrivateMACPrefixString6
				//}
			}
			conf = "{" + conf + "\n}"

			targetNs, err := ns.NewNS()
			Expect(err).NotTo(HaveOccurred())
			defer targetNs.Close()

			args := &skel.CmdArgs{
				ContainerID: "dummy",
				Netns:       targetNs.Path(),
				IfName:      IFNAME,
				StdinData:   []byte(conf),
			}

			var result *types020.Result
			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				r, raw, err := testutils.CmdAddWithResult(targetNs.Path(), IFNAME, []byte(conf), func() error {
					return cmdAdd(args)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Index(string(raw), "\"ip4\":")).Should(BeNumerically(">", 0))

				// We expect a version 0.1.0 result
				result, err = types020.GetResult(r)
				Expect(err).NotTo(HaveOccurred())

				// Make sure bridge link exists
				link, err := netlink.LinkByName(BRNAME)
				Expect(err).NotTo(HaveOccurred())
				Expect(link.Attrs().Name).To(Equal(BRNAME))
				Expect(link).To(BeAssignableToTypeOf(&netlink.Bridge{}))
				hwAddr := fmt.Sprintf("%s", link.Attrs().HardwareAddr)
				Expect(hwAddr).To(HavePrefix(expMACPrefix))

				// Ensure bridge has gateway address(es)
				addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(addrs)).To(BeNumerically(">", 0))
				for _, gw := range gateways {
					found := false
					subnetPrefix, subnetBits := gw.Mask.Size()
					for _, a := range addrs {
						aPrefix, aBits := a.IPNet.Mask.Size()
						if a.IPNet.IP.Equal(gw.IP) && aPrefix == subnetPrefix && aBits == subnetBits {
							found = true
							break
						}
					}
					Expect(found).To(Equal(true))
				}

				// Check for the veth link in the main namespace; can't
				// check the for the specific link since version 0.1.0
				// doesn't report interfaces
				links, err := netlink.LinkList()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(links)).To(Equal(3)) // Bridge, veth, and loopback
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			// Find the veth peer in the container namespace and the default route
			err = targetNs.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				link, err := netlink.LinkByName(IFNAME)
				Expect(err).NotTo(HaveOccurred())
				Expect(link.Attrs().Name).To(Equal(IFNAME))
				Expect(link).To(BeAssignableToTypeOf(&netlink.Veth{}))

				addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(addrs)).To(Equal(expAddrs4))
				addrs, err = netlink.AddrList(link, netlink.FAMILY_V6)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(addrs)).To(Equal(expAddrs6))

				hwAddr := fmt.Sprintf("%s", link.Attrs().HardwareAddr)
				Expect(hwAddr).To(HavePrefix(expMACPrefix))

				// Ensure the default route
				routes, err := netlink.RouteList(link, 0)
				Expect(err).NotTo(HaveOccurred())

				var defaultRouteFound bool
				for _, gw := range gateways {
					for _, route := range routes {
						defaultRouteFound = (route.Dst == nil && route.Src == nil && route.Gw.Equal(gw.IP))
						if defaultRouteFound {
							break
						}
					}
					Expect(defaultRouteFound).To(Equal(true))
				}

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				err := testutils.CmdDelWithResult(targetNs.Path(), IFNAME, func() error {
					return cmdDel(args)
				})
				Expect(err).NotTo(HaveOccurred())
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			// Make sure the container veth has been deleted; cannot check
			// host veth as version 0.1.0 can't report its name
			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				link, err := netlink.LinkByName(IFNAME)
				Expect(err).To(HaveOccurred())
				Expect(link).To(BeNil())
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			// Clean up bridge addresses for next test case
			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				br, err := netlink.LinkByName(BRNAME)
				Expect(err).NotTo(HaveOccurred())
				for _, gw := range gateways {
					addr := netlink.Addr{IPNet: &gw}
					err = netlink.AddrDel(br, &addr)
					Expect(err).NotTo(HaveOccurred())
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("ensure bridge address", func() {
		const IFNAME = "bridge0"

		conf := &NetConf{
			NetConf: types.NetConf{
				CNIVersion: "0.3.1",
				Name:       "testConfig",
				Type:       "bridge",
			},
			BrName: IFNAME,
			IsGW:   true,
			IPMasq: false,
			MTU:    5000,
		}

		testCases := []struct {
			gwCIDRFirst  string
			gwCIDRSecond string
		}{
			{
				// IPv4
				gwCIDRFirst:  "10.0.0.1/8",
				gwCIDRSecond: "10.1.2.3/16",
			},
			{
				// IPv6
				gwCIDRFirst:  "2001:db8::1/32",
				gwCIDRSecond: "fd00:1:2:3::4/64",
			},
		}
		for _, tc := range testCases {

			gwIP, gwSubnet, err := net.ParseCIDR(tc.gwCIDRFirst)
			Expect(err).NotTo(HaveOccurred())
			gwnFirst := net.IPNet{IP: gwIP, Mask: gwSubnet.Mask}
			gwIP, gwSubnet, err = net.ParseCIDR(tc.gwCIDRSecond)
			Expect(err).NotTo(HaveOccurred())
			gwnSecond := net.IPNet{IP: gwIP, Mask: gwSubnet.Mask}

			var family, expNumAddrs int
			switch {
			case gwIP.To4() != nil:
				family = netlink.FAMILY_V4
				expNumAddrs = 1
			default:
				family = netlink.FAMILY_V6
				// Expect configured gw address plus link local
				expNumAddrs = 2
			}

			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				bridge, _, err := setupBridge(conf)
				Expect(err).NotTo(HaveOccurred())
				// Check if ForceAddress has default value
				Expect(conf.ForceAddress).To(Equal(false))

				err = ensureBridgeAddr(bridge, family, &gwnFirst, conf.ForceAddress)
				Expect(err).NotTo(HaveOccurred())

				//Check if IP address is set correctly
				addrs, err := netlink.AddrList(bridge, family)
				Expect(len(addrs)).To(Equal(expNumAddrs))
				addr := addrs[0].IPNet.String()
				Expect(addr).To(Equal(tc.gwCIDRFirst))

				//The bridge IP address has been changed. Error expected when ForceAddress is set to false.
				err = ensureBridgeAddr(bridge, family, &gwnSecond, false)
				Expect(err).To(HaveOccurred())

				//The IP address should stay the same.
				addrs, err = netlink.AddrList(bridge, family)
				Expect(len(addrs)).To(Equal(expNumAddrs))
				addr = addrs[0].IPNet.String()
				Expect(addr).To(Equal(tc.gwCIDRFirst))

				//Reconfigure IP when ForceAddress is set to true and IP address has been changed.
				err = ensureBridgeAddr(bridge, family, &gwnSecond, true)
				Expect(err).NotTo(HaveOccurred())

				//Retrieve the IP address after reconfiguration
				addrs, err = netlink.AddrList(bridge, family)
				Expect(len(addrs)).To(Equal(expNumAddrs))
				addr = addrs[0].IPNet.String()
				Expect(addr).To(Equal(tc.gwCIDRSecond))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
