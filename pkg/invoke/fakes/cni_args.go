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

package fakes

// CNIArgs represents fake CNIArgs.
type CNIArgs struct {
	AsEnvCall struct {
		Returns struct {
			Env []string
		}
	}
}

// AsEnv returns the fake CNIArgs as an array of stringified environment variables.
func (a *CNIArgs) AsEnv() []string {
	return a.AsEnvCall.Returns.Env
}
