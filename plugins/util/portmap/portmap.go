// Copyright 2017 CNI authors
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
	"net"
	"strconv"

	"github.com/containernetworking/cni/pkg/utils"
	"github.com/containernetworking/cni/pkg/utils/sysctl"
	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

// This creates the chains to be added to iptables. The basic structure is
// a bit complex for efficiencies sake. We create 2 chains: a summary chain
// that is shared between invocations, and an invocation (container)-specific
// chain. This minimizes the number of operations on the top level, but allows
// for easy cleanup.
//
// We also create DNAT chains to rewrite destinations, and SNAT chains so that
// connections to localhost work.
//
// The basic setup (all operations are on the nat table) is:
//
// DNAT case (rewrite destination IP and port):
// PREROUTING, OUTPUT: --dst-type local -j CNI-HOSTPORT_DNAT
// CNI-HOSTPORT-DNAT: -j CNI-DN-abcd123
// CNI-DN-abcd123: -p tcp --dport 8080 -j DNAT --to-destination 192.0.2.33:80
// CNI-DN-abcd123: -p tcp --dport 8081 -j DNAT ...
//
// SNAT case (rewrite source IP from localhost after dnat):
// POSTROUTING: -s 127.0.0.1 ! -d 127.0.0.1 -j CNI-HOSTPORT-SNAT
// CNI-HOSTPORT-SNAT: -j CNI-SN-abcd123
// CNI-SN-abcd123: -p tcp -s 127.0.0.1 -d 192.0.2.33 --dport 80 -j MASQUERADE
// CNI-SN-abcd123: -p tcp -s 127.0.0.1 -d 192.0.2.33 --dport 90 -j MASQUERADE

// The names of the top-level summary chains.
// These should never be changed, or else upgrading will require manual
// intervention.
const TopLevelDNATChainName = "CNI-HOSTPORT-DNAT"
const TopLevelSNATChainName = "CNI-HOSTPORT-SNAT"

// forwardPorts establishes port forwarding to a given container IP.
// containerIP can be either v4 or v6.
func forwardPorts(config *PortMapConf, containerIP net.IP, enableSNAT bool) error {
	isV6 := (containerIP.To4() == nil)

	var ipt *iptables.IPTables
	var err error

	if isV6 {
		ipt, err = iptables.NewWithProtocol(iptables.ProtocolIPv6)
	} else {
		ipt, err = iptables.NewWithProtocol(iptables.ProtocolIPv4)
	}
	if err != nil {
		return fmt.Errorf("failed to open iptables: %v", err)
	}

	toplevelDnatChan := genToplevelDnatChain()
	if err := toplevelDnatChan.setup(ipt, nil); err != nil {
		return fmt.Errorf("failed to create top-level DNAT chain: %v", err)
	}

	dnatChain := genDnatChain(config.Name, config.ContainerID)
	_ = dnatChain.teardown(ipt) // If we somehow collide on this container ID + network, cleanup

	dnatRules := dnatRules(config.RuntimeConfig.PortMaps, containerIP)
	if err := dnatChain.setup(ipt, dnatRules); err != nil {
		return fmt.Errorf("unable to setup DNAT: %v", err)
	}

	// Enable SNAT for connections to localhost.
	// This won't work for ipv6, since the kernel doesn't have the equvalent
	// route_localnet sysctl.
	if enableSNAT {
		toplevelSnatChain := genToplevelSnatChain(isV6)
		if err := toplevelSnatChain.setup(ipt, nil); err != nil {
			return fmt.Errorf("failed to create top-level SNAT chain: %v", err)
		}

		snatChain := genSnatChain(config.Name, config.ContainerID)
		_ = snatChain.teardown(ipt)

		snatRules := snatRules(config.RuntimeConfig.PortMaps, containerIP)
		if err := snatChain.setup(ipt, snatRules); err != nil {
			return fmt.Errorf("unable to setup SNAT: %v", err)
		}
		if !isV6 {
			// Set the route_localnet bit on the host interface, so that
			// 127/8 can cross a routing boundary.
			hostIfName := getRoutableHostIF(containerIP)
			if hostIfName != "" {
				if err := enableLocalnetRouting(hostIfName); err != nil {
					return fmt.Errorf("unable to enable route_localnet: %v", err)
				}
			}
		}
	}

	return nil
}

// genToplevelDnatChain creates the top-level summary chain that we'll
// add our chain to. This is easy, because creating chains is idempotent.
// IMPORTANT: do not change this, or else upgrading plugins will require
// manual intervention.
func genToplevelDnatChain() chain {
	return chain{
		table: "nat",
		name:  TopLevelDNATChainName,
		entryRule: []string{
			"-m", "addrtype",
			"--dst-type", "LOCAL",
		},
		entryChains: []string{"PREROUTING", "OUTPUT"},
	}
}

