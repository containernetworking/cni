// Copyright 2016 CNI authors
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

package ip_test

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/ns"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

var _ = Describe("Link", func() {
	const (
		ifaceFormatString string = "i%d"
		mtu               int    = 1400
		ip4onehwaddr             = "0a:58:01:01:01:01"
	)
	var (
		hostNetNS         ns.NetNS
		containerNetNS    ns.NetNS
		ifaceCounter      int = 0
		hostVeth          netlink.Link
		containerVeth     netlink.Link
		hostVethName      string
		containerVethName string
	)

	BeforeEach(func() {
		var err error

		hostNetNS, err = ns.NewNS()
		Expect(err).NotTo(HaveOccurred())

		containerNetNS, err = ns.NewNS()
		Expect(err).NotTo(HaveOccurred())

		_ = containerNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			hostVeth, containerVeth, err = ip.SetupVeth(fmt.Sprintf(ifaceFormatString, ifaceCounter), mtu, hostNetNS)
			if err != nil {
				return err
			}
			Expect(err).NotTo(HaveOccurred())

			hostVethName = hostVeth.Attrs().Name
			containerVethName = containerVeth.Attrs().Name

			return nil
		})
	})

	AfterEach(func() {
		Expect(containerNetNS.Close()).To(Succeed())
		Expect(hostNetNS.Close()).To(Succeed())
		ifaceCounter++
	})

	It("AddRoute sets a link scoped route when nil gateway provided", func() {
		_ = containerNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()
			initRoutes, err := netlink.RouteList(containerVeth, nl.FAMILY_ALL)
			Expect(err).NotTo(HaveOccurred())

			_, linknet, err := net.ParseCIDR("1.2.3.4/32")
			_, vianet, err := net.ParseCIDR("0.0.0.0/0")

			// This should fail, because it can't route to the gateway
			err = ip.AddRoute(vianet, net.ParseIP("1.2.3.4"), containerVeth)
			Expect(err).To(HaveOccurred())

			err = ip.AddRoute(linknet, nil, containerVeth)
			Expect(err).NotTo(HaveOccurred())

			err = ip.AddRoute(vianet, net.ParseIP("1.2.3.4"), containerVeth)
			Expect(err).NotTo(HaveOccurred())

			routes, err := netlink.RouteList(containerVeth, nl.FAMILY_ALL)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(len(initRoutes) + 2))

			return nil
		})
	})

	It("AddHostRoute sets a host scoped route", func() {
		_ = containerNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()
			initRoutes, err := netlink.RouteList(containerVeth, nl.FAMILY_ALL)
			Expect(err).NotTo(HaveOccurred())

			_, linknet, err := net.ParseCIDR("1.2.3.4/32")
			err = ip.AddHostRoute(linknet, nil, containerVeth)
			Expect(err).NotTo(HaveOccurred())

			routes, err := netlink.RouteList(containerVeth, nl.FAMILY_ALL)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(len(initRoutes) + 1))

			return nil
		})
	})

})
