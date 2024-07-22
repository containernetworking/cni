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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

var _ = Describe("Loading configuration from disk", func() {
	Describe("LoadConf", func() {
		var (
			configDir    string
			pluginConfig []byte
		)

		BeforeEach(func() {
			var err error
			configDir, err = os.MkdirTemp("", "plugin-conf")
			Expect(err).NotTo(HaveOccurred())

			pluginConfig = []byte(`{ "name": "some-plugin", "type": "foobar", "some-key": "some-value" }`)
			Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conf"), pluginConfig, 0o600)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(configDir)).To(Succeed())
		})

		It("finds the network config file for the plugin of the given type", func() {
			netConfig, err := libcni.LoadConf(configDir, "some-plugin")
			Expect(err).NotTo(HaveOccurred())
			Expect(netConfig).To(Equal(&libcni.PluginConfig{
				Network: &types.PluginConf{
					Name: "some-plugin",
					Type: "foobar",
				},
				Bytes: pluginConfig,
			}))
		})

		Context("when the config directory does not exist", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(configDir)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.LoadConf(configDir, "some-plugin")
				Expect(err).To(MatchError(libcni.NoConfigsFoundError{Dir: configDir}))
			})
		})

		Context("when the config file is .json extension instead of .conf", func() {
			BeforeEach(func() {
				Expect(os.Remove(configDir + "/50-whatever.conf")).To(Succeed())
				pluginConfig = []byte(`{ "name": "some-plugin", "some-key": "some-value", "type": "foobar" }`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.json"), pluginConfig, 0o600)).To(Succeed())
			})
			It("finds the network config file for the plugin of the given type", func() {
				netConfig, err := libcni.LoadConf(configDir, "some-plugin")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfig).To(Equal(&libcni.PluginConfig{
					Network: &types.PluginConf{
						Name: "some-plugin",
						Type: "foobar",
					},
					Bytes: pluginConfig,
				}))
			})
		})

		Context("when there is no config for the desired plugin", func() {
			It("returns a useful error", func() {
				_, err := libcni.LoadConf(configDir, "some-other-plugin")
				Expect(err).To(MatchError(ContainSubstring(`no net configuration with name "some-other-plugin" in`)))
			})
		})

		Context("when a config file is malformed", func() {
			BeforeEach(func() {
				Expect(os.WriteFile(filepath.Join(configDir, "00-bad.conf"), []byte(`{`), 0o600)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.LoadConf(configDir, "some-plugin")
				Expect(err).To(MatchError(`error parsing configuration: unexpected end of JSON input`))
			})
		})

		Context("when the config is in a nested subdir", func() {
			BeforeEach(func() {
				subdir := filepath.Join(configDir, "subdir1", "subdir2")
				Expect(os.MkdirAll(subdir, 0o700)).To(Succeed())

				pluginConfig = []byte(`{ "name": "deep", "some-key": "some-value" }`)
				Expect(os.WriteFile(filepath.Join(subdir, "90-deep.conf"), pluginConfig, 0o600)).To(Succeed())
			})

			It("will not find the config", func() {
				_, err := libcni.LoadConf(configDir, "deep")
				Expect(err).To(MatchError(HavePrefix("no net configuration with name")))
			})
		})
	})

	Describe("Capabilities", func() {
		var configDir string

		BeforeEach(func() {
			var err error
			configDir, err = os.MkdirTemp("", "plugin-conf")
			Expect(err).NotTo(HaveOccurred())

			pluginConfig := []byte(`{ "name": "some-plugin", "type": "noop", "cniVersion": "0.3.1", "capabilities": { "portMappings": true, "somethingElse": true, "noCapability": false } }`)
			Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conf"), pluginConfig, 0o600)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(configDir)).To(Succeed())
		})

		It("reads plugin capabilities from network config", func() {
			netConfig, err := libcni.LoadConf(configDir, "some-plugin")
			Expect(err).NotTo(HaveOccurred())
			Expect(netConfig.Network.Capabilities).To(Equal(map[string]bool{
				"portMappings":  true,
				"somethingElse": true,
				"noCapability":  false,
			}))
		})
	})

	Describe("ConfFromFile", func() {
		Context("when the file cannot be opened", func() {
			It("returns a useful error", func() {
				_, err := libcni.ConfFromFile("/tmp/nope/not-here")
				Expect(err).To(MatchError(HavePrefix(`error reading /tmp/nope/not-here: open /tmp/nope/not-here`)))
			})
		})

		Context("when the file is missing 'type'", func() {
			var fileName, configDir string
			BeforeEach(func() {
				var err error
				configDir, err = os.MkdirTemp("", "plugin-conf")
				Expect(err).NotTo(HaveOccurred())

				fileName = filepath.Join(configDir, "50-whatever.conf")
				pluginConfig := []byte(`{ "name": "some-plugin", "some-key": "some-value" }`)
				Expect(os.WriteFile(fileName, pluginConfig, 0o600)).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(configDir)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.ConfFromFile(fileName)
				Expect(err).To(MatchError(`error parsing configuration: missing 'type'`))
			})
		})
	})

	Describe("NetworkPluginConfFromBytes", func() {
		Context("when the config is missing 'type'", func() {
			It("returns a useful error", func() {
				_, err := libcni.ConfFromBytes([]byte(`{ "name": "some-plugin", "some-key": "some-value" }`))
				Expect(err).To(MatchError(`error parsing configuration: missing 'type'`))
			})
		})
	})

	Describe("LoadNetworkConf", func() {
		var (
			configDir  string
			configList []byte
		)

		BeforeEach(func() {
			var err error
			configDir, err = os.MkdirTemp("", "plugin-conf")
			Expect(err).NotTo(HaveOccurred())

			configList = []byte(`{
  "name": "some-network",
  "cniVersion": "0.2.0",
  "disableCheck": true,
  "plugins": [
    {
      "type": "host-local",
      "subnet": "10.0.0.1/24"
    },
    {
      "type": "bridge",
      "mtu": 1400
    },
    {
      "type": "port-forwarding",
      "ports": {"20.0.0.1:8080": "80"}
    }
  ]
}`)
			Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(configDir)).To(Succeed())
		})

		It("finds the network config file for the plugin of the given type", func() {
			netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
			Expect(err).NotTo(HaveOccurred())
			Expect(netConfigList).To(Equal(&libcni.NetworkConfigList{
				Name:         "some-network",
				CNIVersion:   "0.2.0",
				DisableCheck: true,
				Plugins: []*libcni.PluginConfig{
					{
						Network: &types.PluginConf{Type: "host-local"},
						Bytes:   []byte(`{"subnet":"10.0.0.1/24","type":"host-local"}`),
					},
					{
						Network: &types.PluginConf{Type: "bridge"},
						Bytes:   []byte(`{"mtu":1400,"type":"bridge"}`),
					},
					{
						Network: &types.PluginConf{Type: "port-forwarding"},
						Bytes:   []byte(`{"ports":{"20.0.0.1:8080":"80"},"type":"port-forwarding"}`),
					},
				},
				Bytes: configList,
			}))
		})

		Context("when there is a config file with the same name as the list", func() {
			BeforeEach(func() {
				configFile := []byte(`{
					"name": "some-network",
					"cniVersion": "0.2.0",
					"type": "bridge"
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "49-whatever.conf"), configFile, 0o600)).To(Succeed())
			})

			It("Loads the config list first", func() {
				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.Plugins).To(HaveLen(3))
			})

			It("falls back to the config file", func() {
				Expect(os.Remove(filepath.Join(configDir, "50-whatever.conflist"))).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.Plugins).To(HaveLen(1))
				Expect(netConfigList.Plugins[0].Network.Type).To(Equal("bridge"))
			})
		})

		Context("when the config directory does not exist", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(configDir)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).To(MatchError(libcni.NoConfigsFoundError{Dir: configDir}))
			})
		})

		Context("when there is no config for the desired network name", func() {
			It("returns a useful error", func() {
				_, err := libcni.LoadNetworkConf(configDir, "some-other-network")
				Expect(err).To(MatchError(libcni.NotFoundError{Dir: configDir, Name: "some-other-network"}))
			})
		})

		Context("when a config file is malformed", func() {
			BeforeEach(func() {
				Expect(os.WriteFile(filepath.Join(configDir, "00-bad.conflist"), []byte(`{`), 0o600)).To(Succeed())
			})

			It("returns a useful error", func() {
				_, err := libcni.LoadNetworkConf(configDir, "some-plugin")
				Expect(err).To(MatchError(`error parsing configuration list: unexpected end of JSON input`))
			})
		})

		Context("when the config is in a nested subdir", func() {
			BeforeEach(func() {
				subdir := filepath.Join(configDir, "subdir1", "subdir2")
				Expect(os.MkdirAll(subdir, 0o700)).To(Succeed())

				configList = []byte(`{
  "name": "deep",
  "cniVersion": "0.2.0",
  "plugins": [
    {
      "type": "host-local",
      "subnet": "10.0.0.1/24"
    },
  ]
}`)
				Expect(os.WriteFile(filepath.Join(subdir, "90-deep.conflist"), configList, 0o600)).To(Succeed())
			})

			It("will not find the config", func() {
				_, err := libcni.LoadNetworkConf(configDir, "deep")
				Expect(err).To(MatchError(HavePrefix("no net configuration with name")))
			})
		})

		Context("when disableCheck is a string not a boolean", func() {
			It("will read a 'true' value and convert to boolean", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "disableCheck": "true",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.DisableCheck).To(BeTrue())
			})

			It("will read a 'false' value and convert to boolean", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "disableCheck": "false",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.DisableCheck).To(BeFalse())
			})

			It("will return an error on an unrecognized value", func() {
				const badValue string = "adsfasdfasf"
				configList = []byte(fmt.Sprintf(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "disableCheck": "%s",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`, badValue))
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).To(MatchError(fmt.Sprintf("error parsing configuration list: invalid value \"%s\" for disableCheck", badValue)))
			})
		})

		Context("for loadOnlyInlinedPlugins", func() {
			It("the value will be parsed", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "loadOnlyInlinedPlugins": true,
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				dirPluginConf := []byte(`{
				      "type": "bro-check-out-my-plugin",
				      "subnet": "10.0.0.1/24"
				}`)

				subDir := filepath.Join(configDir, "some-network")
				Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(subDir, "funky-second-plugin.conf"), dirPluginConf, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.LoadOnlyInlinedPlugins).To(BeTrue())
			})

			It("the value will be false if not in config", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.LoadOnlyInlinedPlugins).To(BeFalse())
			})

			It("will return an error on an unrecognized value", func() {
				const badValue string = "sphagnum"
				configList = []byte(fmt.Sprintf(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "loadOnlyInlinedPlugins": "%s",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`, badValue))
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).To(MatchError(fmt.Sprintf(`error parsing configuration list: invalid value "%s" for loadOnlyInlinedPlugins`, badValue)))
			})

			It("will return an error if `plugins` is missing and `loadOnlyInlinedPlugins` is `true`", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "loadOnlyInlinedPlugins": true
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).To(MatchError("error parsing configuration list: `loadOnlyInlinedPlugins` is true, and no 'plugins' key"))
			})

			It("will return no error if `plugins` is missing and `loadOnlyInlinedPlugins` is false", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "loadOnlyInlinedPlugins": false
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				dirPluginConf := []byte(`{
				      "type": "bro-check-out-my-plugin",
				      "subnet": "10.0.0.1/24"
				}`)

				subDir := filepath.Join(configDir, "some-network")
				Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(subDir, "funky-second-plugin.conf"), dirPluginConf, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.LoadOnlyInlinedPlugins).To(BeFalse())
				Expect(netConfigList.Plugins).To(HaveLen(1))
			})

			It("will return error if `loadOnlyInlinedPlugins` is implicitly false + no conf plugin is defined, but no plugins subfolder with network name exists", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0"
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).To(MatchError("no plugin configs found"))
			})

			It("will return NO error if `loadOnlyInlinedPlugins` is implicitly false + at least 1 conf plugin is defined, but no plugins subfolder with network name exists", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
			})

			It("will return NO error if `loadOnlyInlinedPlugins` is implicitly false + at least 1 conf plugin is defined and network name subfolder exists, but is empty/unreadable", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
				  ]
				}`)
				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				subDir := filepath.Join(configDir, "some-network")
				Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())

				_, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
			})

			It("will merge loaded and inlined plugin lists if both `plugins` is set and `loadOnlyInlinedPlugins` is false", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
	          	  ]
				}`)

				dirPluginConf := []byte(`{
				      "type": "bro-check-out-my-plugin",
				      "subnet": "10.0.0.1/24"
				}`)

				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())

				subDir := filepath.Join(configDir, "some-network")
				Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(subDir, "funky-second-plugin.conf"), dirPluginConf, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.LoadOnlyInlinedPlugins).To(BeFalse())
				Expect(netConfigList.Plugins).To(HaveLen(2))
			})

			It("will ignore loaded plugins if `plugins` is set and `loadOnlyInlinedPlugins` is true", func() {
				configList = []byte(`{
				  "name": "some-network",
				  "cniVersion": "0.4.0",
				  "loadOnlyInlinedPlugins": true,
				  "plugins": [
				    {
				      "type": "host-local",
				      "subnet": "10.0.0.1/24"
				    }
	          	  ]
				}`)

				dirPluginConf := []byte(`{
				      "type": "bro-check-out-my-plugin",
				      "subnet": "10.0.0.1/24"
				}`)

				Expect(os.WriteFile(filepath.Join(configDir, "50-whatever.conflist"), configList, 0o600)).To(Succeed())
				subDir := filepath.Join(configDir, "some-network")
				Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(subDir, "funky-second-plugin.conf"), dirPluginConf, 0o600)).To(Succeed())

				netConfigList, err := libcni.LoadNetworkConf(configDir, "some-network")
				Expect(err).NotTo(HaveOccurred())
				Expect(netConfigList.LoadOnlyInlinedPlugins).To(BeTrue())
				Expect(netConfigList.Plugins).To(HaveLen(1))
				Expect(netConfigList.Plugins[0].Network.Type).To(Equal("host-local"))
			})
		})
	})

	Describe("NetworkConfFromFile", func() {
		Context("when the file cannot be opened", func() {
			It("returns a useful error", func() {
				_, err := libcni.NetworkConfFromFile("/tmp/nope/not-here")
				Expect(err).To(MatchError(HavePrefix(`error reading /tmp/nope/not-here: open /tmp/nope/not-here`)))
			})
		})
	})

	Describe("InjectConf", func() {
		var testNetConfig *libcni.PluginConfig

		BeforeEach(func() {
			testNetConfig = &libcni.PluginConfig{
				Network: &types.PluginConf{Name: "some-plugin", Type: "foobar"},
				Bytes:   []byte(`{ "name": "some-plugin", "type": "foobar" }`),
			}
		})

		Context("when function parameters are incorrect", func() {
			It("returns unmarshal error", func() {
				conf := &libcni.PluginConfig{
					Network: &types.PluginConf{Name: "some-plugin"},
					Bytes:   []byte(`{ cc cc cc}`),
				}

				_, err := libcni.InjectConf(conf, map[string]interface{}{"": nil})
				Expect(err).To(MatchError(HavePrefix(`unmarshal existing network bytes`)))
			})

			It("returns key  error", func() {
				_, err := libcni.InjectConf(testNetConfig, map[string]interface{}{"": nil})
				Expect(err).To(MatchError(HavePrefix(`keys cannot be empty`)))
			})

			It("returns newValue  error", func() {
				_, err := libcni.InjectConf(testNetConfig, map[string]interface{}{"test": nil})
				Expect(err).To(MatchError(HavePrefix(`key 'test' value must not be nil`)))
			})
		})

		Context("when new string value added", func() {
			It("adds the new key & value to the config", func() {
				newPluginConfig := []byte(`{"name":"some-plugin","test":"test","type":"foobar"}`)

				resultConfig, err := libcni.InjectConf(testNetConfig, map[string]interface{}{"test": "test"})
				Expect(err).NotTo(HaveOccurred())
				Expect(resultConfig).To(Equal(&libcni.PluginConfig{
					Network: &types.PluginConf{
						Name: "some-plugin",
						Type: "foobar",
					},
					Bytes: newPluginConfig,
				}))
			})

			It("adds the new value for exiting key", func() {
				newPluginConfig := []byte(`{"name":"some-plugin","test":"changedValue","type":"foobar"}`)

				resultConfig, err := libcni.InjectConf(testNetConfig, map[string]interface{}{"test": "test"})
				Expect(err).NotTo(HaveOccurred())

				resultConfig, err = libcni.InjectConf(resultConfig, map[string]interface{}{"test": "changedValue"})
				Expect(err).NotTo(HaveOccurred())

				Expect(resultConfig).To(Equal(&libcni.PluginConfig{
					Network: &types.PluginConf{
						Name: "some-plugin",
						Type: "foobar",
					},
					Bytes: newPluginConfig,
				}))
			})

			It("adds existing key & value", func() {
				newPluginConfig := []byte(`{"name":"some-plugin","test":"test","type":"foobar"}`)

				resultConfig, err := libcni.InjectConf(testNetConfig, map[string]interface{}{"test": "test"})
				Expect(err).NotTo(HaveOccurred())

				resultConfig, err = libcni.InjectConf(resultConfig, map[string]interface{}{"test": "test"})
				Expect(err).NotTo(HaveOccurred())

				Expect(resultConfig).To(Equal(&libcni.PluginConfig{
					Network: &types.PluginConf{
						Name: "some-plugin",
						Type: "foobar",
					},
					Bytes: newPluginConfig,
				}))
			})

			It("adds sub-fields of NetworkConfig.Network to the config", func() {
				expectedPluginConfig := []byte(`{"dns":{"domain":"local","nameservers":["server1","server2"]},"name":"some-plugin","type":"bridge"}`)
				servers := []string{"server1", "server2"}
				newDNS := &types.DNS{Nameservers: servers, Domain: "local"}

				// inject DNS
				resultConfig, err := libcni.InjectConf(testNetConfig, map[string]interface{}{"dns": newDNS})
				Expect(err).NotTo(HaveOccurred())

				// inject type
				resultConfig, err = libcni.InjectConf(resultConfig, map[string]interface{}{"type": "bridge"})
				Expect(err).NotTo(HaveOccurred())

				Expect(resultConfig).To(Equal(&libcni.PluginConfig{
					Network: &types.PluginConf{Name: "some-plugin", Type: "bridge", DNS: types.DNS{Nameservers: servers, Domain: "local"}},
					Bytes:   expectedPluginConfig,
				}))
			})
		})
	})
})

var _ = Describe("NetworkConfFromBytes", func() {
	Describe("Version selection", func() {
		makeConfig := func(versions ...string) []byte {
			// ugly fake json encoding, but whatever
			vs := []string{}
			for _, v := range versions {
				vs = append(vs, fmt.Sprintf(`"%s"`, v))
			}
			return []byte(fmt.Sprintf(`{"name": "test", "cniVersions": [%s], "plugins": [{"type": "foo"}]}`, strings.Join(vs, ",")))
		}
		It("correctly selects the maximum version", func() {
			conf, err := libcni.NetworkConfFromBytes(makeConfig("1.1.0", "0.4.0", "1.0.0"))
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.CNIVersion).To(Equal("1.1.0"))
		})

		It("selects the highest version supported by libcni", func() {
			conf, err := libcni.NetworkConfFromBytes(makeConfig("99.0.0", "1.1.0", "0.4.0", "1.0.0"))
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.CNIVersion).To(Equal("1.1.0"))
		})

		It("fails when invalid versions are specified", func() {
			_, err := libcni.NetworkConfFromBytes(makeConfig("1.1.0", "0.4.0", "1.0.f"))
			Expect(err).To(HaveOccurred())
		})

		It("falls back to cniVersion", func() {
			conf, err := libcni.NetworkConfFromBytes([]byte(`{"name": "test", "cniVersion": "1.2.3", "plugins": [{"type": "foo"}]}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.CNIVersion).To(Equal("1.2.3"))
		})

		It("merges cniVersions and cniVersion", func() {
			conf, err := libcni.NetworkConfFromBytes([]byte(`{"name": "test", "cniVersion": "1.0.0", "cniVersions": ["0.1.0", "0.4.0"], "plugins": [{"type": "foo"}]}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.CNIVersion).To(Equal("1.0.0"))
		})

		It("handles an empty cniVersions array", func() {
			conf, err := libcni.NetworkConfFromBytes([]byte(`{"name": "test", "cniVersions": [], "plugins": [{"type": "foo"}]}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.CNIVersion).To(Equal(""))
		})
	})
})

