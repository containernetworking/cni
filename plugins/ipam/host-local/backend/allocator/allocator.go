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

package allocator

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/plugins/ipam/host-local/backend"
)

type IPAllocator struct {
	netName  string
	ipRange  Range
	store    backend.Store
	rangeIdx int
}

type RangeIter struct {
	low   net.IP
	high  net.IP
	cur   net.IP
	start net.IP
}

func NewIPAllocator(c *IPAMConfig, rangeIdx int, store backend.Store) *IPAllocator {
	return &IPAllocator{
		netName:  c.Name,
		ipRange:  c.Ranges[rangeIdx],
		store:    store,
		rangeIdx: rangeIdx,
	}
}

// Returns newly allocated IP along with its ipRangeig
func (a *IPAllocator) Get(id string, requestedIP net.IP) (*current.IPConfig, error) {
	a.store.Lock()
	defer a.store.Unlock()

	gw := a.ipRange.Gateway

	var reservedIP net.IP

	if requestedIP != nil {
		if gw != nil && gw.Equal(requestedIP) {
			return nil, fmt.Errorf("requested IP must differ from gateway IP")
		}

		if err := a.ipRange.IPInRange(requestedIP); err != nil {
			return nil, err
		}

		reserved, err := a.store.Reserve(id, requestedIP, a.rangeIdx)
		if err != nil {
			return nil, err
		}
		if !reserved {
			return nil, fmt.Errorf("requested IP address %q is not available in network: %s %s", requestedIP, a.netName, a.ipRange.Subnet.String())
		}
		reservedIP = requestedIP

	} else {
		iter, err := a.GetIter()
		if err != nil {
			return nil, err
		}
		for {
			cur := iter.Next()
			if cur == nil {
				break
			}

			// don't allocate gateway IP
			if gw != nil && cur.Equal(gw) {
				continue
			}

			reserved, err := a.store.Reserve(id, cur, a.rangeIdx)
			if err != nil {
				return nil, err
			}

			if reserved {
				reservedIP = cur
				break
			}
		}
	}

	if reservedIP == nil {
		return nil, fmt.Errorf("no IP addresses available in network: %s %s", a.netName, a.ipRange.Subnet.String())
	}
	version := "4"
	if reservedIP.To4() == nil {
		version = "6"
	}

	return &current.IPConfig{
		Version: version,
		Address: net.IPNet{IP: reservedIP, Mask: a.ipRange.Subnet.Mask},
		Gateway: gw,
	}, nil
}

// Releases all IPs allocated for the container with given ID
func (a *IPAllocator) Release(id string) error {
	a.store.Lock()
	defer a.store.Unlock()

	return a.store.ReleaseByID(id)
}

// GetIter encapsulates the allocation strategy for this allocator.
// We use a round-robin strategy, attempting to evenly use the whole subnet.
// More specifically, a crash-looping container will not see the same IP until
// the entire range has been run through.
// We may wish to consider avoiding recently-released IPs in the future.
func (a *IPAllocator) GetIter() (*RangeIter, error) {
	i := RangeIter{
		low:  a.ipRange.RangeStart,
		high: a.ipRange.RangeEnd,
	}

	// Round-robin by trying to allocate from the last reserved IP + 1
	startFromLastReservedIP := false

	// We might get a last reserved IP that is wrong if the range indexes changed.
	// This is not critical, we just lose round-robin this one time.
	lastReservedIP, err := a.store.LastReservedIP(a.rangeIdx)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Error retriving last reserved ip: %v", err)
	} else if lastReservedIP != nil {
		startFromLastReservedIP = a.ipRange.IPInRange(lastReservedIP) == nil
	}

	if startFromLastReservedIP {
		if i.high.Equal(lastReservedIP) {
			i.start = i.low
		} else {
			i.start = ip.NextIP(lastReservedIP)
		}
	} else {
		i.start = a.ipRange.RangeStart
	}
	return &i, nil
}

// Next returns the next IP in the iterator, or nil if end is reached
func (i *RangeIter) Next() net.IP {
	// If we're at the beginning, time to start
	if i.cur == nil {
		i.cur = i.start
		return i.cur
	}
	//  we returned .high last time, since we're inclusive
	if i.cur.Equal(i.high) {
		i.cur = i.low
	} else {
		i.cur = ip.NextIP(i.cur)
	}

	// If we've looped back to where we started, exit
	if i.cur.Equal(i.start) {
		return nil
	}

	return i.cur
}
