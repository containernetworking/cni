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

package types100_test

import (
	"encoding/json"
	"io"
	"net"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
)

func testResult() *current.Result {
	ipv4, err := types.ParseCIDR("1.2.3.30/24")
	Expect(err).NotTo(HaveOccurred())
	Expect(ipv4).NotTo(BeNil())

	routegwv4, routev4, err := net.ParseCIDR("15.5.6.8/24")
	Expect(err).NotTo(HaveOccurred())
	Expect(routev4).NotTo(BeNil())
	Expect(routegwv4).NotTo(BeNil())

	ipv6, err := types.ParseCIDR("abcd:1234:ffff::cdde/64")
	Expect(err).NotTo(HaveOccurred())
	Expect(ipv6).NotTo(BeNil())

	routegwv6, routev6, err := net.ParseCIDR("1111:dddd::aaaa/80")
	Expect(err).NotTo(HaveOccurred())
	Expect(routev6).NotTo(BeNil())
	Expect(routegwv6).NotTo(BeNil())

	mtu4 := 1450
	mtu6 := 1280

	// Set every field of the struct to ensure source compatibility
	return &current.Result{
		CNIVersion: current.ImplementedSpecVersion,
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
		Routes: []*current.Route{
			{Dst: *routev4, GW: routegwv4, MTU: &mtu4},
			{Dst: *routev6, GW: routegwv6, MTU: &mtu6},
		},
		DNS: types.DNS{
			Nameservers: []string{"1.2.3.4", "1::cafe"},
			Domain:      "acompany.com",
			Search:      []string{"somedomain.com", "otherdomain.net"},
			Options:     []string{"foo", "bar"},
		},
	}
}

