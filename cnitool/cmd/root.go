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

package cmd

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/containernetworking/cni/libcni"
)

// Protocol parameters are passed to the plugins via OS environment variables.
const (
	EnvCNIPath        = "CNI_PATH"
	EnvNetDir         = "NETCONFPATH"
	EnvCapabilityArgs = "CAP_ARGS"
	EnvCNIArgs        = "CNI_ARGS"
	EnvCNIIfname      = "CNI_IFNAME"

	DefaultNetDir = "/etc/cni/net.d"
)

var (
	// Used for flags
	netName string
	netNS   string
	ifName  string

	rootCmd = &cobra.Command{
		Use:   "cnitool",
		Short: "CNI Tool for managing network interfaces in a network namespace",
		Long: `CNI Tool is a simple program that executes a CNI configuration.
It will add, check, remove, gc, or get status of an interface in an already-created network namespace.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&ifName, "ifname", "i", "", "Interface name (defaults to env var CNI_IFNAME or 'eth0')")
}

// parseArgs parses CNI_ARGS environment variable into key-value pairs
func parseArgs(args string) ([][2]string, error) {
	var result [][2]string

	pairs := strings.Split(args, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("invalid CNI_ARGS pair %q", pair)
		}

		result = append(result, [2]string{kv[0], kv[1]})
	}

	return result, nil
}

// setupRuntimeConfig prepares the runtime configuration for CNI operations
func setupRuntimeConfig(cmd *cobra.Command, args []string) (*libcni.NetworkConfigList, *libcni.RuntimeConf, error) {
	if len(args) < 2 {
		return nil, nil, fmt.Errorf("network name and namespace are required")
	}

	netName = args[0]
	netNS = args[1]

	// Get network configuration directory
	netdir := os.Getenv(EnvNetDir)
	if netdir == "" {
		netdir = DefaultNetDir
	}

	// Load network configuration
	netconf, err := libcni.LoadNetworkConf(netdir, netName)
	if err != nil {
		return nil, nil, err
	}

	// Parse capability arguments
	var capabilityArgs map[string]interface{}
	capabilityArgsValue := os.Getenv(EnvCapabilityArgs)
	if len(capabilityArgsValue) > 0 {
		if err = json.Unmarshal([]byte(capabilityArgsValue), &capabilityArgs); err != nil {
			return nil, nil, err
		}
	}

	// Parse CNI arguments
	var cniArgs [][2]string
	args_env := os.Getenv(EnvCNIArgs)
	if len(args_env) > 0 {
		cniArgs, err = parseArgs(args_env)
		if err != nil {
			return nil, nil, err
		}
	}

	// Get interface name from flag or environment variable
	if ifName == "" {
		ifName, _ = os.LookupEnv(EnvCNIIfname)
		if ifName == "" {
			ifName = "eth0"
		}
	}

	// Get absolute path of network namespace
	netNS, err = filepath.Abs(netNS)
	if err != nil {
		return nil, nil, err
	}

	// Generate the containerid by hashing the netns path
	s := sha512.Sum512([]byte(netNS))
	containerID := fmt.Sprintf("cnitool-%x", s[:10])

	// Create runtime configuration
	rt := &libcni.RuntimeConf{
		ContainerID:    containerID,
		NetNS:          netNS,
		IfName:         ifName,
		Args:           cniArgs,
		CapabilityArgs: capabilityArgs,
	}

	return netconf, rt, nil
}

// getCNIConfig returns a CNI configuration
func getCNIConfig() *libcni.CNIConfig {
	return libcni.NewCNIConfig(filepath.SplitList(os.Getenv(EnvCNIPath)), nil)
}
