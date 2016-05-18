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

package ns_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/ns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"
)

func getCurrentThreadNetNSPath() string {
	pid := unix.Getpid()
	tid := unix.Gettid()
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", pid, tid)
}

func getInodeCurNetNS() (uint64, error) {
	return getInode(getCurrentThreadNetNSPath())
}

func getInodeNS(netns ns.NetNS) (uint64, error) {
	return getInodeFd(int(netns.Fd()))
}

func getInode(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return getInodeFd(int(file.Fd()))
}

func getInodeFd(fd int) (uint64, error) {
	stat := &unix.Stat_t{}
	err := unix.Fstat(fd, stat)
	return stat.Ino, err
}

var _ = Describe("Linux namespace operations", func() {
	Describe("WithNetNS", func() {
		var (
			hostInode     uint64
			originalNetNS ns.NetNS
			targetNetNS   ns.NetNS
		)

		var hostErr error
		hostInode, hostErr = getInodeCurNetNS()

		BeforeEach(func() {
			var err error

			Expect(hostErr).NotTo(HaveOccurred())

			originalNetNS, err = ns.NewNS()
			Expect(err).NotTo(HaveOccurred())
			err = originalNetNS.Set()
			Expect(err).NotTo(HaveOccurred())

			targetNetNS, err = ns.NewNS()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(targetNetNS.Close()).To(Succeed())
			Expect(originalNetNS.Close()).To(Succeed())
		})

		It("switches between namespaces correctly", func() {
			curInode, err := getInodeCurNetNS()
			Expect(err).NotTo(HaveOccurred())
			Expect(curInode).NotTo(Equal(hostInode))

			originalInode, err := getInodeNS(originalNetNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(curInode).To(Equal(originalInode))

			targetInode, err := getInodeNS(targetNetNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(targetInode).NotTo(Equal(originalInode))

			err = targetNetNS.Set()
			Expect(err).NotTo(HaveOccurred())
			curInode, err = getInodeCurNetNS()
			Expect(err).NotTo(HaveOccurred())
			Expect(curInode).To(Equal(targetInode))

			err = originalNetNS.Set()
			Expect(err).NotTo(HaveOccurred())
			curInode, err = getInodeCurNetNS()
			Expect(err).NotTo(HaveOccurred())
			Expect(curInode).To(Equal(originalInode))
		})

		It("executes the callback within the target network namespace", func() {
			expectedInode, err := getInodeNS(targetNetNS)
			Expect(err).NotTo(HaveOccurred())

			err = targetNetNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				actualInode, err := getInodeCurNetNS()
				Expect(err).NotTo(HaveOccurred())
				Expect(actualInode).To(Equal(expectedInode))
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("provides the original namespace as the argument to the callback", func() {
			hostNSInode, err := getInodeCurNetNS()
			Expect(err).NotTo(HaveOccurred())
			origNSInode, err := getInodeNS(originalNetNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostNSInode).To(Equal(origNSInode))

			err = targetNetNS.Do(func(inputNS ns.NetNS) error {
				inputNSInode, err := getInodeNS(inputNS)
				Expect(err).NotTo(HaveOccurred())
				Expect(inputNSInode).To(Equal(hostNSInode))
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("restores the calling thread to the original network namespace", func() {
			preTestInode, err := getInodeCurNetNS()
			Expect(err).NotTo(HaveOccurred())
			origNSInode, err := getInodeNS(originalNetNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(preTestInode).To(Equal(origNSInode))

			err = targetNetNS.Do(func(ns.NetNS) error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			postTestInode, err := getInodeCurNetNS()
			Expect(err).NotTo(HaveOccurred())

			Expect(postTestInode).To(Equal(preTestInode))
		})

		Context("when the callback returns an error", func() {
			It("restores the calling thread to the original namespace before returning", func() {
				preTestInode, err := getInodeCurNetNS()
				Expect(err).NotTo(HaveOccurred())
				origNSInode, err := getInodeNS(originalNetNS)
				Expect(err).NotTo(HaveOccurred())
				Expect(preTestInode).To(Equal(origNSInode))

				_ = targetNetNS.Do(func(ns.NetNS) error {
					return errors.New("potato")
				})

				postTestInode, err := getInodeCurNetNS()
				Expect(err).NotTo(HaveOccurred())

				Expect(postTestInode).To(Equal(preTestInode))
			})

			It("returns the error from the callback", func() {
				err := targetNetNS.Do(func(ns.NetNS) error {
					return errors.New("potato")
				})
				Expect(err).To(MatchError("potato"))
			})
		})

		Describe("validating inode mapping to namespaces", func() {
			It("checks that different namespaces have different inodes", func() {
				hostNSInode, err := getInodeCurNetNS()
				Expect(err).NotTo(HaveOccurred())
				origNSInode, err := getInodeNS(originalNetNS)
				Expect(err).NotTo(HaveOccurred())
				Expect(hostNSInode).To(Equal(origNSInode))

				testNsInode, err := getInodeNS(targetNetNS)
				Expect(err).NotTo(HaveOccurred())

				Expect(hostNSInode).NotTo(Equal(0))
				Expect(testNsInode).NotTo(Equal(0))
				Expect(testNsInode).NotTo(Equal(hostNSInode))
			})

			It("should not leak a new netns onto any threads in the process", func() {
				By("creating a new netns")
				createdNetNS, err := ns.NewNS()
				Expect(err).NotTo(HaveOccurred())
				defer createdNetNS.Close()
				// switch back to ensure no thread in the process uses this netns
				err = originalNetNS.Set()
				Expect(err).NotTo(HaveOccurred())

				By("discovering the inode of the created netns")
				createdNetNSInode, err := getInodeNS(createdNetNS)
				Expect(err).NotTo(HaveOccurred())

				By("comparing against the netns inode of every thread in the process")
				for _, netnsPath := range allNetNSInCurrentProcess() {
					netnsInode, err := getInode(netnsPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(netnsInode).NotTo(Equal(createdNetNSInode))
				}
			})
		})
	})
})

func allNetNSInCurrentProcess() []string {
	pid := unix.Getpid()
	paths, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/ns/net", pid))
	Expect(err).NotTo(HaveOccurred())
	return paths
}
