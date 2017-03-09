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

package ip

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/utils/hwaddr"
	"github.com/vishvananda/netlink"
)

func makeVethPair(name, peer string, mtu int) (netlink.Link, error) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  name,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: peer,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}

	return veth, nil
}

func peerExists(name string) bool {
	if _, err := netlink.LinkByName(name); err != nil {
		return false
	}
	return true
}

func makeVeth(name string, mtu int) (peerName string, veth netlink.Link, err error) {
	for i := 0; i < 10; i++ {
		peerName, err = RandomVethName()
		if err != nil {
			return
		}

		veth, err = makeVethPair(name, peerName, mtu)
		switch {
		case err == nil:
			return

		case os.IsExist(err):
			if peerExists(peerName) {
				continue
			}
			err = fmt.Errorf("container veth name provided (%v) already exists", name)
			return

		default:
			err = fmt.Errorf("failed to make veth pair: %v", err)
			return
		}
	}

	// should really never be hit
	err = fmt.Errorf("failed to find a unique veth name")
	return
}

// RandomVethName returns string "veth" with random prefix (hashed from entropy)
func RandomVethName() (string, error) {
	entropy := make([]byte, 4)
	_, err := rand.Reader.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate random veth name: %v", err)
	}

	// NetworkManager (recent versions) will ignore veth devices that start with "veth"
	return fmt.Sprintf("veth%x", entropy), nil
}

func RenameLink(curName, newName string) error {
	link, err := netlink.LinkByName(curName)
	if err == nil {
		err = netlink.LinkSetName(link, newName)
	}
	return err
}

// SetupVeth sets up a virtual ethernet link.
// Should be in container netns, and will switch back to hostNS to set the host
// veth end up.
func SetupVeth(contVethName string, mtu int, hostNS ns.NetNS) (hostVeth, contVeth netlink.Link, err error) {
	var hostVethName string
	hostVethName, contVeth, err = makeVeth(contVethName, mtu)
	if err != nil {
		return
	}

	if err = netlink.LinkSetUp(contVeth); err != nil {
		err = fmt.Errorf("failed to set %q up: %v", contVethName, err)
		return
	}

	hostVeth, err = netlink.LinkByName(hostVethName)
	if err != nil {
		err = fmt.Errorf("failed to lookup %q: %v", hostVethName, err)
		return
	}

	if err = netlink.LinkSetNsFd(hostVeth, int(hostNS.Fd())); err != nil {
		err = fmt.Errorf("failed to move veth to host netns: %v", err)
		return
	}

	err = hostNS.Do(func(_ ns.NetNS) error {
		hostVeth, err = netlink.LinkByName(hostVethName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v", hostVethName, hostNS.Path(), err)
		}

		if err = netlink.LinkSetUp(hostVeth); err != nil {
			return fmt.Errorf("failed to set %q up: %v", hostVethName, err)
		}
		return nil
	})
	return
}

// DelLinkByName removes an interface link.
func DelLinkByName(ifName string) error {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return fmt.Errorf("failed to delete %q: %v", ifName, err)
	}

	return nil
}

// DelLinkByNameAddr remove an interface returns its IP address
// of the specified family
func DelLinkByNameAddr(ifName string, family int) (*net.IPNet, error) {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	addrs, err := netlink.AddrList(iface, family)
	if err != nil || len(addrs) == 0 {
		return nil, fmt.Errorf("failed to get IP addresses for %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return nil, fmt.Errorf("failed to delete %q: %v", ifName, err)
	}

	return addrs[0].IPNet, nil
}

// SetHWAddrByIP sets the hardware address of an interface/link. If an IPv4
// address is provided (for IPv4-only or dual-stack operation), then the
// that IPv4 address is assumed to be locally unique, and the hardware
// address that is configured comprises the 4 bytes of IPv4 address preceded
// by a hard-coded 2-byte prefix (0a:58). If only an IPv6 address is
// provided, then the last 4 bytes of the interface/link's current hardware
// address are assumed to be random, and the hardware address that is
// configured is derived by overwriting the first two bytes of the
// current hardware address with a different hard-coded prefix (6a:58).
func SetHWAddrByIP(ifName string, ip4 net.IP, ip6 net.IP) error {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}
	startHWAddr := iface.Attrs().HardwareAddr.String()

	var hwAddr net.HardwareAddr
	switch {
	case ip4 != nil:
		hwAddr, err = hwaddr.GenerateHardwareAddr4(ip4, hwaddr.PrivateMACPrefix)
		if err != nil {
			return fmt.Errorf("failed to generate hardware addr: %v", err)
		}
	case ip6 != nil:
		attrs := iface.Attrs()
		hwAddr, err = hwaddr.GenerateHardwareAddr6(attrs.HardwareAddr,
			hwaddr.PrivateMACPrefix6)
		if err != nil {
			return fmt.Errorf("failed to generate hardware addr: %v", err)
		}
	default:
		return fmt.Errorf("unable to generate hardware address as neither IPv4 nor IPv6 address specified")
	}

	if hwAddr.String() != startHWAddr {
		// Toggle the interface while changing the hardware address
		// so that a new IPv6 link local address gets generated.
		if err := netlink.LinkSetDown(iface); err != nil {
			return fmt.Errorf("failed to set down %q: %v", ifName, err)
		}
		if err = netlink.LinkSetHardwareAddr(iface, hwAddr); err != nil {
			return fmt.Errorf("failed to add hardware addr to %q: %v", ifName, err)
		}
		if err := netlink.LinkSetUp(iface); err != nil {
			return fmt.Errorf("failed to set up %q: %v", ifName, err)
		}
	}

	return nil
}
