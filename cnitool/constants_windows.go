// Copyright 2023 CNI authors
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

const (
	// NOTE: this is the most reasonable default as most CNI setups on Windows
	// will have been made using the helper script in the containerd repo.
	// https://github.com/containerd/containerd/blob/main/script/setup/install-cni-windows
	DefaultNetDir = "C:\\Program Files\\containerd\\cni\\conf"
)
