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

// debug supports tests that use the noop plugin
package debug

import (
	"encoding/json"
	"os"

	"github.com/containernetworking/cni/pkg/skel"
)

const EmptyReportResultMessage = "set debug.ReportResult and call debug.WriteDebug() before calling this plugin"

// Debug is used to control and record the behavior of the noop plugin
type Debug struct {
	// Report* fields allow the test to control the behavior of the no-op plugin
	ReportResult         string
	ReportError          string
	ReportErrorCode      uint
	ReportStderr         string
	ReportVersionSupport []string
	ExitWithCode         int

	// Command stores the CNI command that the plugin received
	Command string

	// CmdArgs stores the CNI Args and Env Vars that the plugin received
	CmdArgs skel.CmdArgs
}

// CmdLogEntry records a single CNI command as well as its args
type CmdLogEntry struct {
	Command string
	CmdArgs skel.CmdArgs
}

// CmdLog records a list of CmdLogEntry received by the noop plugin
type CmdLog []CmdLogEntry

// ReadDebug will return a debug file recorded by the noop plugin
func ReadDebug(debugFilePath string) (*Debug, error) {
	debugBytes, err := os.ReadFile(debugFilePath)
	if err != nil {
		return nil, err
	}

	var debug Debug
	err = json.Unmarshal(debugBytes, &debug)
	if err != nil {
		return nil, err
	}

	return &debug, nil
}

// WriteDebug will create a debug file to control the noop plugin
func (debug *Debug) WriteDebug(debugFilePath string) error {
	debugBytes, err := json.Marshal(debug)
	if err != nil {
		return err
	}

	err = os.WriteFile(debugFilePath, debugBytes, 0o600)
	if err != nil {
		return err
	}

	return nil
}

// WriteCommandLog appends the executed cni command to the record file
func WriteCommandLog(path string, entry CmdLogEntry) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cmds CmdLog
	if len(buf) > 0 {
		if err = json.Unmarshal(buf, &cmds); err != nil {
			return err
		}
	}
	cmds = append(cmds, entry)
	if buf, err = json.Marshal(&cmds); err != nil {
		return nil
	}
	return os.WriteFile(path, buf, 0o644)
}

func ReadCommandLog(path string) (CmdLog, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cmds CmdLog
	if err = json.Unmarshal(buf, &cmds); err != nil {
		return nil, err
	}
	return cmds, nil
}
