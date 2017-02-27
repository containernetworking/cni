// Copyright 2015-2017 CNI authors
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
	"net"

	"github.com/vishvananda/netlink"
)

func addNextHops(route *netlink.Route, nextHops []net.IP, linkIndex int) {
	switch {
	case len(nextHops) == 0:
		return
	case len(nextHops) == 1:
		route.Gw = nextHops[0]
	case len(nextHops) > 1:
		for _, nh := range nextHops {
			route.MultiPath = append(route.MultiPath, &netlink.NexthopInfo{
				LinkIndex: linkIndex,
				Hops:      1,
				Gw:        nh,
			})
		}
	}
}

// AddRoute adds a universally-scoped route to a device.
func AddRoute(ipn *net.IPNet, nextHops []net.IP, dev netlink.Link) error {
	route := &netlink.Route{
		LinkIndex: dev.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Dst:       ipn,
	}
	addNextHops(route, nextHops, dev.Attrs().Index)
	return netlink.RouteAdd(route)
}

// AddHostRoute adds a host-scoped route to a device.
func AddHostRoute(ipn *net.IPNet, nextHops []net.IP, dev netlink.Link) error {
	route := &netlink.Route{
		LinkIndex: dev.Attrs().Index,
		Scope:     netlink.SCOPE_HOST,
		Dst:       ipn,
	}
	addNextHops(route, nextHops, dev.Attrs().Index)
	return netlink.RouteAdd(route)
}
