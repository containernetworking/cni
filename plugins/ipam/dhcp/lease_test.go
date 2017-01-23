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
	"bytes"
	"net"
	"testing"

	"github.com/d2g/dhcp4"
	"github.com/vishvananda/netlink"
)

func TestAddClientID(t *testing.T) {
	link, err := netlink.LinkByName("lo")
	if err != nil {
		t.Errorf("netlink.LinkByName: %v", err)
		return
	}
	c, err := newDHCPClient(link, true)
	if err != nil {
		t.Errorf("newDHCPClient: %v", err)
		return
	}
	l := &DHCPLease{
		clientID:     "abc123",
		sendClientID: true,
		useBroadcast: true,
		stop:         make(chan struct{}),
	}
	pkt, err := l.sendDiscoverPacket(c)
	options := pkt.ParseOptions()
	if pkt.Broadcast() != true {
		t.Errorf("Broadcast flag should be set, but is not!")
	}
	chaddr := net.HardwareAddr{0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(pkt.CHAddr(), chaddr) {
		t.Errorf("Got CHAddr: %v, expected: %v", pkt.CHAddr(), chaddr)
	}
	clientid := bytes.NewBuffer([]byte{0})
	clientid.Write([]byte(l.clientID))
	if !bytes.Equal(options[dhcp4.OptionClientIdentifier], clientid.Bytes()) {
		t.Errorf("Got ClientID: %v, expected: %v", options[dhcp4.OptionClientIdentifier], clientid)
	}
}
