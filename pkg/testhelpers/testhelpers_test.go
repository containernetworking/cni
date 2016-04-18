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

// Package testhelpers_test contains unit tests of the testhelpers
//
// Some of this stuff is non-trivial and can interact in surprising ways
// with the Go runtime.  Better be safe.
package testhelpers_test

import (
	"fmt"
	"math/rand"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/appc/cni/pkg/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test helper functions", func() {
	Describe("MakeNetworkNS", func() {
		It("should return the filepath to a network namespace", func() {
			containerID := fmt.Sprintf("c-%x", rand.Int31())
			nsPath := testhelpers.MakeNetworkNS(containerID)

			Expect(nsPath).To(BeAnExistingFile())

			testhelpers.RemoveNetworkNS(containerID)
		})

		It("should return a network namespace different from that of the caller", func() {
			containerID := fmt.Sprintf("c-%x", rand.Int31())

			By("discovering the inode of the current netns")
			originalNetNSPath := currentNetNSPath()
			originalNetNSInode, err := testhelpers.GetInode(originalNetNSPath)
			Expect(err).NotTo(HaveOccurred())

			By("creating a new netns")
			createdNetNSPath := testhelpers.MakeNetworkNS(containerID)
			defer testhelpers.RemoveNetworkNS(createdNetNSPath)

			By("discovering the inode of the created netns")
			createdNetNSInode, err := testhelpers.GetInode(createdNetNSPath)
			Expect(err).NotTo(HaveOccurred())

			By("comparing the inodes")
			Expect(createdNetNSInode).NotTo(Equal(originalNetNSInode))
		})

		It("should not leak the new netns onto any threads in the process", func() {
			containerID := fmt.Sprintf("c-%x", rand.Int31())

			By("creating a new netns")
			createdNetNSPath := testhelpers.MakeNetworkNS(containerID)
			defer testhelpers.RemoveNetworkNS(createdNetNSPath)

			By("discovering the inode of the created netns")
			createdNetNSInode, err := testhelpers.GetInode(createdNetNSPath)
			Expect(err).NotTo(HaveOccurred())

			By("comparing against the netns inode of every thread in the process")
			for _, netnsPath := range allNetNSInCurrentProcess() {
				netnsInode, err := testhelpers.GetInode(netnsPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(netnsInode).NotTo(Equal(createdNetNSInode))
			}
		})
	})
})

func currentNetNSPath() string {
	pid := unix.Getpid()
	tid := unix.Gettid()
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", pid, tid)
}

func allNetNSInCurrentProcess() []string {
	pid := unix.Getpid()
	paths, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/ns/net", pid))
	Expect(err).NotTo(HaveOccurred())
	return paths
}
