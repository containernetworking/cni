// Copyright 2014-2016 CNI authors
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

package skel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("envMap", func() {
	It("works for common cases", func() {
		p := []string{"X=", "Y", "Z=foo", "", "=boo"}
		m := envMap(p)
		Expect(len(m)).To(Equal(len(p) - 1))
		Expect(m["X"]).To(Equal(""))
		Expect(m["Y"]).To(Equal(""))
		Expect(m["Z"]).To(Equal("foo"))
		Expect(m[""]).To(Equal("boo"))
	})
})

var _ = Describe("plugin exec", func() {

	q := make(chan string, 1)
	BeforeEach(func() {
		select {
		case <-q:
			Fail("unexpected event")
		default:
		}
	})
	AfterEach(func() {
		select {
		case <-q:
			Fail("unexpected event")
		default:
		}
	})

	stdin := []byte(fmt.Sprintf(
		`{ "name":"test", "cniVersion": "%s" }`, version.Current(),
	))

	var manager = NewDirectPluginManager(func(manager *DirectPluginManager) {
		manager.Plugins["/cni/plugins/test"] = NewDirectPlugin(func(plugin *DirectPlugin) {
			plugin.AddFunc = func(args *Args) (types.Result, error) {
				q <- CmdAdd
				return &current.Result{CNIVersion: version.Current()}, nil
			}
			plugin.DelFunc = func(args *Args) error {
				q <- CmdDel
				return nil
			}
			plugin.CheckFunc = func(args *Args) (types.Result, error) {
				q <- CmdCheck
				return &current.Result{CNIVersion: version.Current()}, nil
			}
			plugin.VersionFunc = func() (version.PluginInfo, error) {
				q <- CmdVersion
				return version.All, nil
			}
		})
	})

	var exec = NewPluginExec(manager)

	Context("when invoke unknown command", func() {
		It("returns error", func() {
			res, err := exec.ExecPlugin(
				context.TODO(),
				"/cni/plugins/test",
				stdin,
				(&invoke.Args{Command: "unknown"}).AsEnv(),
			)
			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())
		})
	})

	Context("when can't find plugin", func() {
		It("all commands returns error", func() {
			for _, c := range []string{CmdAdd, CmdVersion, CmdCheck, CmdDel} {
				res, err := exec.ExecPlugin(
					context.TODO(),
					"/cni/plugins/none",
					stdin,
					(&invoke.Args{Command: c}).AsEnv(),
				)
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			}
		})
	})

	Context("when can't marshal result", func() {
		var manager = NewDirectPluginManager(func(manager *DirectPluginManager) {
			manager.Plugins["/cni/plugins/test"] = NewDirectPlugin(func(plugin *DirectPlugin) {
				plugin.AddFunc = func(args *Args) (types.Result, error) {
					q <- CmdAdd
					return notMarshallableResult{
						Result: &current.Result{CNIVersion: version.Current()},
					}, nil
				}
				plugin.DelFunc = func(args *Args) error {
					q <- CmdDel
					return nil
				}
				plugin.CheckFunc = func(args *Args) (types.Result, error) {
					q <- CmdCheck
					return notMarshallableResult{
						Result: &current.Result{CNIVersion: version.Current()},
					}, nil
				}
				plugin.VersionFunc = func() (version.PluginInfo, error) {
					q <- CmdVersion
					return notMarshallableVersion{
						PluginInfo: version.All,
					}, nil
				}
			})
		})
		var exec = NewPluginExec(manager)
		It("Add, Check command returns error", func() {
			for _, c := range []string{CmdAdd, CmdCheck, CmdVersion} {
				res, err := exec.ExecPlugin(
					context.TODO(),
					"/cni/plugins/test",
					stdin,
					(&invoke.Args{Command: c}).AsEnv(),
				)
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
				select {
				case e := <-q:
					Expect(e).To(Equal(c))
				default:
					Fail("event expected")
				}
			}
		})
	})

})

type notMarshallableResult struct {
	types.Result
}

func (r notMarshallableResult) Print() error {
	return r.PrintTo(os.Stdout)
}

func (r notMarshallableResult) PrintTo(writer io.Writer) error {
	return errors.New("simulated error")
}

type notMarshallableVersion struct {
	version.PluginInfo
}

func (r notMarshallableVersion) Encode(io.Writer) error {
	return errors.New("simulated error")
}
