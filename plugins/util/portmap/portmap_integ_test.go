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

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/testutils"
	"github.com/containernetworking/cni/pkg/types/current"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("portmap integration tests", func() {

	var configList *libcni.NetworkConfigList
	var cniConf *libcni.CNIConfig
	var targetNS ns.NetNS
	var containerPort int
	var closeChan chan interface{}

	BeforeEach(func() {
		var err error
		rawConfig := `{
	"cniVersion": "0.3.0",
	"name": "cni-portmap-unit-test",
	"plugins": [
		{
			"type": "ptp",
			"ipMasq": true,
			"ipam": {
				"type": "host-local",
				"subnet": "172.16.31.0/24"
			}
		},
		{
			"type": "portmap",
			"capabilities": {
				"portMappings": true
			}
		}
	]
}`

		configList, err = libcni.ConfListFromBytes([]byte(rawConfig))
		Expect(err).NotTo(HaveOccurred())

		// turn PATH in to CNI_PATH
		dirs := filepath.SplitList(os.Getenv("PATH"))
		cniConf = &libcni.CNIConfig{Path: dirs}

		targetNS, err = ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprintln(GinkgoWriter, "namespace:", targetNS.Path())

		// Start an echo server and get the port
		containerPort, closeChan, err = testutils.RunEchoServerInNS(targetNS)
		Expect(err).NotTo(HaveOccurred())

	})

	AfterEach(func() {
		if targetNS != nil {
			targetNS.Close()
		}
	})

	// This needs to be done using Ginkgo's asynchronous testing mode.
	It("forwards a TCP port on ipv4", func(done Done) {
		var err error
		hostPort := 9999
		runtimeConfig := libcni.RuntimeConf{
			ContainerID: "unit-test",
			NetNS:       targetNS.Path(),
			IfName:      "eth0",
			CapabilityArgs: map[string]interface{}{
				"portMappings": []map[string]interface{}{
					{
						"hostPort":      hostPort,
						"containerPort": containerPort,
						"protocol":      "tcp",
					},
				},
			},
		}

		// Make delete idempotent, so we can clean up on failure
		netDeleted := false
		deleteNetwork := func() error {
			if netDeleted {
				return nil
			}
			netDeleted = true
			return cniConf.DelNetworkList(configList, &runtimeConfig)
		}

		// Create the network
		resI, err := cniConf.AddNetworkList(configList, &runtimeConfig)
		Expect(err).NotTo(HaveOccurred())
		defer deleteNetwork()

		result, err := current.GetResult(resI)
		Expect(err).NotTo(HaveOccurred())
		var contIP net.IP

		for _, ip := range result.IPs {
			if result.Interfaces[ip.Interface].Sandbox == "" {
				continue
			}
			contIP = ip.Address.IP
		}
		if contIP == nil {
			Fail("could not determine container IP")
		}

		// Sanity check: verify that the container is reachable directly
		contOK := testEchoServer(fmt.Sprintf("%s:%d", contIP.String(), containerPort))

		// Verify that a connection to the forwarded port works
		hostIP := getLocalIP()
		dnatOK := testEchoServer(fmt.Sprintf("%s:%d", hostIP, hostPort))
		snatOK := testEchoServer(fmt.Sprintf("%s:%d", "127.0.0.1", hostPort))

		// Cleanup
		close(closeChan)
		err = deleteNetwork()
		Expect(err).NotTo(HaveOccurred())

		// Check that everything succeeded *after* we clean up the network
		if !contOK {
			Fail("connection direct to " + contIP.String() + " failed")
		}
		if !dnatOK {
			Fail("Connection to " + hostIP + " was not forwarded")
		}
		if !snatOK {
			Fail("connection to 127.0.0.1 was not forwarded")
		}

		close(done)

	}, 5)
})

// testEchoServer returns true if we found an echo server on the port
func testEchoServer(address string) bool {
	fmt.Println("Testing connection to", address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, "connection to ", address, "failed:", err)
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintln(GinkgoWriter, "connected to ", address)

	message := "Aliquid melius quam pessimum optimum non est."
	_, err = fmt.Fprint(conn, message)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, "sending message to", address, " failed:", err)
		return false
	}

	fmt.Fprintln(GinkgoWriter, "reading...")
	response := make([]byte, len(message))
	_, err = conn.Read(response)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, "receiving message from", address, " failed:", err)
		return false
	}

	if string(response) == message {
		return true
	}
	fmt.Fprintln(GinkgoWriter, "returned message didn't match?")
	return false
}

func getLocalIP() string {
	addrs, err := netlink.AddrList(nil, netlink.FAMILY_V4)
	Expect(err).NotTo(HaveOccurred())

	for _, addr := range addrs {
		return addr.IP.String()
	}
	Fail("no live addresses")
	return ""
}
