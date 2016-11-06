// Copyright 2015 CNI authors
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
package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestFlannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flannel Suite")
}

const flannelPackage = "github.com/containernetworking/cni/plugins/meta/flannel"
const noopPackage = "github.com/containernetworking/cni/plugins/test/noop"

var paths testPaths

type testPaths struct {
	PathToPlugin string
	CNIPath      string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	noopBin, err := gexec.Build(noopPackage)
	Expect(err).NotTo(HaveOccurred())
	noopDir, _ := filepath.Split(noopBin)

	pathToPlugin, err := gexec.Build(flannelPackage)
	Expect(err).NotTo(HaveOccurred())
	flannelDir, _ := filepath.Split(pathToPlugin)

	paths := testPaths{
		PathToPlugin: pathToPlugin,
		CNIPath:      fmt.Sprintf("%s:%s", flannelDir, noopDir),
	}

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())
	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
