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

/*
Noop plugin is a CNI plugin designed for use as a test-double.

When calling, set the CNI_ARGS env var equal to the path of a file containing
the JSON encoding of a Debug.
*/

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"
)

// parse extra args i.e. FOO=BAR;ABC=123
func parseExtraArgs(args string) (map[string]string, error) {
	m := make(map[string]string)

	items := strings.Split(args, ";")
	for _, item := range items {
		kv := strings.Split(item, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("CNI_ARGS invalid key/value pair: %s\n", kv)
		}
		m[kv[0]] = kv[1]
	}
	return m, nil
}

func debugBehavior(args *skel.CmdArgs, command string) error {
	extraArgs, err := parseExtraArgs(args.Args)
	if err != nil {
		return err
	}

	debugFilePath, ok := extraArgs["DEBUG"]
	if !ok {
		fmt.Printf(`{}`)
		os.Stderr.WriteString("CNI_ARGS empty, no debug behavior\n")
		return nil
	}

	debug, err := noop_debug.ReadDebug(debugFilePath)
	if err != nil {
		return err
	}

	debug.CmdArgs = *args
	debug.Command = command

	if debug.ReportResult == "" {
		debug.ReportResult = fmt.Sprintf(` { "result": %q }`, noop_debug.EmptyReportResultMessage)
	}

	err = debug.WriteDebug(debugFilePath)
	if err != nil {
		return err
	}

	os.Stderr.WriteString(debug.ReportStderr)

	if debug.ReportError != "" {
		return errors.New(debug.ReportError)
	} else {
		os.Stdout.WriteString(debug.ReportResult)
	}

	return nil
}

func debugGetSupportedVersions() []string {
	vers := []string{"0.-42.0", "0.1.0", "0.2.0"}
	cniArgs := os.Getenv("CNI_ARGS")
	if cniArgs == "" {
		return vers
	}

	extraArgs, err := parseExtraArgs(cniArgs)
	if err != nil {
		panic("test setup error: invalid CNI_ARGS format")
	}

	debugFilePath, ok := extraArgs["DEBUG"]
	if !ok {
		panic("test setup error: missing DEBUG in CNI_ARGS")
	}

	debug, err := noop_debug.ReadDebug(debugFilePath)
	if err != nil {
		panic("test setup error: unable to read debug file: " + err.Error())
	}
	if debug.ReportVersionSupport == nil {
		return vers
	}
	return debug.ReportVersionSupport
}

func cmdAdd(args *skel.CmdArgs) error {
	return debugBehavior(args, "ADD")
}

func cmdDel(args *skel.CmdArgs) error {
	return debugBehavior(args, "DEL")
}

func main() {
	supportedVersions := debugGetSupportedVersions()
	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
