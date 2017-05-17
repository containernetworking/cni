// Copyright 2017 CNI authors
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
	"time"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var linkName = "dummyCNI"

// Launch a pcap, send gratuitous arp, have phun
var _ = Describe("arp", func() {
	var (
		iface *net.Interface
		link  netlink.Link
	)

	// Create a dummy interface
	BeforeEach(func() {
		err := netlink.LinkAdd(&netlink.Dummy{netlink.LinkAttrs{Name: linkName}})
		Expect(err).NotTo(HaveOccurred())

		link, err = netlink.LinkByName(linkName)
		Expect(err).NotTo(HaveOccurred())

		err = netlink.LinkSetUp(link)
		Expect(err).NotTo(HaveOccurred())

		iface, err = net.InterfaceByName(linkName)
		Expect(err).NotTo(HaveOccurred())

	})

	// delete the interface
	AfterEach(func() {
		netlink.LinkDel(link)

	})

	It("Sends a gratuitious arp", func() {
		addr := net.IP{192, 0, 2, 1}

		handle, err := pcap.OpenLive(
			linkName,
			1024,           // snapshotlen
			true,           //promisc
			30*time.Second, //timeout
		)
		Expect(err).NotTo(HaveOccurred())
		defer handle.Close()

		err = handle.SetBPFFilter("arp")
		Expect(err).NotTo(HaveOccurred())

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

		ip.GratuitousArp(iface, addr)

		pkt, err := packetSource.NextPacket()

		Expect(err).NotTo(HaveOccurred())
		fmt.Fprintln(GinkgoWriter, pkt)

		arpLayer := pkt.Layer(layers.LayerTypeARP)
		Expect(arpLayer).ToNot(BeNil())
		arpPkt, ok := arpLayer.(*layers.ARP)
		Expect(ok).To(BeTrue())

		Expect(arpPkt.Operation).Should(Equal(uint16(layers.ARPRequest)))
		Expect(arpPkt.SourceHwAddress).Should(Equal([]byte(iface.HardwareAddr)))
		Expect(arpPkt.SourceProtAddress).Should(Equal([]byte(addr)))
		Expect(arpPkt.DstHwAddress).Should(Equal([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}))
		Expect(arpPkt.DstProtAddress).Should(Equal([]byte(addr)))

	})
})
