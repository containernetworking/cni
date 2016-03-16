package main

import (
	"fmt"
	"os"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"golang.org/x/sys/unix"
	"testing"
)

func TestHostLocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HostLocal Suite")
}

var pathToHostLocalPlugin string

var _ = BeforeSuite(func() {
	var err error
	pathToHostLocalPlugin, err = gexec.Build("github.com/appc/cni/plugins/ipam/host-local")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func makeNetworkNS(containerID string) string {
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

func removeNetworkNS(networkNS string) error {
	err := unix.Unmount(networkNS, unix.MNT_DETACH)

	err = os.RemoveAll(networkNS)
	return err
}
