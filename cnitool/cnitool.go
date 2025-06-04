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
)

// Protocol parameters are passed to the plugins via OS environment variables.
const (
	EnvCNIPath        = "CNI_PATH"
	EnvNetDir         = "NETCONFPATH"
	EnvCapabilityArgs = "CAP_ARGS"
	EnvCNIArgs        = "CNI_ARGS"
	EnvCNIIfname      = "CNI_IFNAME"

	CmdAdd    = "add"
	CmdCheck  = "check"
	CmdDel    = "del"
	CmdGC     = "gc"
	CmdStatus = "status"
)

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

func main() {
	if len(os.Args) < 4 {
		usage()
	}

	netdir := os.Getenv(EnvNetDir)
	if netdir == "" {
		netdir = DefaultNetDir
	}

	if !filepath.IsAbs(netdir) {
		var err error
		netdir, err = filepath.Abs(netdir)
		if err != nil {
			exit(fmt.Errorf("error converting the provided CNI config path ($%s) %q to an absolute path: %w", EnvNetDir, netdir, err))
		}
	}

	if stat, err := os.Stat(netdir); err == nil {
		if !stat.IsDir() {
			exit(fmt.Errorf("the provided CNI config path ($%s) is not a directory: %q", EnvNetDir, netdir))
		}
	} else {
		exit(fmt.Errorf("the provided CNI config path ($%s) does not exist: %q", EnvNetDir, netdir))
	}

	netconf, err := libcni.LoadNetworkConf(netdir, os.Args[2])
	if err != nil {
		exit(err)
	}

	var capabilityArgs map[string]interface{}
	capabilityArgsValue := os.Getenv(EnvCapabilityArgs)
	if len(capabilityArgsValue) > 0 {
		if err = json.Unmarshal([]byte(capabilityArgsValue), &capabilityArgs); err != nil {
			exit(err)
		}
	}

	var cniArgs [][2]string
	args := os.Getenv(EnvCNIArgs)
	if len(args) > 0 {
		cniArgs, err = parseArgs(args)
		if err != nil {
			exit(err)
		}
	}

	ifName, ok := os.LookupEnv(EnvCNIIfname)
	if !ok {
		ifName = "eth0"
	}

	netns := os.Args[3]
	netns, err = filepath.Abs(netns)
	if err != nil {
		exit(err)
	}

	// Generate the containerid by hashing the netns path
	s := sha512.Sum512([]byte(netns))
	containerID := fmt.Sprintf("cnitool-%x", s[:10])

	cninet := libcni.NewCNIConfig(filepath.SplitList(os.Getenv(EnvCNIPath)), nil)

	rt := &libcni.RuntimeConf{
		ContainerID:    containerID,
		NetNS:          netns,
		IfName:         ifName,
		Args:           cniArgs,
		CapabilityArgs: capabilityArgs,
	}

	switch os.Args[1] {
	case CmdAdd:
		result, err := cninet.AddNetworkList(context.TODO(), netconf, rt)
		if result != nil {
			_ = result.Print()
		}
		exit(err)
	case CmdCheck:
		err := cninet.CheckNetworkList(context.TODO(), netconf, rt)
		exit(err)
	case CmdDel:
		exit(cninet.DelNetworkList(context.TODO(), netconf, rt))
	case CmdGC:
		// Currently just invoke GC without args, hence all network interface should be GC'ed!
		exit(cninet.GCNetworkList(context.TODO(), netconf, nil))
	case CmdStatus:
		exit(cninet.GetStatusNetworkList(context.TODO(), netconf))
	}
}

func usage() {
	exe := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, "%s: Add, check, remove, gc or status network interfaces from a network namespace\n", exe)
	fmt.Fprintf(os.Stderr, "  %s add    <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s check  <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s del    <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s gc     <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s status <net> <netns>\n", exe)
	os.Exit(1)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
