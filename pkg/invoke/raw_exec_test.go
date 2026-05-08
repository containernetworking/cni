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
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/invoke"
	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"
)

var _ = Describe("RawExec", func() {
	var (
		debugFileName string
		debug         *noop_debug.Debug
		environ       []string
		stdin         []byte
		execer        *invoke.RawExec
		ctx           context.Context
	)

	const reportResult = `{ "some": "result" }`

	BeforeEach(func() {
		debugFile, err := os.CreateTemp("", "cni_debug")
		Expect(err).NotTo(HaveOccurred())
		Expect(debugFile.Close()).To(Succeed())
		debugFileName = debugFile.Name()

		debug = &noop_debug.Debug{
			ReportResult: reportResult,
			ReportStderr: "some stderr message",
		}
		Expect(debug.WriteDebug(debugFileName)).To(Succeed())

		environ = []string{
			"CNI_COMMAND=ADD",
			"CNI_CONTAINERID=some-container-id",
			"CNI_ARGS=DEBUG=" + debugFileName,
			"CNI_NETNS=/some/netns/path",
			"CNI_PATH=/some/bin/path",
			"CNI_IFNAME=some-eth0",
		}
		stdin = []byte(`{"name": "raw-exec-test", "some":"stdin-json", "cniVersion": "0.3.1"}`)
		execer = &invoke.RawExec{}
		ctx = context.TODO()
	})

	AfterEach(func() {
		Expect(os.Remove(debugFileName)).To(Succeed())
	})

	It("runs the plugin with the given stdin and environment", func() {
		_, err := execer.ExecPlugin(ctx, pathToPlugin, stdin, environ)
		Expect(err).NotTo(HaveOccurred())

		debug, err := noop_debug.ReadDebug(debugFileName)
		Expect(err).NotTo(HaveOccurred())
		Expect(debug.Command).To(Equal("ADD"))
		Expect(debug.CmdArgs.StdinData).To(Equal(stdin))
		Expect(debug.CmdArgs.Netns).To(Equal("/some/netns/path"))
	})

	It("returns the resulting stdout as bytes", func() {
		resultBytes, err := execer.ExecPlugin(ctx, pathToPlugin, stdin, environ)
		Expect(err).NotTo(HaveOccurred())

		Expect(resultBytes).To(BeEquivalentTo(reportResult))
	})

	Context("when the Stderr writer is set", func() {
		var stderrBuffer *bytes.Buffer

		BeforeEach(func() {
			stderrBuffer = &bytes.Buffer{}
			execer.Stderr = stderrBuffer
		})

		It("forwards any stderr bytes to the Stderr writer", func() {
			_, err := execer.ExecPlugin(ctx, pathToPlugin, stdin, environ)
			Expect(err).NotTo(HaveOccurred())

			Expect(stderrBuffer.String()).To(Equal("some stderr message"))
		})
	})

	Context("when the plugin errors", func() {
		BeforeEach(func() {
			debug.ReportResult = ""
		})

		Context("and writes valid error JSON to stdout", func() {
			It("wraps and returns the error", func() {
				debug.ReportError = "banana"
				Expect(debug.WriteDebug(debugFileName)).To(Succeed())
				_, err := execer.ExecPlugin(ctx, pathToPlugin, stdin, environ)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("banana"))
			})
		})

		Context("and writes to stderr", func() {
			It("returns an error message with stderr output", func() {
				debug.ExitWithCode = 1
				Expect(debug.WriteDebug(debugFileName)).To(Succeed())
				_, err := execer.ExecPlugin(ctx, pathToPlugin, stdin, environ)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(`netplugin failed: "some stderr message": exit status 1`))
			})
		})
	})

	Context("when the plugin errors with no output on stdout or stderr", func() {
		It("returns the exec error message", func() {
			debug.ExitWithCode = 1
			debug.ReportResult = ""
			debug.ReportStderr = ""
			Expect(debug.WriteDebug(debugFileName)).To(Succeed())
			_, err := execer.ExecPlugin(ctx, pathToPlugin, stdin, environ)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("netplugin failed with no error message: exit status 1"))
		})
	})

	Context("when the system is unable to execute the plugin", func() {
		It("returns the error", func() {
			_, err := execer.ExecPlugin(ctx, "/tmp/some/invalid/plugin/path", stdin, environ)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("/tmp/some/invalid/plugin/path")))
		})
	})

	Context("when the plugin binary is temporarily busy (ETXTBSY)", func() {
		var (
			scriptPath string
			scriptFile *os.File
		)

		BeforeEach(func() {
			if runtime.GOOS != "linux" {
				Skip("ETXTBSY is Linux-specific")
			}

			// create a trivial executable script simulating a CNI plugin
			var err error
			scriptFile, err = os.CreateTemp("", "cni_etxtbsy_test_*.sh")
			Expect(err).NotTo(HaveOccurred())
			scriptPath = scriptFile.Name()

			_, err = scriptFile.WriteString("#!/bin/sh\necho 'hello from retry test'\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(scriptFile.Chmod(0755)).To(Succeed())
			// keep scriptFile open for writing to trigger ETXTBSY
		})

		AfterEach(func() {
			scriptFile.Close()
			os.Remove(scriptPath)
		})

		It("retries and succeeds after the file is no longer busy", func() {
			// first, verify that exec actually fails with ETXTBSY while
			// the file is still open for writing
			cmd := exec.Command(scriptPath)
			err := cmd.Run()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("text file busy"))

			// close the file after 500ms so the retry (which sleeps 1s)
			// finds the file available on its second attempt
			go func() {
				time.Sleep(500 * time.Millisecond)
				scriptFile.Close()
			}()

			start := time.Now()
			resultBytes, err := execer.ExecPlugin(ctx, scriptPath, []byte{}, []string{})
			elapsed := time.Since(start)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(resultBytes)).To(ContainSubstring("hello from retry test"))

			// the retry sleeps 1s, so elapsed must be >= 1s, proving the
			// first attempt failed and the retry loop was exercised
			Expect(elapsed).To(BeNumerically(">=", time.Second))
		})

		It("does not retry when the file is not busy", func() {
			// close the file so there is no ETXTBSY
			scriptFile.Close()

			start := time.Now()
			resultBytes, err := execer.ExecPlugin(ctx, scriptPath, []byte{}, []string{})
			elapsed := time.Since(start)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(resultBytes)).To(ContainSubstring("hello from retry test"))

			// should complete well under the 1s retry sleep
			Expect(elapsed).To(BeNumerically("<", 500*time.Millisecond))
		})
	})
})
