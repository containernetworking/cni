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

package libcni_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type pluginInfo struct {
	debugFilePath string
	debug         *noop_debug.Debug
	config        string
}

func addNameToConfig(name, config string) ([]byte, error) {
	obj := make(map[string]interface{})
	err := json.Unmarshal([]byte(config), &obj)
	if err != nil {
		return nil, fmt.Errorf("unmarshal existing network bytes: %s", err)
	}
	obj["name"] = name
	return json.Marshal(obj)
}

func newPluginInfo(configKey, configValue, prevResult string, injectDebugFilePath bool, result string) pluginInfo {
	debugFile, err := ioutil.TempFile("", "cni_debug")
	Expect(err).NotTo(HaveOccurred())
	Expect(debugFile.Close()).To(Succeed())
	debugFilePath := debugFile.Name()

	debug := &noop_debug.Debug{
		ReportResult: result,
	}
	Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

	config := fmt.Sprintf(`{"type": "noop", "%s": "%s", "cniVersion": "0.3.0"`, configKey, configValue)
	if prevResult != "" {
		config += fmt.Sprintf(`, "prevResult": %s`, prevResult)
	}
	if injectDebugFilePath {
		config += fmt.Sprintf(`, "debugFile": "%s"`, debugFilePath)
	}
	config += "}"

	return pluginInfo{
		debugFilePath: debugFilePath,
		debug:         debug,
		config:        config,
	}
}

