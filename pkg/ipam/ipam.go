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

package ipam

import (
	"fmt"
	"os"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/types"

	"github.com/vishvananda/netlink"
)

func ExecAdd(plugin string, netconf []byte) (*types.Result, error) {
	return invoke.DelegateAdd(plugin, netconf)
}

func ExecDel(plugin string, netconf []byte) error {
	return invoke.DelegateDel(plugin, netconf)
}

func addIPConfigToLink(ipConfig *types.IPConfig, link netlink.Link, ifName string) error {
	addr := &netlink.Addr{IPNet: &ipConfig.IP, Label: ""}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add addr to %q: %v", ifName, err)
	}

	for _, r := range ipConfig.Routes {
		gw := r.GW
		if gw == nil {
			gw = ipConfig.Gateway
		}
		if err := ip.AddRoute(&r.Dst, gw, link); err != nil {
			// we skip over duplicate routes as we assume the first one wins
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add route '%v via %v dev %v': %v", r.Dst, gw, ifName, err)
			}
		}
	}

	return nil
}

// ConfigureIface takes the result of IPAM plugin and
// applies to the ifName interface
func ConfigureIface(ifName string, res *types.Result) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set %q UP: %v", ifName, err)
	}

	if res.IP4 != nil {
		if err := addIPConfigToLink(res.IP4, link, ifName); err != nil {
			return fmt.Errorf("error configuring IP4: %s", err.Error())
		}
	}
	if res.IP6 != nil {
		if err := addIPConfigToLink(res.IP6, link, ifName); err != nil {
			return fmt.Errorf("error configuring IP6: %s", err.Error())
		}
	}

	return nil
}
