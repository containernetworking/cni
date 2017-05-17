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

package ip

import (
	"net"

	"github.com/j-keck/arping"
)

// GratuitousArp sends an arp for the supplied IP address on the interface ifName
func GratuitousArp(iface *net.Interface, addr net.IP) error {
	return arping.GratuitousArpOverIface(addr, *iface)
}
