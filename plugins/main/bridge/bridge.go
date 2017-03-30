// Copyright 2014 CNI authors
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
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/containernetworking/cni/pkg/bridge"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
)

const defaultBrName = "cni0"

type NetConf struct {
	types.NetConf
	BrName        string                 `json:"bridge"`
	IsGW          bool                   `json:"isGateway"`
	IsDefaultGW   bool                   `json:"isDefaultGateway"`
	ForceAddress  bool                   `json:"forceAddress"`
	IPMasq        bool                   `json:"ipMasq"`
	MTU           int                    `json:"mtu"`
	HairpinMode   bool                   `json:"hairpinMode"`
	RuntimeConfig map[string]interface{} `json:"runtimeConfig"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadNetConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{
		BrName: defaultBrName,
	}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	if n.IsDefaultGW {
		n.IsGW = true
	}

	_, _, err = bridge.Setup(n.BrName, n.MTU)
	if err != nil {
		return err
	}

	return types.PrintResult(&current.Result{}, n.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	_, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}
