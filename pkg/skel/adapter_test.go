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
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("adapter", func() {

	q := make(chan string, 1)
	plugin := NewDirectPlugin(func(plugin *DirectPlugin) {
		plugin.AddFunc = func(args *Args) (types.Result, error) {
			q <- "add"
			return &current.Result{CNIVersion: version.Current()}, nil
		}
		plugin.DelFunc = func(args *Args) error {
			q <- "del"
			return nil
		}
		plugin.CheckFunc = func(args *Args) (types.Result, error) {
			q <- "check"
			return &current.Result{CNIVersion: version.Current()}, nil
		}
		plugin.VersionFunc = func() (version.PluginInfo, error) {
			q <- "version"
			return version.All, nil
		}
	})
	var adapter = NewDirectAdapter(func(manager *DirectPluginManager) {
		manager.Plugins["test"] = plugin
	})
	args := &Args{
		ContainerID: "container-id",
		Netns:       "net-ns",
		IfName:      "if-name",
		Args:        "args",
	}
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

	Describe("extension method Add", func() {
		It("dispatched right", func() {
			res, err := adapter.Add("test", args)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(&current.Result{CNIVersion: version.Current()}))
			Expect(<-q).To(Equal("add"))
		})
	})
	Describe("extension method Check", func() {
		It("dispatched right", func() {
			res, err := adapter.Check("test", args)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(&current.Result{CNIVersion: version.Current()}))
			Expect(<-q).To(Equal("check"))
		})
	})
	Describe("extension method Del", func() {
		It("dispatched right", func() {
			err := adapter.Del("test", args)
			Expect(err).NotTo(HaveOccurred())
			Expect(<-q).To(Equal("del"))
		})
	})
	Describe("extension method Version", func() {
		It("dispatched right", func() {
			res, err := adapter.Version("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(version.All))
			Expect(<-q).To(Equal("version"))

		})
	})

	Context("when plugin doesn't exists", func() {
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
