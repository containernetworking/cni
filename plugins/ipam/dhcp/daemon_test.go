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
	"os"
	"path"
	"syscall"
	"testing"
	"time"
)

func TestGetListener(t *testing.T) {
	l, err := getListener()
	if err != nil {
		t.Errorf("Unable to get listener: %v", err)
	}

	// verify existence of dhcp socket
	_, err = os.Stat(socketPath)
	if err != nil {
		t.Errorf("Unable to Stat socket %v: %v", socketPath, err)
		return
	}

	// clean up
	l.Close()
}

func TestSignalHandler(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("Unable to get current working directory: %v", err)
		return
	}
	var procAttr os.ProcAttr
	proc, err := os.StartProcess(path.Join(cwd, "../../../bin/dhcp"), []string{"dhcp", "daemon"}, &procAttr)
	if err != nil {
		t.Errorf("Unable to start process: %v", err)
		return
	}

	// verify existence of dhcp socket, keep retrying until 1 second has passed
	timeout := time.After(1 * time.Second)
	tick := time.Tick(10 * time.Millisecond)
retryloop:
	for {
		select {
		case <-timeout:
			t.Errorf("Timeout waiting for dhcp socket to present itself")
			return
		case <-tick:
			_, err = os.Stat(socketPath)
			if err == nil {
				break retryloop
			}
		}
	}

	// signal process
	err = proc.Signal(os.Interrupt)
	if err != nil {
		t.Errorf("Unable to kill process: %v", err)
		return
	}

	// wait for process cleanup to finish, check status
	procState, err := proc.Wait()
	status, ok := procState.Sys().(syscall.WaitStatus)
	if !ok {
		t.Errorf("Unable to convert procState.Sys to syscall.WaitStatus. Unsupported platform?")
	} else {
		signum := 0
		switch signal := os.Interrupt.(type) {
		case syscall.Signal:
			signum = int(signal)
		}
		if status.ExitStatus() != 128+signum {
			t.Errorf("Invalid exit status %v should be %v", status.ExitStatus(), 128+signum)
		}
	}

	// verify absence of dhcp socket
	_, err = os.Stat(socketPath)
	if err == nil {
		t.Errorf("Socket still exists when it should not!")
		return
	}
}
