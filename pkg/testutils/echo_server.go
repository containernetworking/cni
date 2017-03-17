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
// limitations under the License.package testutils

package testutils

import (
	"fmt"
	"net"
	"time"

	"github.com/containernetworking/cni/pkg/ns"
)

// OpenEchoServer opens a server that handles one connection before closing.
// It opens on a random port and sends the port number on portChan when
// the server is up and running. If an error is encountered, closes portChan.
// If closeChan is closed, closes the socket.
func OpenEchoServer(portChan chan<- int, closeChan <-chan interface{}) error {
	laddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	if err != nil {
		close(portChan)
		return err
	}
	sock, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		close(portChan)
		return err
	}
	defer sock.Close()

	switch addr := sock.Addr().(type) {
	case *net.TCPAddr:
		portChan <- addr.Port
	default:
		close(portChan)
		return fmt.Errorf("addr cast failed!")
	}

	for {
		select {
		case <-closeChan:
			return nil
		default:
		}

		sock.SetDeadline(time.Now().Add(time.Second))
		con, err := sock.AcceptTCP()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			return err
		}
		defer con.Close()

		buf := make([]byte, 4096)
		_, readErr := con.Read(buf)
		_, writeErr := con.Write(buf)
		if readErr != nil || writeErr != nil {
			return nil
		}
	}
}

func RunEchoServerInNS(netNS ns.NetNS) (int, chan interface{}, error) {
	portChan := make(chan int)
	closeChan := make(chan interface{})

	go func() {
		err := netNS.Do(func(ns.NetNS) error {
			OpenEchoServer(portChan, closeChan)
			return nil
		})
		// Somehow the ns.Do failed
		if err != nil {
			close(portChan)
		}
	}()

	portNum := <-portChan
	if portNum == 0 {
		return 0, nil, fmt.Errorf("failed to execute server")
	}

	return portNum, closeChan, nil
}
