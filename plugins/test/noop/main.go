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
	"github.com/containernetworking/cni/plugins/test/noop/debug"
)

func debugBehavior(args *skel.CmdArgs, command string) error {
	if !strings.HasPrefix(args.Args, "DEBUG=") {
		fmt.Printf(`{}`)
		os.Stderr.WriteString("CNI_ARGS empty, no debug behavior\n")
		return nil
	}
	debugFilePath := strings.TrimPrefix(args.Args, "DEBUG=")
	debug, err := debug.ReadDebug(debugFilePath)
	if err != nil {
		return err
	}

	debug.CmdArgs = *args
	debug.Command = command

	err = debug.WriteDebug(debugFilePath)
	if err != nil {
		return err
	}

	if debug.ReportError != "" {
		return errors.New(debug.ReportError)
	} else {
		os.Stdout.WriteString(debug.ReportResult)
	}

	return nil
}

func cmdAdd(args *skel.CmdArgs) error {
	return debugBehavior(args, "ADD")
}

func cmdDel(args *skel.CmdArgs) error {
	return debugBehavior(args, "DEL")
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
