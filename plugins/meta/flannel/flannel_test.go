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
		subnetFile      string
		stateDir        string
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

	var writeSubnetEnv = func(contents string) string {
		file, err := ioutil.TempFile("", "subnet.env")
		Expect(err).NotTo(HaveOccurred())
		_, err = file.WriteString(contents)
		Expect(err).NotTo(HaveOccurred())
		return file.Name()
	}

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

		// flannel subnet.env
		subnetFile = writeSubnetEnv(flannelSubnetEnv)

		// flannel state dir
		stateDir, err = ioutil.TempDir("", "stateDir")
		Expect(err).NotTo(HaveOccurred())
		input = fmt.Sprintf(inputTemplate, subnetFile, stateDir)
	})

	AfterEach(func() {
		os.Remove(debugFileName)
		os.Remove(subnetFile)
		os.Remove(stateDir)
	})

	Describe("CNI lifecycle", func() {

		BeforeEach(func() {
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

	Describe("loadFlannelNetConf", func() {
		Context("when subnetFile and stateDir are specified", func() {
			It("loads flannel network config", func() {
				conf, err := loadFlannelNetConf([]byte(input))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(conf.Name).To(Equal("cni-flannel"))
				Expect(conf.Type).To(Equal("flannel"))
				Expect(conf.SubnetFile).To(Equal(subnetFile))
				Expect(conf.StateDir).To(Equal(stateDir))
			})
		})

		Context("when defaulting subnetFile and stateDir", func() {
			BeforeEach(func() {
				input = `{
"name": "cni-flannel",
"type": "flannel",
"delegate": ` +
					delegateInput +
					`}`
			})

			It("loads flannel network config with defaults", func() {
				conf, err := loadFlannelNetConf([]byte(input))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(conf.Name).To(Equal("cni-flannel"))
				Expect(conf.Type).To(Equal("flannel"))
				Expect(conf.SubnetFile).To(Equal(defaultSubnetFile))
				Expect(conf.StateDir).To(Equal(defaultStateDir))
			})
		})

		Describe("loadFlannelSubnetEnv", func() {
			Context("when flannel subnet env is valid", func() {
				It("loads flannel subnet config", func() {
					conf, err := loadFlannelSubnetEnv(subnetFile)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(conf.nw.String()).To(Equal("10.1.0.0/16"))
					Expect(conf.sn.String()).To(Equal("10.1.17.0/24"))
					var mtu uint = 1472
					Expect(*conf.mtu).To(Equal(mtu))
					Expect(*conf.ipmasq).To(BeTrue())
				})
			})

			Context("when flannel subnet env is invalid", func() {
				BeforeEach(func() {
					subnetFile = writeSubnetEnv("foo=bar")
				})
				It("returns an error", func() {
					_, err := loadFlannelSubnetEnv(subnetFile)
					Expect(err).To(MatchError(ContainSubstring("missing FLANNEL_NETWORK, FLANNEL_SUBNET, FLANNEL_MTU, FLANNEL_IPMASQ")))
				})
			})
		})
	})
})
