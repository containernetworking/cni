// Copyright 2026 CNI authors
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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/types"
	types040 "github.com/containernetworking/cni/pkg/types/040"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/types/create"
)

func convertedRoutes(result types.Result) []*types.Route {
	switch result := result.(type) {
	case *types040.Result:
		return result.Routes
	case *current.Result:
		return result.Routes
	default:
		Fail(fmt.Sprintf("unexpected result type %T", result))
		return nil
	}
}

var _ = Describe("Route presence during result conversion", func() {
	versions := []string{"0.3.0", "0.3.1", "0.4.0", "1.0.0", "1.1.0"}

	DescribeTable("preserves absent and explicitly empty routes in legacy IPAM results",
		func(input string, absent bool) {
			result, err := create.CreateFromBytes([]byte(input))
			Expect(err).NotTo(HaveOccurred())
			for _, version := range versions {
				converted, err := result.GetAsVersion(version)
				Expect(err).NotTo(HaveOccurred())
				routes := convertedRoutes(converted)
				// IPAM consumers can use nil to decide whether to add a default route.
				if absent {
					Expect(routes).To(BeNil(), "converting to %s", version)
				} else {
					Expect(routes).NotTo(BeNil(), "converting to %s", version)
				}
				Expect(routes).To(BeEmpty())
			}
		},
		Entry("omitted version and IPv4 routes", `{"ip4":{"ip":"10.44.0.2/24","gateway":"10.44.0.1"}}`, true),
		Entry("IPv6 routes", `{"cniVersion":"0.2.0","ip6":{"ip":"2001:db8::2/64","gateway":"2001:db8::1"}}`, true),
		Entry("dual-stack routes", `{"cniVersion":"0.2.0","ip4":{"ip":"10.44.0.2/24"},"ip6":{"ip":"2001:db8::2/64"}}`, true),
		Entry("explicitly empty IPv4 routes", `{"cniVersion":"0.2.0","ip4":{"ip":"10.44.0.2/24","routes":[]}}`, false),
		Entry("explicitly empty IPv6 routes", `{"cniVersion":"0.2.0","ip6":{"ip":"2001:db8::2/64","routes":[]}}`, false),
	)

	DescribeTable("preserves route presence between list-shaped result versions",
		func(routesJSON string, absent bool, count int) {
			for _, fromVersion := range versions {
				input := fmt.Sprintf(`{"cniVersion":%q%s}`, fromVersion, routesJSON)
				result, err := create.CreateFromBytes([]byte(input))
				Expect(err).NotTo(HaveOccurred())
				for _, toVersion := range versions {
					converted, err := result.GetAsVersion(toVersion)
					Expect(err).NotTo(HaveOccurred())
					routes := convertedRoutes(converted)
					if absent {
						Expect(routes).To(BeNil(), "converting %s to %s", fromVersion, toVersion)
					} else {
						Expect(routes).NotTo(BeNil(), "converting %s to %s", fromVersion, toVersion)
					}
					Expect(routes).To(HaveLen(count))
				}
			}
		},
		Entry("absent routes", "", true, 0),
		Entry("explicitly empty routes", `,"routes":[]`, false, 0),
		Entry("populated routes", `,"routes":[{"dst":"0.0.0.0/0","gw":"10.44.0.1"}]`, false, 1),
	)
})
