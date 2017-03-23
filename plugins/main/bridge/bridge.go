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
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"

	"io/ioutil"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/utils"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/vishvananda/netlink"
)

const defaultBrName = "cni0"

type NetConf struct {
	types.NetConf
	BrName       string `json:"bridge"`
	IsGW         bool   `json:"isGateway"`
	IsDefaultGW  bool   `json:"isDefaultGateway"`
	ForceAddress bool   `json:"forceAddress"`
	IPMasq       bool   `json:"ipMasq"`
	MTU          int    `json:"mtu"`
	HairpinMode  bool   `json:"hairpinMode"`
}

type addrGWs struct {
	addr   net.IP
	gws    []net.IPNet
	family int
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

// firstAddrGWs processes the results from the IPAM plugin, and
// determines the following:
//    - The first IPv4 and IPv6 container addresses. These addresses are
//      used to determine the hardware address for the container interface
//    - Default route(s)
//    - Gateway address(es)
func firstAddrGWs(result *current.Result, n *NetConf) (*addrGWs,
	*addrGWs, error) {

	aGWV4 := &addrGWs{}
	aGWV6 := &addrGWs{}

	for _, ipc := range result.IPs {

		// Determine if this config is IPv4 or IPv6
		var aGW *addrGWs
		defaultNet := &net.IPNet{}
		switch {
		case ipc.Address.IP.To4() != nil:
			aGW = aGWV4
			aGW.family = netlink.FAMILY_V4
			defaultNet.IP = net.IPv4zero
		case len(ipc.Address.IP) == net.IPv6len:
			aGW = aGWV6
			aGW.family = netlink.FAMILY_V6
			defaultNet.IP = net.IPv6zero
		default:
			return nil, nil, fmt.Errorf("Unknown IP object: %v", ipc)
		}

		// If this is the first configured address for this family, save it.
		// This will be used to generate a MAC address.
		if aGW.addr == nil {
			aGW.addr = ipc.Address.IP
		}

		// Calculate gateway address corresponding to the selected IP address
		ipc.Interface = 2
		if ipc.Gateway == nil && n.IsGW {
			ipc.Gateway = calcGatewayIP(&ipc.Address)
		}

		// Add the default route for this family if requested
		if n.IsDefaultGW {
			defaultNet.Mask = net.IPMask(defaultNet.IP)
			defaultRouteFound := false
			for _, route := range result.Routes {
				if route.GW != nil && defaultNet.String() == route.Dst.String() {
					defaultRouteFound = true
				}
			}
			if !defaultRouteFound {
				result.Routes = append(
					result.Routes,
					&types.Route{Dst: *defaultNet, GW: ipc.Gateway},
				)
			}
		}

		// Add the gateway address if requested
		if n.IsGW {
			gw := net.IPNet{
				IP:   ipc.Gateway,
				Mask: ipc.Address.Mask,
			}
			aGW.gws = append(aGW.gws, gw)
		}
	}
	return aGWV4, aGWV6, nil
}

func firstGW(aGW *addrGWs) net.IP {
	if aGW.gws != nil {
		return aGW.gws[0].IP
	}
	return nil
}

func ensureBridgeAddr(br *netlink.Bridge, family int, ipn *net.IPNet, forceAddress bool) error {
	addrs, err := netlink.AddrList(br, family)
	if err != nil && err != syscall.ENOENT {
		return fmt.Errorf("could not get list of IP addresses: %v", err)
	}

	// if there're no addresses on the bridge, it's ok -- we'll add one
	if len(addrs) > 0 {
		ipnStr := ipn.String()
		for _, a := range addrs {

			// Ignore IPv6 link local addresses
			if family == netlink.FAMILY_V6 && a.IP.IsLinkLocalUnicast() {
				continue
			}

			// string comp is actually easiest for doing IPNet comps
			if a.IPNet.String() == ipnStr {
				return nil
			}

			// If forceAddress is set to true then reconfigure IP address otherwise throw error
			if forceAddress {
				if err = deleteBridgeAddr(br, a.IPNet); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("%q already has an IP address different from %v", br.Name, ipnStr)
			}
		}
	}

	addr := &netlink.Addr{IPNet: ipn, Label: ""}
	if err := netlink.AddrAdd(br, addr); err != nil {
		return fmt.Errorf("could not add IP address to %q: %v", br.Name, err)
	}
	return nil
}

func deleteBridgeAddr(br *netlink.Bridge, ipn *net.IPNet) error {
	addr := &netlink.Addr{IPNet: ipn, Label: ""}

	if err := netlink.AddrDel(br, addr); err != nil {
		return fmt.Errorf("could not remove IP address from %q: %v", br.Name, err)
	}

	return nil
}

func bridgeByName(name string) (*netlink.Bridge, error) {
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup %q: %v", name, err)
	}
	br, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%q already exists but is not a bridge", name)
	}
	return br, nil
}

