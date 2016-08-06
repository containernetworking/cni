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
	"fmt"

	"github.com/containernetworking/cni/plugins/ipam/host-local/backend/disk"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
)

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}

func cmdAdd(args *skel.CmdArgs) error {
	ipamConf, ipamConf6, err := LoadIPAMConfig(args.StdinData, args.Args)
	if err != nil {
		return err
	}

	r := &types.Result{
		IP4: nil,
		IP6: nil,
	}

	if ipamConf != nil {
		store, err := disk.New(ipamConf.Name)
		if err != nil {
			return err
		}
		defer store.Close()

		allocator, err := NewIPAllocator(ipamConf, store)
		if err != nil {
			return err
		}

		ipConf, err := allocator.Get(args.ContainerID)
		if err != nil {
			return err
		}
		r.IP4 = ipConf
	}

	if ipamConf6 != nil {
		store, err := disk.New(ipamConf6.Name)
		if err != nil {
			return err
		}
		defer store.Close()

		allocator, err := NewIPAllocator(ipamConf6, store)
		if err != nil {
			return err
		}
		ipConf6, err := allocator.Get(args.ContainerID)
		if err != nil {
			return err
		}
		r.IP6 = ipConf6
	}

	return r.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	var result [2]error
	ipamConf, ipamConf6, err := LoadIPAMConfig(args.StdinData, args.Args)
	if err != nil {
		return err
	}

	if ipamConf != nil {
		store, err := disk.New(ipamConf.Name)
		if err != nil {
			return err
		}
		defer store.Close()

		allocator, err := NewIPAllocator(ipamConf, store)
		if err != nil {
			return err
		}

		e := allocator.Release(args.ContainerID)
		if e != nil {
			result[0] = e
		}
	}

	if ipamConf6 != nil {
		store, err := disk.New(ipamConf6.Name)
		if err != nil {
			return err
		}
		allocator, err := NewIPAllocator(ipamConf6, store)
		if err != nil {
			return err
		}
		e := allocator.Release(args.ContainerID)
		if e != nil {
			result[1] = e
		}
	}
	if result[0] != nil {
		if result[1] != nil {
			return fmt.Errorf("%v, %v", result[0], result[1])
		}
	}
	return result[0]
}
