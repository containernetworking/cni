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

package ipam

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
)

var _ = Describe("ipam utils", func() {
	Context("Generate Hardware Addrress", func() {
		It("generate hardware address based on ipv4 address", func() {
			testCases := []struct {
				ip          net.IP
				expectedMAC string
			}{
				{
					ip:          net.ParseIP("10.0.0.2"),
					expectedMAC: privateMACPrefix + ":0a:00:00:02",
				},
				{
					ip:          net.ParseIP("10.250.0.244"),
					expectedMAC: privateMACPrefix + ":0a:fa:00:f4",
				},
				{
					ip:          net.ParseIP("172.17.0.2"),
					expectedMAC: privateMACPrefix + ":ac:11:00:02",
				},
			}

			for _, tc := range testCases {
				mac, err := generateHardwareAddr(tc.ip)
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
				_, err := generateHardwareAddr(tc)
				Expect(err.Error()).To(Equal("generateHardwareAddr only support valid ipv4 address as input"))
			}
		})
	})
})
