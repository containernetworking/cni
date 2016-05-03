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
	"net"
	"strconv"
	"strings"
)

const (
	// private mac prefix safe to use
	privateMACPrefix = "0a:58"

	// veth link dev type
	vethLinkType = "veth"
)

func ExecAdd(plugin string, netconf []byte) (*types.Result, error) {
	return invoke.DelegateAdd(plugin, netconf)
}

func ExecDel(plugin string, netconf []byte) error {
	return invoke.DelegateDel(plugin, netconf)
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

	// only set hardware address to veth when using ipv4
	if link.Type() == vethLinkType && res.IP4 != nil {
		hwAddr, err := generateHardwareAddr(res.IP4.IP.IP)
		if err != nil {
			return fmt.Errorf("failed to generate hardware addr: %v", err)
		}
		if err = netlink.LinkSetHardwareAddr(link, hwAddr); err != nil {
			return fmt.Errorf("failed to add hardware addr to %q: %v", ifName, err)
		}
	}

	// TODO(eyakubovich): IPv6
	addr := &netlink.Addr{IPNet: &res.IP4.IP, Label: ""}
	if err = netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add IP addr to %q: %v", ifName, err)
	}

	for _, r := range res.IP4.Routes {
		gw := r.GW
		if gw == nil {
			gw = res.IP4.Gateway
		}
		if err = ip.AddRoute(&r.Dst, gw, link); err != nil {
			// we skip over duplicate routes as we assume the first one wins
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add route '%v via %v dev %v': %v", r.Dst, gw, ifName, err)
			}
		}
	}

	return nil
}

// generateHardwareAddr generates 48 bit virtual mac addresses based on the IP input.
func generateHardwareAddr(ip net.IP) (net.HardwareAddr, error) {
	if ip.To4() == nil {
		return nil, fmt.Errorf("generateHardwareAddr only support valid ipv4 address as input")
	}
	mac := privateMACPrefix
	sections := strings.Split(ip.String(), ".")
	for _, s := range sections {
		i, _ := strconv.Atoi(s)
		mac = mac + ":" + fmt.Sprintf("%02x", i)
	}
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse mac address %s generated based on ip %s due to: %v", mac, ip, err)
	}
	return hwAddr, nil
}