var _ = Describe("Invoking plugins", func() {
	Describe("Invoking a single plugin", func() {
		var (
			debugFilePath string
			debug         *noop_debug.Debug
			cniBinPath    string
			pluginConfig  string
			cniConfig     libcni.CNIConfig
			netConfig     *libcni.NetworkConfig
			runtimeConfig *libcni.RuntimeConf

			expectedCmdArgs skel.CmdArgs
		)

		BeforeEach(func() {
			debugFile, err := ioutil.TempFile("", "cni_debug")
			Expect(err).NotTo(HaveOccurred())
			Expect(debugFile.Close()).To(Succeed())
			debugFilePath = debugFile.Name()

			debug = &noop_debug.Debug{
				ReportResult: `{ "ips": [{ "version": "4", "address": "10.1.2.3/24" }], "dns": {} }`,
			}
			Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

			cniBinPath = filepath.Dir(pluginPaths["noop"])
			pluginConfig = `{ "type": "noop", "some-key": "some-value", "cniVersion": "0.3.0" }`
			cniConfig = libcni.CNIConfig{Path: []string{cniBinPath}}
			netConfig = &libcni.NetworkConfig{
				Network: &types.NetConf{
					Type: "noop",
				},
				Bytes: []byte(pluginConfig),
			}
			runtimeConfig = &libcni.RuntimeConf{
				ContainerID: "some-container-id",
				NetNS:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        [][2]string{[2]string{"DEBUG", debugFilePath}},
			}

			expectedCmdArgs = skel.CmdArgs{
				ContainerID: "some-container-id",
				Netns:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        "DEBUG=" + debugFilePath,
				Path:        cniBinPath,
				StdinData:   []byte(pluginConfig),
			}
		})

		Describe("AddNetwork", func() {
			It("executes the plugin with command ADD", func() {
				r, err := cniConfig.AddNetwork(netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				result, err := current.GetResult(r)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal(&current.Result{
					IPs: []*current.IPConfig{
						{
							Version: "4",
							Address: net.IPNet{
								IP:   net.ParseIP("10.1.2.3"),
								Mask: net.IPv4Mask(255, 255, 255, 0),
							},
						},
					},
				}))

				debug, err := noop_debug.ReadDebug(debugFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.Command).To(Equal("ADD"))
				Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfig.Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					_, err := cniConfig.AddNetwork(netConfig, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					debug.ReportError = "plugin error: banana"
					Expect(debug.WriteDebug(debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					result, err := cniConfig.AddNetwork(netConfig, runtimeConfig)
					Expect(result).To(BeNil())
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})
		})

		Describe("DelNetwork", func() {
			It("executes the plugin with command DEL", func() {
				err := cniConfig.DelNetwork(netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				debug, err := noop_debug.ReadDebug(debugFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.Command).To(Equal("DEL"))
				Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfig.Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					err := cniConfig.DelNetwork(netConfig, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					debug.ReportError = "plugin error: banana"
					Expect(debug.WriteDebug(debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					err := cniConfig.DelNetwork(netConfig, runtimeConfig)
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})
		})

		Describe("GetVersionInfo", func() {
			It("executes the plugin with the command VERSION", func() {
				versionInfo, err := cniConfig.GetVersionInfo("noop")
				Expect(err).NotTo(HaveOccurred())

				Expect(versionInfo).NotTo(BeNil())
				Expect(versionInfo.SupportedVersions()).To(Equal([]string{
					"0.-42.0", "0.1.0", "0.2.0", "0.3.0",
				}))
			})

			Context("when finding the plugin fails", func() {
				It("returns the error", func() {
					_, err := cniConfig.GetVersionInfo("does-not-exist")
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})
		})
	})

	Describe("Invoking a plugin list", func() {
		var (
			plugins       []pluginInfo
			cniBinPath    string
			cniConfig     libcni.CNIConfig
			netConfigList *libcni.NetworkConfigList
			runtimeConfig *libcni.RuntimeConf

			expectedCmdArgs skel.CmdArgs
		)

		BeforeEach(func() {
			plugins = make([]pluginInfo, 3, 3)
			plugins[0] = newPluginInfo("some-key", "some-value", "", true, `{"dns":{},"ips":[{"version": "4", "address": "10.1.2.3/24"}]}`)
			plugins[1] = newPluginInfo("some-key", "some-other-value", `{"dns":{},"ips":[{"version": "4", "address": "10.1.2.3/24"}]}`, true, "PASSTHROUGH")
			plugins[2] = newPluginInfo("some-key", "yet-another-value", `{"dns":{},"ips":[{"version": "4", "address": "10.1.2.3/24"}]}`, true, "INJECT-DNS")

			configList := []byte(fmt.Sprintf(`{
  "name": "some-list",
  "cniVersion": "0.3.0",
  "plugins": [
    %s,
    %s,
    %s
  ]
}`, plugins[0].config, plugins[1].config, plugins[2].config))

			var err error
			netConfigList, err = libcni.ConfListFromBytes(configList)
			Expect(err).NotTo(HaveOccurred())

			cniBinPath = filepath.Dir(pluginPaths["noop"])
			cniConfig = libcni.CNIConfig{Path: []string{cniBinPath}}
			runtimeConfig = &libcni.RuntimeConf{
				ContainerID: "some-container-id",
				NetNS:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        [][2]string{{"FOO", "BAR"}},
			}

			expectedCmdArgs = skel.CmdArgs{
				ContainerID: "some-container-id",
				Netns:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        "FOO=BAR",
				Path:        cniBinPath,
			}
		})

		Describe("AddNetworkList", func() {
			It("executes all plugins with command ADD and returns an intermediate result", func() {
				r, err := cniConfig.AddNetworkList(netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				result, err := current.GetResult(r)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal(&current.Result{
					// IP4 added by first plugin
					IPs: []*current.IPConfig{
						{
							Version: "4",
							Address: net.IPNet{
								IP:   net.ParseIP("10.1.2.3"),
								Mask: net.IPv4Mask(255, 255, 255, 0),
							},
						},
					},
					// DNS injected by last plugin
					DNS: types.DNS{
						Nameservers: []string{"1.2.3.4"},
					},
				}))

				for i := 0; i < len(plugins); i++ {
					debug, err := noop_debug.ReadDebug(plugins[i].debugFilePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(debug.Command).To(Equal("ADD"))
					newConfig, err := addNameToConfig("some-list", plugins[i].config)
					Expect(err).NotTo(HaveOccurred())

					// Must explicitly match JSON due to dict element ordering
					debugJSON := debug.CmdArgs.StdinData
					debug.CmdArgs.StdinData = nil
					Expect(debugJSON).To(MatchJSON(newConfig))
					Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
				}
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfigList.Plugins[1].Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					_, err := cniConfig.AddNetworkList(netConfigList, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the second plugin errors", func() {
				BeforeEach(func() {
					plugins[1].debug.ReportError = "plugin error: banana"
					Expect(plugins[1].debug.WriteDebug(plugins[1].debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					result, err := cniConfig.AddNetworkList(netConfigList, runtimeConfig)
					Expect(result).To(BeNil())
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})
		})

		Describe("DelNetworkList", func() {
			It("executes all the plugins in reverse order with command DEL", func() {
				err := cniConfig.DelNetworkList(netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				for i := 0; i < len(plugins); i++ {
					debug, err := noop_debug.ReadDebug(plugins[i].debugFilePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(debug.Command).To(Equal("DEL"))
					newConfig, err := addNameToConfig("some-list", plugins[i].config)
					Expect(err).NotTo(HaveOccurred())

					// Must explicitly match JSON due to dict element ordering
					debugJSON := debug.CmdArgs.StdinData
					debug.CmdArgs.StdinData = nil
					Expect(debugJSON).To(MatchJSON(newConfig))
					Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
				}
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfigList.Plugins[1].Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					err := cniConfig.DelNetworkList(netConfigList, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					plugins[1].debug.ReportError = "plugin error: banana"
					Expect(plugins[1].debug.WriteDebug(plugins[1].debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					err := cniConfig.DelNetworkList(netConfigList, runtimeConfig)
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})
		})

	})
})
