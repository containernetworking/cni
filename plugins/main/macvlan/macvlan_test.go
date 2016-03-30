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

	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/ops"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/testutils"
	"github.com/containernetworking/cni/pkg/types"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const MASTER_NAME = "eth0"

var _ = Describe("macvlan Operations", func() {
	var netops ops.NetOps
	var originalNS ns.NetNS

	BeforeEach(func() {
		netops = ops.NewNetOps()

		// Create a new NetNS so we don't modify the host
		var err error
		originalNS, err = netops.NewNS()
		Expect(err).NotTo(HaveOccurred())
		err = originalNS.Set()
		Expect(err).NotTo(HaveOccurred())

		// Add master
		dummy := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name:      MASTER_NAME,
				Namespace: netlink.NsFd(int(originalNS.Fd())),
			},
		}
		err = netops.LinkAdd(dummy)
		Expect(err).NotTo(HaveOccurred())
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err = netops.LinkByName(MASTER_NAME)
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(originalNS.Close()).To(Succeed())
	})

	It("creates an macvlan link in a non-default namespace", func() {
		conf := &NetConf{
			NetConf: types.NetConf{
				Name: "testConfig",
				Type: "macvlan",
			},
			Master: MASTER_NAME,
			Mode:   "bridge",
			MTU:    1500,
		}

		targetNs, err := netops.NewNS()
		Expect(err).NotTo(HaveOccurred())
		defer targetNs.Close()

		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err = createMacvlan(netops, conf, "foobar0", targetNs)
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure macvlan link exists in the target namespace
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netops.LinkByName("foobar0")
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal("foobar0"))
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("configures and deconfigures a macvlan link with ADD/DEL", func() {
		const IFNAME = "macvl0"

		conf := fmt.Sprintf(`{
    "name": "mynet",
    "type": "macvlan",
    "master": "%s",
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24"
    }
}`, MASTER_NAME)

		targetNs, err := netops.NewNS()
		Expect(err).NotTo(HaveOccurred())
		defer targetNs.Close()

		args := &skel.CmdArgs{
			ContainerID: "dummy",
			Netns:       targetNs.Path(),
			IfName:      IFNAME,
			StdinData:   []byte(conf),
		}

		// Make sure macvlan link exists in the target namespace
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			_, err := testutils.CmdAddWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdAddInternal(netops, args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure macvlan link exists in the target namespace
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netops.LinkByName(IFNAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().Name).To(Equal(IFNAME))
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err := testutils.CmdDelWithResult(targetNs.Path(), IFNAME, func() error {
				return cmdDelInternal(netops, args)
			})
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Make sure macvlan link has been deleted
		err = targetNs.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			link, err := netops.LinkByName(IFNAME)
			Expect(err).To(HaveOccurred())
			Expect(link).To(BeNil())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
