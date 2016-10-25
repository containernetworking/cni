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
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
)

// IPAMConfig represents the IP related network configuration.
type IPAMConfig struct {
	Name       string
	Version    string        `json:"version"`
	Type       string        `json:"type"`
	RangeStart net.IP        `json:"rangeStart"`
	RangeEnd   net.IP        `json:"rangeEnd"`
	Subnet     types.IPNet   `json:"subnet"`
	Gateway    net.IP        `json:"gateway"`
	Routes     []types.Route `json:"routes"`
	Args       *IPAMArgs     `json:"-"`
}

type IPAMArgs struct {
	types.CommonArgs
	IP net.IP `json:"ip,omitempty"`
}

type Net struct {
	Name  string      `json:"name"`
	IPAM  *IPAMConfig `json:"ipam,omitempty"`
	IPAM6 *IPAMConfig `json:"ipam6,omitempty"`
}

// LoadIPAMConfig unmarshals a given byte slice to a Net object
func LoadIPAMConfig(bytes []byte, args string) (*IPAMConfig, *IPAMConfig, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return nil, nil, err
	}

	if args != "" {
		if n.IPAM != nil {
			n.IPAM.Args = &IPAMArgs{}
			err := types.LoadArgs(args, n.IPAM.Args)
			if n.IPAM.Version != "4" {
				return nil, nil, fmt.Errorf("Version in the IPAM struct should be 4")
			}
			if err != nil {
				return nil, nil, err
			}
			n.IPAM.Name = n.Name + n.IPAM.Version
		}
		if n.IPAM6 != nil {
			n.IPAM6.Args = &IPAMArgs{}
			err := types.LoadArgs(args, n.IPAM6.Args)
			if n.IPAM6.Version != "6" {
				return nil, nil, fmt.Errorf("Version in the IPAM6 struct should be 6")
			}
			if err != nil {
				return nil, nil, err
			}
			n.IPAM6.Name = n.Name + n.IPAM6.Version
		}
	}

	if n.IPAM == nil && n.IPAM6 == nil {
		return nil, nil, fmt.Errorf("Need at least one of ipam or ipam6 key")
	}

	return n.IPAM, n.IPAM6, nil
}
