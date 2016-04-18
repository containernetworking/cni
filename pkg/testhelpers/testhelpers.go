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

// Package testhelpers provides common support behavior for tests
package testhelpers

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func GetInode(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return GetInodeF(file)
}

func GetInodeF(file *os.File) (uint64, error) {
	stat := &unix.Stat_t{}
	err := unix.Fstat(int(file.Fd()), stat)
	return stat.Ino, err
}

func MakeNetworkNS(containerID string) string {
	namespace := "/var/run/netns/" + containerID
	pid := unix.Getpid()
	tid := unix.Gettid()

	err := os.MkdirAll("/var/run/netns", 0600)
	Expect(err).NotTo(HaveOccurred())

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	go (func() {
		defer GinkgoRecover()

		err = unix.Unshare(unix.CLONE_NEWNET)
		Expect(err).NotTo(HaveOccurred())

		fd, err := os.Create(namespace)
		Expect(err).NotTo(HaveOccurred())
		defer fd.Close()

		err = unix.Mount("/proc/self/ns/net", namespace, "none", unix.MS_BIND, "")
		Expect(err).NotTo(HaveOccurred())
	})()

	Eventually(namespace).Should(BeAnExistingFile())

	fd, err := unix.Open(fmt.Sprintf("/proc/%d/task/%d/ns/net", pid, tid), unix.O_RDONLY, 0)
	Expect(err).NotTo(HaveOccurred())

	defer unix.Close(fd)

	_, _, e1 := unix.Syscall(unix.SYS_SETNS, uintptr(fd), uintptr(unix.CLONE_NEWNET), 0)
	Expect(e1).To(BeZero())

	return namespace
}

func RemoveNetworkNS(networkNS string) error {
	err := unix.Unmount(networkNS, unix.MNT_DETACH)

	err = os.RemoveAll(networkNS)
	return err
}
