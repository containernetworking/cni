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

import "fmt"

// ErrorIncompatible represents a rich plugin incompatibility error.
type ErrorIncompatible struct {
	Config    string
	Supported []string
}

// Details prints the erroneous configuration and the supported configuration of the plugin in use.
func (e *ErrorIncompatible) Details() string {
	return fmt.Sprintf("config is %q, plugin supports %q", e.Config, e.Supported)
}

// Error prints a rich plugin incompatibility error message.
func (e *ErrorIncompatible) Error() string {
	return fmt.Sprintf("incompatible CNI versions: %s", e.Details())
}

// Reconciler represents a plug-in configuration validator.
type Reconciler struct{}

// Check checks if a supported configuration version is in place.
func (r *Reconciler) Check(configVersion string, pluginInfo PluginInfo) *ErrorIncompatible {
	return r.CheckRaw(configVersion, pluginInfo.SupportedVersions())
}

// CheckRaw checks if a supported configuration version is in place.
func (*Reconciler) CheckRaw(configVersion string, supportedVersions []string) *ErrorIncompatible {
	for _, supportedVersion := range supportedVersions {
		if configVersion == supportedVersion {
			return nil
		}
	}

	return &ErrorIncompatible{
		Config:    configVersion,
		Supported: supportedVersions,
	}
}
