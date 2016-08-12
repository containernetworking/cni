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

package version

import (
	"encoding/json"
	"io"
)

// A PluginVersioner can encode information about its version
type PluginVersioner interface {
	Encode(io.Writer) error
}

// BasicVersioner is a PluginVersioner which reports a single cniVersion string
type BasicVersioner struct {
	CNIVersion string `json:"cniVersion"`
}

func (p *BasicVersioner) Encode(w io.Writer) error {
	return json.NewEncoder(w).Encode(p)
}

// Current reports the version of the CNI spec implemented by this library
func Current() string {
	return "0.2.0"
}

// DefaultPluginVersioner reports the Current library spec version as the cniVersion
var DefaultPluginVersioner = &BasicVersioner{CNIVersion: Current()}
