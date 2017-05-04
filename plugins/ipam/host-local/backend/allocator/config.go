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
	"encoding/json"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
)

// IPAMConfig represents the IP related network configuration.
type IPAMConfig struct {
	Name       string
	Type       string      `json:"type"`
	RangeStart net.IP      `json:"rangeStart"`
	RangeEnd   net.IP      `json:"rangeEnd"`
	Subnet     types.IPNet `json:"subnet"`
	Gateway    net.IP      `json:"gateway"`
	Routes     []Route     `json:"routes"`
	DataDir    string      `json:"dataDir"`
	ResolvConf string      `json:"resolvConf"`
	Args       *IPAMArgs   `json:"-"`
}

type IPAMArgs struct {
	types.CommonArgs
	IP net.IP `json:"ip,omitempty"`
}

type Net struct {
	Name       string      `json:"name"`
	CNIVersion string      `json:"cniVersion"`
	IPAM       *IPAMConfig `json:"ipam"`
}

// Custom Route struct to handle old (GW) and new (NextHops) hop options
type Route struct {
	Dst types.IPNet `json:"dst"`
	// If NextHops is empty and GW is given, GW will be added as a NextHop
	NextHops []net.IP `json:"nextHops,omitempty"`
	// GW will be ignored if there are any NextHops
	GW net.IP `json:"gw,omitempty"`
}

func (r *Route) String() string {
	return fmt.Sprintf("%+v", *r)
}

// NewIPAMConfig creates a NetworkConfig from the given network name.
func LoadIPAMConfig(bytes []byte, args string) (*IPAMConfig, string, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return nil, "", err
	}

	if n.IPAM == nil {
		return nil, "", fmt.Errorf("IPAM config missing 'ipam' key")
	}

	if args != "" {
		n.IPAM.Args = &IPAMArgs{}
		err := types.LoadArgs(args, n.IPAM.Args)
		if err != nil {
			return nil, "", err
		}
	}

	// Copy net name into IPAM so not to drag Net struct around
	n.IPAM.Name = n.Name

	// Fix up routes to prefer NextHops over GW, but if NextHops is empty
	// and GW is given, use GW as the NextHop
	for i := range n.IPAM.Routes {
		r := &n.IPAM.Routes[i]
		if len(r.NextHops) == 0 {
			if r.GW != nil {
				r.NextHops = []net.IP{r.GW}
			}
		}
		r.GW = nil
	}

	return n.IPAM, n.CNIVersion, nil
}

func convertRoutesToCurrent(routes []Route) []*current.Route {
	var currentRoutes []*current.Route
	for _, r := range routes {
		currentRoutes = append(currentRoutes, &current.Route{
			Dst:      net.IPNet(r.Dst),
			NextHops: r.NextHops,
		})
	}
	return currentRoutes
}
