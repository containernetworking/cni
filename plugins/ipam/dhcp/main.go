// Copyright 2015 CoreOS, Inc.
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
	"fmt"
	"net/rpc"
	"os"

	"github.com/appc/cni/pkg/plugin"
	"github.com/appc/cni/pkg/skel"
)

const socketPath = "/run/cni/dhcp.sock"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		runDaemon()
	} else {
		skel.PluginMain(cmdAdd, cmdDel)
	}
}

func cmdAdd(args *skel.CmdArgs) error {
	client, err := rpc.DialHTTP("unix", socketPath)
	if err != nil {
		return fmt.Errorf("error dialing DHCP daemon: %v", err)
	}

	result := &plugin.Result{}
	err = client.Call("DHCP.Allocate", args, result)
	if err != nil {
		return fmt.Errorf("error calling DHCP.Add: %v", err)
	}

	return plugin.PrintResult(result)
}

func cmdDel(args *skel.CmdArgs) error {
	client, err := rpc.DialHTTP("unix", socketPath)
	if err != nil {
		return fmt.Errorf("error dialing DHCP daemon: %v", err)
	}

	dummy := struct{}{}
	err = client.Call("DHCP.Release", args, &dummy)
	if err != nil {
		return fmt.Errorf("error calling DHCP.Del: %v", err)
	}

	return nil
}