var _ = Describe("Current types operations", func() {
	It("correctly encodes a 1.1.0 Result", func() {
		res := testResult()

		// Redirect stdout to capture JSON result
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())

		os.Stdout = w
		err = res.Print()
		w.Close()
		Expect(err).NotTo(HaveOccurred())

		// parse the result
		out, err := io.ReadAll(r)
		os.Stdout = oldStdout
		Expect(err).NotTo(HaveOccurred())

		Expect(string(out)).To(MatchJSON(`{
    "cniVersion": "` + current.ImplementedSpecVersion + `",
    "interfaces": [
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:55",
            "sandbox": "/proc/3553/ns/net"
        }
    ],
    "ips": [
        {
            "interface": 0,
            "address": "1.2.3.30/24",
            "gateway": "1.2.3.1"
        },
        {
            "interface": 0,
            "address": "abcd:1234:ffff::cdde/64",
            "gateway": "abcd:1234:ffff::1"
        }
    ],
    "routes": [
        {
            "dst": "15.5.6.0/24",
            "gw": "15.5.6.8",
            "mtu": 1450
        },
        {
            "dst": "1111:dddd::/80",
            "gw": "1111:dddd::aaaa",
            "mtu": 1280
        }
    ],
    "dns": {
        "nameservers": [
            "1.2.3.4",
            "1::cafe"
        ],
        "domain": "acompany.com",
        "search": [
            "somedomain.com",
            "otherdomain.net"
        ],
        "options": [
            "foo",
            "bar"
        ]
    }
}`))
	})

	It("correctly converts a 1.0.0 Result to 1.1.0", func() {
		tr, err := testResult().GetAsVersion("1.0.0")
		Expect(err).NotTo(HaveOccurred())

		trv1, ok := tr.(*current.Result)
		Expect(ok).To(BeTrue())

		// 1.0.0 and 1.1.0 should be the same except for CNI version
		Expect(trv1.CNIVersion).To(Equal("1.0.0"))

		// If we convert 1.0.0 back to 1.1.0 it should be identical
		trv11, err := trv1.GetAsVersion("1.1.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(trv11).To(Equal(testResult()))
	})

	It("correctly encodes a 0.1.0 Result", func() {
		res, err := testResult().GetAsVersion("0.1.0")
		Expect(err).NotTo(HaveOccurred())

		// Redirect stdout to capture JSON result
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())

		os.Stdout = w
		err = res.Print()
		w.Close()
		Expect(err).NotTo(HaveOccurred())

		// parse the result
		out, err := io.ReadAll(r)
		os.Stdout = oldStdout
		Expect(err).NotTo(HaveOccurred())

		Expect(string(out)).To(MatchJSON(`{
    "cniVersion": "0.1.0",
    "ip4": {
        "ip": "1.2.3.30/24",
        "gateway": "1.2.3.1",
        "routes": [
            {
                "dst": "15.5.6.0/24",
                "gw": "15.5.6.8"
            }
        ]
    },
    "ip6": {
        "ip": "abcd:1234:ffff::cdde/64",
        "gateway": "abcd:1234:ffff::1",
        "routes": [
            {
                "dst": "1111:dddd::/80",
                "gw": "1111:dddd::aaaa"
            }
        ]
    },
    "dns": {
        "nameservers": [
            "1.2.3.4",
            "1::cafe"
        ],
        "domain": "acompany.com",
        "search": [
            "somedomain.com",
            "otherdomain.net"
        ],
        "options": [
            "foo",
            "bar"
        ]
    }
}`))
	})

	It("correctly encodes a 0.4.0 Result", func() {
		res, err := testResult().GetAsVersion("0.4.0")
		Expect(err).NotTo(HaveOccurred())

		// Redirect stdout to capture JSON result
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())

		os.Stdout = w
		err = res.Print()
		w.Close()
		Expect(err).NotTo(HaveOccurred())

		// parse the result
		out, err := io.ReadAll(r)
		os.Stdout = oldStdout
		Expect(err).NotTo(HaveOccurred())

		Expect(string(out)).To(MatchJSON(`{
    "cniVersion": "0.4.0",
    "interfaces": [
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:55",
            "sandbox": "/proc/3553/ns/net"
        }
    ],
    "ips": [
        {
            "interface": 0,
            "version": "4",
            "address": "1.2.3.30/24",
            "gateway": "1.2.3.1"
        },
        {
            "interface": 0,
            "version": "6",
            "address": "abcd:1234:ffff::cdde/64",
            "gateway": "abcd:1234:ffff::1"
        }
    ],
    "routes": [
        {
            "dst": "15.5.6.0/24",
            "gw": "15.5.6.8"
        },
        {
            "dst": "1111:dddd::/80",
            "gw": "1111:dddd::aaaa"
        }
    ],
    "dns": {
        "nameservers": [
            "1.2.3.4",
            "1::cafe"
        ],
        "domain": "acompany.com",
        "search": [
            "somedomain.com",
            "otherdomain.net"
        ],
        "options": [
            "foo",
            "bar"
        ]
    }
}`))
	})

	It("correctly marshals and unmarshals interface index 0", func() {
		ipc := &current.IPConfig{
			Interface: current.Int(0),
			Address: net.IPNet{
				IP:   net.ParseIP("10.1.2.3"),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			},
		}

		jsonBytes, err := json.Marshal(ipc)
		Expect(err).NotTo(HaveOccurred())
		Expect(jsonBytes).To(MatchJSON(`{
    "interface": 0,
    "address": "10.1.2.3/24"
}`))

		recovered := &current.IPConfig{}
		Expect(json.Unmarshal(jsonBytes, recovered)).To(Succeed())
		Expect(recovered).To(Equal(ipc))
	})

	Context("when unmarshalling json fails", func() {
		It("returns an error", func() {
			recovered := &current.IPConfig{}
			err := json.Unmarshal([]byte(`{"address": 5}`), recovered)
			Expect(err).To(MatchError(HavePrefix("json: cannot unmarshal")))
		})
	})

	It("correctly marshals a missing interface index", func() {
		ipc := &current.IPConfig{
			Address: net.IPNet{
				IP:   net.ParseIP("10.1.2.3"),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			},
		}

		json, err := json.Marshal(ipc)
		Expect(err).NotTo(HaveOccurred())
		Expect(json).To(MatchJSON(`{
    "address": "10.1.2.3/24"
}`))
	})

	Context("when setting custom routes with mtu 1400", func() {
		notzero := 1400

		exampleWithMTU := current.Route{
			Dst: net.IPNet{
				IP:   net.ParseIP("1.2.3.0"),
				Mask: net.CIDRMask(24, 32),
			},
			GW:  net.ParseIP("1.2.3.1"),
			MTU: &notzero,
		}

		It("marshals to JSON equivalent with mtu set ", func() {
			jsonBytes, err := json.Marshal(&exampleWithMTU)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonBytes).To(MatchJSON(`{ "dst": "1.2.3.0/24", "gw": "1.2.3.1", "mtu": 1400 }`))
		})
	})

	Context("when setting custom routes with mtu 0", func() {
		zero := 0

		exampleZeroMTU := current.Route{
			Dst: net.IPNet{
				IP:   net.ParseIP("1.2.3.0"),
				Mask: net.CIDRMask(24, 32),
			},
			GW:  net.ParseIP("1.2.3.1"),
			MTU: &zero,
		}

		It("marshals to JSON equivalent to an omitted mtu ", func() {
			jsonBytes, err := json.Marshal(&exampleZeroMTU)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonBytes).To(MatchJSON(`{ "dst": "1.2.3.0/24", "gw": "1.2.3.1" }`))
		})
	})
	Context("when setting custom routes without mtu", func() {
		exampleNilMTU := current.Route{
			Dst: net.IPNet{
				IP:   net.ParseIP("1.2.3.0"),
				Mask: net.CIDRMask(24, 32),
			},
			GW: net.ParseIP("1.2.3.1"),
		}

		It("marshals and unmarshals to JSON", func() {
			jsonBytes, err := json.Marshal(&exampleNilMTU)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonBytes).To(MatchJSON(`{ "dst": "1.2.3.0/24", "gw": "1.2.3.1" }`))

			var unmarshaled current.Route
			Expect(json.Unmarshal(jsonBytes, &unmarshaled)).To(Succeed())
			Expect(unmarshaled).To(Equal(exampleNilMTU))
		})

		Context("when the json data is not valid", func() {
			Specify("UnmarshalJSON returns an error", func() {
				route := new(current.Route)
				err := route.UnmarshalJSON([]byte(`{ "dst": "1.2.3.0/24", "gw": "1.2.3.x" }`))
				Expect(err).To(MatchError("invalid IP address: 1.2.3.x"))
			})
		})

		It("formats as a string with a hex mask", func() {
			Expect(exampleNilMTU.String()).To(Equal(`{Dst:{IP:1.2.3.0 Mask:ffffff00} GW:1.2.3.1 MTU:<nil>}`))
		})
	})
})
