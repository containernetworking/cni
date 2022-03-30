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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type pluginInfo struct {
	debugFilePath string
	debug         *noop_debug.Debug
	config        string
	stdinData     []byte
}

type portMapping struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

func stringInList(s string, list []string) bool {
	for _, item := range list {
		if s == item {
			return true
		}
	}
	return false
}

func newPluginInfo(cniVersion, configValue, prevResult string, injectDebugFilePath bool, result string, runtimeConfig map[string]interface{}, capabilities []string) pluginInfo {
	debugFile, err := ioutil.TempFile("", "cni_debug")
	Expect(err).NotTo(HaveOccurred())
	Expect(debugFile.Close()).To(Succeed())
	debugFilePath := debugFile.Name()

	debug := &noop_debug.Debug{
		ReportResult: result,
	}
	Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

	// config is what would be in the plugin's on-disk configuration
	// without runtime injected keys
	config := fmt.Sprintf(`{"type": "noop", "some-key": "%s"`, configValue)
	if prevResult != "" {
		config += fmt.Sprintf(`, "prevResult": %s`, prevResult)
	}
	if injectDebugFilePath {
		config += fmt.Sprintf(`, "debugFile": %q`, debugFilePath)
	}
	if len(capabilities) > 0 {
		config += `, "capabilities": {`
		for i, c := range capabilities {
			if i > 0 {
				config += ", "
			}
			config += fmt.Sprintf(`"%s": true`, c)
		}
		config += "}"
	}
	config += "}"

	// stdinData is what the runtime should pass to the plugin's stdin,
	// including injected keys like 'name', 'cniVersion', and 'runtimeConfig'
	newConfig := make(map[string]interface{})
	err = json.Unmarshal([]byte(config), &newConfig)
	Expect(err).NotTo(HaveOccurred())
	newConfig["name"] = "some-list"
	newConfig["cniVersion"] = cniVersion

	// Only include standard runtime config and capability args that this plugin advertises
	newRuntimeConfig := make(map[string]interface{})
	for key, value := range runtimeConfig {
		if stringInList(key, capabilities) {
			newRuntimeConfig[key] = value
		}
	}
	if len(newRuntimeConfig) > 0 {
		newConfig["runtimeConfig"] = newRuntimeConfig
	}

	stdinData, err := json.Marshal(newConfig)
	Expect(err).NotTo(HaveOccurred())

	return pluginInfo{
		debugFilePath: debugFilePath,
		debug:         debug,
		config:        config,
		stdinData:     stdinData,
	}
}

func makePluginList(cniVersion, ipResult string, runtimeConfig map[string]interface{}) (*libcni.NetworkConfigList, []pluginInfo) {
	plugins := make([]pluginInfo, 3)
	plugins[0] = newPluginInfo(cniVersion, "some-value", "", true, ipResult, runtimeConfig, []string{"portMappings", "otherCapability"})
	plugins[1] = newPluginInfo(cniVersion, "some-other-value", ipResult, true, "PASSTHROUGH", runtimeConfig, []string{"otherCapability"})
	plugins[2] = newPluginInfo(cniVersion, "yet-another-value", ipResult, true, "INJECT-DNS", runtimeConfig, []string{})

	configList := []byte(fmt.Sprintf(`{
"name": "some-list",
"cniVersion": "%s",
"plugins": [
%s,
%s,
%s
]
}`, cniVersion, plugins[0].config, plugins[1].config, plugins[2].config))

	netConfigList, err := libcni.ConfListFromBytes(configList)
	Expect(err).NotTo(HaveOccurred())
	return netConfigList, plugins
}

func resultCacheFilePath(cacheDirPath, netName string, rt *libcni.RuntimeConf) string {
	fName := fmt.Sprintf("%s-%s-%s", netName, rt.ContainerID, rt.IfName)
	return filepath.Join(cacheDirPath, "results", fName)
}

