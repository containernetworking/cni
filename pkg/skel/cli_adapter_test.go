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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/version"
)

var _ = Describe("printResult", func() {
	Context("when args is nil", func() {
		It("writes result to stdout on success", func() {
			out, err := captureStdout(func() error {
				return printResult(func(args *Args) (types.Result, error) {
					return &current.Result{CNIVersion: version.Current()}, nil
				})(nil)
			})
			Expect(err).NotTo(HaveOccurred())

			res, err := current.NewResult(out)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(&current.Result{CNIVersion: version.Current()}))

		})
	})

	It("doesn't write anything to stdout on failure", func() {
		out, err := captureStdout(func() error {
			return printResult(func(args *Args) (types.Result, error) {
				return nil, errors.New("simulated error")
			})(nil)
		})
		Expect(err).To(HaveOccurred())
		Expect(len(out)).To(Equal(0))
	})

	It("writes result to stdout on success", func() {
		out, err := captureStdout(func() error {
			return printResult(func(args *Args) (types.Result, error) {
				return &current.Result{CNIVersion: version.Current()}, nil
			})(&CmdArgs{
				ContainerID: "container-id",
				Netns:       "net-ns",
				IfName:      "if-name",
				Args:        "args",
				Path:        "path",
			})
		})
		Expect(err).NotTo(HaveOccurred())

		res, err := current.NewResult(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(&current.Result{CNIVersion: version.Current()}))

	})
})

var _ = Describe("noResult", func() {
	Context("when args is nil", func() {
		It("passes nil too", func() {
			err := noResult(func(args *Args) error {
				Expect(args).To(BeNil())
				return nil
			})(nil)
			Expect(err).To(BeNil())
		})
	})

	It("returns same error", func() {
		err := noResult(func(args *Args) error {
			return errors.New("test")
		})(nil)
		Expect(err).To(Equal(errors.New("test")))
	})

	It("it passes correct args values", func() {
		err := noResult(func(args *Args) error {
			Expect(args.ContainerID).To(Equal("ContainerID"))
			Expect(args.Netns).To(Equal("Netns"))
			Expect(args.IfName).To(Equal("IfName"))
			Expect(args.Args).To(Equal("Args"))
			return nil
		})(&CmdArgs{
			ContainerID: "ContainerID",
			Netns:       "Netns",
			IfName:      "IfName",
			Args:        "Args",
			Path:        "Path",
		})
		Expect(err).To(BeNil())
	})
})

var _ = Describe("toCmdArgs", func() {
	Context("when args is nil", func() {
		It("returns nil", func() {
			res := toCmdArgs(nil)
			Expect(res).To(BeNil())
		})
	})

	It("returns cmd args with same values", func() {
		res := toCmdArgs(&Args{
			ContainerID: "ContainerID",
			Netns:       "Netns",
			IfName:      "IfName",
			Args:        "Args",
		})
		Expect(res.ContainerID).To(Equal("ContainerID"))
		Expect(res.Netns).To(Equal("Netns"))
		Expect(res.IfName).To(Equal("IfName"))
		Expect(res.Args).To(Equal("Args"))
		Expect(res.Path).To(Equal(""))
	})
})

var _ = Describe("fromCmdArgs", func() {
	Context("when args is nil", func() {
		It("returns nil", func() {
			res := fromCmdArgs(nil)
			Expect(res).To(BeNil())
		})
	})

	It("returns cmd args with same values", func() {
		res := fromCmdArgs(&CmdArgs{
			ContainerID: "ContainerID",
			Netns:       "Netns",
			IfName:      "IfName",
			Args:        "Args",
			Path:        "Path",
		})
		Expect(res.ContainerID).To(Equal("ContainerID"))
		Expect(res.Netns).To(Equal("Netns"))
		Expect(res.IfName).To(Equal("IfName"))
		Expect(res.Args).To(Equal("Args"))
	})
})