// genDnatChain creates the per-container chain.
func genDnatChain(netName, containerID string) chain {
	name := utils.FormatChainName("DN-", netName, containerID)
	comment := fmt.Sprintf(`dnat name: "%s" id: "%s"`, netName, containerID)

	return chain{
		table: "nat",
		name:  name,
		entryRule: []string{
			"-m", "comment",
			"--comment", comment,
		},
		entryChains: []string{TopLevelDNATChainName},
	}
}

// dnatRules generates the destination NAT rules, one per port, to direct
// traffic from hostip:hostport to podip:podport
func dnatRules(entries []PortMapEntry, containerIP net.IP) [][]string {
	out := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rule := []string{
			"-p", entry.Protocol,
			"--dport", strconv.Itoa(entry.HostPort)}

		if entry.HostIP != "" {
			rule = append(rule,
				"-d", entry.HostIP)
		}

		rule = append(rule,
			"-j", "DNAT",
			"--to-destination", fmtIpPort(containerIP, entry.ContainerPort))

		out = append(out, rule)
	}
	return out
}

// genToplevelSnatChain creates the top-level summary snat chain.
// IMPORTANT: do not change this, or else upgrading plugins will require
// manual intervention
func genToplevelSnatChain(isV6 bool) chain {
	return chain{
		table: "nat",
		name:  TopLevelSNATChainName,
		entryRule: []string{
			"-s", localhostIP(isV6),
			"!", "-d", localhostIP(isV6),
		},
		entryChains: []string{"POSTROUTING"},
	}
}

// genSnatChain creates the snat (localhost) chain for this container.
func genSnatChain(netName, containerID string) chain {
	name := utils.FormatChainName("SN-", netName, containerID)
	comment := fmt.Sprintf(`snat name: "%s" id: "%s"`, netName, containerID)

	return chain{
		table: "nat",
		name:  name,
		entryRule: []string{
			"-m", "comment",
			"--comment", comment,
		},
		entryChains: []string{TopLevelSNATChainName},
	}
}

// snatRules sets up masquerading for connections to localhost:hostport,
// rewriting the source so that returning packets are correct.
func snatRules(entries []PortMapEntry, containerIP net.IP) [][]string {
	isV6 := (containerIP.To4() == nil)

	out := make([][]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, []string{
			"-p", entry.Protocol,
			"-s", localhostIP(isV6),
			"-d", containerIP.String(),
			"--dport", strconv.Itoa(entry.ContainerPort),
			"-j", "MASQUERADE",
		})
	}
	return out
}

// enableLocalnetRouting tells the kernel not to treat 127/8 as a martian,
// so that connections with a source ip of 127/8 can cross a routing boundary.
func enableLocalnetRouting(ifName string) error {
	routeLocalnetPath := "net.ipv4.conf." + ifName + ".route_localnet"
	_, err := sysctl.Sysctl(routeLocalnetPath, "1")
	return err
}

// unforwardPorts deletes any iptables rules created by this plugin.
// It should be idempotent - it will not error if the chain does not exist.
func unforwardPorts(config *PortMapConf) error {
	// Teardown v4
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	dnatChain := genDnatChain(config.Name, config.ContainerID)
	if err := dnatChain.teardown(ipt); err != nil {
		return fmt.Errorf("could not teardown ipv4 dnat: %v", err)
	}
	snatChain := genSnatChain(config.Name, config.ContainerID)
	if err := snatChain.teardown(ipt); err != nil {
		return fmt.Errorf("could not teardown ipv4 snat: %v", err)
	}

	ip6t, err := iptables.NewWithProtocol(iptables.ProtocolIPv6)
	if err != nil {
		return err
	}

	if err := dnatChain.teardown(ip6t); err != nil {
		return fmt.Errorf("could not teardown ipv6 dnat: %v", err)
	}
	if err := snatChain.teardown(ip6t); err != nil {
		return fmt.Errorf("could not teardown ipv6 snat: %v", err)
	}
	return nil
}

// getRoutableHostIF will try and determine which interface routes the container's
// traffic. This is the one on which we disable martian filtering.
func getRoutableHostIF(containerIP net.IP) string {
	routes, err := netlink.RouteGet(containerIP)
	if err != nil {
		return ""
	}

	for _, route := range routes {
		link, err := netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			continue
		}

		return link.Attrs().Name
	}

	return ""
}

// fmtIpPort correctly formats ip:port literals for iptables and ip6tables -
// need to wrap v6 literals in a []
func fmtIpPort(ip net.IP, port int) string {
	if ip.To4() == nil {
		return fmt.Sprintf("[%s]:%d", ip.String(), port)
	}
	return fmt.Sprintf("%s:%d", ip.String(), port)
}

func localhostIP(isV6 bool) string {
	if isV6 {
		return "::1"
	}
	return "127.0.0.1"
}
