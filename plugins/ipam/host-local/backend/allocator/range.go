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

package allocator

import (
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/types"
)

// Validate takes a given range and ensures that all information is consistent,
// filling out Start, End, and Gateway with sane values if missing
func (r *Range) Canonicalize() error {
	if err := canonicalizeIP(&r.Subnet.IP); err != nil {
		return err
	}

	// Can't create an allocator for a network with no addresses, eg
	// a /32 or /31
	ones, masklen := r.Subnet.Mask.Size()
	if ones > masklen-2 {
		return fmt.Errorf("Network %s too small to allocate from", r.Subnet.String())
	}

	if len(r.Subnet.IP) != len(r.Subnet.Mask) {
		return fmt.Errorf("IPNet IP and Mask version mismatch")
	}

	// If the gateway is nil, claim .1
	if r.Gateway == nil {
		r.Gateway = ip.NextIP(r.Subnet.IP)
	} else {
		if err := canonicalizeIP(&r.Gateway); err != nil {
			return err
		}
		subnet := (net.IPNet)(r.Subnet)
		if !subnet.Contains(r.Gateway) {
			return fmt.Errorf("gateway %s not in network %s", r.Gateway.String(), subnet.String())
		}
	}

	// RangeStart: If specified, make sure it's sane (inside the subnet),
	// otherwise use .2
	if r.RangeStart != nil {
		if err := canonicalizeIP(&r.RangeStart); err != nil {
			return err
		}

		if err := r.IPInRange(r.RangeStart); err != nil {
			return err
		}
	} else {
		r.RangeStart = ip.NextIP(ip.NextIP(r.Subnet.IP))
	}

	// RangeEnd: If specified, verify sanity. Otherwise, add a sensible default
	// (e.g. for a /24: .254 if IPv4, ::255 if IPv6)
	if r.RangeEnd != nil {
		if err := canonicalizeIP(&r.RangeEnd); err != nil {
			return err
		}

		if err := r.IPInRange(r.RangeEnd); err != nil {
			return err
		}
	} else {
		r.RangeEnd = lastIP(r.Subnet)
	}

	return nil
}

// IsValidIP checks if a given ip is a valid, allocatable address in a given Range
func (r *Range) IPInRange(addr net.IP) error {
	if err := canonicalizeIP(&addr); err != nil {
		return err
	}

	subnet := (net.IPNet)(r.Subnet)

	if len(addr) != len(r.Subnet.IP) {
		return fmt.Errorf("IP %s is not the same protocol as subnet %s",
			addr, subnet.String())
	}

	if !subnet.Contains(addr) {
		return fmt.Errorf("%s not in network %s", addr, subnet.String())
	}

	// We ignore nils here so we can use this function as we initialize the range.
	if r.RangeStart != nil {
		if ip.Cmp(addr, r.RangeStart) < 0 {
			return fmt.Errorf("%s is in network %s but before start %s",
				addr, r.Subnet.String(), r.RangeStart)
		}
	}

	if r.RangeEnd != nil {
		if ip.Cmp(addr, r.RangeEnd) > 0 {
			return fmt.Errorf("%s is in network %s but after end %s",
				addr, r.Subnet.String(), r.RangeEnd)
		}
	}

	return nil
}

// canonicalizeIP makes sure a provided ip is in standard form
func canonicalizeIP(ip *net.IP) error {
	if ip.To4() != nil {
		*ip = ip.To4()
		return nil
	} else if ip.To16() != nil {
		*ip = ip.To16()
		return nil
	}
	return fmt.Errorf("IP %s not v4 nor v6", *ip)
}

// Determine the last IP of a subnet, excluding the broadcast if IPv4
func lastIP(subnet types.IPNet) net.IP {
	var end net.IP
	for i := 0; i < len(subnet.IP); i++ {
		end = append(end, subnet.IP[i]|^subnet.Mask[i])
	}
	if subnet.IP.To4() != nil {
		end[3]--
	}

	return end
}
