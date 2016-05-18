// Copyright 2016 CNI authors
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

package ops

import (
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/utils/sysctl"

	"github.com/vishvananda/netlink"
)

type NetOps interface {
	LinkAdd(link netlink.Link) error
	LinkDel(link netlink.Link) error
	LinkByIndex(index int) (netlink.Link, error)
	LinkByName(name string) (netlink.Link, error)
	LinkSetName(link netlink.Link, name string) error
	LinkSetUp(link netlink.Link) error
	LinkSetDown(link netlink.Link) error
	LinkSetMaster(link netlink.Link, master *netlink.Bridge) error
	LinkSetNsFd(link netlink.Link, fd int) error
	LinkList() ([]netlink.Link, error)

	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)

	RouteAdd(route *netlink.Route) error
	RouteDel(route *netlink.Route) error

	Sysctl(name string, params ...string) (string, error)

	GetNS(nspath string) (ns.NetNS, error)
	NewNS() (ns.NetNS, error)
	WithNetNSPath(nspath string, f func(ns.NetNS) error) error
}

func NewNetOps() NetOps {
	return &defaultOps{}
}

type defaultOps struct{}

func (o *defaultOps) LinkAdd(link netlink.Link) error {
	return netlink.LinkAdd(link)
}

func (o *defaultOps) LinkDel(link netlink.Link) error {
	return netlink.LinkDel(link)
}

func (o *defaultOps) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

func (o *defaultOps) LinkByIndex(index int) (netlink.Link, error) {
	return netlink.LinkByIndex(index)
}

func (o *defaultOps) LinkSetName(link netlink.Link, name string) error {
	return netlink.LinkSetName(link, name)
}

func (o *defaultOps) LinkSetUp(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}

func (o *defaultOps) LinkSetDown(link netlink.Link) error {
	return netlink.LinkSetDown(link)
}

func (o *defaultOps) LinkSetMaster(link netlink.Link, master *netlink.Bridge) error {
	return netlink.LinkSetMaster(link, master)
}

func (o *defaultOps) LinkSetNsFd(link netlink.Link, fd int) error {
	return netlink.LinkSetNsFd(link, fd)
}

func (o *defaultOps) LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

func (o *defaultOps) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

func (o *defaultOps) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}

func (o *defaultOps) RouteAdd(route *netlink.Route) error {
	return netlink.RouteAdd(route)
}

func (o *defaultOps) RouteDel(route *netlink.Route) error {
	return netlink.RouteDel(route)
}

func (o *defaultOps) Sysctl(name string, params ...string) (string, error) {
	return sysctl.Sysctl(name, params...)
}

func (o *defaultOps) GetNS(nspath string) (ns.NetNS, error) {
	return ns.GetNS(nspath)
}

func (o *defaultOps) NewNS() (ns.NetNS, error) {
	return ns.NewNS()
}

func (o *defaultOps) WithNetNSPath(nspath string, f func(ns.NetNS) error) error {
	return ns.WithNetNSPath(nspath, f)
}