var _ = Describe("Invoking plugins", func() {
	var cacheDirPath string

	BeforeEach(func() {
		var err error
		cacheDirPath, err = ioutil.TempDir("", "cni_cachedir")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(cacheDirPath)).To(Succeed())
	})

	Describe("Capabilities", func() {
		var (
			debugFilePath string
			debug         *noop_debug.Debug
			pluginConfig  []byte
			cniConfig     *libcni.CNIConfig
			runtimeConfig *libcni.RuntimeConf
			netConfig     *libcni.NetworkConfig
			ctx           context.Context
		)

		BeforeEach(func() {
			debugFile, err := ioutil.TempFile("", "cni_debug")
			Expect(err).NotTo(HaveOccurred())
			Expect(debugFile.Close()).To(Succeed())
			debugFilePath = debugFile.Name()

			debug = &noop_debug.Debug{
				ReportResult: `{
					"cniVersion": "1.0.0",
					"ips": [{"address": "10.1.2.3/24"}],
					"dns": {}
				}`,
			}
			Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

			pluginConfig = []byte(fmt.Sprintf(`{
				"type": "noop",
				"name": "apitest",
				"cniVersion": "%s",
				"capabilities": {
					"portMappings": true,
					"somethingElse": true,
					"noCapability": false
				}
			}`, current.ImplementedSpecVersion))
			netConfig, err = libcni.ConfFromBytes(pluginConfig)
			Expect(err).NotTo(HaveOccurred())

			cniConfig = libcni.NewCNIConfigWithCacheDir([]string{filepath.Dir(pluginPaths["noop"])}, cacheDirPath, nil)

			runtimeConfig = &libcni.RuntimeConf{
				ContainerID: "some-container-id",
				NetNS:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        [][2]string{{"DEBUG", debugFilePath}},
				CapabilityArgs: map[string]interface{}{
					"portMappings": []portMapping{
						{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
					},
					"somethingElse": []string{"foobar", "baz"},
					"noCapability":  true,
					"notAdded":      []bool{true, false},
				},
			}
			ctx = context.TODO()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(debugFilePath)).To(Succeed())
		})

		It("adds correct runtime config for capabilities to stdin", func() {
			_, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
			Expect(err).NotTo(HaveOccurred())

			debug, err = noop_debug.ReadDebug(debugFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(debug.Command).To(Equal("ADD"))

			conf := make(map[string]interface{})
			err = json.Unmarshal(debug.CmdArgs.StdinData, &conf)
			Expect(err).NotTo(HaveOccurred())

			// We expect runtimeConfig keys only for portMappings and somethingElse
			rawRc := conf["runtimeConfig"]
			rc, ok := rawRc.(map[string]interface{})
			Expect(ok).To(Equal(true))
			expectedKeys := []string{"portMappings", "somethingElse"}
			Expect(len(rc)).To(Equal(len(expectedKeys)))
			for _, key := range expectedKeys {
				_, ok := rc[key]
				Expect(ok).To(Equal(true))
			}
		})

		It("adds no runtimeConfig when the plugin advertises no used capabilities", func() {
			// Replace CapabilityArgs with ones we know the plugin
			// doesn't support
			runtimeConfig.CapabilityArgs = map[string]interface{}{
				"portMappings22": []portMapping{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
				},
				"somethingElse22": []string{"foobar", "baz"},
			}

			_, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
			Expect(err).NotTo(HaveOccurred())

			debug, err = noop_debug.ReadDebug(debugFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(debug.Command).To(Equal("ADD"))

			conf := make(map[string]interface{})
			err = json.Unmarshal(debug.CmdArgs.StdinData, &conf)
			Expect(err).NotTo(HaveOccurred())

			// No intersection of plugin capabilities and CapabilityArgs,
			// so plugin should not receive a "runtimeConfig" key
			_, ok := conf["runtimeConfig"]
			Expect(ok).Should(BeFalse())
		})

		It("outputs correct capabilities for validate", func() {
			caps, err := cniConfig.ValidateNetwork(ctx, netConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(caps).To(ConsistOf("portMappings", "somethingElse"))
		})

	})

	Describe("Invoking a single plugin", func() {
		var (
			debugFilePath string
			debug         *noop_debug.Debug
			pluginConfig  string
			cniConfig     *libcni.CNIConfig
			netConfig     *libcni.NetworkConfig
			runtimeConfig *libcni.RuntimeConf
			ctx           context.Context

			expectedCmdArgs skel.CmdArgs
		)

		BeforeEach(func() {
			debugFile, err := ioutil.TempFile("", "cni_debug")
			Expect(err).NotTo(HaveOccurred())
			Expect(debugFile.Close()).To(Succeed())
			debugFilePath = debugFile.Name()

			debug = &noop_debug.Debug{
				ReportResult: `{
					"cniVersion": "1.0.0",
					"ips": [{"address": "10.1.2.3/24"}],
					"dns": {}
				}`,
			}
			Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

			portMappings := []portMapping{
				{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			}

			cniBinPath := filepath.Dir(pluginPaths["noop"])
			pluginConfig = fmt.Sprintf(`{
				"type": "noop",
				"name": "apitest",
				"some-key": "some-value",
				"cniVersion": "%s",
				"capabilities": { "portMappings": true }
			}`, current.ImplementedSpecVersion)
			cniConfig = libcni.NewCNIConfigWithCacheDir([]string{cniBinPath}, cacheDirPath, nil)
			netConfig, err = libcni.ConfFromBytes([]byte(pluginConfig))
			Expect(err).NotTo(HaveOccurred())
			runtimeConfig = &libcni.RuntimeConf{
				ContainerID: "some-container-id",
				NetNS:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        [][2]string{{"DEBUG", debugFilePath}},
				CapabilityArgs: map[string]interface{}{
					"portMappings": portMappings,
				},
			}

			// inject runtime args into the expected plugin config
			conf := make(map[string]interface{})
			err = json.Unmarshal([]byte(pluginConfig), &conf)
			Expect(err).NotTo(HaveOccurred())
			conf["runtimeConfig"] = map[string]interface{}{
				"portMappings": portMappings,
			}
			newBytes, err := json.Marshal(conf)
			Expect(err).NotTo(HaveOccurred())

			expectedCmdArgs = skel.CmdArgs{
				ContainerID: "some-container-id",
				Netns:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        "DEBUG=" + debugFilePath,
				Path:        cniBinPath,
				StdinData:   newBytes,
			}
			ctx = context.TODO()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(debugFilePath)).To(Succeed())
		})

		Describe("AddNetwork", func() {
			It("executes the plugin with command ADD", func() {
				r, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				result, err := current.GetResult(r)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal(&current.Result{
					CNIVersion: current.ImplementedSpecVersion,
					IPs: []*current.IPConfig{
						{
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
				Expect(string(debug.CmdArgs.StdinData)).To(ContainSubstring("\"portMappings\":"))

				// Ensure the cached config matches the sent one
				cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(reflect.DeepEqual(cachedConfig, []byte(pluginConfig))).To(BeTrue())
				Expect(reflect.DeepEqual(newRt.Args, runtimeConfig.Args)).To(BeTrue())
				// CapabilityArgs are freeform, so we have to match their JSON not
				// the Go structs (which could be unmarshalled differently than the
				// struct that was marshalled).
				expectedCABytes, err := json.Marshal(runtimeConfig.CapabilityArgs)
				Expect(err).NotTo(HaveOccurred())
				foundCABytes, err := json.Marshal(newRt.CapabilityArgs)
				Expect(err).NotTo(HaveOccurred())
				Expect(foundCABytes).To(MatchJSON(expectedCABytes))

				// Ensure the cached result matches the returned one
				cachedResult, err := cniConfig.GetNetworkCachedResult(netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				result2, err := current.GetResult(cachedResult)
				Expect(err).NotTo(HaveOccurred())
				cachedJson, err := json.Marshal(result2)
				Expect(err).NotTo(HaveOccurred())

				returnedJson, err := json.Marshal(result)
				Expect(err).NotTo(HaveOccurred())
				Expect(cachedJson).To(MatchJSON(returnedJson))
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfig.Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					_, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					debug.ReportError = "plugin error: banana"
					Expect(debug.WriteDebug(debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					result, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
					Expect(result).To(BeNil())
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})

			Context("when the cache directory cannot be accessed", func() {
				It("returns an error", func() {
					// Make the results directory inaccessible by making it a
					// file instead of a directory
					tmpPath := filepath.Join(cacheDirPath, "results")
					err := ioutil.WriteFile(tmpPath, []byte("afdsasdfasdf"), 0600)
					Expect(err).NotTo(HaveOccurred())

					result, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
					Expect(result).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("CheckNetwork", func() {
			It("executes the plugin with command CHECK", func() {
				cacheFile := resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
				err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
				Expect(err).NotTo(HaveOccurred())
				cachedJson := `{
					"cniVersion": "1.0.0",
					"ips": [{"address": "10.1.2.3/24"}],
					"dns": {}
				}`
				err = ioutil.WriteFile(cacheFile, []byte(cachedJson), 0600)
				Expect(err).NotTo(HaveOccurred())

				err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				debug, err := noop_debug.ReadDebug(debugFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.Command).To(Equal("CHECK"))
				Expect(string(debug.CmdArgs.StdinData)).To(ContainSubstring("\"portMappings\":"))

				// Explicitly match stdin data as json after
				// inserting the expected prevResult
				var data, data2 map[string]interface{}
				err = json.Unmarshal(expectedCmdArgs.StdinData, &data)
				Expect(err).NotTo(HaveOccurred())
				err = json.Unmarshal([]byte(cachedJson), &data2)
				Expect(err).NotTo(HaveOccurred())
				data["prevResult"] = data2
				expectedStdinJson, err := json.Marshal(data)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.CmdArgs.StdinData).To(MatchJSON(expectedStdinJson))

				debug.CmdArgs.StdinData = nil
				expectedCmdArgs.StdinData = nil
				Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfig.Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					err := cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					debug.ReportError = "plugin error: banana"
					Expect(debug.WriteDebug(debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					err := cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})

			Context("when CHECK is called with a configuration version", func() {
				var cacheFile string

				BeforeEach(func() {
					cacheFile = resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("less than 0.4.0", func() {
					It("fails as CHECK is not supported before 0.4.0", func() {
						// Generate plugin config with older version
						var err error
						netConfig, err = libcni.ConfFromBytes([]byte(`{
							"type": "noop",
							"name": "apitest",
							"cniVersion": "0.3.1"
						}`))
						Expect(err).NotTo(HaveOccurred())
						err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).To(MatchError("configuration version \"0.3.1\" does not support the CHECK command"))

						debug, err := noop_debug.ReadDebug(debugFilePath)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(debug.CmdArgs.StdinData)).NotTo(ContainSubstring("\"prevResult\":"))
					})
				})

				Context("containing only a cached result", func() {
					It("only passes a prevResult to the plugin", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "1.0.0",
							"ips": [{"address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())
						debug, err := noop_debug.ReadDebug(debugFilePath)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(debug.CmdArgs.StdinData)).To(ContainSubstring("\"prevResult\":"))
						Expect(string(debug.CmdArgs.StdinData)).NotTo(ContainSubstring("\"config\":"))
						Expect(string(debug.CmdArgs.StdinData)).NotTo(ContainSubstring("\"kind\":"))
					})
				})

				Context("equal to 0.4.0", func() {
					It("passes a prevResult to the plugin", func() {
						ipResult := `{
							"cniVersion": "0.4.0",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`

						var err error
						netConfig, err = libcni.ConfFromBytes([]byte(`{
							"type": "noop",
							"name": "apitest",
							"cniVersion": "0.4.0",
							"capabilities": {
								"portMappings": true,
								"somethingElse": true,
								"noCapability": false
							}
						}`))
						Expect(err).NotTo(HaveOccurred())

						err = ioutil.WriteFile(cacheFile, []byte(ipResult), 0600)
						Expect(err).NotTo(HaveOccurred())

						debug.ReportResult = ipResult
						Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

						err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())
						debug, err := noop_debug.ReadDebug(debugFilePath)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(debug.CmdArgs.StdinData)).To(ContainSubstring("\"prevResult\":"))
					})
				})
			})

			Context("when the cached result", func() {
				var cacheFile string

				BeforeEach(func() {
					cacheFile = resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("is invalid JSON", func() {
					It("returns an error", func() {
						err := ioutil.WriteFile(cacheFile, []byte("adfadsfasdfasfdsafaf"), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).To(MatchError("failed to get network \"apitest\" cached result: decoding version from network config: invalid character 'a' looking for beginning of value"))
					})
				})

				Context("version doesn't match the config version", func() {
					It("succeeds when the cached result can be converted", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "0.3.1",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())
					})

					It("returns an error when the cached result cannot be converted", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "0.4567.0",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).To(MatchError("failed to get network \"apitest\" cached result: unsupported CNI result version \"0.4567.0\""))
					})
				})
			})
		})

		Describe("DelNetwork", func() {
			It("executes the plugin with command DEL", func() {
				cacheFile := resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
				err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
				Expect(err).NotTo(HaveOccurred())
				cachedJson := `{
					"cniVersion": "1.0.0",
					"ips": [{"address": "10.1.2.3/24"}],
					"dns": {}
				}`
				err = ioutil.WriteFile(cacheFile, []byte(cachedJson), 0600)
				Expect(err).NotTo(HaveOccurred())

				err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				debug, err := noop_debug.ReadDebug(debugFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.Command).To(Equal("DEL"))
				Expect(string(debug.CmdArgs.StdinData)).To(ContainSubstring("\"portMappings\":"))

				// Explicitly match stdin data as json after
				// inserting the expected prevResult
				var data, data2 map[string]interface{}
				err = json.Unmarshal(expectedCmdArgs.StdinData, &data)
				Expect(err).NotTo(HaveOccurred())
				err = json.Unmarshal([]byte(cachedJson), &data2)
				Expect(err).NotTo(HaveOccurred())
				data["prevResult"] = data2
				expectedStdinJson, err := json.Marshal(data)
				Expect(err).NotTo(HaveOccurred())
				Expect(debug.CmdArgs.StdinData).To(MatchJSON(expectedStdinJson))

				debug.CmdArgs.StdinData = nil
				expectedCmdArgs.StdinData = nil
				Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))

				// Ensure the cached result no longer exists
				cachedResult, err := cniConfig.GetNetworkCachedResult(netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(cachedResult).To(BeNil())

				// Ensure the cached config no longer exists
				cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(netConfig, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfig.Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					err := cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					debug.ReportError = "plugin error: banana"
					Expect(debug.WriteDebug(debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					err := cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})

			Context("when DEL is called twice", func() {
				var resultCacheFile string

				BeforeEach(func() {
					resultCacheFile = resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(resultCacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				It("deletes the cached result and config after the first DEL", func() {
					err := ioutil.WriteFile(resultCacheFile, []byte(`{
						"cniVersion": "0.4.0",
						"ips": [{"version": "4", "address": "10.1.2.3/24"}],
						"dns": {}
					}`), 0600)
					Expect(err).NotTo(HaveOccurred())

					err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).NotTo(HaveOccurred())
					_, err = ioutil.ReadFile(resultCacheFile)
					Expect(err).To(HaveOccurred())

					err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when DEL is called with a configuration version", func() {
				var cacheFile string

				BeforeEach(func() {
					cacheFile = resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("less than 0.4.0", func() {
					It("does not pass a prevResult to the plugin", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "0.3.1",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						// Generate plugin config with older version
						netConfig, err = libcni.ConfFromBytes([]byte(`{
							"type": "noop",
							"name": "apitest",
							"cniVersion": "0.3.1"
						}`))
						Expect(err).NotTo(HaveOccurred())
						err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())

						debug, err := noop_debug.ReadDebug(debugFilePath)
						Expect(err).NotTo(HaveOccurred())
						Expect(debug.Command).To(Equal("DEL"))
						Expect(string(debug.CmdArgs.StdinData)).NotTo(ContainSubstring("\"prevResult\":"))
					})
				})

				Context("equal to 0.4.0", func() {
					It("passes a prevResult to the plugin", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "0.4.0",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())

						debug, err := noop_debug.ReadDebug(debugFilePath)
						Expect(err).NotTo(HaveOccurred())
						Expect(debug.Command).To(Equal("DEL"))
						Expect(string(debug.CmdArgs.StdinData)).To(ContainSubstring("\"prevResult\":"))
					})
				})
			})

			Context("when the cached", func() {
				var cacheFile string

				BeforeEach(func() {
					cacheFile = resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("result is invalid JSON", func() {
					It("returns an error", func() {
						err := ioutil.WriteFile(cacheFile, []byte("adfadsfasdfasfdsafaf"), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).To(MatchError("failed to get network \"apitest\" cached result: decoding version from network config: invalid character 'a' looking for beginning of value"))
					})
				})

				Context("result version doesn't match the config version", func() {
					It("succeeds when the cached result can be converted", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "0.3.1",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())
					})

					It("returns an error when the cached result cannot be converted", func() {
						err := ioutil.WriteFile(cacheFile, []byte(`{
							"cniVersion": "0.4567.0",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`), 0600)
						Expect(err).NotTo(HaveOccurred())

						err = cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
						Expect(err).To(MatchError("failed to get network \"apitest\" cached result: unsupported CNI result version \"0.4567.0\""))
					})
				})
			})
		})

		Describe("GetVersionInfo", func() {
			It("executes the plugin with the command VERSION", func() {
				versionInfo, err := cniConfig.GetVersionInfo(ctx, "noop")
				Expect(err).NotTo(HaveOccurred())

				Expect(versionInfo).NotTo(BeNil())
				Expect(versionInfo.SupportedVersions()).To(Equal([]string{
					"0.-42.0", "0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0", "1.0.0",
				}))
			})

			Context("when finding the plugin fails", func() {
				It("returns the error", func() {
					_, err := cniConfig.GetVersionInfo(ctx, "does-not-exist")
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})
		})

		Describe("ValidateNetwork", func() {
			It("validates a good configuration", func() {
				_, err := cniConfig.ValidateNetwork(ctx, netConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("catches non-existent plugins", func() {
				netConfig.Network.Type = "nope"
				_, err := cniConfig.ValidateNetwork(ctx, netConfig)
				Expect(err).To(MatchError("failed to find plugin \"nope\" in path [" + cniConfig.Path[0] + "]"))
			})

			It("catches unsupported versions", func() {
				netConfig.Network.CNIVersion = "broken"
				_, err := cniConfig.ValidateNetwork(ctx, netConfig)
				Expect(err).To(MatchError("plugin noop does not support config version \"broken\""))
			})
			It("allows version to be omitted", func() {
				netConfig.Network.CNIVersion = ""
				_, err := cniConfig.ValidateNetwork(ctx, netConfig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Invoking a plugin list", func() {
		var (
			plugins       []pluginInfo
			cniBinPath    string
			cniConfig     *libcni.CNIConfig
			netConfigList *libcni.NetworkConfigList
			runtimeConfig *libcni.RuntimeConf
			rcMap         map[string]interface{}
			ctx           context.Context
			ipResult      string

			expectedCmdArgs skel.CmdArgs
		)

		BeforeEach(func() {
			capabilityArgs := map[string]interface{}{
				"portMappings": []portMapping{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
				},
				"otherCapability": 33,
			}

			cniBinPath = filepath.Dir(pluginPaths["noop"])
			cniConfig = libcni.NewCNIConfigWithCacheDir([]string{cniBinPath}, cacheDirPath, nil)
			runtimeConfig = &libcni.RuntimeConf{
				ContainerID:    "some-container-id",
				NetNS:          "/some/netns/path",
				IfName:         "some-eth0",
				Args:           [][2]string{{"FOO", "BAR"}},
				CapabilityArgs: capabilityArgs,
			}

			expectedCmdArgs = skel.CmdArgs{
				ContainerID: runtimeConfig.ContainerID,
				Netns:       runtimeConfig.NetNS,
				IfName:      runtimeConfig.IfName,
				Args:        "FOO=BAR",
				Path:        cniBinPath,
			}

			rcMap = map[string]interface{}{
				"containerId": runtimeConfig.ContainerID,
				"netNs":       runtimeConfig.NetNS,
				"ifName":      runtimeConfig.IfName,
				"args": map[string]string{
					"FOO": "BAR",
				},
				"portMappings":    capabilityArgs["portMappings"],
				"otherCapability": capabilityArgs["otherCapability"],
			}

			ipResult = fmt.Sprintf(`{"cniVersion": "%s", "dns":{},"ips":[{"address": "10.1.2.3/24"}]}`, current.ImplementedSpecVersion)
			netConfigList, plugins = makePluginList(current.ImplementedSpecVersion, ipResult, rcMap)

			ctx = context.TODO()
		})

		AfterEach(func() {
			for _, p := range plugins {
				Expect(os.RemoveAll(p.debugFilePath)).To(Succeed())
			}
		})

		Describe("AddNetworkList", func() {
			It("executes all plugins with command ADD and returns an intermediate result", func() {
				r, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				result, err := current.GetResult(r)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal(&current.Result{
					CNIVersion: current.ImplementedSpecVersion,
					// IP4 added by first plugin
					IPs: []*current.IPConfig{
						{
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

					// Must explicitly match JSON due to dict element ordering
					Expect(debug.CmdArgs.StdinData).To(MatchJSON(plugins[i].stdinData))
					debug.CmdArgs.StdinData = nil
					Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
				}
			})

			It("writes the correct cached result", func() {
				r, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				result, err := current.GetResult(r)
				Expect(err).NotTo(HaveOccurred())

				// Ensure the cached result matches the returned one
				cachedResult, err := cniConfig.GetNetworkListCachedResult(netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				result2, err := current.GetResult(cachedResult)
				Expect(err).NotTo(HaveOccurred())
				cachedJson, err := json.Marshal(result2)
				Expect(err).NotTo(HaveOccurred())
				returnedJson, err := json.Marshal(result)
				Expect(err).NotTo(HaveOccurred())
				Expect(cachedJson).To(MatchJSON(returnedJson))
			})

			It("writes the correct cached config", func() {
				_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				// Ensure the cached config matches the returned one
				cachedConfig, newRt, err := cniConfig.GetNetworkListCachedConfig(netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(bytes.Equal(cachedConfig, netConfigList.Bytes)).To(BeTrue())
				Expect(reflect.DeepEqual(newRt.Args, runtimeConfig.Args)).To(BeTrue())
				// CapabilityArgs are freeform, so we have to match their JSON not
				// the Go structs (which could be unmarshalled differently than the
				// struct that was marshalled).
				expectedCABytes, err := json.Marshal(runtimeConfig.CapabilityArgs)
				Expect(err).NotTo(HaveOccurred())
				foundCABytes, err := json.Marshal(newRt.CapabilityArgs)
				Expect(err).NotTo(HaveOccurred())
				Expect(foundCABytes).To(MatchJSON(expectedCABytes))
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfigList.Plugins[1].Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when there is an invalid containerID", func() {
				BeforeEach(func() {
					runtimeConfig.ContainerID = "some-%%container-id"
				})

				It("returns the error", func() {
					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "invalid characters in containerID",
						Details: "some-%%container-id",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})
			})

			Context("when there is an invalid networkName", func() {
				BeforeEach(func() {
					netConfigList.Name = "invalid-%%-name"
				})

				It("returns the error", func() {
					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidNetworkConfig,
						Msg:     "invalid characters found in network name",
						Details: "invalid-%%-name",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})
			})

			Context("return errors when interface name is invalid", func() {
				It("interface name is empty", func() {
					runtimeConfig.IfName = ""

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name is empty",
						Details: "",
					}))
				})

				It("interface name is too long", func() {
					runtimeConfig.IfName = "1234567890123456"

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name is too long",
						Details: "interface name should be less than 16 characters",
					}))
				})

				It("interface name is .", func() {
					runtimeConfig.IfName = "."

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name is . or ..",
						Details: "",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})

				It("interface name is ..", func() {
					runtimeConfig.IfName = ".."

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name is . or ..",
						Details: "",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})

				It("interface name contains invalid characters /", func() {
					runtimeConfig.IfName = "test/test"

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name contains / or : or whitespace characters",
						Details: "",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})

				It("interface name contains invalid characters :", func() {
					runtimeConfig.IfName = "test:test"

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name contains / or : or whitespace characters",
						Details: "",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})

				It("interface name contains invalid characters whitespace", func() {
					runtimeConfig.IfName = "test test"

					_, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(Equal(&types.Error{
						Code:    types.ErrInvalidEnvironmentVariables,
						Msg:     "interface name contains / or : or whitespace characters",
						Details: "",
					}))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})
			})

			Context("when the second plugin errors", func() {
				BeforeEach(func() {
					plugins[1].debug.ReportError = "plugin error: banana"
					Expect(plugins[1].debug.WriteDebug(plugins[1].debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					result, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(result).To(BeNil())
					Expect(errors.Unwrap(err)).To(MatchError("plugin error: banana"))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (add):"))
				})
				It("should not have written cache files", func() {
					resultCacheFile := resultCacheFilePath(cacheDirPath, netConfigList.Name, runtimeConfig)
					_, err := ioutil.ReadFile(resultCacheFile)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the cache directory cannot be accessed", func() {
				It("returns an error when the results cache file cannot be written", func() {
					// Make the results directory inaccessible by making it a
					// file instead of a directory
					tmpPath := filepath.Join(cacheDirPath, "results")
					err := ioutil.WriteFile(tmpPath, []byte("afdsasdfasdf"), 0600)
					Expect(err).NotTo(HaveOccurred())

					result, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(result).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("CheckNetworkList", func() {
			It("executes all plugins with command CHECK", func() {
				cacheFile := resultCacheFilePath(cacheDirPath, netConfigList.Name, runtimeConfig)
				err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(cacheFile, []byte(ipResult), 0600)
				Expect(err).NotTo(HaveOccurred())

				err = cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				for i := 0; i < len(plugins); i++ {
					debug, err := noop_debug.ReadDebug(plugins[i].debugFilePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(debug.Command).To(Equal("CHECK"))

					// Ensure each plugin gets the prevResult from the cache
					asMap := make(map[string]interface{})
					err = json.Unmarshal(debug.CmdArgs.StdinData, &asMap)
					Expect(err).NotTo(HaveOccurred())
					Expect(asMap["prevResult"]).NotTo(BeNil())
					foo, err := json.Marshal(asMap["prevResult"])
					Expect(err).NotTo(HaveOccurred())
					Expect(foo).To(MatchJSON(ipResult))

					debug.CmdArgs.StdinData = nil
					Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
				}
			})

			It("does not executes plugins with command CHECK when disableCheck is true", func() {
				netConfigList.DisableCheck = true
				err := cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				for i := 0; i < len(plugins); i++ {
					debug, err := noop_debug.ReadDebug(plugins[i].debugFilePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(debug.Command).To(Equal(""))
				}
			})

			Context("when the configuration version", func() {
				var cacheFile string

				BeforeEach(func() {
					cacheFile = resultCacheFilePath(cacheDirPath, netConfigList.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("is 0.4.0", func() {
					It("passes a cached result to the first plugin", func() {
						ipResult = `{
							"cniVersion": "0.4.0",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`
						err := ioutil.WriteFile(cacheFile, []byte(ipResult), 0600)
						Expect(err).NotTo(HaveOccurred())

						netConfigList, plugins = makePluginList("0.4.0", ipResult, rcMap)
						for _, p := range plugins {
							p.debug.ReportResult = ipResult
							Expect(p.debug.WriteDebug(p.debugFilePath)).To(Succeed())
						}

						err = cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())

						// Match the first plugin's stdin config to the cached result JSON
						debug, err := noop_debug.ReadDebug(plugins[0].debugFilePath)
						Expect(err).NotTo(HaveOccurred())

						var data map[string]interface{}
						err = json.Unmarshal(debug.CmdArgs.StdinData, &data)
						Expect(err).NotTo(HaveOccurred())
						stdinPrevResult, err := json.Marshal(data["prevResult"])
						Expect(err).NotTo(HaveOccurred())
						Expect(stdinPrevResult).To(MatchJSON(ipResult))
					})
				})

				Context("is less than 0.4.0", func() {
					It("fails as CHECK is not supported before 0.4.0", func() {
						// Set an older CNI version
						confList := make(map[string]interface{})
						err := json.Unmarshal(netConfigList.Bytes, &confList)
						Expect(err).NotTo(HaveOccurred())
						confList["cniVersion"] = "0.3.1"
						newBytes, err := json.Marshal(confList)
						Expect(err).NotTo(HaveOccurred())

						netConfigList, err = libcni.ConfListFromBytes(newBytes)
						Expect(err).NotTo(HaveOccurred())
						err = cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
						Expect(err).To(MatchError("configuration version \"0.3.1\" does not support the CHECK command"))
					})
				})
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfigList.Plugins[1].Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					err := cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the second plugin errors", func() {
				BeforeEach(func() {
					plugins[1].debug.ReportError = "plugin error: banana"
					Expect(plugins[1].debug.WriteDebug(plugins[1].debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					err := cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(err).To(MatchError("plugin error: banana"))
				})
			})

			Context("when the cached result is invalid", func() {
				It("returns an error", func() {
					cacheFile := resultCacheFilePath(cacheDirPath, netConfigList.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
					err = ioutil.WriteFile(cacheFile, []byte("adfadsfasdfasfdsafaf"), 0600)
					Expect(err).NotTo(HaveOccurred())

					err = cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(err).To(MatchError("failed to get network \"some-list\" cached result: decoding version from network config: invalid character 'a' looking for beginning of value"))
				})
			})
		})

		Describe("DelNetworkList", func() {
			It("executes all the plugins in reverse order with command DEL", func() {
				err := cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())

				for i := 0; i < len(plugins); i++ {
					debug, err := noop_debug.ReadDebug(plugins[i].debugFilePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(debug.Command).To(Equal("DEL"))

					// Must explicitly match JSON due to dict element ordering
					Expect(debug.CmdArgs.StdinData).To(MatchJSON(plugins[i].stdinData))
					debug.CmdArgs.StdinData = nil
					Expect(debug.CmdArgs).To(Equal(expectedCmdArgs))
				}

				// Ensure the cached result no longer exists
				cachedResult, err := cniConfig.GetNetworkListCachedResult(netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(cachedResult).To(BeNil())

				// Ensure the cached config no longer exists
				cachedConfig, newRt, err := cniConfig.GetNetworkListCachedConfig(netConfigList, runtimeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())
			})

			Context("when the configuration version", func() {
				var cacheFile string

				BeforeEach(func() {
					cacheFile = resultCacheFilePath(cacheDirPath, netConfigList.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("is 0.4.0", func() {
					It("passes a cached result to the first plugin", func() {
						ipResult = `{
							"cniVersion": "0.4.0",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`
						err := ioutil.WriteFile(cacheFile, []byte(ipResult), 0600)
						Expect(err).NotTo(HaveOccurred())

						netConfigList, plugins = makePluginList("0.4.0", ipResult, rcMap)
						for _, p := range plugins {
							p.debug.ReportResult = ipResult
							Expect(p.debug.WriteDebug(p.debugFilePath)).To(Succeed())
						}

						err = cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())

						// Match the first plugin's stdin config to the cached result JSON
						debug, err := noop_debug.ReadDebug(plugins[0].debugFilePath)
						Expect(err).NotTo(HaveOccurred())

						var data map[string]interface{}
						err = json.Unmarshal(debug.CmdArgs.StdinData, &data)
						Expect(err).NotTo(HaveOccurred())
						stdinPrevResult, err := json.Marshal(data["prevResult"])
						Expect(err).NotTo(HaveOccurred())
						Expect(stdinPrevResult).To(MatchJSON(ipResult))
					})
				})

				Context("is less than 0.4.0", func() {
					It("does not pass a cached result to the first plugin", func() {
						ipResult = `{
							"cniVersion": "0.3.1",
							"ips": [{"version": "4", "address": "10.1.2.3/24"}],
							"dns": {}
						}`
						err := ioutil.WriteFile(cacheFile, []byte(ipResult), 0600)
						Expect(err).NotTo(HaveOccurred())

						netConfigList, plugins = makePluginList("0.3.1", ipResult, rcMap)
						for _, p := range plugins {
							p.debug.ReportResult = ipResult
							Expect(p.debug.WriteDebug(p.debugFilePath)).To(Succeed())
						}

						err = cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
						Expect(err).NotTo(HaveOccurred())

						// Make sure first plugin does not receive a prevResult
						debug, err := noop_debug.ReadDebug(plugins[0].debugFilePath)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(debug.CmdArgs.StdinData)).NotTo(ContainSubstring("\"prevResult\":"))
					})
				})
			})

			Context("when finding the plugin fails", func() {
				BeforeEach(func() {
					netConfigList.Plugins[1].Network.Type = "does-not-exist"
				})

				It("returns the error", func() {
					err := cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(err).To(MatchError(ContainSubstring(`failed to find plugin "does-not-exist"`)))
				})
			})

			Context("when the plugin errors", func() {
				BeforeEach(func() {
					plugins[1].debug.ReportError = "plugin error: banana"
					Expect(plugins[1].debug.WriteDebug(plugins[1].debugFilePath)).To(Succeed())
				})
				It("unmarshals and returns the error", func() {
					err := cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(errors.Unwrap(err)).To(MatchError("plugin error: banana"))
					Expect(err.Error()).To(HavePrefix("plugin type=\"noop\" failed (delete):"))
				})
			})

			Context("when the cached result is invalid", func() {
				It("returns an error", func() {
					cacheFile := resultCacheFilePath(cacheDirPath, netConfigList.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(cacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())
					err = ioutil.WriteFile(cacheFile, []byte("adfadsfasdfasfdsafaf"), 0600)
					Expect(err).NotTo(HaveOccurred())

					err = cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
					Expect(err).To(MatchError("failed to get network \"some-list\" cached result: decoding version from network config: invalid character 'a' looking for beginning of value"))
				})
			})
		})
		Describe("ValidateNetworkList", func() {
			It("Checks that all plugins exist", func() {
				caps, err := cniConfig.ValidateNetworkList(ctx, netConfigList)
				Expect(err).NotTo(HaveOccurred())
				Expect(caps).To(ConsistOf("portMappings", "otherCapability"))

				netConfigList.Plugins[1].Network.Type = "nope"
				_, err = cniConfig.ValidateNetworkList(ctx, netConfigList)
				Expect(err).To(MatchError("[failed to find plugin \"nope\" in path [" + cniConfig.Path[0] + "]]"))
			})

			It("Checks that the plugins support the needed version", func() {
				netConfigList.CNIVersion = "broken"
				_, err := cniConfig.ValidateNetworkList(ctx, netConfigList)

				// The config list is just noop 3 times, so we get 3 errors
				Expect(err).To(MatchError("[plugin noop does not support config version \"broken\" plugin noop does not support config version \"broken\" plugin noop does not support config version \"broken\"]"))
			})
		})
	})

	Describe("Invoking a sleep plugin", func() {
		var (
			debugFilePath string
			debug         *noop_debug.Debug
			cniBinPath    string
			pluginConfig  string
			cniConfig     *libcni.CNIConfig
			netConfig     *libcni.NetworkConfig
			runtimeConfig *libcni.RuntimeConf
			netConfigList *libcni.NetworkConfigList
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

			portMappings := []portMapping{
				{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			}

			pluginConfig = fmt.Sprintf(`{
				"type": "sleep",
				"name": "apitest",
				"some-key": "some-value",
				"cniVersion": "%s",
				"capabilities": { "portMappings": true }
			}`, current.ImplementedSpecVersion)

			cniBinPath = filepath.Dir(pluginPaths["sleep"])
			cniConfig = libcni.NewCNIConfig([]string{cniBinPath}, nil)
			netConfig, err = libcni.ConfFromBytes([]byte(pluginConfig))
			Expect(err).NotTo(HaveOccurred())
			runtimeConfig = &libcni.RuntimeConf{
				ContainerID: "some-container-id",
				NetNS:       "/some/netns/path",
				IfName:      "some-eth0",
				Args:        [][2]string{{"DEBUG", debugFilePath}},
			}

			// inject runtime args into the expected plugin config
			conf := make(map[string]interface{})
			err = json.Unmarshal([]byte(pluginConfig), &conf)
			Expect(err).NotTo(HaveOccurred())
			conf["runtimeConfig"] = map[string]interface{}{
				"portMappings": portMappings,
			}

			configList := []byte(fmt.Sprintf(`{
  "name": "some-list",
  "cniVersion": "%s",
  "plugins": [
    %s
  ]
}`, current.ImplementedSpecVersion, pluginConfig))

			netConfigList, err = libcni.ConfListFromBytes(configList)
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			Expect(os.RemoveAll(debugFilePath)).To(Succeed())
		})

		Describe("AddNetwork", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					result, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
					cancel()
					Expect(result).To(BeNil())
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("DelNetwork", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					err := cniConfig.DelNetwork(ctx, netConfig, runtimeConfig)
					cancel()
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("CheckNetwork", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					err := cniConfig.CheckNetwork(ctx, netConfig, runtimeConfig)
					cancel()
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("GetVersionInfo", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					result, err := cniConfig.GetVersionInfo(ctx, "sleep")
					cancel()
					Expect(result).To(BeNil())
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("ValidateNetwork", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					_, err := cniConfig.ValidateNetwork(ctx, netConfig)
					cancel()
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("AddNetworkList", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					result, err := cniConfig.AddNetworkList(ctx, netConfigList, runtimeConfig)
					cancel()
					Expect(result).To(BeNil())
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("DelNetworkList", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					err := cniConfig.DelNetworkList(ctx, netConfigList, runtimeConfig)
					cancel()
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("CheckNetworkList", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					err := cniConfig.CheckNetworkList(ctx, netConfigList, runtimeConfig)
					cancel()
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

		Describe("ValidateNetworkList", func() {
			Context("when the plugin timeout", func() {
				It("returns the timeout error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
					_, err := cniConfig.ValidateNetworkList(ctx, netConfigList)
					cancel()
					Expect(err).To(MatchError(ContainSubstring("netplugin failed with no error message")))
				})

			})
		})

	})

	Describe("Cache operations", func() {
		var (
			debugFilePath string
			debug         *noop_debug.Debug
			cniBinPath    string
			pluginConfig  string
			cniConfig     *libcni.CNIConfig
			netConfig     *libcni.NetworkConfig
			runtimeConfig *libcni.RuntimeConf

			ctx context.Context
		)
		firstIP := "10.1.2.3/24"
		firstIfname := "eth0"
		secondIP := "10.1.2.5/24"
		secondIfname := "eth1"
		containerID := "some-container-id"
		netName := "cachetest"
		netNS := "/some/netns/path"

		BeforeEach(func() {
			debugFile, err := ioutil.TempFile("", "cni_debug")
			Expect(err).NotTo(HaveOccurred())
			Expect(debugFile.Close()).To(Succeed())
			debugFilePath = debugFile.Name()

			debug = &noop_debug.Debug{
				ReportResult: fmt.Sprintf(`{
					"cniVersion": "%s",
					"ips": [{"version": "4", "address": "%s"}]
				}`, current.ImplementedSpecVersion, firstIP),
			}
			Expect(debug.WriteDebug(debugFilePath)).To(Succeed())

			cniBinPath = filepath.Dir(pluginPaths["noop"])
			pluginConfig = fmt.Sprintf(`{
				"type": "noop",
				"name": "%s",
				"cniVersion": "%s"
			}`, netName, current.ImplementedSpecVersion)
			cniConfig = libcni.NewCNIConfigWithCacheDir([]string{cniBinPath}, cacheDirPath, nil)
			netConfig, err = libcni.ConfFromBytes([]byte(pluginConfig))
			Expect(err).NotTo(HaveOccurred())
			runtimeConfig = &libcni.RuntimeConf{
				ContainerID: containerID,
				NetNS:       netNS,
				IfName:      firstIfname,
				Args:        [][2]string{{"DEBUG", debugFilePath}},
			}
			ctx = context.TODO()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(debugFilePath)).To(Succeed())
		})

		It("creates separate result cache files for multiple attachments to the same network", func() {
			_, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
			Expect(err).NotTo(HaveOccurred())

			debug.ReportResult = fmt.Sprintf(`{
				"cniVersion": "%s",
				"ips": [{"version": "4", "address": "%s"}]
			}`, current.ImplementedSpecVersion, secondIP)
			Expect(debug.WriteDebug(debugFilePath)).To(Succeed())
			runtimeConfig.IfName = secondIfname
			_, err = cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
			Expect(err).NotTo(HaveOccurred())

			resultsDir := filepath.Join(cacheDirPath, "results")
			files, err := ioutil.ReadDir(resultsDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(2))
			var foundFirst, foundSecond bool
			for _, f := range files {
				type cachedConfig struct {
					Kind        string `json:"kind"`
					IfName      string `json:"ifName"`
					ContainerID string `json:"containerId"`
					NetworkName string `json:"networkName"`
				}

				data, err := ioutil.ReadFile(filepath.Join(resultsDir, f.Name()))
				Expect(err).NotTo(HaveOccurred())
				cc := &cachedConfig{}
				err = json.Unmarshal(data, cc)
				Expect(err).NotTo(HaveOccurred())
				Expect(cc.Kind).To(Equal("cniCacheV1"))
				Expect(cc.ContainerID).To(Equal(containerID))
				Expect(cc.NetworkName).To(Equal(netName))
				if strings.HasSuffix(f.Name(), firstIfname) {
					foundFirst = true
					Expect(strings.Contains(string(data), firstIP)).To(BeTrue())
					Expect(cc.IfName).To(Equal(firstIfname))
				} else if strings.HasSuffix(f.Name(), secondIfname) {
					foundSecond = true
					Expect(strings.Contains(string(data), secondIP)).To(BeTrue())
					Expect(cc.IfName).To(Equal(secondIfname))
				}
			}
			Expect(foundFirst).To(BeTrue())
			Expect(foundSecond).To(BeTrue())
		})

		It("returns an updated copy of RuntimeConf filled with cached info", func() {
			_, err := cniConfig.AddNetwork(ctx, netConfig, runtimeConfig)
			Expect(err).NotTo(HaveOccurred())

			// Ensure the cached config matches the sent one
			rt := &libcni.RuntimeConf{
				ContainerID: containerID,
				NetNS:       netNS,
				IfName:      firstIfname,
			}
			_, newRt, err := cniConfig.GetNetworkListCachedConfig(&libcni.NetworkConfigList{Name: netName}, rt)
			Expect(err).NotTo(HaveOccurred())
			Expect(newRt.IfName).To(Equal(runtimeConfig.IfName))
			Expect(newRt.ContainerID).To(Equal(runtimeConfig.ContainerID))
			Expect(reflect.DeepEqual(newRt.Args, runtimeConfig.Args)).To(BeTrue())
			expectedCABytes, err := json.Marshal(runtimeConfig.CapabilityArgs)
			Expect(err).NotTo(HaveOccurred())
			foundCABytes, err := json.Marshal(newRt.CapabilityArgs)
			Expect(err).NotTo(HaveOccurred())
			Expect(foundCABytes).To(MatchJSON(expectedCABytes))
		})

		Context("when the RuntimeConf is incomplete", func() {
			var (
				testRt          *libcni.RuntimeConf
				testNetConf     *libcni.NetworkConfig
				testNetConfList *libcni.NetworkConfigList
			)

			BeforeEach(func() {
				testRt = &libcni.RuntimeConf{}
				testNetConf = &libcni.NetworkConfig{
					Network: &types.NetConf{},
				}
				testNetConfList = &libcni.NetworkConfigList{}
			})

			It("returns an error on missing network name", func() {
				testRt.ContainerID = containerID
				testRt.IfName = firstIfname
				cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(testNetConf, testRt)
				Expect(err).To(MatchError("cache file path requires network name (\"\"), container ID (\"some-container-id\"), and interface name (\"eth0\")"))
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())

				cachedConfig, newRt, err = cniConfig.GetNetworkListCachedConfig(testNetConfList, testRt)
				Expect(err).To(MatchError("cache file path requires network name (\"\"), container ID (\"some-container-id\"), and interface name (\"eth0\")"))
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())
			})

			It("returns an error on missing container ID", func() {
				testNetConf.Network.Name = "foobar"
				testNetConfList.Name = "foobar"
				testRt.IfName = firstIfname

				cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(testNetConf, testRt)
				Expect(err).To(MatchError("cache file path requires network name (\"foobar\"), container ID (\"\"), and interface name (\"eth0\")"))
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())

				cachedConfig, newRt, err = cniConfig.GetNetworkListCachedConfig(testNetConfList, testRt)
				Expect(err).To(MatchError("cache file path requires network name (\"foobar\"), container ID (\"\"), and interface name (\"eth0\")"))
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())
			})

			It("returns an error on missing interface name", func() {
				testNetConf.Network.Name = "foobar"
				testNetConfList.Name = "foobar"
				testRt.ContainerID = containerID

				cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(testNetConf, testRt)
				Expect(err).To(MatchError("cache file path requires network name (\"foobar\"), container ID (\"some-container-id\"), and interface name (\"\")"))
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())

				cachedConfig, newRt, err = cniConfig.GetNetworkListCachedConfig(testNetConfList, testRt)
				Expect(err).To(MatchError("cache file path requires network name (\"foobar\"), container ID (\"some-container-id\"), and interface name (\"\")"))
				Expect(cachedConfig).To(BeNil())
				Expect(newRt).To(BeNil())
			})
		})

		Context("when the cached config", func() {
			Context("is invalid JSON", func() {
				It("returns an error", func() {
					resultCacheFile := resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(resultCacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())

					err = ioutil.WriteFile(resultCacheFile, []byte("adfadsfasdfasfdsafaf"), 0600)
					Expect(err).NotTo(HaveOccurred())

					cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(netConfig, runtimeConfig)
					Expect(err).To(MatchError("failed to unmarshal cached network \"cachetest\" config: invalid character 'a' looking for beginning of value"))
					Expect(cachedConfig).To(BeNil())
					Expect(newRt).To(BeNil())
				})
			})
			Context("is missing", func() {
				It("returns no error", func() {
					resultCacheFile := resultCacheFilePath(cacheDirPath, netConfig.Network.Name, runtimeConfig)
					err := os.MkdirAll(filepath.Dir(resultCacheFile), 0700)
					Expect(err).NotTo(HaveOccurred())

					cachedConfig, newRt, err := cniConfig.GetNetworkCachedConfig(netConfig, runtimeConfig)
					Expect(err).NotTo(HaveOccurred())
					Expect(cachedConfig).To(BeNil())
					Expect(newRt).To(BeNil())
				})
			})
		})

	})
})
