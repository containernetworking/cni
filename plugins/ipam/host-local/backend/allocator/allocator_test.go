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
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	fakestore "github.com/containernetworking/cni/plugins/ipam/host-local/backend/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type AllocatorTestCase struct {
	subnet       string
	ipmap        map[string]string
	expectResult string
	lastIP       string
}

func mkalloc() IPAllocator {
	ipnet, _ := types.ParseCIDR("192.168.1.0/24")

	r := Range{
		Subnet: types.IPNet(*ipnet),
	}
	r.Canonicalize()
	store := fakestore.NewFakeStore(map[string]string{}, map[int]net.IP{})

	alloc := IPAllocator{
		netName:  "netname",
		ipRange:  r,
		store:    store,
		rangeIdx: 0,
	}

	return alloc
}

func (t AllocatorTestCase) run(idx int) (*current.IPConfig, error) {
	fmt.Fprintln(GinkgoWriter, "Index:", idx)
	subnet, err := types.ParseCIDR(t.subnet)
	if err != nil {
		return nil, err
	}

	conf := Range{
		Subnet: types.IPNet(*subnet),
	}

	Expect(conf.Canonicalize()).To(BeNil())

	store := fakestore.NewFakeStore(t.ipmap, map[int]net.IP{0: net.ParseIP(t.lastIP)})

	alloc := IPAllocator{
		"netname",
		conf,
		store,
		0,
	}

	return alloc.Get("ID", nil)
}

