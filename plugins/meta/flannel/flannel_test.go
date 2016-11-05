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
package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"

	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flannel", func() {
	var (
		cmd             *exec.Cmd
		debugFileName   string
		input           string
		debug           *noop_debug.Debug
		expectedCmdArgs skel.CmdArgs
	)

	const delegateInput = `
{
		"type": "noop",
		"some": "other data"
}
`

	const inputTemplate = `
{
  "name": "cni-flannel",
  "type": "flannel",
	"subnetFile": "%s",
	"stateDir": "%s",
	"delegate": ` +
		delegateInput +
		`}`

	const flannelSubnetEnv = `
FLANNEL_NETWORK=10.1.0.0/16
FLANNEL_SUBNET=10.1.17.1/24
FLANNEL_MTU=1472
FLANNEL_IPMASQ=true
`

	var cniCommand = func(command, input string) *exec.Cmd {
		toReturn := exec.Command(paths.PathToPlugin)
		toReturn.Env = []string{
			"CNI_COMMAND=" + command,
			"CNI_CONTAINERID=some-container-id",
			"CNI_NETNS=/some/netns/path",
			"CNI_IFNAME=some-eth0",
			"CNI_PATH=" + paths.CNIPath,
			"CNI_ARGS=DEBUG=" + debugFileName,
		}
		toReturn.Stdin = strings.NewReader(input)
		return toReturn
	}

	BeforeEach(func() {
		debugFile, err := ioutil.TempFile("", "cni_debug")
		Expect(err).NotTo(HaveOccurred())
		Expect(debugFile.Close()).To(Succeed())
		debugFileName = debugFile.Name()

		debug = &noop_debug.Debug{
			ReportResult:         `{ "ip4": { "ip": "1.2.3.4/32" } }`,
			ReportVersionSupport: []string{"0.1.0", "0.2.0", "0.3.0"},
		}
		Expect(debug.WriteDebug(debugFileName)).To(Succeed())
	})

	AfterEach(func() {
		os.Remove(debugFileName)
	})

	Describe("CNI lifecycle", func() {
		Context("when subnetFile and stateDir are specified", func() {
			var (
				subnetFile string
				stateDir   string
			)

			BeforeEach(func() {
				var err error
				file, err := ioutil.TempFile("", "subnet.env")
				Expect(err).NotTo(HaveOccurred())
				_, err = file.WriteString(flannelSubnetEnv)
				Expect(err).NotTo(HaveOccurred())
				subnetFile = file.Name()

				stateDir, err = ioutil.TempDir("", "stateDir")
				Expect(err).NotTo(HaveOccurred())
				input = fmt.Sprintf(inputTemplate, subnetFile, stateDir)

				expectedCmdArgs = skel.CmdArgs{
					ContainerID: "some-container-id",
					Netns:       "/some/netns/path",
					IfName:      "some-eth0",
					Args:        "DEBUG=" + debugFileName,
					Path:        "/some/bin/path",
					StdinData:   []byte(input),
				}
				cmd = cniCommand("ADD", input)
			})

			AfterEach(func() {
				os.Remove(subnetFile)
				os.Remove(stateDir)
			})

			It("uses stateDir for storing network configuration", func() {
				By("calling ADD")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out.Contents()).To(MatchJSON(`{ "ip4": { "ip": "1.2.3.4/32" }, "dns":{} }`))

				By("check that plugin writes to net config to stateDir")
				path := fmt.Sprintf("%s/%s", stateDir, "some-container-id")
				Expect(path).Should(BeAnExistingFile())

				netConfBytes, err := ioutil.ReadFile(path)
				Expect(err).NotTo(HaveOccurred())
				expected := `{
   "name" : "cni-flannel",
   "type" : "noop",
   "ipam" : {
      "type" : "host-local",
      "subnet" : "10.1.17.0/24",
      "routes" : [
         {
            "dst" : "10.1.0.0/16"
         }
      ]
   },
   "mtu" : 1472,
   "ipMasq" : false,
   "some" : "other data"
}
`
				Expect(netConfBytes).Should(MatchJSON(expected))

				By("calling DEL")
				cmd = cniCommand("DEL", input)
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("check that plugin removes net config from state dir")
				Expect(path).ShouldNot(BeAnExistingFile())
			})
		})
	})
})
