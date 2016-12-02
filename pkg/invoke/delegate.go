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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
)

func GetNextPlugin(netconf []byte) (string, []byte, error) {
	conf := struct {
		types.NetConf
		Inner map[string]interface{} `json:"inner,omitempty"`
	}{}

	err := json.Unmarshal(netconf, &conf)
	if err != nil {
		return "", nil, fmt.Errorf("error decoding net config: %v", err)
	}

	if conf.Inner != nil {
		tmp, ok := conf.Inner["type"]
		if !ok {
			return "", nil, fmt.Errorf("inner plugin type not specified")
		}
		innerType, ok := tmp.(string)
		if !ok {
			return "", nil, fmt.Errorf("inner plugin type not a string")
		}

		// Copy network name and CNI version to inner plugin
		conf.Inner["name"] = conf.Name
		conf.Inner["cniVersion"] = conf.CNIVersion
		innerConf, err := json.MarshalIndent(conf.Inner, "", "    ")
		if err != nil {
			return "", nil, fmt.Errorf("error marshalling inner plugin config: %v", err)
		}
		return innerType, innerConf, nil
	} else if conf.IPAM.Type != "" {
		// IPAM plugin gets run with normal netconf
		return conf.IPAM.Type, netconf, nil
	}

	// No delegate to run
	return "", nil, nil
}

func DelegateAdd(delegatePlugin string, netconf []byte) (*types.Result, error) {
	if os.Getenv("CNI_COMMAND") != "ADD" {
		return nil, fmt.Errorf("CNI_COMMAND is not ADD")
	}

	paths := strings.Split(os.Getenv("CNI_PATH"), ":")

	pluginPath, err := FindInPath(delegatePlugin, paths)
	if err != nil {
		return nil, err
	}

	return ExecPluginWithResult(pluginPath, netconf, ArgsFromEnv())
}

func DelegateDel(delegatePlugin string, netconf []byte) error {
	if os.Getenv("CNI_COMMAND") != "DEL" {
		return fmt.Errorf("CNI_COMMAND is not DEL")
	}

	paths := strings.Split(os.Getenv("CNI_PATH"), ":")

	pluginPath, err := FindInPath(delegatePlugin, paths)
	if err != nil {
		return err
	}

	return ExecPluginWithoutResult(pluginPath, netconf, ArgsFromEnv())
}
