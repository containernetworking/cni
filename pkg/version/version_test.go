// Copyright 2018 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version_test

import (
	"encoding/json"
	"net"
	"reflect"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version operations", func() {
	It("computes a list of versions correctly", func() {
		actual := version.VersionsStartingFrom("0.3.1")
		Expect(actual.SupportedVersions()).To(Equal([]string{"0.3.1", "0.4.0", "1.0.0"}))
	})

	Context("when a prevResult is available", func() {
		It("parses the prevResult", func() {
			rawBytes := []byte(`{
				"cniVersion": "1.0.0",
				"interfaces": [
					{
						"name": "eth0",
						"mac": "00:11:22:33:44:55",
						"sandbox": "/proc/3553/ns/net"
					}
				],
				"ips": [
					{
						"version": "4",
						"interface": 0,
						"address": "1.2.3.30/24",
						"gateway": "1.2.3.1"
					}
				]
			}`)
			var raw map[string]interface{}
			err := json.Unmarshal(rawBytes, &raw)
			Expect(err).NotTo(HaveOccurred())

			conf := &types.NetConf{
				CNIVersion:    "1.0.0",
				Name:          "foobar",
				Type:          "baz",
				RawPrevResult: raw,
			}

			err = version.ParsePrevResult(conf)
			Expect(err).NotTo(HaveOccurred())

			expectedResult := &current.Result{
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
						Address: net.IPNet{
							IP:   net.ParseIP("1.2.3.30"),
							Mask: net.IPv4Mask(255, 255, 255, 0),
						},
						Gateway: net.ParseIP("1.2.3.1"),
					},
				},
			}
			Expect(reflect.DeepEqual(conf.PrevResult, expectedResult)).To(BeTrue())
		})

		It("fails if the prevResult version is unknown", func() {
			conf := &types.NetConf{
				CNIVersion: current.ImplementedSpecVersion,
				Name:       "foobar",
				Type:       "baz",
				RawPrevResult: map[string]interface{}{
					"cniVersion": "5678.456",
				},
			}

			err := version.ParsePrevResult(conf)
			Expect(err).To(MatchError("could not parse prevResult: result type supports [1.0.0] but unmarshalled CNIVersion is \"5678.456\""))
		})

		It("fails if the prevResult version does not match the prevResult version", func() {
			conf := &types.NetConf{
				CNIVersion: current.ImplementedSpecVersion,
				Name:       "foobar",
				Type:       "baz",
				RawPrevResult: map[string]interface{}{
					"cniVersion": "0.2.0",
					"ip4": map[string]interface{}{
						"ip":      "1.2.3.30/24",
						"gateway": "1.2.3.1",
					},
				},
			}

			err := version.ParsePrevResult(conf)
			Expect(err).To(MatchError("could not parse prevResult: result type supports [1.0.0] but unmarshalled CNIVersion is \"0.2.0\""))
		})
	})

	Context("when a prevResult is not available", func() {
		It("does not fail", func() {
			conf := &types.NetConf{
				CNIVersion: current.ImplementedSpecVersion,
				Name:       "foobar",
				Type:       "baz",
			}

			err := version.ParsePrevResult(conf)
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.PrevResult).To(BeNil())
		})
	})
})