var _ = Describe("cli adapter", func() {

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

	args := &Args{
		ContainerID: "container-id",
		Netns:       "net-ns",
		IfName:      "if-name",
		Args:        "args",
		StdinData: []byte(fmt.Sprintf(
			`{ "name":"test", "cniVersion": "%s" }`, version.Current(),
		)),
	}

	Context("when plugin's calls succeeds", func() {

		var adapter = NewCliAdapter(func(m *CliPluginManager) {
			plugin := NewDirectPlugin(func(plugin *DirectPlugin) {
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
			var manager = NewDirectPluginManager(func(manager *DirectPluginManager) {
				manager.Plugins["/cni/plugins/test"] = plugin
			})
			m.Paths = []string{"/cni/plugins", "/ext/cni-plugins"}
			m.Exec = NewPluginExec(manager)
		})

		Describe("extension method Add", func() {
			It("dispatched right", func() {
				res, err := adapter.Add("test", args)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(&current.Result{CNIVersion: version.Current()}))
				Expect(<-q).To(Equal(CmdAdd))
			})
		})
		Describe("extension method Check", func() {
			It("dispatched right", func() {
				res, err := adapter.Check("test", args)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(&current.Result{CNIVersion: version.Current()}))
				Expect(<-q).To(Equal(CmdCheck))
			})
		})
		Describe("extension method Del", func() {
			It("dispatched right", func() {
				err := adapter.Del("test", args)
				Expect(err).NotTo(HaveOccurred())
				Expect(<-q).To(Equal(CmdDel))
			})
		})
		Describe("extension method Version", func() {
			It("dispatched right", func() {
				res, err := adapter.Version("test")
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(version.All))
				Expect(<-q).To(Equal(CmdVersion))

			})
		})
	})

	Context("when plugin's calls fails", func() {
		simulatedError := errors.New("simulated error")
		var adapter = NewCliAdapter(func(m *CliPluginManager) {
			plugin := NewDirectPlugin(func(plugin *DirectPlugin) {
				plugin.AddFunc = func(args *Args) (types.Result, error) {
					q <- CmdAdd
					return nil, simulatedError
				}
				plugin.DelFunc = func(args *Args) error {
					q <- CmdDel
					return simulatedError
				}
				plugin.CheckFunc = func(args *Args) (types.Result, error) {
					q <- CmdCheck
					return nil, simulatedError
				}
				plugin.VersionFunc = func() (version.PluginInfo, error) {
					q <- CmdVersion
					return nil, simulatedError
				}
			})
			var manager = NewDirectPluginManager(func(manager *DirectPluginManager) {
				manager.Plugins["test"] = plugin
			})
			m.Paths = []string{"/cni/plugins", "/ext/cni-plugins"}
			m.Exec = NewPluginExec(manager)
		})

		Describe("extension method Add", func() {
			It("propagates error", func() {
				res, err := adapter.Add("test", args)
				Expect(err).To(MatchError(simulatedError))
				Expect(res).To(BeNil())
				Expect(<-q).To(Equal(CmdAdd))
			})
		})
		Describe("extension method Check", func() {
			It("propagates error", func() {
				res, err := adapter.Check("test", args)
				Expect(err).To(MatchError(simulatedError))
				Expect(res).To(BeNil())
				Expect(<-q).To(Equal(CmdCheck))
			})
		})
		Describe("extension method Del", func() {
			It("propagates error", func() {
				err := adapter.Del("test", args)
				Expect(err).To(MatchError(simulatedError))
				Expect(<-q).To(Equal(CmdDel))
			})
		})
		Describe("extension method Version", func() {
			It("propagates error", func() {
				res, err := adapter.Version("test")
				Expect(err).To(MatchError(simulatedError))
				Expect(res).To(BeNil())
				Expect(<-q).To(Equal(CmdVersion))

			})
		})
	})

	Context("when plugin doesn't exists", func() {

		var adapter = NewCliAdapter(func(m *CliPluginManager) {
			m.Paths = []string{}
			m.Exec = NewPluginExec(
				NewDirectPluginManager(),
			)
		})

		Describe("extension method Add", func() {
			It("fails", func() {
				res, err := adapter.Add("foo", args)
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			})
		})
		Describe("extension method Check", func() {
			It("fails", func() {
				res, err := adapter.Check("foo", args)
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			})
		})
		Describe("extension method Del", func() {
			It("fails", func() {
				err := adapter.Del("foo", args)
				Expect(err).To(HaveOccurred())
			})
		})
		Describe("extension method Version", func() {
			It("fails", func() {
				res, err := adapter.Version("foo")
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			})
		})
	})
})

var _ = Describe("NewCliPluginManager", func() {
	Context("when Paths is nil", func() {
		paths := []string{
			"x/y/z", "a/b/c",
		}
		old := os.Getenv("CNI_PATH")
		os.Setenv("CNI_PATH", strings.Join(paths, string(os.PathListSeparator)))
		manager := NewCliPluginManager()
		It("set it from CNI_PATH", func() {
			Expect(manager.Paths).To(Equal(paths))
		})
		os.Setenv("CNI_PATH", old)

	})

	Context("when Exec is nil", func() {
		manager := NewCliPluginManager(func(m *CliPluginManager) {
			m.Paths = []string{}
		})
		It("set it to default", func() {
			Expect(manager.Paths).To(Equal([]string{}))
			Expect(manager.Exec).NotTo(BeNil())
		})
	})

})

func captureStdout(action func() error) ([]byte, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	defer w.Close()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()
	if err = action(); err != nil {
		return nil, err
	}
	os.Stdout = oldStdout
	if err = w.Close(); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}
