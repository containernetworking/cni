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
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
)

// Exec interface implementation for PluginManager
// Allows to wrap PluginManager with Exec interface

type PluginExec struct {
	manger PluginManager
	*version.PluginDecoder
}

// PluginExec implements Exec interface
var _ invoke.Exec = PluginExec{}

func NewPluginExec(manger PluginManager) invoke.Exec {
	return PluginExec{manger: manger}
}

//
//  Exec interface implementation
//

func (e PluginExec) ExecPlugin(ctx context.Context, pluginPath string, stdinData []byte, environ []string) ([]byte, error) {
	plugin, err := e.manger.FindPlugin(pluginPath)
	if err != nil {
		return nil, err
	}
	env := envMap(environ)

	args := Args{
		ContainerID: env["CNI_CONTAINERID"],
		Netns:       env["CNI_NETNS"],
		Args:        env["CNI_ARGS"],
		IfName:      env["CNI_IFNAME"],
		StdinData:   stdinData,
	}

	switch command := env["CNI_COMMAND"]; command {
	case CmdAdd:
		res, err := plugin.Add(&args)
		if err != nil {
			return nil, err
		}
		return marshalResult(res)
	case CmdCheck:
		res, err := plugin.Check(&args)
		if err != nil {
			return nil, err
		}
		return marshalResult(res)
	case CmdDel:
		err := plugin.Del(&args)
		if err != nil {
			return nil, err
		}
		return nil, nil
	case CmdVersion:
		pluginInfo, err := plugin.Version()
		if err != nil {
			return nil, err
		}
		return marshalVersion(pluginInfo)
	default:
		return nil, errors.New(fmt.Sprintf(
			"unexpected command: %q", command,
		))
	}
}

func (e PluginExec) FindInPath(pluginType string, paths []string) (string, error) {

	_, err := e.manger.FindPlugin(pluginType)
	if err == nil {
		return pluginType, nil
	}

	for _, p := range paths {
		pluginPath := path.Join(p, pluginType)
		_, err := e.manger.FindPlugin(pluginPath)
		if err == nil {
			return pluginPath, nil
		}
	}

	return "", fmt.Errorf("failed to find plugin %q in path %s", pluginType, paths)
}

//
//  auxiliary functions
//

func marshalResult(res types.Result) ([]byte, error) {
	b := &bytes.Buffer{}
	if err := res.PrintTo(b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func marshalVersion(pluginInfo version.PluginInfo) ([]byte, error) {
	b := &bytes.Buffer{}
	if err := pluginInfo.Encode(b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func envMap(env []string) map[string]string {
	res := make(map[string]string)
	for _, x := range env {
		i := strings.Index(x, "=")
		if i >= 0 {
			res[x[:i]] = x[i+1:]
		} else {
			res[x] = ""
		}
	}
	return res
}
