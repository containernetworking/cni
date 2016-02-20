// Copyright 2015 CNI authors
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

package libcni

import (
	"strings"

	"github.com/appc/cni/pkg/invoke"
	"github.com/appc/cni/pkg/types"
)

// RuntimeConf is container runtime configuration for network add/del operation
type RuntimeConf struct {
	ContainerID string
	NetNS       string
	IfName      string
	Args        [][2]string
}

// NetworkConfig contains data for a network's add/del operation,
// and the data comes from netconf file.
type NetworkConfig struct {
	Network *types.NetConf
	Bytes   []byte
}

// CNI defines the operations that a CNI plugin needs to support
type CNI interface {
	AddNetwork(net *NetworkConfig, rt *RuntimeConf) (*types.Result, error)
	DelNetwork(net *NetworkConfig, rt *RuntimeConf) error
}

// CNIConfig is the implementation of CNI network operations
type CNIConfig struct {
	Path []string
}

// AddNetwork is the implementation of adding container to network
func (c *CNIConfig) AddNetwork(net *NetworkConfig, rt *RuntimeConf) (*types.Result, error) {
	pluginPath := invoke.FindInPath(net.Network.Type, c.Path)
	return invoke.ExecPluginWithResult(pluginPath, net.Bytes, c.args("ADD", rt))
}

// DelNetwork is the implementation of deleting container from network
func (c *CNIConfig) DelNetwork(net *NetworkConfig, rt *RuntimeConf) error {
	pluginPath := invoke.FindInPath(net.Network.Type, c.Path)
	return invoke.ExecPluginWithoutResult(pluginPath, net.Bytes, c.args("DEL", rt))
}

// =====
func (c *CNIConfig) args(action string, rt *RuntimeConf) *invoke.Args {
	return &invoke.Args{
		Command:     action,
		ContainerID: rt.ContainerID,
		NetNS:       rt.NetNS,
		PluginArgs:  rt.Args,
		IfName:      rt.IfName,
		Path:        strings.Join(c.Path, ":"),
	}
}
