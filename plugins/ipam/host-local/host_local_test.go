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

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/testutils"
	"github.com/containernetworking/cni/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("host-local Operations", func() {
	It("allocates and releases an address with ADD/DEL", func() {
		const ifname string = "eth0"
		const nspath string = "/some/where"

		tmpDir, err := ioutil.TempDir("", "host_local_artifacts")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		err = ioutil.WriteFile(filepath.Join(tmpDir, "resolv.conf"), []byte("nameserver 192.0.2.3"), 0644)
		Expect(err).NotTo(HaveOccurred())

		conf := fmt.Sprintf(`{
    "cniVersion": "0.2.0",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24",
        "dataDir": "%s",
		"resolvConf": "%s/resolv.conf"
    }
}`, tmpDir, tmpDir)

		args := &skel.CmdArgs{
			ContainerID: "dummy",
			Netns:       nspath,
			IfName:      ifname,
			StdinData:   []byte(conf),
		}

		// Allocate the IP
		result, err := testutils.CmdAddWithResult(nspath, ifname, func() error {
			return cmdAdd(args)
		})
		Expect(err).NotTo(HaveOccurred())

		expectedAddress, err := types.ParseCIDR("10.1.2.2/24")
		Expect(err).NotTo(HaveOccurred())
		expectedAddress.IP = expectedAddress.IP.To16()
		Expect(result.IP4.IP).To(Equal(*expectedAddress))

		Expect(result.IP4.Gateway).To(Equal(net.ParseIP("10.1.2.1")))

		ipFilePath := filepath.Join(tmpDir, "mynet", "10.1.2.2")
		contents, err := ioutil.ReadFile(ipFilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(contents)).To(Equal("dummy"))

		lastFilePath := filepath.Join(tmpDir, "mynet", "last_reserved_ip")
		contents, err = ioutil.ReadFile(lastFilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(contents)).To(Equal("10.1.2.2"))

		Expect(result.DNS).To(Equal(types.DNS{Nameservers: []string{"192.0.2.3"}}))

		// Release the IP
		err = testutils.CmdDelWithResult(nspath, ifname, func() error {
			return cmdDel(args)
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = os.Stat(ipFilePath)
		Expect(err).To(HaveOccurred())
	})

	It("ignores whitespace in disk files", func() {
		const ifname string = "eth0"
		const nspath string = "/some/where"

		tmpDir, err := ioutil.TempDir("", "host_local_artifacts")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		conf := fmt.Sprintf(`{
    "cniVersion": "0.2.0",
    "name": "mynet",
    "type": "ipvlan",
    "master": "foo0",
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.2.0/24",
        "dataDir": "%s"
    }
}`, tmpDir)

		args := &skel.CmdArgs{
			ContainerID: "   dummy\n ",
			Netns:       nspath,
			IfName:      ifname,
			StdinData:   []byte(conf),
		}

		// Allocate the IP
		result, err := testutils.CmdAddWithResult(nspath, ifname, func() error {
			return cmdAdd(args)
		})
		Expect(err).NotTo(HaveOccurred())

		ipFilePath := filepath.Join(tmpDir, "mynet", result.IP4.IP.IP.String())
		contents, err := ioutil.ReadFile(ipFilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(contents)).To(Equal("dummy"))

		// Release the IP
		err = testutils.CmdDelWithResult(nspath, ifname, func() error {
			return cmdDel(args)
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = os.Stat(ipFilePath)
		Expect(err).To(HaveOccurred())
	})
})
