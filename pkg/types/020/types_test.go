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

package types020_test

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/create"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testResult(resultCNIVersion, jsonCNIVersion string) (*types020.Result, string) {
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

	// Set every field of the struct to ensure source compatibility
	res := &types020.Result{
		CNIVersion: resultCNIVersion,
		IP4: &types020.IPConfig{
			IP:      *ipv4,
			Gateway: net.ParseIP("1.2.3.1"),
			Routes: []types.Route{
				{Dst: *routev4, GW: routegwv4},
			},
		},
		IP6: &types020.IPConfig{
			IP:      *ipv6,
			Gateway: net.ParseIP("abcd:1234:ffff::1"),
			Routes: []types.Route{
				{Dst: *routev6, GW: routegwv6},
			},
		},
		DNS: types.DNS{
			Nameservers: []string{"1.2.3.4", "1::cafe"},
			Domain:      "acompany.com",
			Search:      []string{"somedomain.com", "otherdomain.net"},
			Options:     []string{"foo", "bar"},
		},
	}

	json := fmt.Sprintf(`{
    "cniVersion": "%s",
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
}`, jsonCNIVersion)

	return res, json
}

var _ = Describe("Ensures compatibility with the 0.1.0/0.2.0 spec", func() {
	It("correctly encodes a 0.2.0 Result", func() {
		res, expectedJSON := testResult(types020.ImplementedSpecVersion, types020.ImplementedSpecVersion)
		out, err := json.Marshal(res)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(MatchJSON(expectedJSON))
	})

	It("correctly encodes a 0.1.0 Result", func() {
		res, expectedJSON := testResult("0.1.0", "0.1.0")
		out, err := json.Marshal(res)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(MatchJSON(expectedJSON))
	})

	It("converts a 0.2.0 result to 0.1.0", func() {
		res, expectedJSON := testResult(types020.ImplementedSpecVersion, "0.1.0")
		res010, err := res.GetAsVersion("0.1.0")
		Expect(err).NotTo(HaveOccurred())
		out, err := json.Marshal(res010)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(MatchJSON(expectedJSON))
	})

	It("converts a 0.1.0 result to 0.2.0", func() {
		res, expectedJSON := testResult("0.1.0", types020.ImplementedSpecVersion)
		res020, err := res.GetAsVersion(types020.ImplementedSpecVersion)
		Expect(err).NotTo(HaveOccurred())
		out, err := json.Marshal(res020)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(MatchJSON(expectedJSON))
	})

	It("creates a 0.1.0 result passing CNIVersion ''", func() {
		_, expectedJSON := testResult("", "")
		resT, err := create.Create("", []byte(expectedJSON))
		Expect(err).NotTo(HaveOccurred())
		res010, ok := resT.(*types020.Result)
		Expect(ok).To(BeTrue())
		Expect(res010.CNIVersion).To(Equal("0.1.0"))
	})
})
