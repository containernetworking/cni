// Copyright 2017 CNI authors
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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/invoke"
)

var _ = Describe("CNIArgs AsEnv", func() {
	Describe("Args AsEnv", func() {
		BeforeEach(func() {
			os.Setenv("CNI_COMMAND", "DEL")
			os.Setenv("CNI_IFNAME", "eth0")
			os.Setenv("CNI_CONTAINERID", "id")
			os.Setenv("CNI_ARGS", "args")
			os.Setenv("CNI_NETNS", "testns")
			os.Setenv("CNI_PATH", "testpath")
		})

		It("places the CNI environment variables in the end to be prepended", func() {
			args := invoke.Args{
				Command:     "ADD",
				ContainerID: "some-container-id",
				NetNS:       "/some/netns/path",
				PluginArgs: [][2]string{
					{"KEY1", "VALUE1"},
					{"KEY2", "VALUE2"},
				},
				IfName: "eth7",
				Path:   "/some/cni/path",
			}

			latentEnvs := os.Environ()
			numLatentEnvs := len(latentEnvs)

			cniEnvs := args.AsEnv()
			Expect(cniEnvs).To(HaveLen(numLatentEnvs))

			Expect(inStringSlice("CNI_COMMAND=ADD", cniEnvs)).To(BeTrue())
			Expect(inStringSlice("CNI_IFNAME=eth7", cniEnvs)).To(BeTrue())
			Expect(inStringSlice("CNI_CONTAINERID=some-container-id", cniEnvs)).To(BeTrue())
			Expect(inStringSlice("CNI_NETNS=/some/netns/path", cniEnvs)).To(BeTrue())
			Expect(inStringSlice("CNI_ARGS=KEY1=VALUE1;KEY2=VALUE2", cniEnvs)).To(BeTrue())
			Expect(inStringSlice("CNI_PATH=/some/cni/path", cniEnvs)).To(BeTrue())

			Expect(inStringSlice("CNI_COMMAND=DEL", cniEnvs)).To(BeFalse())
			Expect(inStringSlice("CNI_IFNAME=eth0", cniEnvs)).To(BeFalse())
			Expect(inStringSlice("CNI_CONTAINERID=id", cniEnvs)).To(BeFalse())
			Expect(inStringSlice("CNI_NETNS=testns", cniEnvs)).To(BeFalse())
			Expect(inStringSlice("CNI_ARGS=args", cniEnvs)).To(BeFalse())
			Expect(inStringSlice("CNI_PATH=testpath", cniEnvs)).To(BeFalse())
		})

		AfterEach(func() {
			os.Unsetenv("CNI_COMMAND")
			os.Unsetenv("CNI_IFNAME")
			os.Unsetenv("CNI_CONTAINERID")
			os.Unsetenv("CNI_ARGS")
			os.Unsetenv("CNI_NETNS")
			os.Unsetenv("CNI_PATH")
		})
	})

	Describe("DelegateArgs AsEnv", func() {
		BeforeEach(func() {
			os.Unsetenv("CNI_COMMAND")
		})

		It("override CNI_COMMAND if it already exists in environment variables", func() {
			os.Setenv("CNI_COMMAND", "DEL")

			delegateArgs := invoke.DelegateArgs{
				Command: "ADD",
			}

			latentEnvs := os.Environ()
			numLatentEnvs := len(latentEnvs)

			cniEnvs := delegateArgs.AsEnv()
			Expect(cniEnvs).To(HaveLen(numLatentEnvs))

			Expect(inStringSlice("CNI_COMMAND=ADD", cniEnvs)).To(BeTrue())
			Expect(inStringSlice("CNI_COMMAND=DEL", cniEnvs)).To(BeFalse())
		})

		It("append CNI_COMMAND if it does not exist in environment variables", func() {
			delegateArgs := invoke.DelegateArgs{
				Command: "ADD",
			}

			latentEnvs := os.Environ()
			numLatentEnvs := len(latentEnvs)

			cniEnvs := delegateArgs.AsEnv()
			Expect(cniEnvs).To(HaveLen(numLatentEnvs + 1))

			Expect(inStringSlice("CNI_COMMAND=ADD", cniEnvs)).To(BeTrue())
		})

		AfterEach(func() {
			os.Unsetenv("CNI_COMMAND")
		})
	})

	Describe("inherited AsEnv", func() {
		It("return nil string slice if we call AsEnv of inherited", func() {
			inheritedArgs := invoke.ArgsFromEnv()

			var nilSlice []string = nil
			Expect(inheritedArgs.AsEnv()).To(Equal(nilSlice))
		})
	})
})

func inStringSlice(in string, slice []string) bool {
	for _, s := range slice {
		if in == s {
			return true
		}
	}
	return false
}
