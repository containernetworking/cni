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

package main_test

import (
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

var _ = Describe("No-op plugin", func() {
	var (
		cmd             *exec.Cmd
		debugFileName   string
		debug           *noop_debug.Debug
		expectedCmdArgs skel.CmdArgs
	)

	const reportResult = `{ "ip4": { "ip": "10.1.2.3/24" }, "dns": {} }`

	BeforeEach(func() {
		debug = &noop_debug.Debug{ReportResult: reportResult}

		debugFile, err := ioutil.TempFile("", "cni_debug")
		Expect(err).NotTo(HaveOccurred())
		Expect(debugFile.Close()).To(Succeed())
		debugFileName = debugFile.Name()

		Expect(debug.WriteDebug(debugFileName)).To(Succeed())

		cmd = exec.Command(pathToPlugin)
		cmd.Env = []string{
			"CNI_COMMAND=ADD",
			"CNI_CONTAINERID=some-container-id",
			"CNI_ARGS=DEBUG=" + debugFileName,
			"CNI_NETNS=/some/netns/path",
			"CNI_IFNAME=some-eth0",
			"CNI_PATH=/some/bin/path",
		}
		cmd.Stdin = strings.NewReader(`{"some":"stdin-json"}`)
		expectedCmdArgs = skel.CmdArgs{
			ContainerID: "some-container-id",
			Netns:       "/some/netns/path",
			IfName:      "some-eth0",
			Args:        "DEBUG=" + debugFileName,
			Path:        "/some/bin/path",
			StdinData:   []byte(`{"some":"stdin-json"}`),
		}
	})

	AfterEach(func() {
		os.Remove(debugFileName)
	})

	It("responds to ADD using the ReportResult debug field", func() {
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out.Contents()).To(MatchJSON(reportResult))
	})

	It("records all the args provided by skel.PluginMain", func() {
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		debug, err := noop_debug.ReadDebug(debugFileName)
		Expect(err).NotTo(HaveOccurred())
		Expect(debug.Command).To(Equal("ADD"))
		Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
	})

	Context("when the ReportError debug field is set", func() {
		BeforeEach(func() {
			debug.ReportError = "banana"
			Expect(debug.WriteDebug(debugFileName)).To(Succeed())
		})

		It("returns an error to skel.PluginMain, causing the process to exit code 1", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Out.Contents()).To(MatchJSON(`{ "code": 100, "msg": "banana" }`))
		})
	})

	Context("when the CNI_COMMAND is DEL", func() {
		BeforeEach(func() {
			cmd.Env[0] = "CNI_COMMAND=DEL"
			debug.ReportResult = `{ "some": "delete-data" }`
			Expect(debug.WriteDebug(debugFileName)).To(Succeed())
		})

		It("still does all the debug behavior", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(MatchJSON(`{
				"some": "delete-data"
      }`))
			debug, err := noop_debug.ReadDebug(debugFileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(debug.Command).To(Equal("DEL"))
			Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
		})

	})
})
