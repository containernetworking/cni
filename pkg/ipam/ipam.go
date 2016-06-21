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
	PrivateMACPrefix = "0a:58"

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
		hwAddr, err := GenerateHardwareAddr4(res.IP4.IP.IP, PrivateMACPrefix)
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

type SupportIp4OnlyErr struct{ msg string }

func (e SupportIp4OnlyErr) Error() string { return e.msg }

type MacParseErr struct{ msg string }

func (e MacParseErr) Error() string { return e.msg }

type InvalidPrefixLengthErr struct{ msg string }

func (e InvalidPrefixLengthErr) Error() string { return e.msg }

// GenerateHardwareAddr4 generates 48 bit virtual mac addresses based on the IP4 input.
func GenerateHardwareAddr4(ip net.IP, prefix string) (net.HardwareAddr, error) {
	switch {

	case ip.To4() == nil:
		return nil, SupportIp4OnlyErr{msg: "GenerateHardwareAddr4 only supports valid IPv4 address as input"}

	case len(prefix) != len(PrivateMACPrefix):
		return nil, InvalidPrefixLengthErr{msg: fmt.Sprintf(
			"Prefix has length %d instead  of %d", len(prefix), len(PrivateMACPrefix)),
		}
	}

	mac := prefix
	sections := strings.Split(ip.String(), ".")
	for _, s := range sections {
		i, _ := strconv.Atoi(s)
		mac = mac + ":" + fmt.Sprintf("%02x", i)
	}

	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, MacParseErr{msg: fmt.Sprintf(
			"Failed to parse mac address %q generated based on IP %q due to: %v", mac, ip, err),
		}
	}
	return hwAddr, nil
}
