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

package invoke

import (
	"os"
	"strings"
)

// CNIArgs represents the arguments to be passed to the plugin via the process environment.
// Sometimes these must be assembled from configuration.
// Sometimes (when calling a plugin recursively) they must be inherited from the calling process.
type CNIArgs interface {
	// For use with os/exec; i.e., return nil to inherit the
	// environment from this process
	AsEnv() []string
}

type inherited struct{}

var inheritArgsFromEnv inherited

func (i *inherited) AsEnv() []string {
	// It stands in for "just use the calling process's environment"
	return nil
}

// ArgsFromEnv returns CNIArgs by inheriting environment variables
func ArgsFromEnv() CNIArgs {
	// Now no environment variables inherited
	return &inheritArgsFromEnv
}

// Args defines the contents of CNIArgs
type Args struct {
	Command       string
	ContainerID   string
	NetNS         string
	PluginArgs    [][2]string
	PluginArgsStr string
	IfName        string
	Path          string
}

// AsEnv returns args serialised as an array of environment variables
func (args *Args) AsEnv() []string {
	env := os.Environ()
	pluginArgsStr := args.PluginArgsStr
	if pluginArgsStr == "" {
		pluginArgsStr = stringify(args.PluginArgs)
	}

	env = append(env,
		"CNI_COMMAND="+args.Command,
		"CNI_CONTAINERID="+args.ContainerID,
		"CNI_NETNS="+args.NetNS,
		"CNI_ARGS="+pluginArgsStr,
		"CNI_IFNAME="+args.IfName,
		"CNI_PATH="+args.Path)
	return env
}

// taken from rkt/networking/net_plugin.go
func stringify(pluginArgs [][2]string) string {
	entries := make([]string, len(pluginArgs))

	for i, kv := range pluginArgs {
		entries[i] = strings.Join(kv[:], "=")
	}

	return strings.Join(entries, ";")
}
