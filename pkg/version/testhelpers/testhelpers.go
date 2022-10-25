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

// Package testhelpers supports testing of CNI components of different versions
//
// For example, to build a plugin against an old version of the CNI library,
// we can pass the plugin's source and the old git commit reference to BuildAt.
// We could then test how the built binary responds when called by the latest
// version of this library.
package testhelpers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const packageBaseName = "github.com/containernetworking/cni"

func run(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err != nil {
		command := strings.Join(cmd.Args, " ")
		return fmt.Errorf("running %q: %s", command, out)
	}
	return nil
}

// unset GOPATH if it's set, so we use modules
func goBuildEnviron() []string {
	out := []string{}
	for _, kvp := range os.Environ() {
		if !strings.HasPrefix(kvp, "GOPATH=") {
			out = append(out, kvp)
		}
	}
	return out
}

func buildGoProgram(modPath, outputFilePath string) error {
	cmd := exec.Command("go", "build", "-o", outputFilePath, ".")
	cmd.Dir = modPath
	cmd.Env = goBuildEnviron()
	return run(cmd)
}

func modInit(path, name string) error {
	cmd := exec.Command("go", "mod", "init", name)
	cmd.Dir = path
	return run(cmd)
}

// addLibcni will execute `go mod edit -replace` to fix libcni at a specified version
func addLibcni(path, gitRef string) error {
	cmd := exec.Command("go", "mod", "edit", "-replace=github.com/containernetworking/cni=github.com/containernetworking/cni@"+gitRef)
	cmd.Dir = path
	return run(cmd)
}

// modTidy will execute `go mod tidy` to ensure all necessary dependencies
func modTidy(path string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = path
	return run(cmd)
}

// BuildAt builds the go programSource using the version of the CNI library
// at gitRef, and saves the resulting binary file at outputFilePath
func BuildAt(programSource []byte, gitRef string, outputFilePath string) error {
	tempDir, err := ioutil.TempDir(os.Getenv("GOTMPDIR"), "cni-test-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	modName := filepath.Base(tempDir)

	if err := modInit(tempDir, modName); err != nil {
		return err
	}

	if err := addLibcni(tempDir, gitRef); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(tempDir, "main.go"), programSource, 0600); err != nil {
		return err
	}

	if err := modTidy(tempDir); err != nil {
		return err
	}

	err = buildGoProgram(tempDir, outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	return nil
}
