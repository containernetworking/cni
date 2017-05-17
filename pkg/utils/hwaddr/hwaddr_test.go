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

package hwaddr_test

import (
	"net"

	"github.com/containernetworking/cni/pkg/utils/hwaddr"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hwaddr", func() {
	Context("Generate Hardware Address for IPv4", func() {
		It("generate hardware address based on ipv4 address", func() {
			testCases := []struct {
				ip          net.IP
				expectedMAC net.HardwareAddr
			}{
				{
					ip:          net.ParseIP("10.0.0.2"),
					expectedMAC: (net.HardwareAddr)(append(hwaddr.PrivateMACPrefix, 0x0a, 0x00, 0x00, 0x02)),
				},
				{
					ip:          net.ParseIP("10.250.0.244"),
					expectedMAC: (net.HardwareAddr)(append(hwaddr.PrivateMACPrefix, 0x0a, 0xfa, 0x00, 0xf4)),
				},
				{
					ip:          net.ParseIP("172.17.0.2"),
					expectedMAC: (net.HardwareAddr)(append(hwaddr.PrivateMACPrefix, 0xac, 0x11, 0x00, 0x02)),
				},
				{
					ip:          net.IPv4(byte(172), byte(17), byte(0), byte(2)),
					expectedMAC: (net.HardwareAddr)(append(hwaddr.PrivateMACPrefix, 0xac, 0x11, 0x00, 0x02)),
				},
			}

			for _, tc := range testCases {
				mac, err := hwaddr.GenerateHardwareAddr4(tc.ip, hwaddr.PrivateMACPrefix)
				Expect(err).NotTo(HaveOccurred())
				Expect(mac).To(Equal(tc.expectedMAC))
			}
		})

		It("return error if IPv4 address is nil", func() {
			_, err := hwaddr.GenerateHardwareAddr4(nil, hwaddr.PrivateMACPrefix)
			Expect(err).To(BeAssignableToTypeOf(hwaddr.InvalidIP4Err{}))
		})

		It("return error if IPv4 address is invalid", func() {
			badIPs := []net.IP{
				net.ParseIP("10.0.0.2")[:1], // Invalid, 3 octets
				net.ParseIP("2001:db8::1"),  // IPv6 used as IPv4 address
			}
			for _, badIP := range badIPs {
				_, err := hwaddr.GenerateHardwareAddr4(badIP, hwaddr.PrivateMACPrefix)
				Expect(err).To(BeAssignableToTypeOf(hwaddr.InvalidIP4Err{}))
			}
		})

		It("return error if IPv4 hardware address prefix is invalid", func() {
			_, err := hwaddr.GenerateHardwareAddr4(net.ParseIP("10.0.0.2"), []byte{0x58})
			Expect(err).To(BeAssignableToTypeOf(hwaddr.InvalidPrefixLengthErr{}))
		})
	})

	Context("Generate Hardware Address for IPv6", func() {
		It("generate hardware address that includes a hard-coded prefix", func() {
			prefix := hwaddr.PrivateMACPrefixString6
			testCases := []struct {
				startMAC    string
				expectedMAC string
			}{
				{
					startMAC:    "02:42:d1:0e:5d:54",
					expectedMAC: prefix + ":d1:0e:5d:54",
				},
				{
					startMAC:    "1e:f7:4c:4e:1b:91",
					expectedMAC: prefix + ":4c:4e:1b:91",
				},
				{
					startMAC:    "52:54:00:80:f2:81",
					expectedMAC: prefix + ":00:80:f2:81",
				},
			}
			for _, tc := range testCases {
				hwaddrBefore, err := net.ParseMAC(tc.startMAC)
				Expect(err).NotTo(HaveOccurred())
				hwaddrAfter, err := hwaddr.GenerateHardwareAddr6(
					hwaddrBefore, hwaddr.PrivateMACPrefix6)
				Expect(err).NotTo(HaveOccurred())
				hwaddrExpected, err := net.ParseMAC(tc.expectedMAC)
				Expect(err).NotTo(HaveOccurred())
				Expect(hwaddrAfter).To(Equal(hwaddrExpected))
			}
		})

		It("return error if IPv6 hardware address prefix is invalid", func() {
			hwaddrBefore, err := net.ParseMAC("0e:01:02:03:04:05")
			Expect(err).NotTo(HaveOccurred())
			_, err = hwaddr.GenerateHardwareAddr6(hwaddrBefore, []byte{0x58})
			Expect(err).To(BeAssignableToTypeOf(hwaddr.InvalidPrefixLengthErr{}))
		})

	})
})
