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

package allocator

import (
	"net"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IPAM config", func() {
	It("Should parse an old-style config", func() {
		input := `{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24",
		"rangeStart": "10.1.2.9",
		"rangeEnd": "10.1.2.20",
		"gateway": "10.1.2.30"
    }
}`
		conf, version, err := LoadIPAMConfig([]byte(input), "")
		Expect(err).NotTo(HaveOccurred())
		Expect(version).Should(Equal("0.3.1"))

		Expect(conf).To(Equal(&IPAMConfig{
			Name: "mynet",
			Type: "host-local",
			Ranges: []Range{
				{
					RangeStart: net.IP{10, 1, 2, 9},
					RangeEnd:   net.IP{10, 1, 2, 20},
					Gateway:    net.IP{10, 1, 2, 30},
					Subnet: types.IPNet{
						IP:   net.IP{10, 1, 2, 0},
						Mask: net.CIDRMask(24, 32),
					},
				},
			},
		}))
	})
	It("Should parse a new-style config", func() {
		input := `{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
    "ipam": {
        "type": "host-local",
		"ranges": [
			{
				"subnet": "10.1.2.0/24",
				"rangeStart": "10.1.2.9",
				"rangeEnd": "10.1.2.20",
				"gateway": "10.1.2.30"
			},
			{
				"subnet": "11.1.2.0/24",
				"rangeStart": "11.1.2.9",
				"rangeEnd": "11.1.2.20",
				"gateway": "11.1.2.30"
			}
		]
    }
}`
		conf, version, err := LoadIPAMConfig([]byte(input), "")
		Expect(err).NotTo(HaveOccurred())
		Expect(version).Should(Equal("0.3.1"))

		Expect(conf).To(Equal(&IPAMConfig{
			Name: "mynet",
			Type: "host-local",
			Ranges: []Range{
				{
					RangeStart: net.IP{10, 1, 2, 9},
					RangeEnd:   net.IP{10, 1, 2, 20},
					Gateway:    net.IP{10, 1, 2, 30},
					Subnet: types.IPNet{
						IP:   net.IP{10, 1, 2, 0},
						Mask: net.CIDRMask(24, 32),
					},
				},
				{
					RangeStart: net.IP{11, 1, 2, 9},
					RangeEnd:   net.IP{11, 1, 2, 20},
					Gateway:    net.IP{11, 1, 2, 30},
					Subnet: types.IPNet{
						IP:   net.IP{11, 1, 2, 0},
						Mask: net.CIDRMask(24, 32),
					},
				},
			},
		}))
	})

	It("Should parse a mixed config", func() {
		input := `{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24",
		"rangeStart": "10.1.2.9",
		"rangeEnd": "10.1.2.20",
		"gateway": "10.1.2.30",
		"ranges": [
			{
				"subnet": "11.1.2.0/24",
				"rangeStart": "11.1.2.9",
				"rangeEnd": "11.1.2.20",
				"gateway": "11.1.2.30"
			}
		]
    }
}`
		conf, version, err := LoadIPAMConfig([]byte(input), "")
		Expect(err).NotTo(HaveOccurred())
		Expect(version).Should(Equal("0.3.1"))

		Expect(conf).To(Equal(&IPAMConfig{
			Name: "mynet",
			Type: "host-local",
			Ranges: []Range{
				{
					RangeStart: net.IP{10, 1, 2, 9},
					RangeEnd:   net.IP{10, 1, 2, 20},
					Gateway:    net.IP{10, 1, 2, 30},
					Subnet: types.IPNet{
						IP:   net.IP{10, 1, 2, 0},
						Mask: net.CIDRMask(24, 32),
					},
				},
				{
					RangeStart: net.IP{11, 1, 2, 9},
					RangeEnd:   net.IP{11, 1, 2, 20},
					Gateway:    net.IP{11, 1, 2, 30},
					Subnet: types.IPNet{
						IP:   net.IP{11, 1, 2, 0},
						Mask: net.CIDRMask(24, 32),
					},
				},
			},
		}))
	})

	It("Should parse CNI_ARGS env", func() {
		input := `{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
    "ipam": {
        "type": "host-local",
		"ranges": [
			{
				"subnet": "10.1.2.0/24",
				"rangeStart": "10.1.2.9",
				"rangeEnd": "10.1.2.20",
				"gateway": "10.1.2.30"
			},
			{
				"subnet": "11.1.2.0/24",
				"rangeStart": "11.1.2.9",
				"rangeEnd": "11.1.2.20",
				"gateway": "11.1.2.30"
			}
		]
    }
}`

		envArgs := "IP=10.1.2.10"

		conf, _, err := LoadIPAMConfig([]byte(input), envArgs)
		Expect(err).NotTo(HaveOccurred())
		Expect(conf.IPArgs).To(Equal([]net.IP{{10, 1, 2, 10}}))

	})
	It("Should parse config args", func() {
		input := `{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
	"args": {
		"cni": {
			"ips": [ "10.1.2.11", "11.11.11.11"]
		}
	},
    "ipam": {
        "type": "host-local",
		"ranges": [
			{
				"subnet": "10.1.2.0/24",
				"rangeStart": "10.1.2.9",
				"rangeEnd": "10.1.2.20",
				"gateway": "10.1.2.30"
			},
			{
				"subnet": "11.1.2.0/24",
				"rangeStart": "11.1.2.9",
				"rangeEnd": "11.1.2.20",
				"gateway": "11.1.2.30"
			}
		]
    }
}`

		envArgs := "IP=10.1.2.10"

		conf, _, err := LoadIPAMConfig([]byte(input), envArgs)
		Expect(err).NotTo(HaveOccurred())
		Expect(conf.IPArgs).To(Equal([]net.IP{{10, 1, 2, 10}, {10, 1, 2, 11}, {11, 11, 11, 11}}))

	})
})
