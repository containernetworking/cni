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

package ns

import (
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"runtime"
	"sync"

	"golang.org/x/sys/unix"
)

type NetNS interface {
	// Executes the passed closure in this object's network namespace,
	// restoring the original namespace afterwards. If the closure returns
	// an error, Do attempts to restore the original namespace before
	// returning.
	Do(toRun func(NetNS) error) error

	// Sets the current network namespace to this object's network namespace
	Set() error

	// Returns the filesystem path representing this object's network namespace
	Path() string

	// Returns a file descriptor representing this object's network namespace
	Fd() uintptr

	// Cleans up this instance of the network namespace; if this instance
	// is the last user the namespace will be destroyed
	Close() error
}

type netNS struct {
	file    *os.File
	mounted bool
}

func getCurrentThreadNetNSPath() string {
	// /proc/self/ns/net returns the namespace of the main thread, not
	// of whatever thread this goroutine is running on.  Make sure we
	// use the thread's net namespace since the thread is switching around
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), unix.Gettid())
}

// Returns an object representing the namespace referred to by @path
func GetNS(nspath string) (NetNS, error) {
	fd, err := os.Open(nspath)
	if err != nil {
		return nil, err
	}
	return &netNS{file: fd}, nil
}

// Creates a new persistent network namespace and returns an object
// representing that namespace, without switching to it
func NewNS() (NetNS, error) {
	const nsRunDir = "/var/run/netns"

	b := make([]byte, 16)
	_, err := rand.Reader.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random netns name: %v", err)
	}

	err = os.MkdirAll(nsRunDir, 0755)
	if err != nil {
		return nil, err
	}

	// create an empty file at the mount point
	nsName := fmt.Sprintf("cni-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	nsPath := path.Join(nsRunDir, nsName)
	mountPointFd, err := os.Create(nsPath)
	if err != nil {
		return nil, err
	}
	mountPointFd.Close()

	// Ensure the mount point is cleaned up on errors; if the namespace
	// was successfully mounted this will have no effect because the file
	// is in-use
	defer os.RemoveAll(nsPath)

	var wg sync.WaitGroup
	wg.Add(1)

	// do namespace work in a dedicated goroutine, so that we can safely
	// Lock/Unlock OSThread without upsetting the lock/unlock state of
	// the caller of this function
	var fd *os.File
	go (func() {
		defer wg.Done()
		runtime.LockOSThread()

		var origNS NetNS
		origNS, err = GetNS(getCurrentThreadNetNSPath())
		if err != nil {
			return
		}
		defer origNS.Close()

		// create a new netns on the current thread
		err = unix.Unshare(unix.CLONE_NEWNET)
		if err != nil {
			return
		}
		defer origNS.Set()

		// bind mount the new netns from the current thread onto the mount point
		err = unix.Mount(getCurrentThreadNetNSPath(), nsPath, "none", unix.MS_BIND, "")
		if err != nil {
			return
		}

		fd, err = os.Open(nsPath)
		if err != nil {
			return
		}
	})()
	wg.Wait()

	if err != nil {
		unix.Unmount(nsPath, unix.MNT_DETACH)
		return nil, fmt.Errorf("failed to create namespace: %v", err)
	}

	return &netNS{file: fd, mounted: true}, nil
}

func (ns *netNS) Path() string {
	return ns.file.Name()
}

func (ns *netNS) Fd() uintptr {
	return ns.file.Fd()
}

func (ns *netNS) Close() error {
	ns.file.Close()

	if ns.mounted {
		if err := unix.Unmount(ns.file.Name(), unix.MNT_DETACH); err != nil {
			return fmt.Errorf("Failed to unmount namespace %s: %v", ns.file.Name(), err)
		}
		if err := os.RemoveAll(ns.file.Name()); err != nil {
			return fmt.Errorf("Failed to clean up namespace %s: %v", ns.file.Name(), err)
		}
	}
	return nil
}

func (ns *netNS) Do(toRun func(NetNS) error) error {
	containedCall := func(hostNS NetNS) error {
		threadNS, err := GetNS(getCurrentThreadNetNSPath())
		if err != nil {
			return fmt.Errorf("failed to open current netns: %v", err)
		}
		defer threadNS.Close()

		// switch to target namespace
		if err = ns.Set(); err != nil {
			return fmt.Errorf("error switching to ns %v: %v", ns.file.Name(), err)
		}
		defer threadNS.Set() // switch back

		return toRun(hostNS)
	}

	// save a handle to current network namespace
	hostNS, err := GetNS(getCurrentThreadNetNSPath())
	if err != nil {
		return fmt.Errorf("Failed to open current namespace: %v", err)
	}
	defer hostNS.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	var innerError error
	go func() {
		defer wg.Done()
		runtime.LockOSThread()
		innerError = containedCall(hostNS)
	}()
	wg.Wait()

	return innerError
}

func (ns *netNS) Set() error {
	if _, _, err := unix.Syscall(unix.SYS_SETNS, ns.Fd(), uintptr(unix.CLONE_NEWNET), 0); err != 0 {
		return fmt.Errorf("Error switching to ns %v: %v", ns.file.Name(), err)
	}

	return nil
}

// WithNetNSPath executes the passed closure under the given network
// namespace, restoring the original namespace afterwards.
func WithNetNSPath(nspath string, toRun func(NetNS) error) error {
	ns, err := GetNS(nspath)
	if err != nil {
		return fmt.Errorf("Failed to open %v: %v", nspath, err)
	}
	defer ns.Close()
	return ns.Do(toRun)
}
