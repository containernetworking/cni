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

package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Protocol parameters are passed to the plugins via OS environment variables.
const (
	VarCNIArgs          = "CNI_ARGS"
	VarCNICommand       = "CNI_COMMAND"
	VarCNIContainerId   = "CNI_CONTAINERID"
	VarCNIIfname        = "CNI_IFNAME"
	VarCNINetNs         = "CNI_NETNS"
	VarCNINetNsOverride = "CNI_NETNS_OVERRIDE"
	VarCNIPath          = "CNI_PATH"

	DefaultCNIArgs = ""
	DefaultCNIPath = ""

	// supported CNI_COMMAND values

	CmdAdd     = "ADD"
	CmdCheck   = "CHECK"
	CmdDel     = "DEL"
	CmdVersion = "VERSION"
)

// GetValue is a wrapper around os.GetEnv, which
// returns the given fallback value in case the
// environment variable is not set or an empty
// string.
func GetValue(key, fallbackValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallbackValue
	}

	return val
}

// GetCNIArgs gets the plugin arguments
// from the environment
func GetCNIArgs() string {
	return GetValue(VarCNIArgs, DefaultCNIArgs)
}

// GetCNIPath gets the plugin lookup path
// from the environment
func GetCNIPath() string {
	return GetValue(VarCNIPath, DefaultCNIPath)
}

// ParseCNIArgs returns a list of tuples
// representing extra arguments
func ParseCNIArgs() ([][2]string, error) {
	pairs := strings.Split(GetCNIArgs(), ";")
	result := make([][2]string, len(pairs))

	for i, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("invalid CNI_ARGS pair %q", pair)
		}

		result[i] = [2]string{kv[0], kv[1]}
	}

	return result, nil
}

// ParseCNIPath returns a list of directories to
// search for executables
func ParseCNIPath() []string {
	return filepath.SplitList(GetCNIPath())
}
