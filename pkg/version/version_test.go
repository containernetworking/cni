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

package version_test

import (
	"github.com/containernetworking/cni/pkg/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Decode", func() {
	It("returns a PluginInfo that represents the given json bytes", func() {
		pluginInfo, err := version.Decode([]byte(`{
			"cniVersion": "some-library-version",
			"supportedVersions": [ "some-version", "some-other-version" ]
		}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(pluginInfo).NotTo(BeNil())
		Expect(pluginInfo.SupportedVersions()).To(Equal([]string{
			"some-version",
			"some-other-version",
		}))
	})

	Context("when the bytes cannot be decoded as json", func() {
		It("returns a meaningful error", func() {
			_, err := version.Decode([]byte(`{{{`))
			Expect(err).To(MatchError("decoding version info: invalid character '{' looking for beginning of object key string"))
		})
	})

	Context("when the json bytes are missing the required CNIVersion field", func() {
		It("returns a meaningful error", func() {
			_, err := version.Decode([]byte(`{ "supportedVersions": [ "foo" ] }`))
			Expect(err).To(MatchError("decoding version info: missing field cniVersion"))
		})
	})

	Context("when there are no supported versions", func() {
		Context("when the cniVersion is 0.2.0", func() {
			It("infers the supported versions are 0.1.0 and 0.2.0", func() {
				pluginInfo, err := version.Decode([]byte(`{ "cniVersion": "0.2.0" }`))
				Expect(err).NotTo(HaveOccurred())
				Expect(pluginInfo).NotTo(BeNil())
				Expect(pluginInfo.SupportedVersions()).To(Equal([]string{
					"0.1.0",
					"0.2.0",
				}))
			})
		})

		Context("when the cniVersion is >= 0.3.0", func() {
			It("returns a meaningful error", func() {
				_, err := version.Decode([]byte(`{ "cniVersion": "0.3.0" }`))
				Expect(err).To(MatchError("decoding version info: missing field supportedVersions"))
			})
		})
	})

})
