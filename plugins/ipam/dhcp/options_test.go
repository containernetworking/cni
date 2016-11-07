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
	"net"
	"testing"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/d2g/dhcp4"
)

func validateRoutes(t *testing.T, routes []types.Route) {
	expected := []types.Route{
		types.Route{
			Dst: net.IPNet{
				IP:   net.IPv4(10, 0, 0, 0),
				Mask: net.CIDRMask(8, 32),
			},
			GW: net.IPv4(10, 1, 2, 3),
		},
		types.Route{
			Dst: net.IPNet{
				IP:   net.IPv4(192, 168, 1, 0),
				Mask: net.CIDRMask(24, 32),
			},
			GW: net.IPv4(192, 168, 2, 3),
		},
	}

	if len(routes) != len(expected) {
		t.Fatalf("wrong length slice; expected %v, got %v", len(expected), len(routes))
	}

	for i := 0; i < len(routes); i++ {
		a := routes[i]
		e := expected[i]

		if a.Dst.String() != e.Dst.String() {
			t.Errorf("route.Dst mismatch: expected %v, got %v", e.Dst, a.Dst)
		}

		if !a.GW.Equal(e.GW) {
			t.Errorf("route.GW mismatch: expected %v, got %v", e.GW, a.GW)
		}
	}
}

func TestParseRoutes(t *testing.T) {
	opts := make(dhcp4.Options)
	opts[dhcp4.OptionStaticRoute] = []byte{10, 0, 0, 0, 10, 1, 2, 3, 192, 168, 1, 0, 192, 168, 2, 3}
	routes := parseRoutes(opts)

	validateRoutes(t, routes)
}

func TestParseCIDRRoutes(t *testing.T) {
	opts := make(dhcp4.Options)
	opts[dhcp4.OptionClasslessRouteFormat] = []byte{8, 10, 10, 1, 2, 3, 24, 192, 168, 1, 192, 168, 2, 3}
	routes := parseCIDRRoutes(opts)

	validateRoutes(t, routes)
}

func validateDNSServers(t *testing.T, nameservers []string, expected []string) {
	if len(nameservers) != len(expected) {
		t.Fatalf("wrong number of records; expected %v, got %v",
			len(expected), len(nameservers))
	}

	for i := 0; i < len(nameservers); i++ {

		if nameservers[i] != expected[i] {
			t.Errorf("nameserver mismatch: expected %v, got %v",
				expected[i], nameservers[i])
		}
	}
}

func TestParseDNSServers(t *testing.T) {
	opts := make(dhcp4.Options)

	var tests = []struct {
		nameservers []byte
		expected    []string
	}{
		{[]byte{8, 8, 8, 8, 8, 8, 4, 4, 1, 2, 3, 4}, []string{"8.8.8.8", "8.8.4.4", "1.2.3.4"}},
		{[]byte{8, 8, 8, 8, 8, 8, 4, 4}, []string{"8.8.8.8", "8.8.4.4"}},
		{[]byte{8, 8, 8, 8, 4, 4}, []string{"8.8.8.8"}},
		{[]byte{8, 8, 8, 8, 4}, []string{"8.8.8.8"}},
		{[]byte{8, 8, 8}, nil},
	}

	for _, test := range tests {
		opts[dhcp4.OptionDomainNameServer] = test.nameservers
		var nameservers = parseDNSServers(opts)
		validateDNSServers(t, nameservers, test.expected)
	}
}

func validateDNSDomain(t *testing.T, domainname string, expected string) {
	if expected != domainname {
		t.Errorf("domain name mismatch: expected %v, got %v",
			expected, domainname)
	}
}

func TestParseDNSDomain(t *testing.T) {
	opts := make(dhcp4.Options)

	var tests = []struct {
		domainname []byte
		expected   string
	}{
		// Let's add example.com
		// python -c "print [ord(i) for i in 'example.com']"
		{[]byte{101, 120, 97, 109, 112, 108, 101, 46, 99, 111, 109}, "example.com"},
		// "veryveryverylonglonglonglonglonglonglonglongname.com"
		{[]byte{118, 101, 114, 121, 118, 101, 114, 121, 118, 101, 114, 121, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 110, 97, 109, 101, 46, 99, 111, 109},
			"veryveryverylonglonglonglonglonglonglonglongname.com"},
		// 64 bytes label: python -c 'print [ord(i) for i in "long"*16+".new.com"]'
		{[]byte{108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 108, 111, 110, 103, 108, 111, 110, 103, 46, 110, 101, 119, 46, 99,
			111, 109}, ""},
		// over 255
		{[]byte{108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 108, 111, 110, 103, 46, 108, 111, 110, 103, 108, 111, 110, 103,
			108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111,
			110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103,
			108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111,
			110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 46, 108, 111, 110,
			103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108,
			111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110,
			103, 46, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103,
			108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111,
			110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103,
			108, 111, 110, 103, 108, 111, 110, 103, 108, 111, 110, 103, 108, 111,
			110, 103, 108, 111, 110, 103, 46, 109, 111, 114, 101, 116, 104, 97, 110,
			50, 53, 53, 98, 121, 116, 101, 115, 46, 99, 111, 109}, ""},
	}

	for _, test := range tests {
		opts[dhcp4.OptionDomainName] = test.domainname
		domainname := parseDNSDomain(opts)
		validateDNSDomain(t, domainname, test.expected)
	}

}
