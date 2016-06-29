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
	)
	var (
		hostNetNS         ns.NetNS
		containerNetNS    ns.NetNS
		ifaceCounter      int = 0
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

			hostVeth, containerVeth, err := ip.SetupVeth(fmt.Sprintf(ifaceFormatString, ifaceCounter), mtu, hostNetNS)
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

	It("SetupVeth must put the veth endpoints into the separate namespaces", func() {
		_ = containerNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err := netlink.LinkByName(containerVethName)
			Expect(err).NotTo(HaveOccurred())

			return nil
		})

		_ = hostNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err := netlink.LinkByName(hostVethName)
			Expect(err).NotTo(HaveOccurred())

			return nil
		})
	})

	It("DelLinkByName must delete the veth endpoints", func() {
		_ = containerNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			// this will delete the host endpoint too
			err := ip.DelLinkByName(containerVethName)
			Expect(err).NotTo(HaveOccurred())

			_, err = netlink.LinkByName(containerVethName)
			Expect(err).To(HaveOccurred())

			return nil
		})

		_ = hostNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err := netlink.LinkByName(hostVethName)
			Expect(err).To(HaveOccurred())

			return nil
		})
	})

	It("DelLinkByNameAddr must throw an error for configured interfaces", func() {
		_ = containerNetNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			// this will delete the host endpoint too
			addr, err := ip.DelLinkByNameAddr(containerVethName, nl.FAMILY_V4)
			Expect(err).To(HaveOccurred())

			var ipNetNil *net.IPNet
			Expect(addr).To(Equal(ipNetNil))
			return nil
		})
	})
})
