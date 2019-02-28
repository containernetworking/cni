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
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/pkg/version"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
)

// CNI commands
const (
	CmdAdd     = "ADD"
	CmdDel     = "DEL"
	CmdCheck   = "CHECK"
	CmdVersion = "VERSION"
)

// CliPlugin exposes Plugin interface
// and intended to forward invocations to specified
// cni executable plugin. ExecPath member points
// to this executable.
type CliPlugin struct {
	ExecPath string
	Exec     invoke.Exec
	Path     string
}

// CliPlugin implements Plugin interface
var _ Plugin = &CliPlugin{}

// CliResolver exposes Resolver interface
// and intended to lookup for available cni executable
// plugins. Paths member specify directories where to
// look for plugin executables.
type CliPluginManager struct {
	// paths used to look for plugin executables
	Paths []string
	// plugin executor
	Exec invoke.Exec
}

// CliPluginManager implements PluginManager interface
var _ PluginManager = &CliPluginManager{}

func CliPluginMain(plugin Plugin, about string) {
	if e := CliPluginMainWithError(plugin, about); e != nil {
		if err := e.Print(); err != nil {
			log.Print("Error writing error JSON to stdout: ", err)
		}
		os.Exit(1)
	}
}

func CliPluginMainWithError(plugin Plugin, about string) *types.Error {
	versionInfo, err := plugin.Version()
	if err != nil {
		return createTypedError(err.Error())
	}
	return PluginMainWithError(
		printResult(plugin.Add),
		printResult(plugin.Check),
		noResult(plugin.Del),
		versionInfo,
		about,
	)
}
func NewCliAdapter(opts ...func(*CliPluginManager)) Adapter {
	return NewAdapter(NewCliPluginManager(opts...))
}

func NewCliPluginManager(opts ...func(*CliPluginManager)) *CliPluginManager {
	mgr := &CliPluginManager{}
	for _, opt := range opts {
		opt(mgr)
	}
	if mgr.Paths == nil {
		mgr.Paths = filepath.SplitList(os.Getenv("CNI_PATH"))
	}
	if mgr.Exec == nil {
		mgr.Exec = &invoke.DefaultExec{
			RawExec: &invoke.RawExec{Stderr: os.Stderr},
		}
	}
	return mgr
}

//
// PluginManager interface implementation for CliPluginManager
//

func (mgr *CliPluginManager) FindPlugin(pluginType string) (Plugin, error) {
	execPath, err := mgr.Exec.FindInPath(pluginType, mgr.Paths)
	if err != nil {
		return nil, err
	}
	return &CliPlugin{
		ExecPath: execPath,
		Exec:     mgr.Exec,
		Path:     strings.Join(mgr.Paths, string(filepath.ListSeparator)),
	}, nil
}

//
// Plugin interface implementation for CliPlugin
//

func (p *CliPlugin) Add(args *Args) (types.Result, error) {
	return p.execWithResult(CmdAdd, toCmdArgs(args))
}

func (p *CliPlugin) Del(args *Args) error {
	return p.execWithoutResult(CmdDel, toCmdArgs(args))
}

func (p *CliPlugin) Check(args *Args) (types.Result, error) {
	return p.execWithResult(CmdCheck, toCmdArgs(args))
}

func (p *CliPlugin) Version() (version.PluginInfo, error) {
	stdoutBytes, err := p.Exec.ExecPlugin(
		context.TODO(), p.ExecPath, nil, p.invokeArgs(CmdVersion, nil).AsEnv(),
	)
	if err != nil {
		return nil, err
	}
	return (&version.PluginDecoder{}).Decode(stdoutBytes)
}

func (p *CliPlugin) execWithResult(command string, args *CmdArgs) (types.Result, error) {
	return invoke.ExecPluginWithResult(
		context.TODO(), p.ExecPath, args.StdinData, p.invokeArgs(command, args), p.Exec,
	)
}

func (p *CliPlugin) execWithoutResult(command string, args *CmdArgs) error {
	return invoke.ExecPluginWithoutResult(
		context.TODO(), p.ExecPath, args.StdinData, p.invokeArgs(command, args), p.Exec,
	)
}

func (p *CliPlugin) invokeArgs(command string, args *CmdArgs) *invoke.Args {
	res := &invoke.Args{
		Command: command,
		Path:    p.Path,
	}
	if args != nil {
		res.ContainerID = args.ContainerID
		res.NetNS = args.Netns
		res.PluginArgsStr = args.Args
		res.IfName = args.IfName
	}
	return res
}

//
// auxiliary functions
//

func toCmdArgs(args *Args) *CmdArgs {
	if args == nil {
		return nil
	}
	return &CmdArgs{
		ContainerID: args.ContainerID,
		Netns:       args.Netns,
		IfName:      args.IfName,
		Args:        args.Args,
		StdinData:   args.StdinData,
	}
}

func fromCmdArgs(args *CmdArgs) *Args {
	if args == nil {
		return nil
	}
	return &Args{
		ContainerID: args.ContainerID,
		Netns:       args.Netns,
		IfName:      args.IfName,
		Args:        args.Args,
		StdinData:   args.StdinData,
	}
}

func printResult(f func(args *Args) (types.Result, error)) func(args *CmdArgs) error {
	return func(cmdArgs *CmdArgs) error {
		res, err := f(fromCmdArgs(cmdArgs))
		if err != nil {
			return err
		}
		return res.Print()
	}
}

func noResult(f func(args *Args) error) func(args *CmdArgs) error {
	return func(cmdArgs *CmdArgs) error {
		return f(fromCmdArgs(cmdArgs))
	}
}
