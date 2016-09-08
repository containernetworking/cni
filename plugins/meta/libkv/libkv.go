// Copyright 2016 Roman Mohr <rmohr@redhat.com>
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

// This is a "meta-plugin". According to its config, it loads a JSON array
// of types.NetConf from the specified key/value store. Then it delegates
// one loaded NetConf after the other to the specified plugin.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/docker/libkv/store/etcd"
	"github.com/docker/libkv/store/zookeeper"
	"log"
	"strings"
	"time"
)

const (
	stateDir = "/var/run/cni/libkv"
)

type LibKvConf struct {
	StoreBackend store.Backend `json:"storeBackend"`
	Uri          string        `json:"uri"`
	BasePath     string        `json:"basePath,omitempty"`
	StoreConfig  *StoreConfig  `json:"storeConfig,omitempty"`
}

type StoreConfig struct {
	// TODO ClientTLS         *ClientTLSConfig
	// TODO TLS               *tls.Config
	// TODO needed? Bucket            string
	// TODO needed? PersistConnection bool
	ConnectionTimeout uint   `json:"connectionTimeout,omitempty"`
	Username          string `json:"username,omitempty"`
	Password          string `json:"password,omitempty"`
}

func cmdAdd(args *skel.CmdArgs) error {
	config, err := loadPluginConfig(args.StdinData)
	if err != nil {
		return err
	}

	// Populate store config
	storeConfig := &store.Config{}
	if config.StoreConfig.ConnectionTimeout != 0 {
		storeConfig.ConnectionTimeout = time.Duration(config.StoreConfig.ConnectionTimeout) * time.Second
	}
	if config.StoreConfig.Username != "" {
		storeConfig.Username = config.StoreConfig.Username
	}
	if config.StoreConfig.Password != "" {
		storeConfig.Password = config.StoreConfig.Password
	}

	// Initialize a new store with the provided properties
	kv, err := libkv.NewStore(
		config.StoreBackend,
		[]string{config.Uri},
		storeConfig,
	)
	if err != nil {
		log.Fatal("Cannot create %s store", config.StoreBackend)
	}
	key := config.BasePath + args.ContainerID
	pair, err := kv.Get(key)
	if err != nil {
		return fmt.Errorf("Error trying accessing value at key: %v", key)
	}

	var netconfs []map[string]interface{}
	if err = json.Unmarshal(pair.Value, &netconfs); err != nil {
		return fmt.Errorf("Could not unmarshal store value: %v", err)
	}

	if err = saveScratchNetConf(args.ContainerID, pair.Value); err != nil {
		return fmt.Errorf("Could not save generated cni configs: %v", err)
	}

	var result *types.Result

	for index, conf := range netconfs {
		confBytes, err := json.Marshal(conf)
		// If not the first interface, we need to get the right interface name
		if index > 0 && conf["type"] != "loopback" {
			os.Setenv("CNI_IFNAME", conf["args"].(map[string]interface{})["ifName"].(string))
		}
		if err != nil {
			return fmt.Errorf("Could not marshal subconfig at index %d: %v", index, err)
		}
		res, err := invoke.DelegateAdd(conf["type"].(string), confBytes)
		if err != nil {
			return err
		}
		// The first configuration in the array is the management interface
		if index == 0 {
			result = res
		}
	}

	return result.Print()
}

func loadPluginConfig(bytes []byte) (*LibKvConf, error) {
	config := &LibKvConf{
		BasePath:     "/",
		StoreConfig: &StoreConfig{},
	}
	if err := json.Unmarshal(bytes, config); err != nil {
		return nil, fmt.Errorf("failed to load libkv config: %v", err)
	}
	if !strings.HasSuffix(config.BasePath, "/") {
		config.BasePath += "/"
	}
	// TODO Config validation
	return config, nil
}

func saveScratchNetConf(containerID string, netconf []byte) error {
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}
	path := filepath.Join(stateDir, containerID)
	return ioutil.WriteFile(path, netconf, 0600)
}

func consumeScratchNetConf(containerID string) ([]byte, error) {
	path := filepath.Join(stateDir, containerID)
	defer os.Remove(path)

	return ioutil.ReadFile(path)
}

func init() {
	// TODO: Only load store when it is really needed?
	consul.Register()
	etcd.Register()
	zookeeper.Register()
}

func cmdDel(args *skel.CmdArgs) error {
	netconfBytes, err := consumeScratchNetConf(args.ContainerID)
	if err != nil {
		return err
	}

	var netconfs []map[string]interface{}
	if err = json.Unmarshal(netconfBytes, &netconfs); err != nil {
		return fmt.Errorf("failed to parse netconf: %v", err)
	}

	for index, conf := range netconfs {
		// If not the first interface, we need to get the right interface name
		if index > 0 && conf["type"] != "loopback" {
			os.Setenv("CNI_IFNAME", conf["args"].(map[string]interface{})["ifName"].(string))
		}
		confBytes, err := json.Marshal(conf)
		if err != nil {
			return fmt.Errorf("Could not marshal subconfig at index %d: %v", index, err)
		}
		if err = invoke.DelegateDel(conf["type"].(string), confBytes); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