var _ = Describe("ConfListFromConf", func() {
	var testNetConfig *libcni.PluginConfig

	BeforeEach(func() {
		pb := []byte(`{"name":"some-plugin","cniVersion":"0.3.1", "type":"foobar"}`)
		tc, err := libcni.ConfFromBytes(pb)
		Expect(err).NotTo(HaveOccurred())
		testNetConfig = tc
	})

	It("correctly upconverts a NetworkConfig to a NetworkConfigList", func() {
		ncl, err := libcni.ConfListFromConf(testNetConfig)
		Expect(err).NotTo(HaveOccurred())
		bytes := ncl.Bytes

		// null out the json - we don't care about the exact marshalling
		ncl.Bytes = nil
		ncl.Plugins[0].Bytes = nil
		testNetConfig.Bytes = nil

		Expect(ncl).To(Equal(&libcni.NetworkConfigList{
			Name:       "some-plugin",
			CNIVersion: "0.3.1",
			Plugins:    []*libcni.PluginConfig{testNetConfig},
		}))

		// Test that the json unmarshals to the same data
		ncl2, err := libcni.NetworkConfFromBytes(bytes)
		Expect(err).NotTo(HaveOccurred())
		ncl2.Bytes = nil
		ncl2.Plugins[0].Bytes = nil

		Expect(ncl2).To(Equal(ncl))
	})
})
