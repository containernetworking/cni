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

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
)

type DirectPluginManager struct {
	Plugins map[string]Plugin
}

// DirectPluginManager implements PluginManager interface
var _ PluginManager = &DirectPluginManager{}

type DirectPlugin struct {
	AddFunc     func(args *Args) (types.Result, error)
	CheckFunc   func(args *Args) (types.Result, error)
	DelFunc     func(args *Args) error
	VersionFunc func() (version.PluginInfo, error)
}

// DirectPlugin implements PluginManager interface
var _ Plugin = &DirectPlugin{}

func NewDirectAdapter(opts ...func(*DirectPluginManager)) Adapter {
	return NewAdapter(NewDirectPluginManager(opts...))
}

func NewDirectPluginManager(opts ...func(*DirectPluginManager)) *DirectPluginManager {
	mgr := &DirectPluginManager{
		Plugins: make(map[string]Plugin),
	}
	for _, opt := range opts {
		opt(mgr)
	}
	return mgr
}

func NewDirectPlugin(opts ...func(*DirectPlugin)) *DirectPlugin {
	plugin := &DirectPlugin{}
	for _, opt := range opts {
		opt(plugin)
	}
	return plugin
}

//
// DirectPluginManager implementation
//

func (p *DirectPluginManager) FindPlugin(pluginType string) (Plugin, error) {
	plugin, ok := p.Plugins[pluginType]
	if !ok {
		return nil, errors.New(fmt.Sprintf("plugin %q not found", pluginType))
	}
	return plugin, nil
}

//
// DirectPlugin implementation
//

func (p *DirectPlugin) Add(args *Args) (types.Result, error) {
	if p.AddFunc == nil {
		return nil, errors.New("not implemented")
	}
	return p.AddFunc(args)
}

func (p *DirectPlugin) Del(args *Args) error {
	if p.DelFunc == nil {
		return errors.New("not implemented")
	}
	return p.DelFunc(args)
}

func (p *DirectPlugin) Check(args *Args) (types.Result, error) {
	if p.CheckFunc == nil {
		return nil, errors.New("not implemented")
	}
	return p.CheckFunc(args)
}

func (p *DirectPlugin) Version() (version.PluginInfo, error) {
	if p.VersionFunc == nil {
		return nil, errors.New("not implemented")
	}
	return p.VersionFunc()
}
