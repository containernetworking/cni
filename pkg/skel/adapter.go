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

// Adapter semantically is PluginManager interface, wrapped in
// structure type in order to define useful extension methods

type Adapter struct {
	PluginManager
}

func NewAdapter(mgr PluginManager) Adapter {
	return Adapter{PluginManager: mgr}
}

func (a Adapter) Add(pluginType string, args *Args) (types.Result, error) {
	plugin, err := a.FindPlugin(pluginType)
	if err != nil {
		return nil, err
	}
	return plugin.Add(args)
}

func (a Adapter) Check(pluginType string, args *Args) (types.Result, error) {
	plugin, err := a.FindPlugin(pluginType)
	if err != nil {
		return nil, err
	}
	return plugin.Check(args)
}

func (a Adapter) Del(pluginType string, args *Args) error {
	plugin, err := a.FindPlugin(pluginType)
	if err != nil {
		return err
	}
	return plugin.Del(args)
}

func (a Adapter) Version(pluginType string) (version.PluginInfo, error) {
	plugin, err := a.FindPlugin(pluginType)
	if err != nil {
		return nil, err
	}
	return plugin.Version()
}
