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
	"github.com/containernetworking/cni/pkg/version"
)

// Args captures all the arguments passed in to the plugin
// via both env vars and stdin, except CNI_PATH, which should
// be handled differently in the beginning and should be used
// only by specific adapters like CliPlugin
type Args struct {
	ContainerID string
	Netns       string
	IfName      string
	Args        string
	StdinData   []byte
}

type Plugin interface {
	Add(args *Args) (types.Result, error)
	Check(args *Args) (types.Result, error)
	Del(args *Args) error
	Version() (version.PluginInfo, error)
}

type PluginManager interface {
	FindPlugin(pluginType string) (Plugin, error)
}