var _ = Describe("host-local ip allocator", func() {
	Context("RangeIter", func() {
		It("should loop correctly from the beginning", func() {
			r := RangeIter{
				start: net.IP{10, 0, 0, 0},
				low:   net.IP{10, 0, 0, 0},
				high:  net.IP{10, 0, 0, 5},
			}
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 0}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 1}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 2}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 3}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 4}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 5}))
			Expect(r.Next()).To(BeNil())
		})

		It("should loop correctly from the end", func() {
			r := RangeIter{
				start: net.IP{10, 0, 0, 5},
				low:   net.IP{10, 0, 0, 0},
				high:  net.IP{10, 0, 0, 5},
			}
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 5}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 0}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 1}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 2}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 3}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 4}))
			Expect(r.Next()).To(BeNil())
		})

		It("should loop correctly from the middle", func() {
			r := RangeIter{
				start: net.IP{10, 0, 0, 3},
				low:   net.IP{10, 0, 0, 0},
				high:  net.IP{10, 0, 0, 5},
			}
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 3}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 4}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 5}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 0}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 1}))
			Expect(r.Next()).To(Equal(net.IP{10, 0, 0, 2}))
			Expect(r.Next()).To(BeNil())
		})

	})

	Context("when has free ip", func() {
		It("should allocate ips in round robin", func() {
			testCases := []AllocatorTestCase{
				// fresh start
				{
					subnet:       "10.0.0.0/29",
					ipmap:        map[string]string{},
					expectResult: "10.0.0.2",
					lastIP:       "",
				},
				{
					subnet:       "2001:db8:1::0/64",
					ipmap:        map[string]string{},
					expectResult: "2001:db8:1::2",
					lastIP:       "",
				},
				{
					subnet:       "10.0.0.0/30",
					ipmap:        map[string]string{},
					expectResult: "10.0.0.2",
					lastIP:       "",
				},
				{
					subnet: "10.0.0.0/29",
					ipmap: map[string]string{
						"10.0.0.2": "id",
					},
					expectResult: "10.0.0.3",
					lastIP:       "",
				},
				// next ip of last reserved ip
				{
					subnet:       "10.0.0.0/29",
					ipmap:        map[string]string{},
					expectResult: "10.0.0.6",
					lastIP:       "10.0.0.5",
				},
				{
					subnet: "10.0.0.0/29",
					ipmap: map[string]string{
						"10.0.0.4": "id",
						"10.0.0.5": "id",
					},
					expectResult: "10.0.0.6",
					lastIP:       "10.0.0.3",
				},
				// round robin to the beginning
				{
					subnet: "10.0.0.0/29",
					ipmap: map[string]string{
						"10.0.0.6": "id",
					},
					expectResult: "10.0.0.2",
					lastIP:       "10.0.0.5",
				},
				// lastIP is out of range
				{
					subnet: "10.0.0.0/29",
					ipmap: map[string]string{
						"10.0.0.2": "id",
					},
					expectResult: "10.0.0.3",
					lastIP:       "10.0.0.128",
				},
				// wrap around and reserve lastIP
				{
					subnet: "10.0.0.0/29",
					ipmap: map[string]string{
						"10.0.0.2": "id",
						"10.0.0.4": "id",
						"10.0.0.5": "id",
						"10.0.0.6": "id",
					},
					expectResult: "10.0.0.3",
					lastIP:       "10.0.0.3",
				},
			}

			for idx, tc := range testCases {
				res, err := tc.run(idx)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Address.IP.String()).To(Equal(tc.expectResult))
			}
		})

		It("should not allocate the broadcast address", func() {
			alloc := mkalloc()
			for i := 2; i < 255; i++ {
				res, err := alloc.Get("ID", nil)
				Expect(err).ToNot(HaveOccurred())
				s := fmt.Sprintf("192.168.1.%d/24", i)
				Expect(s).To(Equal(res.Address.String()))
				fmt.Fprintln(GinkgoWriter, "got ip", res.Address.String())
			}

			x, err := alloc.Get("ID", nil)
			fmt.Fprintln(GinkgoWriter, "got ip", x)
			Expect(err).To(HaveOccurred())
		})

		It("should allocate in a round-robin fashion", func() {
			alloc := mkalloc()
			res, err := alloc.Get("ID", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Address.String()).To(Equal("192.168.1.2/24"))

			err = alloc.Release("ID")
			Expect(err).ToNot(HaveOccurred())

			res, err = alloc.Get("ID", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Address.String()).To(Equal("192.168.1.3/24"))

		})

		It("should allocate RangeStart first", func() {
			alloc := mkalloc()
			alloc.ipRange.RangeStart = net.IP{192, 168, 1, 10}
			res, err := alloc.Get("ID", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Address.String()).To(Equal("192.168.1.10/24"))

			res, err = alloc.Get("ID", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Address.String()).To(Equal("192.168.1.11/24"))
		})

		It("should allocate RangeEnd but not past RangeEnd", func() {
			alloc := mkalloc()
			alloc.ipRange.RangeEnd = net.IP{192, 168, 1, 5}

			for i := 1; i < 5; i++ {
				res, err := alloc.Get("ID", nil)
				Expect(err).ToNot(HaveOccurred())
				// i+1 because the gateway address is skipped
				Expect(res.Address.String()).To(Equal(fmt.Sprintf("192.168.1.%d/24", i+1)))
			}

			_, err := alloc.Get("ID", nil)
			Expect(err).To(HaveOccurred())
		})

		Context("when requesting a specific IP", func() {
			It("must allocate the requested IP", func() {
				alloc := mkalloc()
				requestedIP := net.IP{192, 168, 1, 5}
				res, err := alloc.Get("ID", requestedIP)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Address.IP.String()).To(Equal(requestedIP.String()))
			})

			It("must fail when the requested IP is allocated", func() {
				alloc := mkalloc()
				requestedIP := net.IP{192, 168, 1, 5}
				res, err := alloc.Get("ID", requestedIP)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Address.IP.String()).To(Equal(requestedIP.String()))

				_, err = alloc.Get("ID", requestedIP)
				Expect(err).To(MatchError(`requested IP address "192.168.1.5" is not available in network: netname 192.168.1.0/24`))
			})

			It("must return an error when the requested IP is after RangeEnd", func() {
				alloc := mkalloc()
				alloc.ipRange.RangeEnd = net.IP{192, 168, 1, 5}
				requestedIP := net.IP{192, 168, 1, 6}
				_, err := alloc.Get("ID", requestedIP)
				Expect(err).To(HaveOccurred())
			})

			It("must return an error when the requested IP is before RangeStart", func() {
				alloc := mkalloc()
				alloc.ipRange.RangeStart = net.IP{192, 168, 1, 6}
				requestedIP := net.IP{192, 168, 1, 5}
				_, err := alloc.Get("ID", requestedIP)
				Expect(err).To(HaveOccurred())
			})
		})

	})
	Context("when out of ips", func() {
		It("returns a meaningful error", func() {
			testCases := []AllocatorTestCase{
				{
					subnet: "10.0.0.0/30",
					ipmap: map[string]string{
						"10.0.0.2": "id",
						"10.0.0.3": "id",
					},
				},
				{
					subnet: "10.0.0.0/29",
					ipmap: map[string]string{
						"10.0.0.2": "id",
						"10.0.0.3": "id",
						"10.0.0.4": "id",
						"10.0.0.5": "id",
						"10.0.0.6": "id",
						"10.0.0.7": "id",
					},
				},
			}
			for idx, tc := range testCases {
				_, err := tc.run(idx)
				Expect(err).To(MatchError("no IP addresses available in network: netname " + tc.subnet))
			}
		})
	})
})
