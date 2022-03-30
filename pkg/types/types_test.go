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

package types_test

import (
	"encoding/json"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Types", func() {

	Describe("ParseCIDR", func() {
		DescribeTable("Parse and stringify",
			func(input, expectedIP string, expectedMask int) {
				ipn, err := types.ParseCIDR(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(ipn.String()).To(Equal(input))

				Expect(ipn.IP.String()).To(Equal(expectedIP))
				ones, _ := ipn.Mask.Size()
				Expect(ones).To(Equal(expectedMask))
			},
			Entry("ipv4", "1.2.3.4/24", "1.2.3.4", 24),
			Entry("ipv6", "2001:db8::/32", "2001:db8::", 32),
		)
		It("returns an error when given invalid inputs", func() {
			ipn, err := types.ParseCIDR("1.2.3/45")
			Expect(ipn).To(BeNil())
			Expect(err).To(MatchError("invalid CIDR address: 1.2.3/45"))
		})
	})

	Describe("custom IPNet type", func() {
		It("marshals and unmarshals to JSON as a string", func() {
			ipn := types.IPNet{
				IP:   net.ParseIP("1.2.3.4"),
				Mask: net.CIDRMask(24, 32),
			}
			jsonBytes, err := json.Marshal(ipn)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonBytes).To(MatchJSON(`"1.2.3.4/24"`))

			var unmarshaled types.IPNet
			Expect(json.Unmarshal(jsonBytes, &unmarshaled)).To(Succeed())
			Expect(unmarshaled).To(Equal(ipn))
		})

		Context("when the json data is not syntactically valid", func() {
			Specify("UnmarshalJSON returns an error", func() {
				ipn := new(types.IPNet)
				err := ipn.UnmarshalJSON([]byte("1"))
				Expect(err).To(MatchError("json: cannot unmarshal number into Go value of type string"))
			})
		})

		Context("when the json data is not semantically valid", func() {
			Specify("UnmarshalJSON returns an error", func() {
				ipn := new(types.IPNet)
				err := ipn.UnmarshalJSON([]byte(`"1.2.3.4/99"`))
				Expect(err).To(MatchError("invalid CIDR address: 1.2.3.4/99"))
			})
		})
	})

	Describe("custom Route type", func() {
		var example types.Route
		BeforeEach(func() {
			example = types.Route{
				Dst: net.IPNet{
					IP:   net.ParseIP("1.2.3.0"),
					Mask: net.CIDRMask(24, 32),
				},
				GW: net.ParseIP("1.2.3.1"),
			}
		})

		It("marshals and unmarshals to JSON", func() {
			jsonBytes, err := json.Marshal(example)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonBytes).To(MatchJSON(`{ "dst": "1.2.3.0/24", "gw": "1.2.3.1" }`))

			var unmarshaled types.Route
			Expect(json.Unmarshal(jsonBytes, &unmarshaled)).To(Succeed())
			Expect(unmarshaled).To(Equal(example))
		})

		Context("when the json data is not valid", func() {
			Specify("UnmarshalJSON returns an error", func() {
				route := new(types.Route)
				err := route.UnmarshalJSON([]byte(`{ "dst": "1.2.3.0/24", "gw": "1.2.3.x" }`))
				Expect(err).To(MatchError("invalid IP address: 1.2.3.x"))
			})
		})

		It("formats as a string with a hex mask", func() {
			Expect(example.String()).To(Equal(`{Dst:{IP:1.2.3.0 Mask:ffffff00} GW:1.2.3.1}`))
		})
	})

	Describe("Error type", func() {
		var example *types.Error
		BeforeEach(func() {
			example = &types.Error{
				Code:    1234,
				Msg:     "some message",
				Details: "some details",
			}
		})

		Describe("Error() method (basic string)", func() {
			It("returns a formatted string", func() {
				Expect(example.Error()).To(Equal("some message; some details"))
			})
			Context("when details are not present", func() {
				BeforeEach(func() {
					example.Details = ""
				})
				It("returns only the message", func() {
					Expect(example.Error()).To(Equal("some message"))
				})
			})
		})

		It("NewError method", func() {
			err := types.NewError(1234, "some message", "some details")
			Expect(err).To(Equal(example))
		})
	})

	Describe("Result conversion", func() {
		var result *current.Result
		BeforeEach(func() {
			ipv4, err := types.ParseCIDR("1.2.3.30/24")
			Expect(err).NotTo(HaveOccurred())
			Expect(ipv4).NotTo(BeNil())

			ipv6, err := types.ParseCIDR("abcd:1234:ffff::cdde/64")
			Expect(err).NotTo(HaveOccurred())
			Expect(ipv6).NotTo(BeNil())

			result = &current.Result{
				CNIVersion: "1.0.0",
				Interfaces: []*current.Interface{
					{
						Name:    "eth0",
						Mac:     "00:11:22:33:44:55",
						Sandbox: "/proc/3553/ns/net",
					},
				},
				IPs: []*current.IPConfig{
					{
						Interface: current.Int(0),
						Address:   *ipv4,
						Gateway:   net.ParseIP("1.2.3.1"),
					},
					{
						Interface: current.Int(0),
						Address:   *ipv6,
						Gateway:   net.ParseIP("abcd:1234:ffff::1"),
					},
				},
				DNS: types.DNS{
					Nameservers: []string{"1.2.3.4", "1::cafe"},
					Domain:      "acompany.com",
					Search:      []string{"somedomain.com", "otherdomain.net"},
					Options:     []string{"foo", "bar"},
				},
			}
		})

		It("can create a CNIVersion '' (0.1.0) result", func() {
			newResult, err := result.GetAsVersion("")
			Expect(err).NotTo(HaveOccurred())
			Expect(newResult.Version()).To(Equal("0.1.0"))
		})
	})
})