func ensureBridge(brName string, mtu int) (*netlink.Bridge, error) {
	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: brName,
			MTU:  mtu,
			// Let kernel use default txqueuelen; leaving it unset
			// means 0, and a zero-length TX queue messes up FIFO
			// traffic shapers which use TX queue length as the
			// default packet limit
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(br)
	if err != nil && err != syscall.EEXIST {
		return nil, fmt.Errorf("could not add %q: %v", brName, err)
	}

	// Re-fetch link to read all attributes and if it already existed,
	// ensure it's really a bridge with similar configuration
	br, err = bridgeByName(brName)
	if err != nil {
		return nil, err
	}

	if err := netlink.LinkSetUp(br); err != nil {
		return nil, err
	}

	return br, nil
}

func setupVeth(netns ns.NetNS, br *netlink.Bridge, ifName string, mtu int, hairpinMode bool) (*current.Interface, *current.Interface, error) {
	contIface := &current.Interface{}
	hostIface := &current.Interface{}

	err := netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		hostVeth, containerVeth, err := ip.SetupVeth(ifName, mtu, hostNS)
		if err != nil {
			return err
		}
		contIface.Name = containerVeth.Name
		contIface.Mac = containerVeth.HardwareAddr.String()
		contIface.Sandbox = netns.Path()
		hostIface.Name = hostVeth.Name
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup %q: %v", hostIface.Name, err)
	}
	hostIface.Mac = hostVeth.Attrs().HardwareAddr.String()

	// connect host veth end to the bridge
	if err := netlink.LinkSetMaster(hostVeth, br); err != nil {
		return nil, nil, fmt.Errorf("failed to connect %q to bridge %v: %v", hostVeth.Attrs().Name, br.Attrs().Name, err)
	}

	// set hairpin mode
	if err = netlink.LinkSetHairpin(hostVeth, hairpinMode); err != nil {
		return nil, nil, fmt.Errorf("failed to setup hairpin mode for %v: %v", hostVeth.Attrs().Name, err)
	}

	return hostIface, contIface, nil
}

func calcGatewayIP(ipn *net.IPNet) net.IP {
	nid := ipn.IP.Mask(ipn.Mask)
	return ip.NextIP(nid)
}

func setupBridge(n *NetConf) (*netlink.Bridge, *current.Interface, error) {
	// create bridge if necessary
	br, err := ensureBridge(n.BrName, n.MTU)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create bridge %q: %v", n.BrName, err)
	}

	return br, &current.Interface{
		Name: br.Attrs().Name,
		Mac:  br.Attrs().HardwareAddr.String(),
	}, nil
}

// disableIPV6DAD disables IPv6 Duplicate Address Detection (DAD)
// for an interface.
func disableIPV6DAD(ifName string) error {
	f := fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/accept_dad", ifName)
	return ioutil.WriteFile(f, []byte("0"), 0644)
}

func enableIPForward(family int) error {
	if family == netlink.FAMILY_V4 {
		return ip.EnableIP4Forward()
	}
	return ip.EnableIP6Forward()
}

func ipamPluginType(n *NetConf) string {
	// TODO: Add logic to handle configurations that have different IPv4 vs
	// IPv6 IPAM plugin types. This will require splitting up the V4 vs V6
	// config, marshalling the separate configs into stdin (bitstream)
	// format, calling the different IPAM plugins in sequence, and
	// combining the results.
	if n.IPAM.Type != "" {
		return n.IPAM.Type
	}
	return n.IPAM6.Type
}

