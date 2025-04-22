// Copyright 2015 CNI authors
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

package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check <network-name> <netns>",
	Short: "Check network interface in a network namespace",
	Long: `Check network interface in a network namespace.
This command will check if the network interface is properly configured in the specified network namespace.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		netconf, rt, err := setupRuntimeConfig(cmd, args)
		if err != nil {
			return err
		}

		cninet := getCNIConfig()
		return cninet.CheckNetworkList(context.TODO(), netconf, rt)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
