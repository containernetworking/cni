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

package ops

import (
	"net"

	"github.com/containernetworking/cni/pkg/ns"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Net Operations", func() {
	Describe("Interfaces", func() {
		var netops NetOps
		var originalNS ns.NetNS

		BeforeEach(func() {
			netops = NewNetOps()

			// Create a new NetNS so we don't modify the host
			var err error
			originalNS, err = netops.NewNS()
			Expect(err).NotTo(HaveOccurred())
			err = originalNS.Set()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(originalNS.Close()).To(Succeed())
		})

		It("creates a veth pair in the default namespace", func() {
			veth := &netlink.Veth{
				LinkAttrs: netlink.LinkAttrs{
					Name:  "veth0",
					Flags: net.FlagUp,
				},
				PeerName: "veth1",
			}
			err := netops.LinkAdd(veth)
			Expect(err).NotTo(HaveOccurred())

			link, err := netops.LinkByName("veth0")
			Expect(err).NotTo(HaveOccurred())
			Expect(link).NotTo(Equal(nil))
			Expect(link.Attrs().Name).To(Equal("veth0"))
			// Even though we pass Flags: net.FlagUp it doesn't seem to happen
			Expect(link.Attrs().Flags & net.FlagUp).To(Equal(net.Flags(0)))

			peer, err := netops.LinkByName("veth1")
			Expect(err).NotTo(HaveOccurred())
			Expect(peer).NotTo(Equal(nil))
			Expect(peer.Attrs().Name).To(Equal("veth1"))
		})

		It("creates a veth pair in different namespaces and deletes them", func() {
			targetNs, err := netops.NewNS()
			Expect(err).NotTo(HaveOccurred())
			defer targetNs.Close()

			veth := &netlink.Veth{
				LinkAttrs: netlink.LinkAttrs{
					Name:  "veth0",
					Flags: net.FlagUp,
				},
				PeerName: "veth1",
			}

			// Create a dummy base link and a macvlan link in targetNs
			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				err = netops.LinkAdd(veth)
				Expect(err).NotTo(HaveOccurred())

				peer, err := netops.LinkByName("veth1")
				Expect(err).NotTo(HaveOccurred())
				Expect(peer).NotTo(Equal(nil))
				Expect(peer.Attrs().Name).To(Equal("veth1"))

				err = netops.LinkSetNsFd(peer, int(targetNs.Fd()))
				Expect(err).NotTo(HaveOccurred())
				return nil
			})

			// Make sure peer is in other namespace
			err = targetNs.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				peer, err := netops.LinkByName("veth1")
				Expect(err).NotTo(HaveOccurred())
				Expect(peer.Attrs().Name).To(Equal("veth1"))
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				// And not in the default namespace
				_, err := netops.LinkByName("veth1")
				Expect(err).To(HaveOccurred())

				// When deleted both links should no longer exist
				err = netops.LinkDel(veth)
				Expect(err).NotTo(HaveOccurred())
				_, err = netops.LinkByName("veth0")
				Expect(err).To(HaveOccurred())
				return nil
			})

			err = targetNs.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				_, err := netops.LinkByName("veth1")
				Expect(err).To(HaveOccurred())
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates an ipvlan link in a non-default namespace", func() {
			targetNs, err := netops.NewNS()
			Expect(err).NotTo(HaveOccurred())
			defer targetNs.Close()

			// Create a dummy base link and an ipvlan link in targetNs
			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				dummy := &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name: "dummy0",
					},
				}
				err = netops.LinkAdd(dummy)
				Expect(err).NotTo(HaveOccurred())
				dummyLink, err := netops.LinkByName("dummy0")
				Expect(err).NotTo(HaveOccurred())

				ipvl := &netlink.IPVlan{
					LinkAttrs: netlink.LinkAttrs{
						Name:        "ipvl0",
						ParentIndex: dummyLink.Attrs().Index,
						Namespace:   netlink.NsFd(int(targetNs.Fd())),
					},
					Mode: netlink.IPVLAN_MODE_L2,
				}
				err = netops.LinkAdd(ipvl)
				Expect(err).NotTo(HaveOccurred())
				return nil
			})

			// Make sure ipvlan link exists in other namespace
			err = targetNs.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				link, err := netops.LinkByName("ipvl0")
				Expect(err).NotTo(HaveOccurred())
				Expect(link.Attrs().Name).To(Equal("ipvl0"))
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
