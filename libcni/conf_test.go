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

package libcni_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Loading configuration from disk", func() {
	var (
		configDir    string
		pluginConfig []byte
	)

	BeforeEach(func() {
		var err error
		configDir, err = ioutil.TempDir("", "plugin-conf")
		Expect(err).NotTo(HaveOccurred())

		pluginConfig = []byte(`{ "name": "some-plugin", "some-key": "some-value" }`)
		Expect(ioutil.WriteFile(filepath.Join(configDir, "50-whatever.conf"), pluginConfig, 0600)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(configDir)).To(Succeed())
	})

	Describe("LoadConf", func() {
		It("finds the network config file for the plugin of the given type", func() {
			netConfig, err := libcni.LoadConf(configDir, "some-plugin")
			Expect(err).NotTo(HaveOccurred())
			Expect(netConfig).To(Equal(&libcni.NetworkConfig{
				Network: &types.NetConf{Name: "some-plugin"},
				Bytes:   pluginConfig,
			}))
		})

		Context("when the config directory does not exist", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(configDir)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.LoadConf(configDir, "some-plugin")
				Expect(err).To(MatchError("no net configurations found"))
			})
		})

		Context("when there is no config for the desired plugin", func() {
			It("returns a useful error", func() {
				_, err := libcni.LoadConf(configDir, "some-other-plugin")
				Expect(err).To(MatchError(ContainSubstring(`no net configuration with name "some-other-plugin" in`)))
			})
		})

		Context("when a config file is malformed", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(configDir, "00-bad.conf"), []byte(`{`), 0600)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.LoadConf(configDir, "some-plugin")
				Expect(err).To(MatchError(`error parsing configuration: unexpected end of JSON input`))
			})
		})

		Context("when the config is in a nested subdir", func() {
			BeforeEach(func() {
				subdir := filepath.Join(configDir, "subdir1", "subdir2")
				Expect(os.MkdirAll(subdir, 0700)).To(Succeed())

				pluginConfig = []byte(`{ "name": "deep", "some-key": "some-value" }`)
				Expect(ioutil.WriteFile(filepath.Join(subdir, "90-deep.conf"), pluginConfig, 0600)).To(Succeed())
			})

			It("will not find the config", func() {
				_, err := libcni.LoadConf(configDir, "deep")
				Expect(err).To(MatchError(HavePrefix("no net configuration with name")))
			})
		})
	})

	Describe("ConfFromFile", func() {
		Context("when the file cannot be opened", func() {
			It("returns a useful error", func() {
				_, err := libcni.ConfFromFile("/tmp/nope/not-here")
				Expect(err).To(MatchError(HavePrefix(`error reading /tmp/nope/not-here: open /tmp/nope/not-here`)))
			})
		})
	})
})
