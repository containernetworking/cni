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
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/testutils"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ptp Operations", func() {
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

	It("configures and deconfigures a ptp link with ADD/DEL", func() {
		const IFNAME = "ptp0"

		conf := `{
    "name": "mynet",
    "type": "ptp",
    "ipMasq": true,
    "mtu": 5000,
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24"
    }
}`

		targetNs, err := ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
		defer targetNs.Close()

		args := &skel.CmdArgs{
			ContainerID: "dummy",
			Netns:       targetNs.Path(),
			IfName:      IFNAME,
			StdinData:   []byte(conf),
		}

		// Execute the plugin with the ADD command, creating the veth endpoints
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err := testutils.CmdAddWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdAdd(args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure ptp link exists in the target namespace
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netlink.LinkByName(IFNAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(IFNAME))
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Call the plugins with the DEL command, deleting the veth endpoints
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err := testutils.CmdDelWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdDel(args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure ptp link has been deleted
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netlink.LinkByName(IFNAME)
			Expect(err).To(HaveOccurred())
			Expect(link).To(BeNil())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("configures and deconfigures a ptp link with ADD/DEL using TAP device", func() {
		// Used only to fill CNI_IFNAME, not actually used in this test
		const IFNAME = "tap"

		const TAP_PREFIX = "tap"

		conf := `{
    "name": "mynet",
    "type": "ptp",
    "ipMasq": false,
    "mtu": 5000,
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24"
    }
}`

		targetNs, err := ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
		defer targetNs.Close()

		args := &skel.CmdArgs{
			ContainerID:   "dummy",
			Netns:         targetNs.Path(),
			IfName:        IFNAME,
			StdinData:     []byte(conf),
			UsesTapDevice: true,
		}

		// Execute the plugin with the ADD command, creating the TAP device in the original namespace
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err := testutils.CmdAddWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdAdd(args)
			})
			Expect(err).NotTo(HaveOccurred())

			// Get the second link (lo being first one)
			link, err := netlink.LinkByIndex(2)
			// Tap devices are created with a tap prefix and a random id
			Expect(link.Attrs().Name[:3]).To(Equal(TAP_PREFIX))
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Call the plugins with the DEL command, deleting the TAP device
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netlink.LinkByIndex(2)
			Expect(err).NotTo(HaveOccurred())
			args.IfName = link.Attrs().Name
			Expect(args.IfName[:3]).To(Equal(TAP_PREFIX))

			err = testutils.CmdDelWithResult(targetNs.Path(), args.IfName, func() error {
				return cmdDel(args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure TAP ptp link has been deleted
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			// There should be no more interfaces/devices other than lo
			link, err := netlink.LinkByIndex(2)
			Expect(err).To(HaveOccurred())
			Expect(link).To(BeNil())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
