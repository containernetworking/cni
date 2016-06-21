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

package ipam_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"

	"github.com/containernetworking/cni/pkg/ipam"
)

var _ = Describe("ipam utils", func() {
	Context("Generate Hardware Address", func() {
		It("generate hardware address based on ipv4 address", func() {
			testCases := []struct {
				ip          net.IP
				expectedMAC string
			}{
				{
					ip:          net.ParseIP("10.0.0.2"),
					expectedMAC: ipam.PrivateMACPrefix + ":0a:00:00:02",
				},
				{
					ip:          net.ParseIP("10.250.0.244"),
					expectedMAC: ipam.PrivateMACPrefix + ":0a:fa:00:f4",
				},
				{
					ip:          net.ParseIP("172.17.0.2"),
					expectedMAC: ipam.PrivateMACPrefix + ":ac:11:00:02",
				},
			}

			for _, tc := range testCases {
				mac, err := ipam.GenerateHardwareAddr4(tc.ip, ipam.PrivateMACPrefix)
				Expect(err).NotTo(HaveOccurred())
				Expect(mac.String()).To(Equal(tc.expectedMAC))
			}
		})

		It("return error if input is not ipv4 address", func() {
			testCases := []net.IP{
				net.ParseIP(""),
				net.ParseIP("2001:db8:0:1:1:1:1:1"),
			}
			for _, tc := range testCases {
				_, err := ipam.GenerateHardwareAddr4(tc, ipam.PrivateMACPrefix)
				Expect(err).To(BeAssignableToTypeOf(ipam.SupportIp4OnlyErr{}))
			}
		})

		It("return error if prefix is invalid", func() {
			_, err := ipam.GenerateHardwareAddr4(net.ParseIP("10.0.0.2"), "")
			Expect(err).To(BeAssignableToTypeOf(ipam.InvalidPrefixLengthErr{}))
		})
	})
})