func cmdAdd(args *skel.CmdArgs) error {
	n, cniVersion, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	if n.IsDefaultGW {
		n.IsGW = true
	}

	br, brInterface, err := setupBridge(n)
	if err != nil {
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	hostInterface, containerInterface, err := setupVeth(netns, br, args.IfName, n.MTU, n.HairpinMode)
	if err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	r, err := ipam.ExecAdd(ipamPluginType(n), args.StdinData)
	if err != nil {
		return err
	}

	// Convert whatever the IPAM result was into the current Result type
	result, err := current.NewResultFromResult(r)
	if err != nil {
		return err
	}

	if len(result.IPs) == 0 {
		return errors.New("IPAM plugin returned missing IP config")
	}

	result.Interfaces = []*current.Interface{brInterface, hostInterface, containerInterface}

	addrGWV4, addrGWV6, err := firstAddrGWs(result, n)
	if err != nil {
		return err
	}

	// Configure the container hardware address and IP address(es)
	if err := netns.Do(func(_ ns.NetNS) error {

		// Temporary workaround for Kubernetes Issue #32291. Disable
		// IPv6 DAD since kubelet enables hairpin mode on the bridge.
		// Hairpin mode causes echos of neighbor solicitation packets,
		// which causes DAD failures. Long term fix: enable enhanced DAD
		// when that becomes available in kernels.
		if err := disableIPV6DAD(args.IfName); err != nil {
			return err
		}

		err := ip.SetHWAddrByIP(args.IfName, addrGWV4.addr, addrGWV6.addr)
		if err != nil {
			return err
		}

		if err := ipam.ConfigureIface(args.IfName, result); err != nil {
			return err
		}

		// Refetch the veth since its MAC address may have changed
		link, err := netlink.LinkByName(args.IfName)
		if err != nil {
			return fmt.Errorf("could not lookup %q: %v", args.IfName, err)
		}
		containerInterface.Mac = link.Attrs().HardwareAddr.String()

		return nil
	}); err != nil {
		return err
	}

	if n.IsGW {

		// Set the hardware address on the bridge interface
		if addrGWV4.gws != nil || addrGWV6.gws != nil {
			err := ip.SetHWAddrByIP(n.BrName, firstGW(addrGWV4), firstGW(addrGWV6))
			if err != nil {
				return err
			}
		}

		// Set the IP address(es) on the bridge interface and enable forwarding
		for _, aGW := range []*addrGWs{addrGWV4, addrGWV6} {
			for _, gw := range aGW.gws {
				err = ensureBridgeAddr(br, aGW.family, &gw, n.ForceAddress)
				if err != nil {
					return fmt.Errorf("failed to set bridge addr: %v", err)
				}
			}
			if aGW.gws != nil {
				if err = enableIPForward(aGW.family); err != nil {
					return fmt.Errorf("failed to enable forwarding: %v", err)
				}
			}
		}
	}

	if n.IPMasq {
		chain := utils.FormatChainName(n.Name, args.ContainerID)
		comment := utils.FormatComment(n.Name, args.ContainerID)
		for _, ipc := range result.IPs {
			if err = ip.SetupIPMasq(ip.Network(&ipc.Address), chain, comment); err != nil {
				return err
			}
		}
	}

	// Refetch the bridge since its MAC address may change when the first
	// veth is added or after its IP address is set
	br, err = bridgeByName(n.BrName)
	if err != nil {
		return err
	}
	brInterface.Mac = br.Attrs().HardwareAddr.String()

	result.DNS = n.DNS

	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	if err := ipam.ExecDel(ipamPluginType(n), args.StdinData); err != nil {
		return err
	}

	if args.Netns == "" {
		return nil
	}

	// There is a netns so try to clean up. Delete can be called multiple times
	// so don't return an error if the device is already removed.
	// If the device isn't there then don't try to clean up IP masq either.
	var ipn *net.IPNet
	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		var err error
		ipn, err = ip.DelLinkByNameAddr(args.IfName, netlink.FAMILY_ALL)
		if err != nil && err == ip.ErrLinkNotFound {
			return nil
		}
		return err
	})

	if err != nil {
		return err
	}

	if ipn != nil && n.IPMasq {
		chain := utils.FormatChainName(n.Name, args.ContainerID)
		comment := utils.FormatComment(n.Name, args.ContainerID)
		err = ip.TeardownIPMasq(ipn, chain, comment)
	}

	return err
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}
