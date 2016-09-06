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

package invoke_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/cni/pkg/version/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetVersion, integration tests", func() {
	var (
		pluginDir  string
		pluginPath string
	)

	BeforeEach(func() {
		pluginDir, err := ioutil.TempDir("", "plugins")
		Expect(err).NotTo(HaveOccurred())
		pluginPath = filepath.Join(pluginDir, "test-plugin")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(pluginDir)).To(Succeed())
	})

	DescribeTable("correctly reporting plugin versions",
		func(gitRef string, pluginSource string, expectedVersions version.PluginInfo) {
			Expect(testhelpers.BuildAt([]byte(pluginSource), gitRef, pluginPath)).To(Succeed())
			versionInfo, err := invoke.GetVersionInfo(pluginPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(versionInfo.SupportedVersions()).To(ConsistOf(expectedVersions.SupportedVersions()))
		},
		Entry("old plugin, before VERSION was introduced", git_ref_v010, plugin_source_v010, version.PluginSupports("0.1.0")),
		Entry("when VERSION was introduced", git_ref_v020, plugin_source_v010, version.PluginSupports("0.1.0", "0.2.0")),
		Entry("when plugins report their own version support", git_ref_v030, plugin_source_v030, version.PluginSupports("0.3.0", "0.999.0")),
		Entry("HEAD", "HEAD", plugin_source_v030, version.PluginSupports("0.3.0", "0.999.0")),
	)
})

// a 0.3.0 plugin that can report its own versions
const plugin_source_v030 = `package main

import (
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"fmt"
)

func c(_ *skel.CmdArgs) error { fmt.Println("{}"); return nil }

func main() { skel.PluginMain(c, c, version.PluginSupports("0.3.0", "0.999.0")) }
`
const git_ref_v030 = "bf31ed15"

// a minimal 0.1.0 / 0.2.0 plugin
const plugin_source_v010 = `package main

import "github.com/containernetworking/cni/pkg/skel"
import "fmt"

func c(_ *skel.CmdArgs) error { fmt.Println("{}"); return nil }

func main() { skel.PluginMain(c, c) }
`

const git_ref_v010 = "2c482f4"
const git_ref_v020 = "349d66d"
