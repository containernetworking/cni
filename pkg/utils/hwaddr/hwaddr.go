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

package hwaddr

import (
	"fmt"
	"net"
)

const (
	ipRelevantByteLen       = 4
	PrivateMACPrefixString  = "0a:58"
	PrivateMACPrefixString6 = "6a:58"
)

var (
	// private mac prefix safe to use
	PrivateMACPrefix  = []byte{0x0a, 0x58}
	PrivateMACPrefix6 = []byte{0x6a, 0x58}
)

type InvalidIP4Err struct{ msg string }

func (e InvalidIP4Err) Error() string { return e.msg }

type InvalidPrefixLengthErr struct{ msg string }

func (e InvalidPrefixLengthErr) Error() string { return e.msg }

// GenerateHardwareAddr4 generates 48 bit virtual mac addresses based on
// an IPv4 address.
func GenerateHardwareAddr4(ip net.IP, prefix []byte) (net.HardwareAddr, error) {
	switch {

	case ip.To4() == nil:
		return nil, InvalidIP4Err{msg: fmt.Sprintf(
			"Invalid IPv4 address: %s", ip.String()),
		}

	case len(prefix) != len(PrivateMACPrefix):
		return nil, InvalidPrefixLengthErr{msg: fmt.Sprintf(
			"Prefix has length %d instead of %d", len(prefix), len(PrivateMACPrefix)),
		}
	}

	ipByteLen := len(ip)
	return (net.HardwareAddr)(
		append(
			prefix,
			ip[ipByteLen-ipRelevantByteLen:ipByteLen]...),
	), nil
}

// GenerateHardwareAddr6 generates 48 bit virtual mac address for an
// interface by taking the existing, presumably random mac address
// and overwriting the first 2 octets with an easily discernible
// 2-byte hard-coded prefix (6a:58).
func GenerateHardwareAddr6(mac net.HardwareAddr, prefix []byte) (net.HardwareAddr, error) {

	if len(prefix) != len(PrivateMACPrefix) {
		return nil, InvalidPrefixLengthErr{msg: fmt.Sprintf(
			"Prefix has length %d instead of %d", len(prefix),
			len(PrivateMACPrefix6)),
		}
	}

	return (net.HardwareAddr)(append(prefix, mac[len(prefix):]...)), nil
}
