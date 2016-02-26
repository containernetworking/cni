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

// Package firewalld provides functions to integrate with Fedora's FirewallD.
// See https://fedoraproject.org/wiki/FirewallD
package firewalld

import (
	"net"

	"github.com/godbus/dbus"
)

const (
	firewalldName          = "org.fedoraproject.FirewallD1"
	firewalldPath          = "/org/fedoraproject/FirewallD1"
	firewalldZoneInterface = "org.fedoraproject.FirewallD1.zone"
)

func AddTrustedSource(source net.IP, zone string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	firewalldObj := conn.Object(firewalldName, firewalldPath)
	var res string
	err = firewalldObj.Call(firewalldZoneInterface+".addSource", 0, zone, source.String()).Store(&res)
	if err != nil {
		return err
	}

	return nil
}

func RemoveTrustedSource(source net.IP, zone string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	firewalldObj := conn.Object(firewalldName, firewalldPath)
	var res string
	err = firewalldObj.Call(firewalldZoneInterface+".removeSource", 0, zone, source.String()).Store(&res)
	if err != nil {
		return err
	}

	return nil
}
