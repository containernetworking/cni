// Copyright 2019 CNI authors
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

package utils

import (
	"fmt"
	"regexp"

	"github.com/containernetworking/cni/pkg/types"
)

// ValidInputString is the regexp used to validate valid characters in
// containerID and networkName
const ValidInputString = "^[a-zA-Z0-9_-]+$"

// ValidateContainerID will validate that the supplied containerID does not contain invalid characters
func ValidateContainerID(containerID string) *types.Error {

	reg := regexp.MustCompile(ValidInputString)
	if containerID == "" {
		return types.NewError(types.ErrUnknownContainer, "missing containerID", "")
	}
	if !reg.MatchString(containerID) {
		return types.NewError(types.ErrDecodingFailure, fmt.Sprintf("error: invalid characters in containerID: %v", containerID), "")
	}
	return nil
}

// ValidateNetworkName will validate that the supplied networkName does not contain invalid characters
func ValidateNetworkName(networkName string) *types.Error {

	reg := regexp.MustCompile(ValidInputString)
	if networkName != "" {
		if !reg.MatchString(networkName) {
			return types.NewError(types.ErrInvalidNetworkConfig, "invalid characters found in network name", "")
		}
	}
	return nil
}
