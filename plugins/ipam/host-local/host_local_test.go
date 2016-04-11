package main

import (
	"github.com/appc/cni/pkg/skel"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HostLocal", func() {
	var (
		networkNS   string
		containerID string
		args        skel.CmdArgs
	)

	BeforeEach(func() {
		containerID = "some-container-id"
		networkNS = makeNetworkNS(containerID)

		args.ContainerID = containerID
		args.Netns = networkNS
		args.IfName = "blah"
		args.Path = "/some/test"
	})

	AfterEach(func() {
		Expect(removeNetworkNS(networkNS)).To(Succeed())
	})

	Context("test host local IPAM", func() {
		It("assigns an IPv4 address", func() {
			args.StdinData = []byte(`{
						"name": "default",
						"ipam": {
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
								"type": "host-local",
								"subnet": "203.0.113.0/24"
							},
							"ipam6": {
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
})
