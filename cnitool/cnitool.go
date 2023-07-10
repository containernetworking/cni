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

package main

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/env"
)

// Protocol parameters are passed to the plugins via OS environment variables.
const (
	EnvNetDir         = "NETCONFPATH"
	EnvCapabilityArgs = "CAP_ARGS"

	DefaultIfname       = "eth0"
	DefaultNetDir       = "/etc/cni/net.d"
	DefaultCapabilities = ""
)

func main() {
	if len(os.Args) < 4 {
		usage()
	}

	netdir := env.GetValue(EnvNetDir, DefaultNetDir)
	netconf, err := libcni.LoadConfList(netdir, os.Args[2])
	if err != nil {
		exit(err)
	}

	var capabilityArgs map[string]interface{}
	capabilityArgsValue := env.GetValue(EnvCapabilityArgs, DefaultCapabilities)
	if len(capabilityArgsValue) > 0 {
		if err = json.Unmarshal([]byte(capabilityArgsValue), &capabilityArgs); err != nil {
			exit(err)
		}
	}

	cniArgs, err := env.ParseCNIArgs()
	if err != nil {
		exit(err)
	}

	ifName := env.GetValue(env.VarCNIIfname, DefaultIfname)
	netns := os.Args[3]
	netns, err = filepath.Abs(netns)
	if err != nil {
		exit(err)
	}

	// Generate the containerid by hashing the netns path
	s := sha512.Sum512([]byte(netns))
	containerID := fmt.Sprintf("cnitool-%x", s[:10])

	cninet := libcni.NewCNIConfig(env.ParseCNIPath(), nil)

	rt := &libcni.RuntimeConf{
		ContainerID:    containerID,
		NetNS:          netns,
		IfName:         ifName,
		Args:           cniArgs,
		CapabilityArgs: capabilityArgs,
	}

	switch strings.ToUpper(os.Args[1]) {
	case env.CmdAdd:
		result, err := cninet.AddNetworkList(context.TODO(), netconf, rt)
		if result != nil {
			_ = result.Print()
		}
		exit(err)
	case env.CmdCheck:
		err := cninet.CheckNetworkList(context.TODO(), netconf, rt)
		exit(err)
	case env.CmdDel:
		exit(cninet.DelNetworkList(context.TODO(), netconf, rt))
	}
}

func usage() {
	exe := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, "%s: Add, check, or remove network interfaces from a network namespace\n", exe)
	fmt.Fprintf(os.Stderr, "  %s add   <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s check <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s del   <net> <netns>\n", exe)
	os.Exit(1)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
