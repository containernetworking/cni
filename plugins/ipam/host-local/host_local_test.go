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

package main

import (
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const MASTER_NAME = "eth0"

var _ = Describe("host-local Operations", func() {
	var originalNS ns.NetNS
	var args skel.CmdArgs
	BeforeEach(func() {
		var err error
		originalNS, err = ns.NewNS()
		Expect(err).NotTo(HaveOccurred())

		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			err = netlink.LinkAdd(&netlink.Dummy{
				LinkAttrs: netlink.LinkAttrs{
					Name: MASTER_NAME,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			_, err = netlink.LinkByName(MASTER_NAME)
			Expect(err).NotTo(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(originalNS.Close()).To(Succeed())
	})

	It("assigns an IPv4 address", func() {
		args.StdinData = []byte(`{
					"name": "default",
					"ipam": {
						"version": "4",
						"type": "host-local",
						"subnet": "203.0.113.0/24"
					}
				}`)
		err1 := cmdAdd(&args)
		Expect(err1).ShouldNot(HaveOccurred())
		err2 := cmdDel(&args)
		Expect(err2).ShouldNot(HaveOccurred())
	})

	It("assigns an IPv6 address", func() {
		args.StdinData = []byte(`{
					"name": "default",
					"ipam6": {
						"version": "6",
						"type": "host-local",
						"subnet": "2001:db8::/32"
					}
				}`)
		err1 := cmdAdd(&args)
		Expect(err1).ShouldNot(HaveOccurred())
		err2 := cmdDel(&args)
		Expect(err2).ShouldNot(HaveOccurred())

	})

	It("assigns an IPv4 and IPv6 address", func() {
		args.StdinData = []byte(`{
					"name": "default",
						"ipam": {
							"version": "4",
							"type": "host-local",
							"subnet": "203.0.113.0/24"
						},
						"ipam6": {
							"version": "6",
							"type": "host-local",
							"subnet": "2001:db8::/32"
						}
					}`)
		err1 := cmdAdd(&args)
		Expect(err1).ShouldNot(HaveOccurred())
		err2 := cmdDel(&args)
		Expect(err2).ShouldNot(HaveOccurred())
	})
})
