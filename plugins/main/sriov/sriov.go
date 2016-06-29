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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"

	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/vishvananda/netlink"
)

type NetConf struct {
	types.NetConf
	Master string `json:"master"`
	MAC    string `json:"mac"`
	VF     int    `json:"vf"`
	Vlan   int    `json:"vlan"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadConf(bytes []byte) (*NetConf, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}
	if n.Master == "" {
		return nil, fmt.Errorf(`"master" field is required. It specifies the host interface name to virtualize`)
	}
	return n, nil
}

func setupVF(conf *NetConf, ifName string, netns ns.NetNS) error {

	masterName := conf.Master
	vfIdx := conf.VF

	m, err := netlink.LinkByName(masterName)
	if err != nil {
		return fmt.Errorf("failed to lookup master %q: %v", conf.Master, err)
	}

	vfDir := fmt.Sprintf("/sys/class/net/%s/device/virtfn%d/net", masterName, vfIdx)
	if _, err := os.Lstat(vfDir); err != nil {
		return err
	}

	infos, err := ioutil.ReadDir(vfDir)
	if err != nil {
		return err
	}

	if len(infos) != 1 {
		return fmt.Errorf("Mutiple network devices in directory %s", vfDir)
	}

	// VF NIC name
	vfDevName := infos[0].Name()
	vfDev, err := netlink.LinkByName(vfDevName)
	if err != nil {
		return fmt.Errorf("failed to lookup vf device %q: %v", vfDevName, err)
	}

	// set hardware address
	if conf.MAC != "" {
		macAddr, err := net.ParseMAC(conf.MAC)
		if err != nil {
			return err
		}
		if err = netlink.LinkSetVfHardwareAddr(m, conf.VF, macAddr); err != nil {
			return fmt.Errorf("failed to set vf %d macaddress: %v", conf.VF, err)
		}
	}

	if conf.Vlan != 0 {
		if err = netlink.LinkSetVfVlan(m, conf.VF, conf.Vlan); err != nil {
			return fmt.Errorf("failed to set vf %d vlan: %v", conf.VF, err)
		}
	}

	if err = netlink.LinkSetUp(vfDev); err != nil {
		return fmt.Errorf("failed to setup vf device: %v", err)
	}

	// move vf device to ns
	if err = netlink.LinkSetNsFd(vfDev, int(netns.Fd())); err != nil {
		return fmt.Errorf("failed to move vf %d to netnamespace: %v", conf.VF, err)
	}

	return netns.Do(func(_ ns.NetNS) error {
		err := renameLink(vfDevName, ifName)
		if err != nil {
			return fmt.Errorf("failed to rename vf device %q to %q: %v", vfDevName, ifName, err)
		}
		return nil
	})
}

func cmdAdd(args *skel.CmdArgs) error {
	n, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", netns, err)
	}
	defer netns.Close()

	if err = setupVF(n, args.IfName, netns); err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	result, err := ipam.ExecAdd(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}
	if result.IP4 == nil {
		return errors.New("IPAM plugin returned missing IPv4 config")
	}

	err = netns.Do(func(_ ns.NetNS) error {
		return ipam.ConfigureIface(args.IfName, result)
	})
	if err != nil {
		return err
	}

	result.DNS = n.DNS
	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	n, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	err = ipam.ExecDel(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	return nil
}

func renameLink(curName, newName string) error {
	link, err := netlink.LinkByName(curName)
	if err != nil {
		return err
	}

	return netlink.LinkSetName(link, newName)
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
