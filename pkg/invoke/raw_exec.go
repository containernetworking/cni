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

package invoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/containernetworking/cni/pkg/types"
)

var ErrorPluginExecTimeout = fmt.Errorf("Waiting for plugin to return timed out")

type RawExec struct {
	Stderr io.Writer
}

func (e *RawExec) ExecPluginWithTimeout(pluginPath string, stdinData []byte, environ []string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	stdout := &bytes.Buffer{}
	cmd, err := e.start(pluginPath, stdinData, stdout, environ)
	if err != nil {
		return nil, pluginErr(err, stdout.Bytes())
	}
	cmdDone := make(chan error)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// kill the plugin process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, pluginErr(ErrorPluginExecTimeout, stdout.Bytes())
	case err := <-cmdDone:
		if err != nil {
			return nil, pluginErr(err, stdout.Bytes())
		}
		return stdout.Bytes(), nil
	}
}

func (e *RawExec) ExecPlugin(pluginPath string, stdinData []byte, environ []string) ([]byte, error) {
	stdout := &bytes.Buffer{}
	cmd, err := e.start(pluginPath, stdinData, stdout, environ)
	if err != nil {
		return nil, pluginErr(err, stdout.Bytes())
	}

	err = cmd.Wait()
	if err != nil {
		return nil, pluginErr(err, stdout.Bytes())
	}

	return stdout.Bytes(), nil
}

func (e *RawExec) start(pluginPath string, stdinData []byte, stdout *bytes.Buffer, environ []string) (exec.Cmd, error) {
	c := exec.Cmd{
		Env:    environ,
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Stdin:  bytes.NewBuffer(stdinData),
		Stdout: stdout,
		Stderr: e.Stderr,
	}

	err := c.Start()
	return c, err
}

func pluginErr(err error, output []byte) error {
	if _, ok := err.(*exec.ExitError); ok {
		emsg := types.Error{}
		if perr := json.Unmarshal(output, &emsg); perr != nil {
			emsg.Msg = fmt.Sprintf("netplugin failed but error parsing its diagnostic message %q: %v", string(output), perr)
		}
		return &emsg
	}

	return err
}
