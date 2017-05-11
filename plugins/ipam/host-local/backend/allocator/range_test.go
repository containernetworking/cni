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

package allocator

import (
	"net"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IP ranges", func() {
	It("should generate sane defaults for ipv4", func() {
		snstr := "192.0.2.0/24"
		r := Range{Subnet: mustSubnet(snstr)}

		err := r.Canonicalize()
		Expect(err).NotTo(HaveOccurred())

		Expect(r).To(Equal(Range{
			Subnet:     mustSubnet(snstr),
			RangeStart: net.IP{192, 0, 2, 2},
			RangeEnd:   net.IP{192, 0, 2, 254},
			Gateway:    net.IP{192, 0, 2, 1},
		}))
	})
	It("should generate sane defaults for a smaller ipv4 subnet", func() {
		snstr := "192.0.2.0/25"
		r := Range{Subnet: mustSubnet(snstr)}

		err := r.Canonicalize()
		Expect(err).NotTo(HaveOccurred())

		Expect(r).To(Equal(Range{
			Subnet:     mustSubnet(snstr),
			RangeStart: net.IP{192, 0, 2, 2},
			RangeEnd:   net.IP{192, 0, 2, 126},
			Gateway:    net.IP{192, 0, 2, 1},
		}))
	})
	It("should generate sane defaults for ipv6", func() {
		snstr := "2001:DB8:1::/64"
		r := Range{Subnet: mustSubnet(snstr)}

		err := r.Canonicalize()
		Expect(err).NotTo(HaveOccurred())

		Expect(r).To(Equal(Range{
			Subnet:     mustSubnet(snstr),
			RangeStart: net.ParseIP("2001:DB8:1::2"),
			RangeEnd:   net.ParseIP("2001:DB8:1::ffff:ffff:ffff:ffff"),
			Gateway:    net.ParseIP("2001:DB8:1::1"),
		}))
	})

	It("Should reject a network that's too small", func() {
		r := Range{Subnet: mustSubnet("192.0.2.0/31")}
		err := r.Canonicalize()
		Expect(err).Should(MatchError("Network 192.0.2.0/31 too small to allocate from"))
	})

	It("should reject invalid RangeStart and RangeEnd specifications", func() {
		r := Range{Subnet: mustSubnet("192.0.2.0/24"), RangeStart: net.ParseIP("192.0.3.0")}
		err := r.Canonicalize()
		Expect(err).Should(MatchError("192.0.3.0 not in network 192.0.2.0/24"))

		r = Range{Subnet: mustSubnet("192.0.2.0/24"), RangeEnd: net.ParseIP("192.0.4.0")}
		err = r.Canonicalize()
		Expect(err).Should(MatchError("192.0.4.0 not in network 192.0.2.0/24"))

		r = Range{
			Subnet:     mustSubnet("192.0.2.0/24"),
			RangeStart: net.ParseIP("192.0.2.50"),
			RangeEnd:   net.ParseIP("192.0.2.40"),
		}
		err = r.Canonicalize()
		Expect(err).Should(MatchError("192.0.2.50 is in network 192.0.2.0/24 but after end 192.0.2.40"))
	})

	It("should reject invalid gateways", func() {
		r := Range{Subnet: mustSubnet("192.0.2.0/24"), Gateway: net.ParseIP("192.0.3.0")}
		err := r.Canonicalize()
		Expect(err).Should(MatchError("gateway 192.0.3.0 not in network 192.0.2.0/24"))
	})

	It("should parse all fields correctly", func() {
		r := Range{
			Subnet:     mustSubnet("192.0.2.0/24"),
			RangeStart: net.ParseIP("192.0.2.40"),
			RangeEnd:   net.ParseIP("192.0.2.50"),
			Gateway:    net.ParseIP("192.0.2.254"),
		}
		err := r.Canonicalize()
		Expect(err).NotTo(HaveOccurred())

		Expect(r).To(Equal(Range{
			Subnet:     mustSubnet("192.0.2.0/24"),
			RangeStart: net.IP{192, 0, 2, 40},
			RangeEnd:   net.IP{192, 0, 2, 50},
			Gateway:    net.IP{192, 0, 2, 254},
		}))
	})

	It("should accept IPs in range and reject IPs out of range", func() {
		r := Range{
			Subnet:     mustSubnet("192.0.2.0/24"),
			RangeStart: net.ParseIP("192.0.2.40"),
			RangeEnd:   net.ParseIP("192.0.2.50"),
			Gateway:    net.ParseIP("192.0.2.254"),
		}
		err := r.Canonicalize()
		Expect(err).NotTo(HaveOccurred())

		Expect(r.IPInRange(net.ParseIP("192.0.3.0"))).Should(MatchError(
			"192.0.3.0 not in network 192.0.2.0/24"))

		Expect(r.IPInRange(net.ParseIP("192.0.2.39"))).Should(MatchError(
			"192.0.2.39 is in network 192.0.2.0/24 but before start 192.0.2.40"))
		Expect(r.IPInRange(net.ParseIP("192.0.2.40"))).Should(BeNil())
		Expect(r.IPInRange(net.ParseIP("192.0.2.50"))).Should(BeNil())
		Expect(r.IPInRange(net.ParseIP("192.0.2.51"))).Should(MatchError(
			"192.0.2.51 is in network 192.0.2.0/24 but after end 192.0.2.50"))
	})
})

func mustSubnet(s string) types.IPNet {
	n, err := types.ParseCIDR(s)
	if err != nil {
		Fail(err.Error())
	}
	canonicalizeIP(&n.IP)
	return types.IPNet(*n)
}
