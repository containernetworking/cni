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
	"strings"
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
		{[]byte("example.com"), "example.com"},
		// "veryveryverylonglonglonglonglonglonglonglongname.com"
		{[]byte("veryveryverylonglonglonglonglonglonglonglongname.com"),
			"veryveryverylonglonglonglonglonglonglonglongname.com"},
		{[]byte("a.com"), "a.com"},
		{[]byte("a-b.com"), "a-b.com"},
		{[]byte("1-b.com"), "1-b.com"},
		{[]byte("1-3.com"), "1-3.com"},
		{[]byte("1a-3.com"), "1a-3.com"},
		{[]byte("1a-3.com"), "1a-3.com"},
		// 63 bytes long label
		{[]byte(strings.Repeat("a", 63) + ".com"),
			strings.Repeat("a", 63) + ".com"},
		{[]byte("1a-3.ca"), "1a-3.ca"},
		{[]byte("a-.com"), "a-.com"},
		{[]byte("-a.com"), "-a.com"},
		{[]byte("a.c"), "a.c"},
		// 64 bytes long label
		{[]byte(strings.Repeat("a", 64) + ".com"), ""},
		// over 255
		{[]byte(strings.Repeat("a", 63) + "." + strings.Repeat("a", 63) +
			"." + strings.Repeat("a", 63) + "." + strings.Repeat("a", 63) + ".aero"), ""},
	}

	for _, test := range tests {
		opts[dhcp4.OptionDomainName] = test.domainname
		domainname := parseDNSDomain(opts)
		validateDNSDomain(t, domainname, test.expected)
	}

}
